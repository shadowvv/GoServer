package gameController

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gloryArenaService"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/raid"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("honorArena", &GloryArenaController{})
}

type GloryArenaController struct{}

var _ LogicControllerInterface = (*GloryArenaController)(nil)

func (p *GloryArenaController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_REQ, &pb.GetHonorArenaInfoReq{}, GetHonorArenaInfo, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HONOR_ARENA_START_REQ, &pb.HonorArenaStartReq{}, HonorArenaStart, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_REQ, &pb.HonorArenaGetBoxRewardReq{}, HonorArenaGetBoxReward, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_REQ, &pb.HonorArenaChallengeListReq{}, HonorArenaChallengeList, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_REQ, &pb.StartHonorArenaBattleReq{}, StartHonorArenaBattle, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_REQ, &pb.HonorArenaSelectHeroReq{}, HonorArenaSelectHero, enum.FUNCTION_ID_GLORY_ARENA)
	RegisterRpcPlayerMessageHandler(enum.MSG_TYPE_PLAYER, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_MY_ARENA_RANK_RESP, &rpcPb.GetMyRankResp{}, BackHonorArenaInfoArenaRankFromRankBoardNode)
}

func GetHonorArenaInfo(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetHonorArenaInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	player.PlayerGloryArenaModel.ForceSyncRoundState(tool.UnixNowMilli())
	round := player.PlayerGloryArenaModel.Entity.RoundId
	season := player.PlayerGloryArenaModel.GetSeasonType()
	startTime, endTime := player.PlayerGloryArenaModel.GetCurrentRoundTimeWindow()
	if player.PlayerGloryArenaModel.IsEnrolled() {
		sendHonorArenaInfoResp(player, 0, round, season, startTime, endTime, 1)
		return
	}
	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc == nil {
		return
	}
	opsState, err := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
	if err != nil || opsState == nil || !opsState.IsRoundOpen {
		// 轮次未开启：显示上一轮最大胜场，不取竞技场排名。
		sendHonorArenaInfoResp(player, 0, round, season, startTime, endTime, 0)
		return
	}
	qualified, err := isQualifiedByQualifyPool(player.GetUserId(), opsState.GroupVersion)
	if err != nil {
		logger.ErrorBySprintf("[honorArenaController] query qualify pool failed while get info userId:%d serverId:%d groupVersion:%s roundStart:%d err:%v", player.GetUserId(), player.GetUserServerId(), opsState.GroupVersion, opsState.RoundStart, err)
		return
	}
	if qualified {
		// 有资格：不需要取竞技场排名，直接返回0。
		sendHonorArenaInfoResp(player, 0, round, season, startTime, endTime, 0)
		return
	}
	if err = requestArenaRankForHonorArenaInfo(player); err != nil {
		logger.ErrorBySprintf("[honorArenaController] request arena rank failed userId:%d serverId:%d err:%v", player.GetUserId(), player.GetUserServerId(), err)
		return
	}
}

func requestArenaRankForHonorArenaInfo(player *model.PlayerModel) error {
	if player == nil {
		return errors.New("player is nil")
	}
	version := ""
	if player.PlayerArenaModel != nil {
		version = player.PlayerArenaModel.GetVersion()
	}
	if version == "" {
		version = logicCommon.GetArenaRankVersionByTime(player.GetUserServerId(), tool.UnixNowMilli())
	}
	rankID, err := logicCommon.GetRankUniqueId(gameConfig.GetArenaRankId(), 0, 0, player.GetUserServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		return err
	}
	return rpcController.SendMessageToRankBoardWithRespMsgId(
		player.GetUserId(),
		rankID,
		int32(pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP),
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_MY_RANK_REQ,
		rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_GET_MY_ARENA_RANK_RESP,
		&rpcPb.GetMyRankReq{},
	)
}

