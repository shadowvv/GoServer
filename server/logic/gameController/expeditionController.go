package gameController

import (
	"math"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("expedition", &ExpeditionController{})
}

type ExpeditionController struct {
}

var _ LogicControllerInterface = (*ExpeditionController)(nil)

func (d *ExpeditionController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_INFO_REQ, &pb.ExpeditionInfoReq{}, ExpeditionInfoHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_START_REQ, &pb.ExpeditionStartReq{}, StartExpeditionHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_FINISH_REQ, &pb.ExpeditionFinishReq{}, FinishSlotHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_CLAIM_REWARD_REQ, &pb.ExpeditionClaimRewardReq{}, ClaimRewardHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_CANCEL_REQ, &pb.ExpeditionCancelReq{}, CancelSlotHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_CLAIM_FREE_STAMINA_REQ, &pb.ExpeditionClaimFreeStaminaReq{}, ClaimFreeStaminaHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_REQ, &pb.ExpeditionSpeedUpReq{}, ExpeditionSpeedUpHandler, enum.FUNCTION_ID_EXPEDITION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_REQ, &pb.ExpeditionChangeLevelReq{}, ExpeditionChangeLevelHandler, enum.FUNCTION_ID_EXPEDITION)
}

func ClaimRewardHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionClaimRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	point := player.ExpeditionModel.GetPointInfo(req.BattleFiledId, req.PointId)
	if point == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_REWARD_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if point.Status != enum.ExpeditionPointStatusReward {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_REWARD_RESP, pb.ERROR_CODE_SLOT_IS_NOT_OVER)
		return
	}
	reward := make([]*pb.ItemBasicInfo, 0)
	for _, item := range point.RewardItem {
		reward = append(reward, &pb.ItemBasicInfo{
			ItemId: item.ID,
			Count:  item.Num,
		})
	}
	resp := &pb.ExpeditionClaimRewardResp{
		IsWin:  point.IsWin,
		Reward: reward,
	}
	_ = itemService.AddItems(player, point.RewardItem, enum.ITEM_CHANGE_REASON_CLIAM_EXPEDITION_REWARD)
	player.ExpeditionModel.ClaimPointReward(req.BattleFiledId, req.PointId)
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_REWARD_RESP, resp)
}

func ClaimFreeStaminaHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.ExpeditionClaimFreeStaminaReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_FREE_STAMINA_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player.ExpeditionModel.ExpeditionData.DailyFreeStaminaTimes+1 > gameConfig.GetDailyLimitOnFreeStaminaClaims() {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_FREE_STAMINA_RESP, pb.ERROR_CODE_FREE_TIME_IS_OVER)
		return
	}
	cd := int64(gameConfig.GetFreeStaminaClaimCooldown() * 1000)
	if tool.UnixNowMilli()-player.ExpeditionModel.ExpeditionData.LastDailyFreeStaminaTime <= cd {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_RESP, pb.ERROR_CODE_FREE_STAMINA_IS_CD)
		return
	}
	player.ExpeditionModel.ClaimFreeStamina()
	itemService.AddItem(player, &gameConfig.ItemConfig{
		ID:  enum.STAMINA_ITEM_ID,
		Num: int64(gameConfig.GetFreeStaminaNumb()),
	}, enum.ITEM_CHANGE_REASON_FREE_STAMINA)
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_CLAIM_FREE_STAMINA_RESP, &pb.ExpeditionClaimFreeStaminaResp{
		FreeCDTimeStamp: player.ExpeditionModel.ExpeditionData.LastDailyFreeStaminaTime + cd,
	})
}

func ExpeditionChangeLevelHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionChangeLevelReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	cfg := gameConfig.GetCityDispatchCfg(req.BattlefieldId, req.Level)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if !unlockService.CheckUnlock(cfg.Unlock, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_RESP, pb.ERROR_CODE_BATTLEFIELD_IS_LOCKED)
		return
	}
	pointChange := player.ExpeditionModel.ChangeLevel(req.BattlefieldId, req.Level)
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_LEVEL_RESP, &pb.ExpeditionChangeLevelResp{})
	if len(pointChange) > 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_CHANGE_PUSH, &pb.ExpeditionChangePush{
			Points: pointChange,
		})
	}
}

func ExpeditionSpeedUpHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionSpeedUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	slot := player.ExpeditionModel.GetSlotById(req.SlotId)
	if slot == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	if slot.StartTime == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	lossTime := int64(0)
	for itemId, num := range req.Items {
		itemCfg := gameConfig.GetItemCfg(itemId)
		if itemCfg == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		itemInfo := &gameConfig.ItemConfig{
			ID:  itemId,
			Num: int64(num),
		}
		ok, err := itemService.CheckItemCount(player, itemInfo)
		if !ok || err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		lossTime += int64(itemCfg.TargetId*1000) * itemInfo.Num
		err = itemService.RemoveItem(player, itemInfo, enum.ITEM_CHANGE_REASON_ARCHITECTURE)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
	}
	slot = player.ExpeditionModel.SpeedUpSlot(req.SlotId, lossTime)

	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_SPEED_UP_RESP, &pb.ExpeditionSpeedUpResp{
		Slot: &pb.ExpeditionSlotInfo{
			SlotId:        slot.SlotId,
			Status:        1,
			BattlefieldId: slot.BattlefieldId,
			PointId:       slot.PointId,
			OverTime:      slot.EndTime,
		},
	})
}

func CancelSlotHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionCancelReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CANCEL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	slot := player.ExpeditionModel.GetSlotById(req.SlotId)
	if slot == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CANCEL_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if slot.StartTime <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CANCEL_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	monsterId := int32(0)
	point := player.ExpeditionModel.GetPointInfo(slot.BattlefieldId, slot.PointId)
	if point != nil {
		monsterId = point.MonsterId
	}
	slot = player.ExpeditionModel.CancelSlot(req.SlotId)
	if slot == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_CANCEL_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_CANCEL_RESP, &pb.ExpeditionCancelResp{
		Slot: &pb.ExpeditionSlotInfo{
			SlotId: slot.SlotId,
		},
	})
	operationLogService.OnUserExpeditionStart(player.GetUserId(), monsterId)
}

func FinishSlotHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionFinishReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	slot := player.ExpeditionModel.GetSlotById(req.SlotId)
	if slot == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if slot.StartTime <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	currentTime := tool.UnixNowMilli()
	if currentTime > slot.EndTime {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_SLOT_IS_OVER)
		return
	}
	point := player.ExpeditionModel.GetPointInfo(slot.BattlefieldId, slot.PointId)
	if point == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	leftMin := math.Ceil(float64(slot.EndTime-currentTime) / 1000 / 60)
	cost := &gameConfig.ItemConfig{
		ID:  enum.DIAMOND_ITEM_ID, // 钻石
		Num: int64(leftMin) * int64(gameConfig.GetDispatchAccelerationCost()),
	}
	if ok, _ := itemService.CheckItemCount(player, cost); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	_ = itemService.RemoveItem(player, cost, enum.ITEM_CHANGE_REASON_FINISH_EXPEDITION)

	slot = player.ExpeditionModel.FinishSlot(req.SlotId)
	if slot == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_FINISH_RESP, &pb.ExpeditionFinishResp{
		Slot: &pb.ExpeditionSlotInfo{
			SlotId: slot.SlotId,
		},
		MonsterInfos: &pb.ExpeditionPointInfo{
			PointId:         point.PointId,
			MonsterId:       point.MonsterId,
			NextRefreshTime: point.NextRefreshTime,
			IsReward:        1,
		},
	})
	eventBusService.SubmitDispatchKillMonsterEvent(player.GetUserId(), player.GetUserServerId())
}

func StartExpeditionHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ExpeditionStartReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.SlotId <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	slotNum := enum.DefaultExpeditionUnlockedSlots + player.StaticData.GetBuyDispatchFormationNum()
	// 特权解锁派遣队列
	vipCards, err := vipCard.Service.GetAllFunctionValues(player)
	if err == nil {
		if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_FIRST]; ok {
			slotNum++
		}
		if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_SECOND]; ok {
			slotNum++
		}
	}
	if req.SlotId > slotNum {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	slot := player.ExpeditionModel.GetSlotById(req.SlotId)
	if slot == nil {
		slot = player.ExpeditionModel.UnlockSlot(req.SlotId)
	} else if slot.StartTime > 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_SLOT_IS_BUSY)
		return
	}

	pointInfo := player.ExpeditionModel.GetPointInfo(req.BattlefieldId, req.PointId)
	if pointInfo == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_POINT_NOT_EXIST)
		return
	}
	if pointInfo.Status != enum.ExpeditionPointStatusIdle {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_POINT_IS_BUSY)
		return
	}
	cfg := gameConfig.GetCityMonsterCfg(pointInfo.MonsterId)
	if cfg == nil {
		platformLogger.InfoWithUser("city monster cfg is null", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	formation := player.HeroFormationModel.GetHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_EXPEDITION), slot.SlotId)
	if formation == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	bfCfg := gameConfig.GetCityDispatchCfg(req.BattlefieldId, pointInfo.Level)
	if bfCfg == nil {
		platformLogger.ErrorWithUser("Invalid battlefieldId", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if ok, _ := itemService.CheckItemCount(player, cfg.Energy); !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, pb.ERROR_CODE_ENERGY_IS_NOT_ENOUGH)
		return
	}
	_ = itemService.RemoveItem(player, cfg.Energy, enum.ITEM_CHANGE_RESON_START_EXPEDITION)

	totalPower := float64(0)
	for _, heroId := range formation.HeroOwnIDList {
		info := player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroId)
		if info == nil {
			continue
		}
		totalPower += float64(info.Attributes[enum.AttributeBasicCombatPower])
	}
	win := false
	winPercent := (totalPower - float64(cfg.MonsterPower)*0.8) / (float64(cfg.MonsterPower) * 0.2)
	if winPercent >= 0 {
		winPercent = winPercent * winPercent
		if winPercent > 1 {
			win = true
		} else {
			weight := tool.RandInt32(0, 10000)
			if float64(weight) < winPercent*10000 {
				win = true
			}
		}
	}

	items := make([]*pb.ItemBasicInfo, 0)
	reward := make([]*gameConfig.ItemConfig, 0)
	dropItem := gameConfig.Drop(bfCfg.Drop1)
	for _, item := range dropItem {
		items = append(items, &pb.ItemBasicInfo{
			ItemId: item.ID,
			Count:  item.Num,
		})
		reward = append(reward, item)
	}
	if win {
		winDrop := gameConfig.Drop(cfg.DropId)
		for _, item := range winDrop {
			items = append(items, &pb.ItemBasicInfo{
				ItemId: item.ID,
				Count:  item.Num,
			})
			reward = append(reward, item)
		}
	}

	player.ExpeditionModel.StartExpedition(slot, req.BattlefieldId, pointInfo, reward, win)
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_START_RESP, &pb.ExpeditionStartResp{
		Active: &pb.ExpeditionSlotInfo{
			SlotId:        slot.SlotId,
			Status:        1,
			BattlefieldId: req.BattlefieldId,
			PointId:       pointInfo.PointId,
			OverTime:      slot.EndTime,
		},
	})
	operationLogService.OnUserExpeditionStart(player.GetUserId(), pointInfo.MonsterId)
}

