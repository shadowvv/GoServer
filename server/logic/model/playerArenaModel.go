package model

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

const (
	UPDATE_INIT_INTERVAL = 1 * 1000
	UPDATE_MAX_INTERVAL  = 15 * 1000
	ARENA_RESOLVE_TIME   = 30 * tool.MINUTE_MILLI
)

type PlayerChallengeBasicData struct {
	UserId  int64 `json:"user_id"`
	Score   int32 `json:"score"`
	IsRobot int32 `json:"is_robot"`
}

type PlayerArenaEntity struct {
	UserId         int64  `gorm:"column:user_Id;primaryKey"`
	ChallengeList  string `gorm:"column:challenge_list"`
	ChallengeCount int32  `gorm:"column:challenge_num"`
	RefreshCount   int32  `gorm:"column:refresh_count"`
	Score          int32  `gorm:"column:score"`
	Version        string `gorm:"column:version"`
	LastRewardTime int64  `gorm:"column:last_reward_time"`

	ChallengeListData []*PlayerChallengeBasicData `gorm:"-"`
}

func (u *PlayerArenaEntity) TableName() string {
	return "player_arena_data"
}

type PlayerArenaLogEntity struct {
	BattleId          int64  `gorm:"column:battle_id;primaryKey"`
	AttackUserId      int64  `gorm:"column:attack_user_id;"`
	AttackScoreChange int32  `gorm:"column:attack_score_change;"`
	DefendUserId      int64  `gorm:"column:defend_user_id;"`
	DefendScoreChange int32  `gorm:"column:defend_score_change;"`
	DefendResolved    int32  `gorm:"column:defend_resolved;"`
	ChallengeTime     int64  `gorm:"column:challenge_time;"`
	Version           string `gorm:"column:version"`
}

func (u *PlayerArenaLogEntity) TableName() string {
	return "player_arena_log"
}

type PlayerArenaModel struct {
	Player              *PlayerModel
	entity              *PlayerArenaEntity
	Changed             map[string]interface{}
	lastTickTime        int64
	UpdateScoreInterval int64
}

var _ logicCommon.PlayerModelInterface = (*PlayerArenaModel)(nil)

func (p *PlayerArenaModel) SaveModelToDB() {
	if p.Changed == nil || len(p.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(p.entity, p.Changed, p.Player.GetUserId())
	p.Changed = make(map[string]interface{})
}

// ExportPlayerData 瀹炵幇 PlayerInfoExportable 鎺ュ彛锛岀敓鎴?INSERT SQL 璇彞
func (p *PlayerArenaModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		p.entity.RefreshCount = 0
		p.Changed["refresh_count"] = p.entity.RefreshCount
		p.entity.ChallengeCount = 0
		p.Changed["challenge_num"] = p.entity.ChallengeCount
		if p.Player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.ARENA_INSTANCE_ID {
			p.RefreshChallengeList()
		}
	}

	// arena version refresh after daily settle window (00:30).
	if currentTime-tool.GetTodayZeroByTimeStamp(currentTime) >= ARENA_RESOLVE_TIME {
		version := logicCommon.GetArenaRankVersionByTime(p.Player.GetUserServerId(), currentTime)
		if p.entity.Version != version {
			logger.InfoWithSprintf("arena func ArenaRefresh passDay userId:%d,oldVersion:%s,currentVersion:%s", p.Player.GetUserId(), p.entity.Version, version)
			p.entity.Score = gameConfig.GetArenaInitScore()
			p.Changed["score"] = p.entity.Score
			p.entity.Version = version
			p.Changed["version"] = p.entity.Version
			p.Player.UpdatePlayerBasicInfoToRedis()
			logicCommon.UpdateAreanScoreRank(p.Player.GetUserServerId(), version, p.Player.GetUserId(), p.entity.Score)
			operationLogService.OnUserArenaChange(p.Player.GetUserId(), operationLogService.ARENA_OPER_INIT, 0, p.entity.Score, 0)
			if p.Player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.ARENA_INSTANCE_ID {
				p.RefreshChallengeList()
			}
		}
	}

	// update defended-score changes
	if currentTime-p.lastTickTime >= p.UpdateScoreInterval {
		p.lastTickTime = currentTime
		logs, err := easyDB.GetPlayerEntitiesByRaw[PlayerArenaLogEntity](enum.SELECT_ARENA_DEFEND_NOT_RESOLVE_SQL, p.Player.GetUserId(), p.entity.Version)
		if err != nil {
			return
		}
		if len(logs) == 0 {
			p.UpdateScoreInterval = p.UpdateScoreInterval * 2
			if p.UpdateScoreInterval > UPDATE_MAX_INTERVAL {
				p.UpdateScoreInterval = UPDATE_MAX_INTERVAL
			}
			return
		}
		p.UpdateScoreInterval = UPDATE_INIT_INTERVAL
		totalScore := int32(0)
		for _, log := range logs {
			totalScore += log.DefendScoreChange
		}
		logger.InfoWithSprintf("arena func point change userId:%d,scoreChange:%d", p.Player.GetUserId(), totalScore)
		err = easyDB.UpdatePlayerEntityByRaw(buildUpdateLogsSql(logs))
		if err != nil {
			logger.ErrorBySprintf("arena func point update error userId:%d", p.Player.GetUserId())
			return
		}
		p.AddScore(totalScore)
	}
}