func BackHonorArenaInfoArenaRankFromRankBoardNode(message proto.Message, player *model.PlayerModel) {
	resp, ok := message.(*rpcPb.GetMyRankResp)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	round := player.PlayerGloryArenaModel.Entity.RoundId
	season := player.PlayerGloryArenaModel.GetSeasonType()
	startTime, endTime := player.PlayerGloryArenaModel.GetCurrentRoundTimeWindow()
	rank := resp.Rank.Rank
	if rank == 0 {
		rank = -1
	}
	sendHonorArenaInfoResp(player, rank, round, season, startTime, endTime, 0)
}

func sendHonorArenaInfoResp(player *model.PlayerModel, arenaRank int32, round int32, season int32, startTime int64, endTime int64, isCompete int32) {
	gotBoxCount := player.PlayerGloryArenaModel.GetRoundGotBoxCount()
	resp := &pb.GetHonorArenaInfoResp{
		StartTime:   startTime,
		ArenaRank:   arenaRank,
		WinCount:    player.PlayerGloryArenaModel.GetWinCount(),
		HpCount:     player.PlayerGloryArenaModel.GetLife(),
		IsCompete:   isCompete,
		MaxWinCount: player.PlayerGloryArenaModel.GetRoundBestWinCount(),
		GotBoxCount: gotBoxCount,
		Round:       round,
		EndTime:     endTime,
		Season:      season,
		Hero:        player.PlayerGloryArenaModel.BuildSelectedHeroes(),
		HeroSelect:  player.PlayerGloryArenaModel.BuildSelectableHeroes(),
		IsFree:      player.PlayerGloryArenaModel.CanFreeCompete(),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_HONOR_ARENA_INFO_RESP, resp)
	player.PlayerGloryArenaModel.EnterCount = player.PlayerGloryArenaModel.EnterCount + 1
}

func HonorArenaSelectHero(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.HonorArenaSelectHeroReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil || player.HeroDetailsModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	selectedHero := player.PlayerGloryArenaModel.FindSelectableHero(req.GetHeroId())
	if selectedHero == nil || selectedHero.Id <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	if err := player.PlayerGloryArenaModel.UpsertSelectedHero(selectedHero); err != nil {
		logger.ErrorBySprintf("[honorArenaController] cache selected hero failed userId:%d heroCid:%d err:%v", player.GetUserId(), selectedHero.Id, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	selectedHeroResp := &pb.HonorArenaHeroInfo{
		HeroId: req.GetHeroId(),
		Cid:    int32(selectedHero.Id),
		Level:  selectedHero.Level,
		Star:   selectedHero.Star,
		Job:    selectedHero.ClassId,
		Fight:  selectedHero.Attr[enum.AttributeBasicCombatPower],
	}
	player.PlayerGloryArenaModel.ClearDefeatedOpponentCache()
	messageSender.SendMessage(player, pb.MESSAGE_ID_HONOR_ARENA_SELECT_HERO_RESP, &pb.HonorArenaSelectHeroResp{
		Hero: selectedHeroResp,
	})
}

func HonorArenaStart(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.HonorArenaStartReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	player.PlayerGloryArenaModel.ForceSyncRoundState(tool.UnixNowMilli())

	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	opsState, err := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
	if err != nil || opsState == nil || !opsState.IsRoundOpen {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_OPEN)
		return
	}

	ok, err = isQualifiedByQualifyPool(player.GetUserId(), opsState.GroupVersion)
	if err != nil {
		logger.ErrorBySprintf("[honorArenaController] query qualify pool failed userId:%d serverId:%d groupVersion:%s roundStart:%d err:%v", player.GetUserId(), player.GetUserServerId(), opsState.GroupVersion, opsState.RoundStart, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_QUALIFIED)
		return
	}

	if player.PlayerGloryArenaModel.IsEnrolled() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_GLORY_ARENA_ALREADY_START)
		return
	}
	if !isGloryArenaChallengeTimeOpen(tool.UnixNowMilli()) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_GLORY_ARENA_WRONG_TIME)
		return
	}

	if player.PlayerGloryArenaModel.CanFreeCompete() == 0 {
		instanceCfg := gameConfig.GetInstanceCfg(int32(enum.GLORY_ARENA_INSTANCE_ID))
		if instanceCfg == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		if len(instanceCfg.TicketID) > 0 {
			check, checkErr := itemService.CheckItemsCount(player, instanceCfg.TicketID)
			if !check || checkErr != nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
				return
			}
			if removeErr := itemService.RemoveItems(player, instanceCfg.TicketID, enum.ITEM_CHANGE_REASON_CHALLENGE_INSTANCE); removeErr != nil {
				logger.ErrorBySprintf("[honorArenaController] remove glory arena ticket failed userId:%d err:%v", player.GetUserId(), removeErr)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
		}
	}

	player.PlayerGloryArenaModel.SetEnrollStatus(true)
	player.PlayerGloryArenaModel.AddEnrollCount(1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_HONOR_ARENA_START_RESP, &pb.HonorArenaStartResp{})
}

