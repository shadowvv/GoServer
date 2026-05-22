package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/raid"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("arena", &ArenaController{})
}

type ArenaController struct {
}

var _ LogicControllerInterface = (*ArenaController)(nil)

func (p *ArenaController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_ARENA_INFO_REQ, &pb.GetArenaInfoReq{}, GetArenaInfo, enum.FUNCTION_ID_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_Get_CHALLENGE_LIST_REQ, &pb.GetChallengeListReq{}, GetChallengeList, enum.FUNCTION_ID_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_REFRESH_CHALLENGE_LIST_REQ, &pb.RefreshChallengeListReq{}, RefreshChallengeList, enum.FUNCTION_ID_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_Get_Arena_LOG_REQ, &pb.GetArenaLogReq{}, GetArenaLog, enum.FUNCTION_ID_ARENA)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_START_ARENA_BATTLE_REQ, &pb.StartArenaBattleReq{}, StartArenaBattle, enum.FUNCTION_ID_ARENA)
}

func GetArenaInfo(message proto.Message, player *model.PlayerModel) {
	refreshLeftTime := gameConfig.GetArenaDailyFreeRefreshTimes() - player.PlayerArenaModel.GetRefreshTime()
	if refreshLeftTime < 0 {
		refreshLeftTime = 0
	}
	freeChallengeTime := gameConfig.GetArenaDailyFreeChallengeTimes() - player.PlayerArenaModel.GetChallengeTime()
	if freeChallengeTime < 0 {
		freeChallengeTime = 0
	}
	resp := &pb.GetArenaInfoResp{
		Score:             player.PlayerArenaModel.GetScore(),
		RefreshTime:       refreshLeftTime,
		FreeChallengeTime: freeChallengeTime,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_ARENA_INFO_RESP, resp)
}

func GetArenaLog(message proto.Message, player *model.PlayerModel) {
	areanLogs := player.PlayerArenaModel.GetAllArenaLog()
	resp := &pb.GetArenaLogResp{
		AreaLogs: make([]*pb.ArenaLog, 0),
	}
	for _, log := range areanLogs {
		pbLog := &pb.ArenaLog{
			Opponent: &pb.OpponentBasicInfo{
				Opponent: &pb.PlayerBasicInfo{
					UserId: log.AttackUserId,
				},
			},
			ChangeScore: -log.DefendScoreChange,
			BattleTime:  log.ChallengeTime,
		}

		playerInfo := logicCommon.GetPlayerRedisInfo(log.AttackUserId)
		if playerInfo != nil {
			pbLog.Opponent.Opponent.NickName = playerInfo.BasicInfo.Name
			battlePower := int64(0)
			if formation, ok := playerInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF)]; ok {
				battlePower = formation.BattlePower
			}
			pbLog.Opponent.OpponentBattlePower = battlePower
			pbLog.Opponent.OpponentScore = playerInfo.BasicInfo.ArenaScore
			for _, hero := range playerInfo.BattleInfo.FormationHeroes {
				pbLog.Opponent.Units = append(pbLog.Opponent.Units, hero.Units)
			}
		}
		resp.AreaLogs = append(resp.AreaLogs, pbLog)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_Get_Arena_LOG_RESP, resp)
}

func GetChallengeList(message proto.Message, player *model.PlayerModel) {
	lists := player.PlayerArenaModel.GetChallengeList()
	if len(lists) == 0 {
		lists = player.PlayerArenaModel.RefreshChallengeList()
	}
	resp := &pb.GetChallengeListResp{
		Opponents: make([]*pb.OpponentBasicInfo, 0),
	}
	for _, data := range lists {
		if data == nil {
			continue
		}
		opponent := &pb.OpponentBasicInfo{
			Opponent: &pb.PlayerBasicInfo{
				UserId: data.UserId,
			},
			OpponentScore: data.Score,
		}
		if data.IsRobot != 1 {
			basicInfo := logicCommon.GetPlayerRedisInfo(data.UserId)
			if basicInfo != nil {
				opponent.Opponent.NickName = basicInfo.BasicInfo.Name
				opponent.OpponentScore = basicInfo.BasicInfo.ArenaScore
				if formation, ok := basicInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF)]; ok {
					opponent.OpponentBattlePower = formation.BattlePower
				}
				opponent.Units = make([]int32, 0)
				for _, hero := range basicInfo.BattleInfo.FormationHeroes {
					opponent.Units = append(opponent.Units, hero.Units)
				}
			}
		} else {
			opponent.IsRobot = 1
		}
		resp.Opponents = append(resp.Opponents, opponent)

	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_Get_CHALLENGE_LIST_RESP, resp)
}

