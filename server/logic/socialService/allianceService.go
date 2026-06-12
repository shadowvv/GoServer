package socialService

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var errAllianceNameAlreadyExists = errors.New("alliance name already exists")
var errAllianceNotPermission = errors.New("alliance not permission")
var errAllianceConditionNotMet = errors.New("alliance condition not met")
var errAllianceAlreadyInAlliance = errors.New("alliance already in alliance")

var service *AllianceService
var serviceInitOnce sync.Once

type AllianceService struct {
	idGenerator     *tool.IdGenerator
	mailIdGenerator *tool.IdGenerator
	manager         *AllianceManager
}

func InitService() {
	serviceInitOnce.Do(func() {
		s := &AllianceService{
			idGenerator:     tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_ALLIANCE)),
			mailIdGenerator: tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_RANK_SOCIAL_MAIL)),
			manager:         NewAllianceManager(enum.AllianceUpdateFlushInterval),
		}
		s.manager.LoadAlliances()
		service = s
	})
}

func GetService() *AllianceService {
	InitService()
	return service
}

func (s *AllianceService) CreateAlliance(req *rpcPb.CreateAllianceReq) *rpcPb.CreateAllianceResp {
	name := req.Name
	announce := req.Announce

	var alliance *model.AllianceEntity
	var leaderMember *model.AllianceMemberEntity
	err := easyDB.GetPlayerDB().Transaction(func(tx *gorm.DB) error {
		exists, err := s.manager.nameExists(req.ServerId, name, 0)
		if err != nil {
			return err
		}
		if exists {
			return errAllianceNameAlreadyExists
		}

		now := tool.UnixNowMilli()
		alliance = &model.AllianceEntity{
			AllianceId:          s.idGenerator.NextId(),
			ServerId:            req.ServerId,
			Name:                name,
			Announce:            announce,
			BadgeId:             req.BadgeId,
			Notice:              "",
			Level:               1,
			ApplyType:           0,
			PowerApplyCondition: 0,
			CityLevelCondition:  0,
			CreateTime:          now,
			MemberNum:           1,
		}
		if err := tx.Create(alliance).Error; err != nil {
			return err
		}

		leaderMember = &model.AllianceMemberEntity{
			AllianceId: alliance.AllianceId,
			UserId:     req.UserId,
			Role:       int32(pb.ALLIANCE_POSITION_ALLIANCE_LEADER),
			JoinTime:   now,
		}
		if err := tx.Create(leaderMember).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return &rpcPb.CreateAllianceResp{
			ErrorCode: int32(mapErrCode(err)),
		}
	}

	playerBattleInfo := logicCommon.GetPlayerBattleInfoFromRedis(req.UserId)
	if playerBattleInfo != nil {
		alliance.AllianceTotalPower += playerBattleInfo.GetMainFormationPower()
	}
	s.manager.attachCreatedAlliance(alliance, []*model.AllianceMemberEntity{leaderMember})

	roundBestWin := logicCommon.GetOtherPlayerGloryArenaRoundBestWin(req.UserId)
	logicCommon.UpdatePlayerAllianceInfo(&logicCommon.PlayerAllianceInfo{
		ArenaJoined:  false,
		RoundBestWin: roundBestWin,
		UserId:       req.UserId,
		AllianceId:   alliance.AllianceId,
		AllianceName: alliance.Name,
		JoinTime:     leaderMember.JoinTime,
	})
	if roundBestWin > 0 {
		notifyAllianceGloryArenaRoundRankDelta(req.UserId, alliance.ServerId, alliance.AllianceId, int64(roundBestWin))
	}
	return &rpcPb.CreateAllianceResp{
		Alliance:  allianceToPb(alliance, req.UserId),
		ErrorCode: int32(pb.ERROR_CODE_SUCCESS),
	}
}

func (s *AllianceService) SendAllianceMail(mailTemplateId int32, playerId int64, allianceName string) {
	cfg := gameConfig.GetMailContentCfg(mailTemplateId)
	if cfg == nil {
		logger.ErrorBySprintf("mail template not found: %v,playerId:%d", mailTemplateId, playerId)
		return
	}
	expireTime := int64(0)
	if cfg.MailExpTime > 0 {
		expireTime = tool.UnixNow() + int64(cfg.MailExpTime)*3600
	}
	mailID := s.mailIdGenerator.NextId()

	contentParam := make([]string, 0)
	contentParam = append(contentParam, allianceName)

	playerMail := &mail.Mail{
		MailID:        mailID,
		UserID:        playerId,
		MailType:      cfg.MailType,
		Title:         strconv.FormatInt(int64(cfg.MailTitle), 10),
		Content:       strconv.FormatInt(int64(cfg.MailWords), 10),
		ContentParams: contentParam,
		SenderID:      0,
		SenderName:    strconv.FormatInt(int64(cfg.SendName), 10),
		TemplateID:    cfg.ID,
		Status:        mail.MailStatusUnread,
		IsConvenient:  cfg.IsConvenient,
		ExpireTime:    expireTime,
		SendTime:      tool.UnixNow(),
	}

	entity := mail.MailToEntity(playerMail)
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		logger.ErrorBySprintf("create player mail error: %v,entity:%v", err, entity)
		return
	}

	// 写入 Redis 通知游戏服刷新该玩家邮件缓存
	if dbService.RDB != nil {
		_ = dbService.RDB.SAdd(context.Background(), enum.REDIS_MAIL_REFRESH_USERS, strconv.FormatInt(playerId, 10)).Err()
	}
}