func buildUpdateLogsSql(logs []*PlayerArenaLogEntity) string {
	if len(logs) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("UPDATE player_arena_log SET defend_resolved = 1 WHERE defend_resolved = 0 AND battle_id IN (")

	for i, log := range logs {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(strconv.FormatInt(log.BattleId, 10))
	}

	builder.WriteString(");")

	return builder.String()
}
func (p *PlayerArenaModel) AddScore(point int32) {
	p.entity.Score += point
	if p.entity.Score < 0 {
		p.entity.Score = 0
	}
	p.Changed["score"] = p.entity.Score
	p.Player.UpdatePlayerBasicInfoToRedis()
	logicCommon.UpdateAreanScoreRank(p.Player.GetUserServerId(), p.entity.Version, p.Player.GetUserId(), p.entity.Score)

	version := p.entity.Version
	if version == "" {
		version = logicCommon.GetArenaRankVersionByTime(p.Player.GetUserServerId(), tool.UnixNowMilli())
	}
	rankId, err := logicCommon.GetRankUniqueId(gameConfig.GetArenaRankId(), 0, 0, p.Player.GetUserServerId(), version)
	if err != nil {
		logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
		return
	}
	updateRankReq := &rpcPb.NotifyUpdateRankInfo{
		Id:    p.Player.GetUserId(),
		Score: int64(p.entity.Score - gameConfig.GetArenaInitScore()),
	}
	_ = rpcMessageSender.SendMessageToRankBoard(p.Player.GetUserId(), rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, updateRankReq)

	// alliance arena rank update request
	allianceRankReq := &rpcPb.NotifyUpdateRankInfo{
		Id:                p.Player.GetUserId(),
		Score:             int64(point),
		IncrementalUpdate: true,
	}
	allianceRankIds := logicCommon.GetCommonRankUniqueIdsByPointType(
		p.Player.GetUserServerId(),
		enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA,
		p.entity.Version,
	)
	for _, allianceRankId := range allianceRankIds {
		_ = rpcMessageSender.SendMessageToRankBoard(
			p.Player.GetUserId(),
			allianceRankId,
			0,
			rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO,
			allianceRankReq,
		)
	}
}

func (p *PlayerArenaModel) updateChallengeList(lists map[int64]*PlayerChallengeBasicData) {
	p.entity.ChallengeListData = make([]*PlayerChallengeBasicData, 0)
	for _, data := range lists {
		p.entity.ChallengeListData = append(p.entity.ChallengeListData, data)
	}
	listString, err := json.Marshal(p.entity.ChallengeListData)
	if err == nil {
		p.Changed["challenge_list"] = listString
	}
}