func ExpeditionInfoHandler(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.ExpeditionInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player.ExpeditionModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	slotNum := enum.DefaultExpeditionUnlockedSlots + player.StaticData.GetBuyDispatchFormationNum()
	// 特权解锁派遣队列
	vipCards, err := vipCard.Service.GetAllFunctionValues(player)
	if err == nil {
		if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_FIRST]; ok {
			slotNum++
		}
		if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_SECOND]; ok {
			slotNum++
		}
	}
	err = player.ExpeditionModel.CheckExpeditionUnlock(slotNum)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EXPEDITION_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	resp := buildExpeditionInfoResp(player, slotNum)
	messageSender.SendMessage(player, pb.MESSAGE_ID_EXPEDITION_INFO_RESP, resp)
}

func buildExpeditionInfoResp(player *model.PlayerModel, slotNum int32) *pb.ExpeditionInfoResp {
	expeditionModel := player.ExpeditionModel

	resp := &pb.ExpeditionInfoResp{
		Battlefields: make([]*pb.ExpeditionBattlefieldInfo, 0),
		Slots:        make([]*pb.ExpeditionSlotInfo, 0),
	}
	for _, battlefield := range expeditionModel.BattlefieldEntities {
		if battlefield == nil {
			continue
		}
		bfPB := &pb.ExpeditionBattlefieldInfo{
			BattlefieldId:    battlefield.BattlefieldId,
			BattlefieldLevel: battlefield.BattlefieldLevel,
		}
		for _, monsterInfo := range battlefield.PointMonsterInfos {
			point := &pb.ExpeditionPointInfo{
				PointId:         monsterInfo.PointId,
				MonsterId:       monsterInfo.MonsterId,
				NextRefreshTime: monsterInfo.NextRefreshTime,
			}
			point.IsReward = 0
			if monsterInfo.Status == enum.ExpeditionPointStatusReward {
				point.IsReward = 1
			}
			bfPB.MonsterInfos = append(bfPB.MonsterInfos, point)
		}
		resp.Battlefields = append(resp.Battlefields, bfPB)
	}

	for _, slot := range expeditionModel.SlotEntities {
		if slot.SlotId > slotNum {
			continue
		}
		slotPb := &pb.ExpeditionSlotInfo{
			SlotId: slot.SlotId,
		}
		if slot.StartTime > 0 {
			slotPb.BattlefieldId = slot.BattlefieldId
			slotPb.PointId = slot.PointId
			slotPb.OverTime = slot.EndTime
			slotPb.Status = 1
		}
		resp.Slots = append(resp.Slots, slotPb)
	}

	resp.FreeClaimCount = expeditionModel.ExpeditionData.DailyFreeStaminaTimes
	if expeditionModel.ExpeditionData.LastDailyFreeStaminaTime == 0 {
		resp.FreeCDTimeStamp = 0
	} else {
		resp.FreeCDTimeStamp = expeditionModel.ExpeditionData.LastDailyFreeStaminaTime + int64(gameConfig.GetFreeStaminaClaimCooldown())*1000
	}
	resp.LastPowerRecoveryTimeStamp = expeditionModel.ExpeditionData.LastRecoveryStaminaTime
	return resp
}