func (s *AllianceService) GetAllianceById(allianceId int64) *AllianceModel {
	if s == nil || s.manager == nil || allianceId <= 0 {
		return nil
	}
	return s.manager.GetAllianceById(allianceId)
}

func (s *AllianceService) FlushDirtyByProcessor(processorID int32, processorNum int) {
	if s == nil || s.manager == nil {
		return
	}
	s.manager.FlushDirtyByProcessor(processorID, processorNum)
}

func (s *AllianceService) HeartbeatByProcessor(processorID int32, processorNum int) {
	if s == nil || s.manager == nil {
		return
	}
	s.manager.HeartbeatByProcessor(processorID, processorNum)
}

func allianceToPb(entity *model.AllianceEntity, leaderUserId int64) *rpcPb.AllianceInfo {
	if entity == nil {
		return nil
	}
	cfg := gameConfig.GetAllianceLevelCfg(entity.Level)
	if cfg == nil {
		return nil
	}
	return &rpcPb.AllianceInfo{
		AllianceId:          entity.AllianceId,
		ServerId:            entity.ServerId,
		Name:                entity.Name,
		LeaderUserId:        leaderUserId,
		Notice:              entity.Notice,
		MemberCount:         entity.MemberNum,
		MaxMember:           cfg.Num,
		CreateTime:          entity.CreateTime,
		Announce:            entity.Announce,
		BadgeId:             entity.BadgeId,
		Level:               entity.Level,
		ApplyType:           entity.ApplyType,
		PowerApplyCondition: entity.PowerApplyCondition,
		CityLevel:           entity.CityLevelCondition,
		AllianceTotalPower:  entity.AllianceTotalPower,
	}
}

func memberToPb(entity *model.AllianceMemberEntity, info *logicCommon.PlayerRedisInfo) *rpcPb.AllianceMember {
	if entity == nil {
		return nil
	}
	memberPb := &rpcPb.AllianceMember{
		UserId: entity.UserId,
		Role:   entity.Role,
	}
	if info != nil {
		memberPb.NickName = info.BasicInfo.Name
		memberPb.Head = info.BasicInfo.HeadId
		memberPb.CityLevel = info.BasicInfo.MainCityLevel
		memberPb.Contribute = 0 // TODO 联盟贡献
		memberPb.OfflineTime = info.BasicInfo.LastOfflineTime
		memberPb.HeadFrame = info.BasicInfo.FrameId
		if info.BattleInfo != nil && info.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] != nil {
			memberPb.Power = info.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)].BattlePower
		}
	}
	return memberPb
}

func mapErrCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if errors.Is(err, errAllianceNameAlreadyExists) {
		return pb.ERROR_CODE_ALLIANCE_NAME_ALREADY_EXISTS
	}
	if errors.Is(err, errAllianceNotPermission) {
		return pb.ERROR_CODE_ALLIANCE_NOT_PERMISSION
	}
	if errors.Is(err, errAllianceAlreadyInAlliance) {
		return pb.ERROR_CODE_ALLIANCE_ALREADY_IN_ALLIANCE
	}
	if errors.Is(err, errAllianceConditionNotMet) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if errors.Is(err, gorm.ErrInvalidData) {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	logger.ErrorBySprintf("[socialService] db error:%v", err)
	return pb.ERROR_CODE_SYSTEM_ERROR
}