func (p *PlayerArenaModel) RefreshChallengeList() []*PlayerChallengeBasicData {

	result := make(map[int64]*PlayerChallengeBasicData)

	playerScore := p.GetScore()
	userId := p.Player.GetUserId()

	lessScorePlayerNum := tool.RandInt32(1, 3)
	// ---------- 鏂版墜淇濇姢 ----------
	if p.Player.StaticData.GetArenaChallengeTimes() < 3 {
		robots := gameConfig.GetBotCfgByScore(playerScore-30, playerScore, lessScorePlayerNum)
		for _, r := range robots {
			result[int64(r.Id)] = &PlayerChallengeBasicData{
				UserId:  int64(r.Id),
				Score:   r.ArenaPoints,
				IsRobot: 1,
			}
		}
		if int32(len(result)) < lessScorePlayerNum {
			robots = gameConfig.GetBotCfgByScore(playerScore-50, playerScore, lessScorePlayerNum-int32(len(result)))
			for _, r := range robots {
				result[int64(r.Id)] = &PlayerChallengeBasicData{
					UserId:  int64(r.Id),
					Score:   r.ArenaPoints,
					IsRobot: 1,
				}
			}
		}
		robots = gameConfig.GetBotCfgByScore(playerScore, playerScore+50, 5-int32(len(result)))
		for _, r := range robots {
			result[int64(r.Id)] = &PlayerChallengeBasicData{
				UserId:  int64(r.Id),
				Score:   r.ArenaPoints,
				IsRobot: 1,
			}
		}
		if len(result) < 5 {
			robots := gameConfig.GetRankBotList()
			for _, r := range robots {
				result[int64(r.Id)] = &PlayerChallengeBasicData{
					UserId:  int64(r.Id),
					Score:   r.ArenaPoints,
					IsRobot: 1,
				}
				if len(result) >= 5 {
					break
				}
			}
		}
		p.updateChallengeList(result)
		return p.GetChallengeList()
	}

	ctx := context.Background()
	serverId := p.Player.GetUserServerId()
	redisKey := enum.GetArenaScoreInfoKey(serverId, p.entity.Version)

	p.matchByScore(ctx, redisKey, playerScore-30, playerScore, lessScorePlayerNum, result)
	if int32(len(result)) < lessScorePlayerNum {
		p.matchByScore(ctx, redisKey, playerScore-50, playerScore, lessScorePlayerNum-int32(len(result)), result)
	}
	if len(result) < 5 {
		p.matchByRank(ctx, redisKey, userId, int32(5-len(result)), result)
	}

	if len(result) >= 5 {
		findLess := false
		id := int64(0)
		for _, data := range result {
			id = data.UserId
			if data.Score < playerScore {
				findLess = true
				break
			}
		}
		if !findLess {
			delete(result, id)
			robots := gameConfig.GetBotCfgByScore(playerScore-100, playerScore, 1)
			for _, r := range robots {
				result[int64(r.Id)] = &PlayerChallengeBasicData{
					UserId:  int64(r.Id),
					Score:   r.ArenaPoints,
					IsRobot: 1,
				}
			}
		}
	}

	// ---------- 琛ユ満鍣ㄤ汉 ----------
	if len(result) < 5 {
		robots := gameConfig.GetBotCfgByScore(playerScore-100, playerScore+100, int32(5-len(result)))
		for _, r := range robots {
			result[int64(r.Id)] = &PlayerChallengeBasicData{
				UserId:  int64(r.Id),
				Score:   r.ArenaPoints,
				IsRobot: 1,
			}
		}
	}
	if len(result) < 5 {
		robots := gameConfig.GetRankBotList()
		for _, r := range robots {
			result[int64(r.Id)] = &PlayerChallengeBasicData{
				UserId:  int64(r.Id),
				Score:   r.ArenaPoints,
				IsRobot: 1,
			}
			if len(result) >= 5 {
				break
			}
		}
	}

	p.updateChallengeList(result)
	return p.GetChallengeList()
}

func (p *PlayerArenaModel) matchByScore(ctx context.Context, redisKey string, scoreLeft int32, scoreRight int32, num int32, result map[int64]*PlayerChallengeBasicData) {

	players, err := dbService.RDB.ZRangeByScoreWithScores(
		ctx,
		redisKey,
		&redis.ZRangeBy{
			Min: strconv.Itoa(int(scoreLeft)),
			Max: strconv.Itoa(int(scoreRight)),
		},
	).Result()

	if err != nil {
		return
	}

	for _, z := range players {
		uid, _ := strconv.ParseInt(z.Member.(string), 10, 64)
		if uid == p.Player.GetUserId() {
			continue
		}
		result[uid] = &PlayerChallengeBasicData{
			UserId: uid,
			Score:  int32(z.Score),
		}
		if int32(len(result)) >= num {
			break
		}
	}
	return
}

