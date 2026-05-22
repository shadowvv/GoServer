package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

const (
	AccessoryLuckyTenTimes           = 11
	AccessoryLuckyThirtyTimes        = 35
	AccessoryLuckyFreeTimeUpdateTime = 5 * tool.MINUTE_MILLI
	DailyMaxFreeTimes                = 3
)

func init() {
	RegisterController("Accessory", &AccessoryController{})
}

type AccessoryController struct {
}

var _ LogicControllerInterface = (*AccessoryController)(nil)

func (a *AccessoryController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ACCESSORY_LEVEL_UP_REQ, &pb.AccessoryLevelUpReq{}, AccessoryLevelUpReqHandle, enum.FUNCTION_ID_HERO_ACCESSORY)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ACCESSORY_OPERATION_REQ, &pb.AccessoryOperationReq{}, AccessoryWearableHandle, enum.FUNCTION_ID_HERO_ACCESSORY)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_USER_ACCESSORY_DETAIL_REQ, &pb.GetUserAccessoryDetailReq{}, GetAccessoryDetailHandle, enum.FUNCTION_ID_HERO_ACCESSORY)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_ACCESSORY_LUCKY_DETAIL_REQ, &pb.GetAccessoryLuckyDetailReq{}, GetAccessoryLuckyDetailHandle, enum.FUNCTION_ID_ACCESSORY_LUCKY)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ACCESSORY_LUCKY_REQ, &pb.AccessoryLuckyReq{}, AccessoryLuckyHandle, enum.FUNCTION_ID_ACCESSORY_LUCKY)
}

func AccessoryLevelUpReqHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AccessoryLevelUpReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	levelUpType := req.LevelUpType
	accessoryId := req.AccessoryId

	res := make([]*pb.AccessoryDetail, 0)
	oldLevels := make(map[int32]int32)
	if levelUpType == 0 {
		if player.AccessoryModel.Entities[accessoryId] == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LEVEL_UP_RESP, pb.ERROR_CODE_ACCESSORY_NOT_FOUND)
			return
		}
		oldLevels[accessoryId] = player.AccessoryModel.Entities[accessoryId].AccessoryLevel
		for _, v := range player.AccessoryModel.LevelUp(accessoryId, player.GetLevel()) {
			res = append(res, v)
		}

	} else {
		visited := make(map[int32]bool) // 记录已访问
		// 101是首个饰品
		for detail := gameConfig.GetAccessoryBaseCfg(101); detail != nil; detail = gameConfig.GetAccessoryBaseCfg(detail.NextId) {
			if visited[detail.Id] {
				continue // 发现循环，退出
			}
			visited[detail.Id] = true
			if entity := player.AccessoryModel.Entities[detail.Id]; entity != nil {
				oldLevels[detail.Id] = entity.AccessoryLevel
				v := player.AccessoryModel.LevelUp(entity.AccessoryId, player.GetLevel())
				res = append(res, v...)
			}
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_ACCESSORY_LEVEL_UP_RESP, &pb.AccessoryLevelUpResp{
		InfoList: res,
	})
	for _, detail := range res {
		oldLevel, ok := oldLevels[detail.AccessoryId]
		if !ok {
			continue
		}
		if detail.AccessoryLevel > oldLevel {
			eventBusService.SubmitAccessoryLevelUpEvent(player.GetUserId(), detail.AccessoryId, oldLevel, detail.AccessoryLevel)
		}
	}
}

func AccessoryWearableHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AccessoryOperationReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_OPERATION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	heroOwnId := req.HeroOwnId
	accessoryId := req.AccessoryId
	operationType := req.OperationType

	detail := player.AccessoryModel.Entities[accessoryId]
	res := make([]*pb.AccessoryDetail, 0)
	if detail == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_OPERATION_RESP, pb.ERROR_CODE_ACCESSORY_NOT_FOUND)
		return
	}
	if detail.HeroOwnId == heroOwnId && req.OperationType == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_OPERATION_RESP, pb.ERROR_CODE_ACCESSORY_HERO_ALREADY_WEAR)
		return
	}
	if player.HeroDetailsModel.Entities[heroOwnId] == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_OPERATION_RESP, pb.ERROR_CODE_ACCESSORY_HERO_NOT_FOUND)
		return
	}
	if operationType == 0 {
		pbs := player.AccessoryModel.WearAccessory(accessoryId, heroOwnId, player.GetLevel())
		if len(pbs) > 0 {
			res = append(res, pbs...)
		}
	} else {
		pbs := player.AccessoryModel.UnloadAccessory(accessoryId, heroOwnId, player.GetLevel())
		if len(pbs) > 0 {
			res = append(res, pbs...)
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_ACCESSORY_OPERATION_RESP, &pb.AccessoryOperationResp{
		Info: res,
	})
}

func GetAccessoryDetailHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetUserAccessoryDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_USER_ACCESSORY_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	res := make([]*pb.AccessoryDetail, 0)
	if req.GetGetType() == 2 {
		res = append(res, &pb.AccessoryDetail{
			AccessoryId:    req.GetAccessoryId(),
			AccessoryLevel: 1,
			AccessoryNum:   0,
			HeroOwnId:      0,
			Power:          gameConfig.GetAccessoryPower(req.GetAccessoryId(), 1, player.GetLevel()),
		})
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_USER_ACCESSORY_DETAIL_RESP, &pb.GetUserAccessoryDetailResp{
			InfoList: res,
		})
		return
	}
	if req.AccessoryId == 0 {
		for _, entity := range player.AccessoryModel.Entities {
			res = append(res, &pb.AccessoryDetail{
				AccessoryId:    entity.AccessoryId,
				AccessoryLevel: entity.AccessoryLevel,
				AccessoryNum:   entity.Num - gameConfig.GetAccessoryLevelUpCfg(entity.AccessoryLevel).Sum,
				HeroOwnId:      entity.HeroOwnId,
				Power:          gameConfig.GetAccessoryPower(entity.AccessoryId, entity.AccessoryLevel, player.GetLevel()),
			})
		}
	} else {
		if entity := player.AccessoryModel.Entities[req.AccessoryId]; entity != nil {
			res = append(res, &pb.AccessoryDetail{
				AccessoryId:    entity.AccessoryId,
				AccessoryLevel: entity.AccessoryLevel,
				AccessoryNum:   entity.Num - gameConfig.GetAccessoryLevelUpCfg(entity.AccessoryLevel).Sum,
				HeroOwnId:      entity.HeroOwnId,
				Power:          gameConfig.GetAccessoryPower(entity.AccessoryId, entity.AccessoryLevel, player.GetLevel()),
			})
		}
	}

	resp := &pb.GetUserAccessoryDetailResp{
		InfoList: res,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_USER_ACCESSORY_DETAIL_RESP, resp)
}

func GetAccessoryLuckyDetailHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetAccessoryLuckyDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_ACCESSORY_LUCKY_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	res := make([]*pb.AccessoryLuckyDetail, 0)
	for _, v := range player.AccessoryLuckyModel.Entities {
		todayZeroAMTimestamp := tool.GetTodayZeroByTimeStamp(tool.UnixNowMilli())
		if v.FreeUpdateTime < todayZeroAMTimestamp && todayZeroAMTimestamp < tool.UnixNowMilli() {
			player.AccessoryLuckyModel.UpdateFreeNum(v.LuckyId, 0)
			player.AccessoryLuckyModel.UpdateFreeUpdateTime(v.LuckyId, 0)
		}
		res = append(res, &pb.AccessoryLuckyDetail{
			LuckyId:        v.LuckyId,
			LuckyLevel:     v.LuckyLevel,
			LuckyNum:       v.LuckyNum,
			FreeNum:        v.FreeNum,
			FreeLuckyNum:   min(v.FreeUsedNum+11, 35),
			FreeUpdateTime: v.FreeUpdateTime,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_ACCESSORY_LUCKY_DETAIL_RESP, &pb.GetAccessoryLuckyDetailResp{
		InfoList: res,
	})
}

func AccessoryLuckyHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.AccessoryLuckyReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	luckyId := req.LuckyId
	luckyType := req.LuckyType
	detail := player.AccessoryLuckyModel.Entities[luckyId]
	if detail == nil {
		var err error
		detail, err = player.AccessoryLuckyModel.AddAccessoryLucky(luckyId, 0)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ACCESSORY_DETAIL_NOT_FOUND)
			return
		}
	}
	var luckyNum int32 = 0
	if luckyType == 0 {
		luckyNum = AccessoryLuckyTenTimes
	} else if luckyType == 1 {
		luckyNum = AccessoryLuckyThirtyTimes
	} else {
		if tool.UnixNowMilli()-detail.FreeUpdateTime < AccessoryLuckyFreeTimeUpdateTime {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ACCESSORY_LUCKY_TIME_TOO_SHORT)
			return
		}
		luckyNum = min(detail.FreeUsedNum+AccessoryLuckyTenTimes, AccessoryLuckyThirtyTimes)
	}

	num := detail.LuckyNum
	level := detail.LuckyLevel
	freeNum := detail.FreeNum

	cfgBase := gameConfig.GetLuckyCfg(luckyId)
	if cfgBase == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	itemCost := make([]*gameConfig.ItemConfig, 0)
	if luckyType == 0 || luckyType == 1 {
		itemNum, err := invService.GetItemCount(player.GetUserId(), cfgBase.LuckyCoin)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return

		}
		if itemNum < int64(cfgBase.Num[luckyType]) {
			itemCost = append(itemCost, &gameConfig.ItemConfig{
				ID:  cfgBase.LuckyCoin,
				Num: itemNum,
			})
			itemNeedNum := int64(cfgBase.Num[luckyType]) - itemNum
			flag, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{
				ID:  cfgBase.LuckyCoin2,
				Num: itemNeedNum * int64(cfgBase.Num2),
			})
			if err != nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
				return
			}

			if !flag {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
				return
			}
			itemCost = append(itemCost, &gameConfig.ItemConfig{
				ID:  cfgBase.LuckyCoin2,
				Num: itemNeedNum * int64(cfgBase.Num2),
			})

		} else {
			itemCost = append(itemCost, &gameConfig.ItemConfig{
				ID:  cfgBase.LuckyCoin,
				Num: int64(cfgBase.Num[luckyType]),
			})
		}
	} else {
		if freeNum+1 > DailyMaxFreeTimes {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
	}

	res := make([]*pb.ItemBasicInfo, 0)
	allItems := make([]*gameConfig.ItemConfig, 0)
	for i := int32(1); i <= luckyNum; i++ {
		cfg := gameConfig.GetLuckyDropCfg(luckyId, level)
		if cfg == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		cfgNextLevel := gameConfig.GetLuckyDropCfg(luckyId, level+1)
		itemList := gameConfig.DropGroupItems(cfg.DropGroup, nil)
		num++
		if cfgNextLevel != nil {
			if num >= cfg.Num {
				if cfg.Unlock == 0 || unlockService.CheckUnlock(cfg.Unlock, player) {
					level++
				}
			}
		}
		for _, v := range itemList {
			res = append(res, &pb.ItemBasicInfo{
				ItemId: v.ID,
				Count:  v.Num,
			})
			allItems = append(allItems, v)
		}
	}

	if luckyType == 0 || luckyType == 1 {
		err := itemService.RemoveItems(player, itemCost, enum.ITEM_CHANGE_REASON_ACCESSORY_LUCKY)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
	} else {
		freeNum++
		player.AccessoryLuckyModel.UpdateFreeNum(luckyId, freeNum)
		player.AccessoryLuckyModel.UpdateFreeUpdateTime(luckyId, tool.UnixNowMilli())
		player.AccessoryLuckyModel.UpdateFreeUsedNum(luckyId, detail.FreeUsedNum+1)
	}
	err := itemService.AddItems(player, allItems, enum.ITEM_CHANGE_REASON_ACCESSORY_LUCKY)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}

	player.AccessoryLuckyModel.UpdateLuckyLevel(luckyId, level)
	player.AccessoryLuckyModel.UpdateLuckyNum(luckyId, num)

	messageSender.SendMessage(player, pb.MESSAGE_ID_ACCESSORY_LUCKY_RESP, &pb.AccessoryLuckyResp{
		Info: &pb.AccessoryLuckyDetail{
			LuckyId:        luckyId,
			LuckyLevel:     level,
			LuckyNum:       num,
			FreeNum:        freeNum,
			FreeLuckyNum:   min(detail.FreeUsedNum+11, 35),
			FreeUpdateTime: detail.FreeUpdateTime,
		},
		ItemList: res,
	})

	// 上报饰品幸运值日志
	operationLogService.OnUserAccessoryLucky(player.GetUserId(), luckyNum)

	eventBusService.SubmitLuckyLotteryEvent(player.GetUserId(), "accessory", luckyNum, allItems)
}