func notifyAllianceArenaRankDelta(playerId int64, serverID int32, allianceId int64, deltaScore int64) {
	if playerId <= 0 || serverID <= 0 || allianceId <= 0 || deltaScore == 0 {
		return
	}
	// TODO:之后优化为opsState := logicCommon.LoadGloryArenaOpsStateByServerID(serverID)类似这个样子，从redis读取
	now := tool.UnixNowMilli()
	arenaVersion := logicCommon.GetArenaRankVersionByTime(serverID, now)
	if basicInfo := logicCommon.GetPlayerBasicInfoFromRedis(playerId); basicInfo != nil && basicInfo.ServerId == serverID && basicInfo.ArenaVersion != "" {
		arenaVersion = basicInfo.ArenaVersion
	}
	rankIds := logicCommon.GetCommonRankUniqueIdsByPointType(
		serverID,
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA,
		arenaVersion,
	)
	if len(rankIds) == 0 {
		return
	}
	req := &rpcPb.NotifyUpdateRankInfo{
		AllianceId:        allianceId,
		Score:             deltaScore,
		IncrementalUpdate: true,
	}
	for _, rankId := range rankIds {
		_ = rpcController.SendMessageToRankBoard(playerId, rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, req)
	}
}

func notifyAllianceGloryArenaRoundRankDelta(playerId int64, serverID int32, allianceID int64, deltaScore int64) {
	if playerId <= 0 || serverID <= 0 || allianceID <= 0 || deltaScore == 0 {
		return
	}
	opsState := logicCommon.LoadGloryArenaOpsStateByServerID(serverID)
	if opsState == nil {
		return
	}
	rankIDs := logicCommon.GetCommonRankUniqueIdsByPointType(
		serverID,
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT,
		opsState.GroupVersion,
	)
	if len(rankIDs) == 0 {
		return
	}
	req := &rpcPb.NotifyUpdateRankInfo{
		AllianceId:        allianceID,
		Score:             deltaScore,
		IncrementalUpdate: true,
	}
	for _, rankID := range rankIDs {
		_ = rpcController.SendMessageToRankBoard(playerId, rankID, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, req)
	}
}

func syncAllianceRankFinalScores(playerID int64, serverID int32, allianceID int64, arenaScore int64, gloryRoundBestWin int64) {
	if serverID <= 0 || allianceID <= 0 {
		return
	}
	if playerID <= 0 {
		playerID = allianceID
	}
	// TODO:之后优化为opsState := logicCommon.LoadGloryArenaOpsStateByServerID(serverID)类似这个样子，从redis读取
	now := tool.UnixNowMilli()
	arenaVersion := logicCommon.GetArenaRankVersionByTime(serverID, now)
	if basicInfo := logicCommon.GetPlayerBasicInfoFromRedis(playerID); basicInfo != nil && basicInfo.ServerId == serverID && basicInfo.ArenaVersion != "" {
		arenaVersion = basicInfo.ArenaVersion
	}
	arenaRankIDs := logicCommon.GetCommonRankUniqueIdsByPointType(
		serverID,
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA,
		arenaVersion,
	)
	if len(arenaRankIDs) > 0 {
		arenaReq := &rpcPb.NotifyUpdateRankInfo{
			AllianceId:        allianceID,
			Score:             arenaScore,
			IncrementalUpdate: false,
		}
		for _, rankID := range arenaRankIDs {
			_ = rpcController.SendMessageToRankBoard(playerID, rankID, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, arenaReq)
		}
	}

	opsState := logicCommon.LoadGloryArenaOpsStateByServerID(serverID)
	if opsState == nil {
		return
	}
	gloryRankIDs := logicCommon.GetCommonRankUniqueIdsByPointType(
		serverID,
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT,
		opsState.GroupVersion,
	)
	if len(gloryRankIDs) > 0 {
		gloryReq := &rpcPb.NotifyUpdateRankInfo{
			AllianceId:        allianceID,
			Score:             gloryRoundBestWin,
			IncrementalUpdate: false,
		}
		for _, rankID := range gloryRankIDs {
			_ = rpcController.SendMessageToRankBoard(playerID, rankID, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, gloryReq)
		}
	}
}

