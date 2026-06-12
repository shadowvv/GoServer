package gameController

import (
	"fmt"
	"unicode/utf8"

	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/eventService"
	"github.com/drop/GoServer/server/logic/rpcController"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/service/wordFilter"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("player", &PlayerController{})
}

type PlayerController struct {
}

var _ LogicControllerInterface = (*PlayerController)(nil)

func (p *PlayerController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HEART_REQ, &pb.HeartReq{}, onPlayerHeartbeatHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CHANGE_NICKNAME_REQ, &pb.ChangeNicknameReq{}, ChangeNickNameHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXCHANGE_ITEM_REQ, &pb.ExchangeItemReq{}, exchangeItem, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_ANNOUNCE_REQ, &pb.GetAnnounceReq{}, GetAnnounceHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_REQ, &pb.GetSystemUnlockRewardReq{}, GetSystemUnlockRewardHandler, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_REQ, &pb.GetTempBattleSpeedUpReq{}, GetTempBattleSpeedHandler, enum.FUNCTION_ID_BATTLE_SPEED)

	RegisterPlayerInnerTask(enum.INNER_MSG_EVENT_UPDATE_RANK_BOARD, updatePlayerRankInfoHandler)
}

func updatePlayerRankInfoHandler(messageTask serviceInterface.InnerTaskInterface) (any, error) {
	innerTask, ok := messageTask.(*dispatcherService.InnerTask)
	if !ok {
		return nil, fmt.Errorf("invalid task type")
	}

	event, ok := innerTask.ReqParameter.(eventService.GameEvent)

	playerId := event.GetObjectID()
	eventType := event.GetEventType()

	p := sessionManager.GetPlayerBasicInfoByUserId(playerId)
	if p == nil {
		return nil, fmt.Errorf("player not exist")
	}
	player := p.(*model.PlayerModel)

	if _, ok := enum.PlayerEventTypes[eventType]; ok {
		switch eventType {
		case enum.EventTypePassInstance:
			ev := event.(*eventService.PassInstanceEvent)
			if ev.InstanceTypeId == enum.MAIN_INSTANCE_ID {
				info := &rpcPb.NotifyUpdateRankInfo{
					PlayerId: ev.PlayerID,
					Score:    int64(player.PlayerInstanceModel.GetMainInstanceMaxStageId()),
				}
				updatePlayerRankInfo(player, info, enum.RANK_BOARD_SCORE_TYPE_MAIN_INSTANCE)
			} else if ev.InstanceTypeId == enum.FIVE_VS_FIVE_TOWER_INSTANCE_ID {
				towerCfg := gameConfig.GetTowerCfg(ev.InstanceId)
				if towerCfg == nil {
					return nil, nil
				}
				info := &rpcPb.NotifyUpdateRankInfo{
					PlayerId: ev.PlayerID,
					Score:    int64(towerCfg.Level),
				}
				updatePlayerRankInfo(player, info, enum.RANK_BOARD_SCORE_TYPE_TOWER)
			}
		case enum.EventTypeBuildLevelUp:
			ev := event.(*eventService.BuildLevelUpEvent)
			if ev.BuildId == enum.ARCHITECTURE_TYPE_MAIN {
				info := &rpcPb.NotifyUpdateRankInfo{
					PlayerId: ev.PlayerID,
					Score:    int64(ev.BuildLevel),
				}
				updatePlayerRankInfo(player, info, enum.RANK_BOARD_SCORE_TYPE_LEVEL)
			}
		}
	}
	return nil, nil
}

func updatePlayerRankInfo(player *model.PlayerModel, info *rpcPb.NotifyUpdateRankInfo, scoreType enum.RankBoardScoreType) {
	commonRankConfigs := gameConfig.GetAllRankCfg()
	for _, rankCfgMap := range commonRankConfigs {
		for _, rankCfg := range rankCfgMap {
			fullRankCfg := gameConfig.GetRankCfgByIds(rankCfg.ActId, rankCfg.Id)
			if fullRankCfg == nil {
				continue
			}
			rankId := ""
			var err error
			if rankCfg.ActId != 0 {
				settled, version := player.PlayerActivityModel.CheckActivitySettled(rankCfg.ActId)
				if settled {
					continue
				}
				if int64(fullRankCfg.RankThreshold) > info.Score {
					continue
				}
				rankId, err = logicCommon.GetRankUniqueId(0, rankCfg.ActId, rankCfg.Id, player.GetUserServerId(), version)
				if err != nil {
					logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
					return
				}
			} else {
				if int64(fullRankCfg.RankThreshold) > info.Score {
					continue
				}
				rankId, err = logicCommon.GetRankUniqueId(rankCfg.Id, 0, 0, player.GetUserServerId(), "")
				if err != nil {
					logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
					return
				}
			}
			if rankCfg.PointType == int32(scoreType) {
				err = rpcController.SendMessageToRankBoard(player.GetUserId(), rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, info)
				if err != nil {
					logger.ErrorBySprintf("notify rank info to rankBoard node error: %v", err)
					continue
				}
			}
		}
	}
}

