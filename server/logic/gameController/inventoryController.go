package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/inventory"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("inventory", &InventoryController{})
}

type InventoryController struct {
}

var _ LogicControllerInterface = (*InventoryController)(nil)

// 注册背包路由
func (i *InventoryController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_INVENTORY_LIST_REQ, &pb.GetInventoryListReq{}, GetInventoryListHandle, enum.FUNCTION_ID_PACK)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_DISCARD_ITEM_REQ, &pb.DiscardItemReq{}, DiscardItemHandle, enum.FUNCTION_ID_PACK)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_ITEM_COUNT_REQ, &pb.GetItemCountReq{}, GetItemCountHandle, enum.FUNCTION_ID_PACK)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HAS_ITEM_REQ, &pb.HasItemReq{}, HasItemHandle, enum.FUNCTION_ID_PACK)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_USE_ITEMS_REQ, &pb.UseItemsReq{}, UseItemsHandle, enum.FUNCTION_ID_PACK)
}

var invService inventory.InventoryServiceInterface

func InitinvService() {
	invService = inventory.NewInventoryService(sessionManager)
}

func GetInventoryListHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get inventory list", player)

	stacks, err := invService.GetItemList(player.GetUserId())
	if err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_INVENTORY_LIST_RESP, &pb.GetInventoryListResp{
			Result: pb.InventoryResult_INVENTORY_RESULT_SYSTEM_ERROR,
			Items:  []*pb.ItemInfo{},
		})
		return
	}

	items := make([]*pb.ItemInfo, 0, len(stacks))
	for _, s := range stacks {
		items = append(items, &pb.ItemInfo{ItemId: s.ItemId, ItemNum: s.ItemNum, ItemUid: 0})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_INVENTORY_LIST_RESP, &pb.GetInventoryListResp{
		Result: pb.InventoryResult_INVENTORY_RESULT_SUCCESS,
		Items:  items,
	})
}

func DiscardItemHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.DiscardItemReq)
	if !ok {
		return
	}
	platformLogger.InfoWithUser("discard item", player)

	err := itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: req.GetItemId(), Num: int64(req.GetQuantity())}, enum.ITEM_CHANGE_REASON_DISCARD_ITEM)
	if err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_DISCARD_ITEM_RESP, &pb.DiscardItemResp{Result: pb.InventoryResult_INVENTORY_RESULT_SYSTEM_ERROR})
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_DISCARD_ITEM_RESP, &pb.DiscardItemResp{Result: pb.InventoryResult_INVENTORY_RESULT_SUCCESS})
}

func GetItemCountHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetItemCountReq)
	if !ok {
		return
	}
	platformLogger.InfoWithUser("get item count", player)

	count, err := invService.GetItemCount(player.GetUserId(), req.GetItemId())
	if err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_ITEM_COUNT_RESP, &pb.GetItemCountResp{Result: pb.InventoryResult_INVENTORY_RESULT_SYSTEM_ERROR, Count: 0})
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_ITEM_COUNT_RESP, &pb.GetItemCountResp{Result: pb.InventoryResult_INVENTORY_RESULT_SUCCESS, Count: count})
}

func HasItemHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.HasItemReq)
	if !ok {
		return
	}
	platformLogger.InfoWithUser("has item", player)

	has, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: req.GetItemId(), Num: int64(req.GetQuantity())})
	if err != nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_HAS_ITEM_RESP, &pb.HasItemResp{Result: pb.InventoryResult_INVENTORY_RESULT_SYSTEM_ERROR, HasItem: false})
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HAS_ITEM_RESP, &pb.HasItemResp{Result: pb.InventoryResult_INVENTORY_RESULT_SUCCESS, HasItem: has})
}

func toPbResult(r enum.InventoryResult) pb.InventoryResult {
	switch r {
	case enum.INVENTORY_RESULT_SUCCESS:
		return pb.InventoryResult_INVENTORY_RESULT_SUCCESS
	case enum.INVENTORY_RESULT_INVALID_ITEM:
		return pb.InventoryResult_INVENTORY_RESULT_INVALID_ITEM
	case enum.INVENTORY_RESULT_NOT_ENOUGH_ITEMS, enum.INVENTORY_RESULT_INSUFFICIENT_ITEM_NUM:
		return pb.InventoryResult_INVENTORY_RESULT_NOT_ENOUGH_ITEMS
	case enum.INVENTORY_RESULT_INVALID_INVENTORY:
		return pb.InventoryResult_INVENTORY_RESULT_INVENTORY_NOT_FOUND
	default:
		return pb.InventoryResult_INVENTORY_RESULT_SYSTEM_ERROR
	}
}

func UseItemsHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.UseItemsReq)
	if !ok {
		return
	}
	platformLogger.InfoWithUser("has item", player)

	itemCfg := gameConfig.GetItemCfg(req.UseItemId)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("item cfg not found", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	itemType := itemCfg.ShowGroup
	itemList := make([]*gameConfig.ItemConfig, 0)

	flag, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: req.UseItemId, Num: int64(req.UseNum)})
	if err != nil || !flag {
		platformLogger.ErrorWithUser("item count error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	switch itemType {
	case int32(enum.ITEM_TYPE_NORMAL_CHEST):
		for i := int32(0); i < req.UseNum; i++ {
			allItems := gameConfig.Drop(itemCfg.TargetId)
			itemList = append(itemList, allItems...)
		}
	case int32(enum.ITEM_TYPE_PICK_CHEST):
		dropCfg := gameConfig.GetDropCfg(itemCfg.TargetId)
		if dropCfg == nil {
			platformLogger.ErrorWithUser("item cfg not found", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		flag := false
		itemNum := int64(0)
		for _, v := range dropCfg.FixedItem {
			if v.ID == req.ChooseId {
				flag = true
				itemNum = v.Num
				break
			}
		}
		if !flag {
			platformLogger.ErrorWithUser("item cfg not found", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		itemList = append(itemList, &gameConfig.ItemConfig{ID: req.ChooseId, Num: itemNum * int64(req.UseNum)})
	case int32(enum.ITEM_TYPE_APPEARANCE):
		player.AppearanceModel.UseAppearanceItem(req.UseItemId, req.UseNum)
		err := itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: req.UseItemId, Num: int64(req.UseNum)}, enum.ITEM_CHANGE_REASON_USE_ITEM)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
		messageSender.SendMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, &pb.UseItemsResp{})
		return
	}

	err = itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: req.UseItemId, Num: int64(req.UseNum)}, enum.ITEM_CHANGE_REASON_USE_ITEM)
	if err != nil {
		platformLogger.ErrorWithUser("remove item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}
	err = itemService.AddItems(player, itemList, enum.ITEM_CHANGE_REASON_USE_ITEM)
	if err != nil {
		platformLogger.ErrorWithUser("add item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}
	res := make([]*pb.ItemBasicInfo, 0)
	for _, v := range itemList {
		res = append(res, &pb.ItemBasicInfo{
			ItemId: v.ID,
			Count:  v.Num,
		})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_USE_ITEMS_RESP, &pb.UseItemsResp{
		ItemList: res,
	})
}