func getAllianceApplyTimeFromRedis(allianceID, userID int64) (bool, error) {
	if allianceID <= 0 || userID <= 0 || dbService.RDB == nil {
		return false, errors.New("invalid alliance apply redis args")
	}
	now := tool.UnixNowMilli()
	key := enum.GetAllianceApplyListKey(allianceID)
	ctx := context.Background()
	expireBefore := now - enum.AllianceApplyExpireDurationMillis
	if err := dbService.RDB.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(expireBefore, 10)).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] cleanup alliance apply expired failed allianceId:%d err:%v", allianceID, err)
		return false, err
	}

	score, err := dbService.RDB.ZScore(ctx, key, strconv.FormatInt(userID, 10)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		logger.ErrorBySprintf("[allianceManager] get alliance apply score redis failed allianceId:%d userId:%d err:%v", allianceID, userID, err)
		return false, err
	}
	if int64(score) <= expireBefore {
		removeAllianceApplyFromRedis(allianceID, userID)
		return false, nil
	}
	return true, nil
}

func removeAllianceApplyFromRedis(allianceID, userID int64) {
	if allianceID <= 0 || userID <= 0 || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	key := enum.GetAllianceApplyListKey(allianceID)
	if err := dbService.RDB.ZRem(ctx, key, strconv.FormatInt(userID, 10)).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] remove alliance apply redis failed allianceId:%d userId:%d err:%v", allianceID, userID, err)
	}
}

func syncAllianceBasicToRedis(alliance *model.AllianceEntity) {
	if alliance == nil || dbService.RDB == nil {
		return
	}
	data, err := json.Marshal(alliance)
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] marshal alliance basic failed allianceId:%d err:%v", alliance.AllianceId, err)
		return
	}
	ctx := context.Background()
	infoKey := enum.GetAllianceBasicInfoKey(alliance.AllianceId)
	serverKey := enum.GetServerAllianceSetKey(alliance.ServerId)
	pip := dbService.RDB.Pipeline()
	pip.Set(ctx, infoKey, string(data), 0)
	pip.ZAdd(ctx, serverKey, &redis.Z{
		Score:  float64(alliance.AllianceTotalPower),
		Member: strconv.FormatInt(alliance.AllianceId, 10),
	})
	if alliance.Name != "" {
		pip.HSet(ctx, enum.GetAllianceNameIndexKey(alliance.ServerId), alliance.Name, strconv.FormatInt(alliance.AllianceId, 10))
	}
	if _, err = pip.Exec(ctx); err != nil {
		logger.ErrorBySprintf("[allianceManager] sync alliance basic to redis failed allianceId:%d err:%v", alliance.AllianceId, err)
	}
}

func syncMemberAllianceInfoToRedis(alliance *model.AllianceEntity, members map[int64]*model.AllianceMemberEntity) {
	if alliance == nil || dbService.RDB == nil || len(members) == 0 {
		return
	}

	ctx := context.Background()

	_, err := dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, member := range members {
			if member == nil {
				continue
			}

			allianceKey := enum.GetPlayerAllianceInfoKey(member.UserId)
			oldInfo := logicCommon.GetPlayerAllianceInfoFromRedis(member.UserId)
			arenaJoined := false
			roundBestWin := int32(0)
			if oldInfo != nil {
				arenaJoined = oldInfo.ArenaJoined
				roundBestWin = oldInfo.RoundBestWin
			}
			allianceInfo := &logicCommon.PlayerAllianceInfo{
				ArenaJoined:  arenaJoined,
				RoundBestWin: roundBestWin,
				UserId:       member.UserId,
				AllianceId:   member.AllianceId,
				AllianceName: alliance.Name,
				JoinTime:     member.JoinTime,
			}

			data, err := json.Marshal(allianceInfo)
			if err != nil {
				logger.ErrorBySprintf(
					"[allianceManager] marshal alliance info failed userId:%d allianceId:%d err:%v",
					member.UserId, member.AllianceId, err,
				)
				continue
			}

			pipe.Set(ctx, allianceKey, data, 0)
		}
		return nil
	})

	if err != nil {
		logger.ErrorBySprintf(
			"[allianceManager] sync alliance info to redis failed allianceId:%d err:%v",
			alliance.AllianceId, err,
		)
	}

	syncAllianceMemberSetToRedis(alliance.AllianceId, members)
}