func isQualifiedByQualifyPool(userID int64, groupVersion string) (bool, error) {
	if userID <= 0 || groupVersion == "" {
		return false, nil
	}
	key := enum.GetGloryArenaPoolQualifyRoundKey(groupVersion)
	member := strconv.FormatInt(userID, 10)
	return dbService.RDB.SIsMember(context.Background(), key, member).Result()
}

func HonorArenaGetBoxReward(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.HonorArenaGetBoxRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	gotBoxCount := player.PlayerGloryArenaModel.GetRoundGotBoxCount()
	eligibleCount := calcHonorArenaEligibleBoxCount(player.PlayerGloryArenaModel.GetRoundBestWinCount())
	if gotBoxCount >= eligibleCount {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, pb.ERROR_CODE_GLORY_ARENA_REWARD_ALREADY_RECEIVED)
		return
	}

	rewardItems, err := collectHonorArenaBoxRewardItems(gotBoxCount, eligibleCount)
	if err != nil {
		logger.ErrorBySprintf("[honorArenaController] collect box reward failed userId:%d got:%d eligible:%d err:%v", player.GetUserId(), gotBoxCount, eligibleCount, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if len(rewardItems) > 0 {
		if err = itemService.AddItems(player, rewardItems, enum.ITEM_CHANGE_REASON_GLORY_ARENA_OPEN_AWARD); err != nil {
			logger.ErrorBySprintf("[honorArenaController] send box reward failed userId:%d got:%d eligible:%d err:%v", player.GetUserId(), gotBoxCount, eligibleCount, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
	}
	player.PlayerGloryArenaModel.SetRoundGotBoxCount(eligibleCount)
	messageSender.SendMessage(player, pb.MESSAGE_ID_HONOR_ARENA_GET_BOX_REWARD_RESP, &pb.HonorArenaGetBoxRewardResp{
		GotBoxCount: player.PlayerGloryArenaModel.GetRoundGotBoxCount(),
	})
}

func collectHonorArenaBoxRewardItems(gotBoxCount int32, eligibleCount int32) ([]*gameConfig.ItemConfig, error) {
	configs := gameConfig.GetAllGloryArenaRewardCfg()
	if eligibleCount > int32(len(configs)) {
		return nil, fmt.Errorf("eligible count out of range eligible:%d max:%d", eligibleCount, len(configs))
	}

	items := make([]*gameConfig.ItemConfig, 0)
	for i := gotBoxCount; i < eligibleCount; i++ {
		cfg := configs[i]
		if cfg == nil || cfg.Drop <= 0 {
			return nil, fmt.Errorf("reward config invalid index:%d", i)
		}
		dropItems := gameConfig.Drop(cfg.Drop)
		if len(dropItems) == 0 {
			return nil, fmt.Errorf("reward drop empty cfgId:%d drop:%d", cfg.Id, cfg.Drop)
		}
		items = append(items, dropItems...)
	}
	return items, nil
}

func calcHonorArenaEligibleBoxCount(maxWin int32) int32 {
	count := int32(0)
	configs := gameConfig.GetAllGloryArenaRewardCfg()
	for _, cfg := range configs {
		if cfg.Id <= maxWin {
			count++
		}
	}
	return count
}

func HonorArenaChallengeList(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.HonorArenaChallengeListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if req.GetIsRefresh() != 0 && req.GetIsRefresh() != 1 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	player.PlayerGloryArenaModel.ForceSyncRoundState(tool.UnixNowMilli())
	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	opsState, err := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
	if err != nil || opsState == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_OPEN)
		return
	}

	if !isGloryArenaChallengeTimeOpen(tool.UnixNowMilli()) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_GLORY_ARENA_WRONG_TIME)
		return
	}
	if !player.PlayerGloryArenaModel.IsEnrolled() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_START)
		return
	}
	if player.PlayerGloryArenaModel.IsFinished() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_OPEN)
		return
	}

	var opponentIDs []int64
	var opponentInfos map[int64]*logicCommon.PlayerRedisInfo
	existing := player.PlayerGloryArenaModel.GetCurrentMatchCandidates()
	if req.GetIsRefresh() == 0 {
		if len(existing) < gloryArenaService.DefaultGloryArenaMatchCount {
			matchReq := &gloryArenaService.GloryArenaMatchRequest{
				PlayerId:       player.GetUserId(),
				PoolVersion:    opsState.GroupVersion,
				WinCount:       player.PlayerGloryArenaModel.GetWinCount(),
				SelfPower:      player.GetMainFormationPower(),
				DefeatedSet:    player.PlayerGloryArenaModel.GetDefeatedSet(),
				LastOpponents:  make([]int64, 0),
				NeedCount:      gloryArenaService.DefaultGloryArenaMatchCount,
				ForceDifferent: req.GetIsRefresh() == 1,
			}
			members, poolVersion, matchErr := gloryArenaSvc.GetChallengeList(matchReq)
			if matchErr != nil || len(members) < gloryArenaService.DefaultGloryArenaMatchCount {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}

			opponentIDs = make([]int64, 0, len(members))
			for _, member := range members {
				if member == nil || member.PlayerId <= 0 {
					continue
				}
				opponentIDs = append(opponentIDs, member.PlayerId)
			}
			if len(opponentIDs) < gloryArenaService.DefaultGloryArenaMatchCount {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			opponentInfos = gloryArenaSvc.LoadChallengePlayerInfos(opponentIDs)
			player.PlayerGloryArenaModel.SetCurrentMatchCandidates(opponentIDs, opponentInfos)
			player.PlayerGloryArenaModel.SetPoolVersion(poolVersion)
		} else {
			opponentIDs = existing
		}
		opponentInfos = player.PlayerGloryArenaModel.GetCurrentMatchCandidateInfos()
	} else {
		items := make([]*gameConfig.ItemConfig, 0)
		items = append(items, &gameConfig.ItemConfig{
			ID:  gameConfig.GetGloryArenaRefreshItem(),
			Num: 1,
		})
		check, checkErr := itemService.CheckItemsCount(player, items)
		if !check || checkErr != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		if removeErr := itemService.RemoveItems(player, items, enum.ITEM_CHANGE_REASON_GLORY_ARENA_REFRESH_LIST); removeErr != nil {
			logger.ErrorBySprintf("[honorArenaController] remove glory arena refresh item failed userId:%d err:%v", player.GetUserId(), removeErr)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}

		matchReq := &gloryArenaService.GloryArenaMatchRequest{
			PlayerId:       player.GetUserId(),
			PoolVersion:    opsState.GroupVersion,
			WinCount:       player.PlayerGloryArenaModel.GetWinCount(),
			SelfPower:      player.GetMainFormationPower(),
			DefeatedSet:    player.PlayerGloryArenaModel.GetDefeatedSet(),
			LastOpponents:  existing,
			NeedCount:      gloryArenaService.DefaultGloryArenaMatchCount,
			ForceDifferent: req.GetIsRefresh() == 1,
		}
		members, poolVersion, matchErr := gloryArenaSvc.GetChallengeList(matchReq)
		if matchErr != nil || len(members) < gloryArenaService.DefaultGloryArenaMatchCount {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}

		opponentIDs = make([]int64, 0, len(members))
		for _, member := range members {
			if member == nil || member.PlayerId <= 0 {
				continue
			}
			opponentIDs = append(opponentIDs, member.PlayerId)
		}
		if len(opponentIDs) < gloryArenaService.DefaultGloryArenaMatchCount {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		opponentInfos = gloryArenaSvc.LoadChallengePlayerInfos(opponentIDs)
		player.PlayerGloryArenaModel.SetCurrentMatchCandidates(opponentIDs, opponentInfos)
		player.PlayerGloryArenaModel.SetPoolVersion(poolVersion)
	}
	if shouldReloadHonorArenaChallengeInfos(opponentIDs, opponentInfos) {
		opponentInfos = gloryArenaSvc.LoadChallengePlayerInfos(opponentIDs)
		player.PlayerGloryArenaModel.SetCurrentMatchCandidates(opponentIDs, opponentInfos)
	}

	resp := &pb.HonorArenaChallengeListResp{
		Players: buildHonorArenaChallengePlayers(opponentIDs, opponentInfos),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HONOR_ARENA_CHALLENGE_LIST_RESP, resp)
}

func StartHonorArenaBattle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.StartHonorArenaBattleReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PlayerGloryArenaModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	player.PlayerGloryArenaModel.ForceSyncRoundState(tool.UnixNowMilli())
	gloryArenaSvc := gamePlatform.GetGloryArenaService()
	if gloryArenaSvc == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	opsState, err := gloryArenaSvc.GetOpsStateByServerID(player.GetUserServerId())
	if err != nil || opsState == nil || !opsState.IsRoundOpen {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_OPEN)
		return
	}

	if !isGloryArenaChallengeTimeOpen(tool.UnixNowMilli()) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_GLORY_ARENA_WRONG_TIME)
		return
	}
	if !player.PlayerGloryArenaModel.IsEnrolled() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_START)
		return
	}
	if player.PlayerGloryArenaModel.IsFinished() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_GLORY_ARENA_NOT_OPEN)
		return
	}

	opponentID := req.GetOpponentId()
	if opponentID <= 0 || !isHonorArenaCandidate(player.PlayerGloryArenaModel.GetCurrentMatchCandidates(), opponentID) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	config := gameConfig.GetInstanceCfg(int32(enum.GLORY_ARENA_INSTANCE_ID))
	if config == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	if int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID) == config.Id {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_ENTER_SCENE_REPEAT)
		return
	}
	if player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.MAIN_INSTANCE_ID {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	raidData := &logicCommon.PlayerInstanceRaid{
		PlayerId:         player.GetUserId(),
		InstanceID:       enum.GLORY_ARENA_INSTANCE_ID,
		FormationType:    int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA),
		TargetTd:         opponentID,
		IsRobot:          false,
		SubStageInfo:     make(map[int32]*logicCommon.SubStageData),
		SubStageIds:      make([]int32, 0),
		MonsterTemplates: make(map[int64]*logicCommon.MonsterTemplate),
	}
	raidData.StageInfo = logicCommon.NewInstanceStageInfo()
	if !raid.CanEnterInstanceStage(config.Id, 0, 0) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	raid.OnLeaveRaid(player.PlayerInstanceModel.CurrentRaidInfo)

	err = raid.BuildInstanceRaid(raidData)
	if err != nil {
		platformLogger.ErrorWithUser("enter instance error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	loginMutexService.EnterMutex(player.GetUserAccount(), player.GetUserId())
	err = raid.EnterScene(player, raidData)
	loginMutexService.ExitMutex(player.GetUserAccount(), player.GetUserId())
	if err != nil {
		platformLogger.ErrorWithUser("enter scene error", player, err)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_START_HONOR_ARENA_BATTLE_RESP, &pb.StartHonorArenaBattleResp{
		Data: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	player.StaticData.UpdateGloryArenaJoinCount(player.StaticData.GetGloryArenaJoinCount() + 1)
	eventBusService.SubmitJoinInstanceEvent(player.GetUserId(), player.GetUserServerId(), raidData.InstanceID, raidData.CurrentStageId)
}

func isHonorArenaCandidate(candidates []int64, opponentID int64) bool {
	for _, candidate := range candidates {
		if candidate == opponentID {
			return true
		}
	}
	return false
}

func isGloryArenaChallengeTimeOpen(nowMilli int64) bool {
	startHour, endHour := gameConfig.GetGloryArenaChallengeTime()
	now := time.UnixMilli(nowMilli)
	year, month, day := now.Date()
	location := now.Location()
	start := time.Date(year, month, day, int(startHour), 0, 0, 0, location)
	end := time.Date(year, month, day, int(endHour), 0, 0, 0, location)
	return !now.Before(start) && now.Before(end)
}

func buildHonorArenaChallengePlayers(opponentIDs []int64, infos map[int64]*logicCommon.PlayerRedisInfo) []*pb.HonorPlayerBasicInfo {
	respPlayers := make([]*pb.HonorPlayerBasicInfo, 0, len(opponentIDs))
	for _, opponentID := range opponentIDs {
		if opponentID <= 0 {
			continue
		}

		playerInfo := infos[opponentID]
		item := &pb.HonorPlayerBasicInfo{
			Opponent: &pb.PlayerBasicInfo{
				UserId: opponentID,
			},
			Power:  0,
			Heroes: make([]*pb.HeroShowInfo, 0),
		}

		if playerInfo != nil && playerInfo.BasicInfo != nil {
			item.Opponent.ServerId = playerInfo.BasicInfo.ServerId
			item.Opponent.NickName = playerInfo.BasicInfo.Name
			item.Opponent.HeadId = playerInfo.BasicInfo.HeadId
			item.Opponent.HeadFrameId = playerInfo.BasicInfo.FrameId
			item.Opponent.TitleId = playerInfo.BasicInfo.Title
			item.Opponent.Level = playerInfo.BasicInfo.MainCityLevel
			item.Opponent.ImageId = playerInfo.BasicInfo.ImageId
			item.Opponent.BubbleId = playerInfo.BasicInfo.BubbleId
		}

		if playerInfo != nil && playerInfo.BattleInfo != nil {
			if formation, ok := playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]; ok && formation != nil {
				item.Power = formation.BattlePower
			}
			for _, hero := range playerInfo.BattleInfo.FormationHeroes {
				if hero == nil {
					continue
				}
				info := &pb.HeroShowInfo{
					Unit: hero.Units,
				}
				if hero.PetInfo != nil {
					info.PetId = hero.PetInfo.PetId
				}
				item.Heroes = append(item.Heroes, info)
			}
		}

		respPlayers = append(respPlayers, item)
	}
	return respPlayers
}

func shouldReloadHonorArenaChallengeInfos(opponentIDs []int64, infos map[int64]*logicCommon.PlayerRedisInfo) bool {
	if len(opponentIDs) == 0 {
		return false
	}
	if len(infos) < len(opponentIDs) {
		return true
	}
	for _, opponentID := range opponentIDs {
		if opponentID <= 0 {
			return true
		}
		info := infos[opponentID]
		if info == nil || info.BasicInfo == nil || info.BattleInfo == nil {
			return true
		}
	}
	return false
}