func (p *PlayerArenaModel) matchByRank(ctx context.Context, redisKey string, userId int64, num int32, result map[int64]*PlayerChallengeBasicData) {
	rank, err := dbService.RDB.ZRevRank(
		ctx,
		redisKey,
		strconv.FormatInt(userId, 10),
	).Result()
	if err != nil {
		return
	}
	start := rank - 20
	if start < 0 {
		start = 0
	}
	end := rank + 20
	players, err := dbService.RDB.ZRevRangeWithScores(ctx, redisKey, start, end).Result()

	if err != nil {
		return
	}

	for _, z := range players {
		uid, _ := strconv.ParseInt(z.Member.(string), 10, 64)
		if uid == userId {
			continue
		}
		info := logicCommon.GetPlayerRedisInfo(uid)
		if info == nil {
			continue
		}
		result[uid] = &PlayerChallengeBasicData{
			UserId: uid,
			Score:  int32(z.Score),
		}
		if len(result) >= 5 {
			break
		}
	}
	return
}

func (p *PlayerArenaModel) GetAllArenaLog() []*PlayerArenaLogEntity {
	logEntities, err := easyDB.GetPlayerEntitiesByRaw[PlayerArenaLogEntity](enum.SELECT_ARENA_DEFEND_LOG_SQL, p.Player.GetUserId(), p.entity.Version)
	if err != nil {
		return nil
	}
	return logEntities
}

func (p *PlayerArenaModel) AddRefreshTime(refreshTimes int32) {
	p.entity.RefreshCount += refreshTimes
	p.Changed["refresh_count"] = p.entity.RefreshCount
}

func (p *PlayerArenaModel) AddChallengeTime(challengeTimes int) {
	p.entity.ChallengeCount += int32(challengeTimes)
	p.Changed["challenge_num"] = p.entity.ChallengeCount
}

func (p *PlayerArenaModel) UpdateLastRewardTime(currentTime int64) {
	p.entity.LastRewardTime = currentTime
	p.Changed["last_reward_time"] = p.entity.LastRewardTime
}

func (p *PlayerArenaModel) GetChallengeOpponent(id int64) *PlayerChallengeBasicData {
	for _, data := range p.entity.ChallengeListData {
		if data.UserId == id {
			return data
		}
	}
	return nil
}

func (p *PlayerArenaModel) GetChallengeList() []*PlayerChallengeBasicData {
	return p.entity.ChallengeListData
}

func (p *PlayerArenaModel) GetScore() int32 {
	return p.entity.Score
}

func (p *PlayerArenaModel) GetRefreshTime() int32 {
	return p.entity.RefreshCount
}

func (p *PlayerArenaModel) GetChallengeTime() int32 {
	return p.entity.ChallengeCount
}

func (p *PlayerArenaModel) GetLastRewardTime() int64 {
	return p.entity.LastRewardTime
}

func (p *PlayerArenaModel) GetVersion() string {
	return p.entity.Version
}

func NewPlayerArenaModel(player *PlayerModel) *PlayerArenaModel {
	return &PlayerArenaModel{
		Player: player,
		entity: &PlayerArenaEntity{
			UserId:            player.GetUserId(),
			ChallengeList:     "",
			ChallengeListData: make([]*PlayerChallengeBasicData, 0),
			Score:             gameConfig.GetArenaInitScore(),
			Version:           logicCommon.GetArenaRankVersionByTime(player.GetUserServerId(), tool.UnixNowMilli()),
		},
		Changed:             make(map[string]interface{}),
		UpdateScoreInterval: UPDATE_INIT_INTERVAL,
	}
}

func LoadPlayerArenaModel(player *PlayerModel) (*PlayerArenaModel, error) {
	entity, err := easyDB.GetPlayerEntityByID[PlayerArenaEntity](player.GetUserId())
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if entity == nil {
		model, err := CreatePlayerArenaModel(player)
		return model, err
	}
	var basicData = make([]*PlayerChallengeBasicData, 0)
	err = json.Unmarshal([]byte(entity.ChallengeList), &basicData)
	if err != nil {
		basicData = make([]*PlayerChallengeBasicData, 0)
	}
	entity.ChallengeListData = basicData

	model := &PlayerArenaModel{
		Player:              player,
		entity:              entity,
		Changed:             make(map[string]interface{}),
		UpdateScoreInterval: UPDATE_INIT_INTERVAL,
	}
	return model, nil
}

func CreatePlayerArenaModel(player *PlayerModel) (*PlayerArenaModel, error) {
	model := NewPlayerArenaModel(player)
	err := easyDB.CreatePlayerEntity(model.entity)
	if err != nil {
		return nil, err
	}
	return model, nil
}