func syncAllianceMemberSetToRedis(allianceID int64, members map[int64]*model.AllianceMemberEntity) {
	if allianceID <= 0 || dbService.RDB == nil {
		return
	}

	key := enum.GetAllianceMemberInfoKey(allianceID)
	ctx := context.Background()
	memberIDs := make([]interface{}, 0, len(members))
	for _, member := range members {
		if member == nil || member.UserId <= 0 {
			continue
		}
		memberIDs = append(memberIDs, strconv.FormatInt(member.UserId, 10))
	}

	pip := dbService.RDB.Pipeline()
	pip.Del(ctx, key)
	if len(memberIDs) > 0 {
		pip.SAdd(ctx, key, memberIDs...)
	}
	if _, err := pip.Exec(ctx); err != nil {
		logger.ErrorBySprintf("[allianceManager] sync alliance member set failed allianceId:%d err:%v", allianceID, err)
	}
}

func rebuildAlliancesBasicToRedis(alliances []*model.AllianceEntity) {
	if len(alliances) == 0 || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	serverKeys := make(map[string]struct{})
	nameIndexKeys := make(map[string]struct{})
	for _, alliance := range alliances {
		if alliance == nil {
			continue
		}
		serverKeys[enum.GetServerAllianceSetKey(alliance.ServerId)] = struct{}{}
		nameIndexKeys[enum.GetAllianceNameIndexKey(alliance.ServerId)] = struct{}{}
	}
	pip := dbService.RDB.Pipeline()
	for key := range serverKeys {
		pip.Del(ctx, key)
	}
	for key := range nameIndexKeys {
		pip.Del(ctx, key)
	}
	for _, alliance := range alliances {
		if alliance == nil {
			continue
		}
		data, err := json.Marshal(alliance)
		if err != nil {
			logger.ErrorBySprintf("[allianceManager] marshal alliance basic failed allianceId:%d err:%v", alliance.AllianceId, err)
			continue
		}
		pip.Set(ctx, enum.GetAllianceBasicInfoKey(alliance.AllianceId), string(data), 0)
		pip.ZAdd(ctx, enum.GetServerAllianceSetKey(alliance.ServerId), &redis.Z{
			Score:  float64(alliance.AllianceTotalPower),
			Member: strconv.FormatInt(alliance.AllianceId, 10),
		})
		if alliance.Name != "" {
			pip.HSet(ctx, enum.GetAllianceNameIndexKey(alliance.ServerId), alliance.Name, strconv.FormatInt(alliance.AllianceId, 10))
		}
	}
	if _, err := pip.Exec(ctx); err != nil {
		logger.ErrorBySprintf("[allianceManager] rebuild alliance basic redis failed err:%v", err)
	}
}

func rebuildAllianceMemberInfoToRedis(actors map[int64]*AllianceModel) {
	if dbService.RDB == nil {
		return
	}
	clearAllPlayerAllianceInfo()

	ctx := context.Background()
	for _, actor := range actors {
		memberInfoMap := make(map[int64]*logicCommon.PlayerAllianceInfo)
		if actor == nil || actor.alliance == nil {
			continue
		}
		for _, member := range actor.members {
			if member == nil {
				continue
			}
			memberInfoMap[member.UserId] = &logicCommon.PlayerAllianceInfo{
				ArenaJoined:  false,
				RoundBestWin: 0,
				UserId:       member.UserId,
				AllianceId:   member.AllianceId,
				AllianceName: actor.alliance.Name,
				JoinTime:     member.JoinTime,
			}
		}
		if len(memberInfoMap) == 0 {
			clearAllianceMemberSetFromRedis(actor.alliance.AllianceId)
			continue
		}
		_, err := dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for userId, info := range memberInfoMap {
				data, marshalErr := json.Marshal(info)
				if marshalErr != nil {
					logger.ErrorBySprintf("[allianceManager] marshal alliance info failed userId:%d allianceId:%d err:%v",
						userId, info.AllianceId, marshalErr)
					continue
				}

				pipe.Set(ctx, enum.GetPlayerAllianceInfoKey(userId), data, 0)
			}
			return nil
		})
		if err != nil {
			logger.ErrorBySprintf("[allianceManager] rebuild alliance member info failed err:%v", err)
			return
		}
		syncAllianceMemberSetToRedis(actor.alliance.AllianceId, actor.members)
	}
}

