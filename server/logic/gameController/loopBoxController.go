package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("LoopBox", &LoopBoxController{})
}

type LoopBoxController struct {
}

var _ LogicControllerInterface = (*LoopBoxController)(nil)

func (l *LoopBoxController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LOOP_LEVEL_UP_REQ, &pb.LoopLevelUpReq{}, LoopLevelUpReqHandle, enum.FUNCTION_ID_CALL_BOX)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_LOOP_BOX_DETAIL_INFO_REQ, &pb.GetLoopBoxDetailInfoReq{}, GetLoopBoxDetailInfoHandle, enum.FUNCTION_ID_CALL_BOX)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LOOP_POINT_REQ, &pb.LoopPointReq{}, LoopPointHandle, enum.FUNCTION_ID_CALL_BOX)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_OPEN_LOOP_BOX_REQ, &pb.OpenLoopBoxReq{}, OpenLoopBoxHandle, enum.FUNCTION_ID_CALL_BOX)
}

func LoopLevelUpReqHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Loop box level req", player)

	_, ok := message.(*pb.LoopLevelUpReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	nextLevelCfg := gameConfig.GetLevelCfg(player.LoopBoxModel.LoopBoxEntity.SystemLevel + 1)
	if nextLevelCfg == nil {
		platformLogger.InfoWithUser("Loop box level max", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_LEVEL_UP_RESP, pb.ERROR_CODE_LOOP_BOX_LEVEL_UP_MAX)
		return
	}
	cfg := gameConfig.GetLevelCfg(player.LoopBoxModel.LoopBoxEntity.SystemLevel)
	if cfg.Unlock != 0 && !unlockService.CheckUnlock(cfg.Unlock, player) {
		platformLogger.InfoWithUser("Loop box unLock not full", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_LEVEL_UP_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	oldLevel := player.LoopBoxModel.LoopBoxEntity.SystemLevel
	if player.LoopBoxModel.LoopBoxEntity.SystemEx >= cfg.LevelUpExp {
		player.LoopBoxModel.UpdateSystemEx(player.LoopBoxModel.LoopBoxEntity.SystemEx - cfg.LevelUpExp)
		player.LoopBoxModel.UpdateSystemLevel(player.LoopBoxModel.LoopBoxEntity.SystemLevel + 1)
	} else {
		platformLogger.InfoWithUser("Loop box level up ex not enough", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_LEVEL_UP_RESP, pb.ERROR_CODE_LOOP_BOX_EX_NOT_ENOUGH)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOOP_LEVEL_UP_RESP, &pb.LoopLevelUpResp{
		LoopLevel: player.LoopBoxModel.LoopBoxEntity.SystemLevel,
		LoopEx:    player.LoopBoxModel.LoopBoxEntity.SystemEx,
	})
	if player.LoopBoxModel.LoopBoxEntity.SystemLevel > oldLevel {
		eventBusService.SubmitLoopBoxLevelUpEvent(player.GetUserId(), player.GetUserServerId(), oldLevel, player.LoopBoxModel.LoopBoxEntity.SystemLevel, player.LoopBoxModel.LoopBoxEntity.SystemEx)
	}
}

func GetLoopBoxDetailInfoHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Loop box get detail req", player)

	_, ok := message.(*pb.GetLoopBoxDetailInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOOP_BOX_DETAIL_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOOP_BOX_DETAIL_INFO_RESP, &pb.GetLoopBoxDetailInfoResp{
		SystemEx:    player.LoopBoxModel.LoopBoxEntity.SystemEx,
		SystemLevel: player.LoopBoxModel.LoopBoxEntity.SystemLevel,
		SystemPoint: player.LoopBoxModel.LoopBoxEntity.SystemPoint,
		BoxList:     player.LoopBoxModel.LoopBoxEntity.BoxList,
		LoopId:      player.LoopBoxModel.LoopBoxEntity.LoopId,
	})
}

func LoopPointHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Loop box point req", player)

	_, ok := message.(*pb.LoopPointReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_POINT_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	loopId := player.LoopBoxModel.LoopBoxEntity.LoopId
	point := player.LoopBoxModel.LoopBoxEntity.SystemPoint
	boxList := player.LoopBoxModel.LoopBoxEntity.BoxList
	changeList := make([]int32, len(boxList))
	for i := player.LoopBoxModel.LoopBoxEntity.LoopId; ; {
		cfg := gameConfig.GetProgressRewardCfg(i)
		if cfg == nil {
			platformLogger.InfoWithUser("loop progress reward cfg is null", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOOP_POINT_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		if point < cfg.BoxLoopPoint {
			break
		}
		point -= cfg.BoxLoopPoint
		changeList[cfg.BoxLoopUnlock-1]++

		if gameConfig.GetProgressRewardCfg(i+1) == nil {
			i = 1
			loopId = i
		} else {
			i++
			loopId++
		}
	}
	player.LoopBoxModel.UpdateLoopId(loopId)
	player.LoopBoxModel.UpdateSystemPoint(point)
	submitItemsList := make([]*gameConfig.ItemConfig, 0)
	for i, v := range changeList {
		boxList[i] += v
		if v > 0 {
			submitItemsList = append(submitItemsList, &gameConfig.ItemConfig{
				ID:  int32(gameConfig.GetLoopBoxItemMap()[int32(i+1)]),
				Num: int64(v),
			})
		}
	}
	// 上报积分兑换获得秘罐（增加）
	for _, si := range submitItemsList {
		if si.ID > 0 && si.Num > 0 {
			itemService.ReportItemChange(player, si, enum.ITEM_CHANGE_REASON_LOOP_BOX_POINT, 0, 0, true)
		}
	}
	eventBusService.SubmitItemCollectEvent(player.GetUserId(), submitItemsList)
	player.LoopBoxModel.UpdateBoxList(boxList)
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOOP_POINT_RESP, &pb.LoopPointResp{
		BoxList: changeList,
		Point:   point,
		LoopId:  loopId,
	})
}

func OpenLoopBoxHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("Open loop box req", player)

	req, ok := message.(*pb.OpenLoopBoxReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_LOOP_BOX_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	boxId := req.BoxId
	num := req.OpenNum
	sysEx := player.LoopBoxModel.LoopBoxEntity.SystemEx
	sysLevel := player.LoopBoxModel.LoopBoxEntity.SystemLevel
	sysPoint := player.LoopBoxModel.LoopBoxEntity.SystemPoint
	boxList := player.LoopBoxModel.LoopBoxEntity.BoxList
	res := make([]*pb.ItemBasicInfo, 0)
	cfg := gameConfig.GetLevelCfg(sysLevel)
	num = min(num, boxList[boxId-1])
	// 上报秘罐消耗（开箱前）
	boxItemId := gameConfig.GetLoopBoxItemMap()[boxId]
	if boxItemId > 0 && num > 0 {
		itemService.ReportItemChange(player, &gameConfig.ItemConfig{ID: boxItemId, Num: int64(num)}, enum.ITEM_CHANGE_REASON_LOOP_BOX_OPEN, 0, 0, false)
	}
	for i := int32(0); i < num; i++ {
		itemList := gameConfig.DropGroupItems(cfg.DropGroupId[5-boxId], nil)
		for _, v := range itemList {
			res = append(res, &pb.ItemBasicInfo{
				ItemId: v.ID,
				Count:  v.Num,
			})
		}
		sysEx += gameConfig.GetBoxPropCfg(boxId).LoopBoxExp
		sysPoint += gameConfig.GetBoxPropCfg(boxId).LoopBoxPoint
		boxList[boxId-1]--
	}
	player.LoopBoxModel.UpdateSystemEx(sysEx)
	player.LoopBoxModel.UpdateBoxList(boxList)
	player.LoopBoxModel.UpdateSystemPoint(min(99999, sysPoint))

	items := make([]*gameConfig.ItemConfig, 0)
	for _, v := range res {
		items = append(items, &gameConfig.ItemConfig{
			ID:  v.ItemId,
			Num: v.Count,
		})
	}
	err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_LOTTERY)
	if err != nil {
		platformLogger.ErrorWithUser("add item error", player, err)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_OPEN_LOOP_BOX_RESP, &pb.OpenLoopBoxResp{
		InfoList:    res,
		BoxNum:      boxList[boxId-1],
		SystemPoint: sysPoint,
		SystemEx:    sysEx,
	})

	// 上报循环宝箱开启日志
	operationLogService.OnUserOpenLoopBox(player.GetUserId(), boxId, num)

	eventBusService.SubmitLuckyLotteryEvent(player.GetUserId(), "loopBox", num, items)
}