func RefreshChallengeList(message proto.Message, player *model.PlayerModel) {
	if tool.UnixNowMilli()-tool.GetTodayZeroByTimeStamp(tool.UnixNowMilli()) < model.ARENA_RESOLVE_TIME {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_REFRESH_CHALLENGE_LIST_RESP, pb.ERROR_CODE_ARENA_IS_SETTLING)
		return
	}
	if player.PlayerArenaModel.GetRefreshTime() < gameConfig.GetArenaDailyFreeRefreshTimes() {
		player.PlayerArenaModel.AddRefreshTime(1)
	} else {
		item := &gameConfig.ItemConfig{
			ID:  enum.DIAMOND_ITEM_ID,
			Num: int64(gameConfig.GetRefreshDiamondConsumptionQuantity()),
		}
		result, _ := itemService.CheckItemCount(player, item)
		if !result {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_REFRESH_CHALLENGE_LIST_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		_ = itemService.RemoveItem(player, item, enum.ITEM_CHANGE_REASON_ARENA_REFRESH)
	}

	lists := player.PlayerArenaModel.RefreshChallengeList()
	resp := &pb.RefreshChallengeListResp{
		Opponents: make([]*pb.OpponentBasicInfo, 0),
	}
	for _, data := range lists {
		opponent := &pb.OpponentBasicInfo{
			Opponent: &pb.PlayerBasicInfo{
				UserId: data.UserId,
			},
			OpponentScore: data.Score,
		}
		if data.IsRobot != 1 {
			basicInfo := logicCommon.GetPlayerRedisInfo(data.UserId)
			if basicInfo != nil {
				opponent.Opponent.NickName = basicInfo.BasicInfo.Name
				opponent.OpponentScore = basicInfo.BasicInfo.ArenaScore
				if formation, ok := basicInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF)]; ok {
					opponent.OpponentBattlePower = formation.BattlePower
				}
				opponent.Units = make([]int32, 0)
				for _, hero := range basicInfo.BattleInfo.FormationHeroes {
					opponent.Units = append(opponent.Units, hero.Units)
				}
			}
		} else {
			opponent.IsRobot = 1
		}
		resp.Opponents = append(resp.Opponents, opponent)

	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_REFRESH_CHALLENGE_LIST_RESP, resp)
}

func StartArenaBattle(message proto.Message, player *model.PlayerModel) {
	if tool.UnixNowMilli()-tool.GetTodayZeroByTimeStamp(tool.UnixNowMilli()) < model.ARENA_RESOLVE_TIME {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_ARENA_IS_SETTLING)
		return
	}

	req, ok := message.(*pb.StartArenaBattleReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	config := gameConfig.GetArenaInstanceCfg()
	if config == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if int32(player.PlayerInstanceModel.CurrentRaidInfo.InstanceID) == config.Id {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_ENTER_SCENE_REPEAT)
		return
	}

	if player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.MAIN_INSTANCE_ID {
		platformLogger.ErrorWithUser("enter scene repeat", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if player.PlayerArenaModel.GetChallengeTime() >= gameConfig.GetArenaDailyFreeChallengeTimes() {
		check, err := itemService.CheckItemsCount(player, config.TicketID)
		if !check || err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		_ = itemService.RemoveItems(player, config.TicketID, enum.ITEM_CHANGE_REASON_ARENA_CHALLENGE)
	}

	var isRobot int32
	if req.IsRevenge == 1 {
		areanLogs := player.PlayerArenaModel.GetAllArenaLog()
		find := false
		for _, log := range areanLogs {
			if log.AttackUserId == req.OpponentId {
				find = true
				break
			}
		}
		if !find {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
	} else {
		opponent := player.PlayerArenaModel.GetChallengeOpponent(req.OpponentId)
		if opponent == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
		isRobot = opponent.IsRobot
	}

	raidData := &logicCommon.PlayerInstanceRaid{
		PlayerId:         player.GetUserId(),
		InstanceID:       enum.ARENA_INSTANCE_ID,
		TargetTd:         req.OpponentId,
		IsRobot:          isRobot == 1,
		SubStageInfo:     make(map[int32]*logicCommon.SubStageData),
		SubStageIds:      make([]int32, 0),
		MonsterTemplates: make(map[int64]*logicCommon.MonsterTemplate),
	}
	raidData.StageInfo = logicCommon.NewInstanceStageInfo()
	if !raid.CanEnterInstanceStage(config.Id, 0, 0) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	raid.OnLeaveRaid(player.PlayerInstanceModel.CurrentRaidInfo)

	err := raid.BuildInstanceRaid(raidData)
	if err != nil {
		platformLogger.ErrorWithUser("enter instance error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	loginMutexService.EnterMutex(player.GetUserAccount(), player.GetUserId())
	err = raid.EnterScene(player, raidData)
	loginMutexService.ExitMutex(player.GetUserAccount(), player.GetUserId())

	if err != nil {
		platformLogger.ErrorWithUser("enter scene error", player, err)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
		return
	}
	player.StaticData.AddArenaChallengeTimes(1)
	player.PlayerArenaModel.AddChallengeTime(1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_START_ARENA_BATTLE_RESP, &pb.StartArenaBattleResp{
		Data: raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo),
	})
	eventBusService.SubmitJoinInstanceEvent(player.GetUserId(), player.GetUserServerId(), raidData.InstanceID, raidData.CurrentStageId)
}