func clearAllPlayerAllianceInfo() {
	if dbService.RDB == nil {
		return
	}

	ctx := context.Background()
	var cursor uint64
	pattern := enum.REDIS_PLAYER_ALLIANCE_INFO + "*"
	count := int64(200) // 每次扫描数量（可调）

	for {
		keys, nextCursor, err := dbService.RDB.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			logger.ErrorBySprintf("[redis] scan alliance keys failed err:%v", err)
			return
		}

		if len(keys) > 0 {
			_, err = dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				for _, key := range keys {
					pipe.Del(ctx, key)
				}
				return nil
			})
			if err != nil {
				logger.ErrorBySprintf("[redis] delete alliance keys failed err:%v", err)
				return
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func removeAllianceBasicFromRedis(allianceID int64, serverID int32) {
	if allianceID <= 0 || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	pip := dbService.RDB.Pipeline()
	pip.Del(ctx, enum.GetAllianceBasicInfoKey(allianceID))
	if serverID > 0 {
		pip.ZRem(ctx, enum.GetServerAllianceSetKey(serverID), strconv.FormatInt(allianceID, 10))
	}
	if _, err := pip.Exec(ctx); err != nil {
		logger.ErrorBySprintf("[allianceManager] remove alliance basic redis failed allianceId:%d serverId:%d err:%v", allianceID, serverID, err)
	}
}

func removeAllianceApplyListFromRedis(allianceID int64) {
	if allianceID <= 0 || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	if err := dbService.RDB.Del(ctx, enum.GetAllianceApplyListKey(allianceID)).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] remove alliance apply list redis failed allianceId:%d err:%v", allianceID, err)
	}
}

func removeAllianceNameIndexFromRedis(serverID int32, name string) {
	if serverID <= 0 || name == "" || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	key := enum.GetAllianceNameIndexKey(serverID)
	if err := dbService.RDB.HDel(ctx, key, name).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] remove alliance name index failed serverId:%d name:%s err:%v", serverID, name, err)
	}
}

func removeAllianceMemberInfoFromRedis(alliance *AllianceModel) {
	if alliance == nil || alliance.alliance == nil {
		return
	}
	allianceID := alliance.alliance.AllianceId
	if dbService.RDB == nil {
		return
	}
	defer clearAllianceMemberSetFromRedis(allianceID)

	ctx := context.Background()
	memberInfoMap := make(map[int64]*logicCommon.PlayerAllianceInfo)
	for _, member := range alliance.members {
		if member == nil {
			continue
		}
		memberInfoMap[member.UserId] = &logicCommon.PlayerAllianceInfo{
			ArenaJoined:  false,
			RoundBestWin: 0,
			UserId:       member.UserId,
			AllianceId:   0,
			AllianceName: "",
			JoinTime:     0,
		}
	}
	if len(memberInfoMap) == 0 {
		return
	}
	_, err := dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for userId, info := range memberInfoMap {
			data, marshalErr := json.Marshal(info)
			if marshalErr != nil {
				logger.ErrorBySprintf("[allianceManager] marshal alliance info failed userId:%d allianceId:%d err:%v",
					userId, info.AllianceId, marshalErr)
				continue
			}

			pipe.Set(ctx, enum.GetPlayerAllianceInfoKey(userId), data, 0)
		}
		return nil
	})
	if err != nil {
		logger.ErrorBySprintf("[allianceManager] rebuild alliance member info failed err:%v", err)
		return
	}
}

func clearAllianceMemberSetFromRedis(allianceID int64) {
	if allianceID <= 0 || dbService.RDB == nil {
		return
	}
	ctx := context.Background()
	if err := dbService.RDB.Del(ctx, enum.GetAllianceMemberInfoKey(allianceID)).Err(); err != nil {
		logger.ErrorBySprintf("[allianceManager] remove alliance member set redis failed allianceId:%d err:%v", allianceID, err)
	}
}
