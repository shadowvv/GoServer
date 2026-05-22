package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/adChest"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("adChest", &AdChestController{})
}

type AdChestController struct{}

var _ LogicControllerInterface = (*AdChestController)(nil)

func (c *AdChestController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_AD_CHEST_LIST_REQ, &pb.GetAdChestListReq{}, GetAdChestListHandle, enum.FUNCTION_ID_AD_CHEST)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_OPEN_AD_CHEST_REQ, &pb.OpenAdChestReq{}, OpenAdChestHandle, enum.FUNCTION_ID_AD_CHEST)
}

func GetAdChestListHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.GetAdChestListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_AD_CHEST_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	adChestModel := adChest.Service.GetOrLoadAdChestModel(player)
	if adChestModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_AD_CHEST_LIST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	currentTime := tool.UnixNowMilli()
	chests := adChestModel.GetAllChests()
	chestList := make([]*pb.AdChestInfo, 0, len(chests))
	for _, e := range chests {
		cfg := gameConfig.GetLimitedAdChestCfg(e.CfgIndex)
		if cfg == nil {
			continue
		}
		expireTime := e.CreateTime + int64(cfg.Duration)*tool.MINUTE_MILLI // Duration 单位：分钟
		if currentTime > expireTime {
			adChestModel.RemoveChest(e.UniqueId) // 过期则移除，不发给客户端
			continue
		}
		chestList = append(chestList, &pb.AdChestInfo{
			UniqueId:   e.UniqueId,
			ItemId:     e.ItemId,
			CfgIndex:   e.CfgIndex,
			CreateTime: e.CreateTime,
			ExpireTime: expireTime,
		})
	}

	todayOpenCount := adChestModel.GetTodayOpenCount(currentTime)
	todayOpenLimit := adChest.Service.GetDailyOpenLimit(player)

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_AD_CHEST_LIST_RESP, &pb.GetAdChestListResp{
		ChestList:      chestList,
		TodayOpenCount: todayOpenCount,
		TodayOpenLimit: todayOpenLimit,
	})
}

func OpenAdChestHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.OpenAdChestReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	// 开启前先记录 chest 的 ItemId（开启后 chest 将被删除，无法再查）
	var chestItemId int32
	if chestModel := adChest.Service.GetOrLoadAdChestModel(player); chestModel != nil {
		if chest := chestModel.GetChest(req.UniqueId); chest != nil {
			chestItemId = chest.ItemId
		}
	}

	items, err := adChest.Service.OpenAdChest(player, req.UniqueId, req.WatchAd)
	if err != nil {
		switch err.Error() {
		case "chest not found":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, pb.ERROR_CODE_AD_CHEST_NOT_FOUND)
		case "chest expired":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, pb.ERROR_CODE_AD_CHEST_EXPIRED)
		case "daily open limit reached":
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, pb.ERROR_CODE_AD_CHEST_OPEN_LIMIT)
		default:
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		}
		return
	}

	// 上报广告宝箱消耗（实体已删除，无数量统计意义，仅记录 chg=-1）
	if chestItemId > 0 {
		itemService.ReportItemChange(player, &gameConfig.ItemConfig{ID: chestItemId, Num: 1}, enum.ITEM_CHANGE_REASON_USE_ITEM, 0, 0, false)
	}

	// 发放奖励
	if len(items) > 0 {
		_ = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_USE_ITEM)
	}

	itemList := make([]*pb.AdChestRewardItem, 0, len(items))
	for _, v := range items {
		itemList = append(itemList, &pb.AdChestRewardItem{
			ItemId: v.ID,
			Count:  v.Num,
		})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_OPEN_AD_CHEST_RESP, &pb.OpenAdChestResp{
		ItemList: itemList,
	})
	eventBusService.SubmitAdChestOpenEvent(player.GetUserId(), 1)
}