func onPlayerHeartbeatHandler(message proto.Message, player *model.PlayerModel) {
	messageSender.SendMessage(player, pb.MESSAGE_ID_HEART_RESP, &pb.HeartResp{
		Timestamp: tool.UnixNowMilli(),
	})
}

func ChangeNickNameHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ChangeNicknameReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.NickName == player.User.GetNickname() {
		platformLogger.ErrorWithUser("same nickname", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, pb.ERROR_CODE_CHANGE_NICKNAME_SAME_NICKNAME)
		return
	}
	if !CheckNickNameLegal(req.NickName) {
		platformLogger.ErrorWithUser("nickname is invalid", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, pb.ERROR_CODE_CHANGE_NICKNAME_IS_INVALID)
		return
	}

	changeItem := &gameConfig.ItemConfig{
		ID:  gameConfig.GetNicknameChangeItem().ID,
		Num: gameConfig.GetNicknameChangeItem().Num,
	}
	if player.StaticData.GetChangeNicknameTimes() >= gameConfig.GetChangeNicknameFreeTimes() {
		ok, err := itemService.CheckItemCount(player, changeItem)
		if err != nil || !ok {
			platformLogger.ErrorWithUser("get item count error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		err = itemService.RemoveItem(player, changeItem, enum.ITEM_CHANGE_REASON_CHANGE_NICKNAME)
		if err != nil {
			platformLogger.ErrorWithUser("use item error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
	}
	player.User.UpdateNickname(req.NickName)
	player.StaticData.UpdateChangeNicknameTimes(player.StaticData.GetChangeNicknameTimes() + 1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_CHANGE_NICKNAME_RESP, &pb.ChangeNicknameResp{})
}

func CheckNickNameLegal(name string) bool {
	length := utf8.RuneCountInString(name)
	if length < 1 || length > int(gameConfig.GetMaxNicknameLength()) {
		return false
	}
	if !tool.IsUnicodeLetterOrDigit(name) {
		return false
	}
	return !wordFilter.HasSensitive(name)
}

func exchangeItem(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExchangeItemReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.ExchangeNum <= 0 {
		platformLogger.ErrorWithUser("exchange num <= 0", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	exchangeCfg := gameConfig.GetExchangeCfg(req.ExchangeId)
	if exchangeCfg == nil {
		platformLogger.ErrorWithUser("item id is 0", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if exchangeCfg.SystemId != 0 && !unlockService.CheckSystemUnlock(exchangeCfg.SystemId, player) {
		platformLogger.ErrorWithUser("system unlock error", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
	}
	if exchangeCfg.ActivityId != 0 {
		if ok, _ := player.PlayerActivityModel.CheckActivitySettled(exchangeCfg.ActivityId); ok {
			platformLogger.ErrorWithUser("activity unlock error", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		}
	}
	costItems := make([]*gameConfig.ItemConfig, 0)
	for _, target := range exchangeCfg.ExchangeId {
		costItems = append(costItems, &gameConfig.ItemConfig{
			ID:  target.ID,
			Num: target.Num * int64(req.ExchangeNum),
		})
	}
	ok, err := itemService.CheckItemsCount(player, costItems)
	if err != nil || !ok {
		platformLogger.ErrorWithUser("get item count error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	addItems := make([]*gameConfig.ItemConfig, 0)
	for _, target := range exchangeCfg.TargetId {
		addItems = append(addItems, &gameConfig.ItemConfig{
			ID:  target.ID,
			Num: target.Num * int64(req.ExchangeNum),
		})
	}
	err = itemService.ExchangeItem(player, costItems, addItems, enum.ITEM_CHANGE_REASON_EXCHANGE_ITEM)
	if err != nil {
		// 补丁代码，英雄背包满了提示背包已满
		if err.Error() == "hero bag full" {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_LOTTERY_HERO_BAGE_IS_FULL)
			return
		}
		platformLogger.ErrorWithUser("add item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXCHANGE_ITEM_RESP, &pb.ExchangeItemResp{})
}

func GetAnnounceHandle(message proto.Message, player *model.PlayerModel) {
	all := gamePlatform.GetServerInfoService().GetAllAnnounceInfo(player.GetUserServerId())
	playerAnnounces := make([]*pb.AnnounceInfo, 0)
	for _, info := range all {
		allCheck := true
		for _, stop := range info.UnlockStopIds {
			if !unlockService.CheckUnlock(stop, player) {
				allCheck = false
				break
			}
		}
		if !allCheck {
			continue
		}
		allCheck = true
		for _, unlock := range info.UnlockIds {
			if !unlockService.CheckUnlock(unlock, player) {
				allCheck = false
				break
			}
		}
		if !allCheck {
			continue
		}
		playerAnnounces = append(playerAnnounces, &pb.AnnounceInfo{
			Content:    info.Content,
			Id:         info.Id,
			PicAddress: info.PicAddress,
			ShowType:   info.ShowType,
			Title:      info.Title,
			Type:       info.AnnounceType,
			StartTime:  info.StartTime,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_ANNOUNCE_RESP, &pb.GetAnnounceResp{
		Announces: playerAnnounces,
	})
}

func GetSystemUnlockRewardHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetSystemUnlockRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if !unlockService.CheckSystemUnlock(req.SystemId, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}

	cfg := gameConfig.GetSystemUnlockCfg(req.SystemId)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	entity := player.PlayerFunctionModel.Get(req.SystemId)
	if entity != nil && entity.RewardCommited == 1 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	_ = itemService.AddItem(player, cfg.UnlockReward, enum.ITEM_CHANGE_REASON_SYSTEM_UNLOCK_REWARD)
	player.PlayerFunctionModel.CommitReward(req.SystemId)

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_SYSTEM_UNLOCK_REWARD_RESP, &pb.GetSystemUnlockRewardResp{})
}

func GetTempBattleSpeedHandler(message proto.Message, player *model.PlayerModel) {
	currentSpeed := player.VipCardModel.GetFunctionValue(enum.VIP_PRIVILEGE_BATTLE_SPEED, tool.UnixNowMilli())
	if currentSpeed > 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, pb.ERROR_CODE_ERROR_CODE_ALREADY_SPEED_UP)
		return
	}
	itemCfg := gameConfig.GetItemCfg(gameConfig.GetBattleSpeedUpItemId())
	if itemCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	entity := player.VipCardModel.GetVipCardByItemId(itemCfg.TargetId, tool.UnixNowMilli())
	if len(entity) != 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, pb.ERROR_CODE_ERROR_CODE_ALREADY_SPEED_UP)
		return
	}
	if player.StaticData.GetBattleSpeedUpTimes() == 0 {
		player.StaticData.UpdateBattleSpeedUpTimes(1)
		player.VipCardModel.AddVipCardMinute(itemCfg.TargetId, int64(gameConfig.GetBattleSpeedUpTime()/60))
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, &pb.GetTempBattleSpeedUpResp{})
	} else {
		if player.StaticData.GetDailyBattleSpeedUpTimes() >= gameConfig.GetDailyBattleSpeedUpTimes() {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, pb.ERROR_CODE_ERROR_CODE_DAILY_SPEED_UP_OVER)
			return
		}
		player.StaticData.UpdateDailyBattleSpeedUpTimes(player.StaticData.GetDailyBattleSpeedUpTimes() + 1)
		player.VipCardModel.AddVipCardMinute(itemCfg.TargetId, int64(gameConfig.GetBattleSpeedUpTime()/60))
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_TEMP_BATTLE_SPEED_UP_RESP, &pb.GetTempBattleSpeedUpResp{})
	}

	vipPush := &pb.VipCardInfo{}
	entity = player.VipCardModel.GetVipCardByItemId(itemCfg.TargetId, tool.UnixNowMilli())
	for _, card := range entity {
		if card == nil || card.ItemId != itemCfg.TargetId {
			continue
		}
		cfg := gameConfig.GetVipCardCfg(card.ItemId)
		if cfg == nil {
			continue
		}
		privs := make([]*pb.VipPrivilegeData, 0, len(cfg.Functions))
		for privType, value := range cfg.Functions {
			privs = append(privs, &pb.VipPrivilegeData{
				Type:  pb.VipPrivilegeType(privType),
				Value: value,
			})
		}
		vipPush.ItemId = card.ItemId
		vipPush.ExpireTime = card.ExpireTime
		vipPush.Privs = privs
		break
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_VIP_CARD_INFO, &pb.PushVipCardInfo{
		Info: vipPush,
	})
}
