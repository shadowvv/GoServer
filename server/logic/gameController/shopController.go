package gameController

import (
	"encoding/json"
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/logic/platform/payOrderService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("shop", &ShopController{})
}

type ShopController struct {
}

var _ LogicControllerInterface = (*ShopController)(nil)

func (p *ShopController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_SHOP_INFO_REQ, &pb.GetShopInfoReq{}, GetShopHandler, enum.FUNCTION_ID_SHOP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BUY_SHOP_ITEM_REQ, &pb.BuyShopItemReq{}, BuyShopItemHandler, enum.FUNCTION_ID_SHOP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_PASS_AWARD_REQ, &pb.GetPassAwardReq{}, GetPassAwardHandler, enum.FUNCTION_ID_NONE)

	RegisterPlayerInnerTask(enum.INNER_MSG_DELIVER_RECHARGE_ITEM, DeliverRechargeItemHandler)

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_REQ, &pb.GetTokenShopInfoReq{}, GetTokenShopHandler, enum.FUNCTION_ID_SHOP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_REQ, &pb.BuyTokenShopItemReq{}, BuyTokenShopItemHandler, enum.FUNCTION_ID_SHOP)
}

func GetTokenShopHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetTokenShopInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	shopCfg := gameConfig.GetTokenShopMainCfg(req.ShopId)
	if shopCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if shopCfg.SystemId != 0 && !unlockService.CheckSystemUnlock(shopCfg.SystemId, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}
	if shopCfg.ActId != 0 {
		if ok, _ := player.PlayerActivityModel.CheckActivityOpen(shopCfg.ActId); !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
	}
	shopItems := make([]*pb.TokenShopItemInfo, 0)
	for _, item := range gameConfig.GetAllTokenShopItem() {
		if item.ShopId != req.ShopId {
			continue
		}
		if item.UnlockShow != 0 && !unlockService.CheckUnlock(item.UnlockShow, player) {
			continue
		}
		if item.UnlockStop != 0 && unlockService.CheckUnlock(item.UnlockStop, player) {
			continue
		}
		shopItem := player.PlayerTokenShopModel.GetItem(item.Id)
		if shopItem == nil && len(item.Discount) > 0 {
			shopItem = player.PlayerTokenShopModel.UnlockItem(item.Id, item)
		}
		if shopItem == nil {
			continue
		}
		if shopCfg.Refresh == int32(enum.ITEM_REFRESH_TYPE_DAY) {
			if tool.GetNatureDayDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
				player.PlayerTokenShopModel.RefreshItem(shopItem)
			}
		} else if shopCfg.Refresh == int32(enum.ITEM_REFRESH_TYPE_WEEK) {
			if tool.GetNatureWeekDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
				player.PlayerTokenShopModel.RefreshItem(shopItem)
			}
		}
		shopItems = append(shopItems, &pb.TokenShopItemInfo{
			ItemId:   shopItem.ShopItemId,
			BuyCount: shopItem.BuyCount,
			Discount: shopItem.Discount,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_TOKEN_SHOP_INFO_RESP, &pb.GetTokenShopInfoResp{
		ItemInfo: shopItems,
	})
}

func BuyTokenShopItemHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.BuyTokenShopItemReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if req.BuyNum <= 0 {
		platformLogger.ErrorWithUser("exchange num <= 0", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	itemCfg := gameConfig.GetTokenShopItemCfg(req.ItemId)
	if itemCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	shopCfg := gameConfig.GetTokenShopMainCfg(itemCfg.ShopId)
	if shopCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if shopCfg.SystemId != 0 && !unlockService.CheckSystemUnlock(shopCfg.SystemId, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_FUNCTION_NOT_OPEN)
		return
	}
	if shopCfg.ActId != 0 {
		if ok, _ := player.PlayerActivityModel.CheckActivityOpen(shopCfg.ActId); !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
	}

	if itemCfg.UnlockShow != 0 && !unlockService.CheckUnlock(itemCfg.UnlockShow, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_LOCKED)
		return
	}
	if itemCfg.UnlockStop != 0 && unlockService.CheckUnlock(itemCfg.UnlockStop, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_LOCKED)
		return
	}

	shopItem := player.PlayerTokenShopModel.GetItem(itemCfg.Id)
	discount := int64(0)
	if shopItem != nil {
		if itemCfg.LimitBuy != 0 && shopItem.BuyCount+req.BuyNum > itemCfg.LimitBuy {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_BUY_LIMIT)
			return
		}
		discount = int64(shopItem.Discount)
	} else {
		if itemCfg.LimitBuy != 0 {
			player.PlayerTokenShopModel.UnlockItem(itemCfg.Id, itemCfg)
		}
		shopItem = player.PlayerTokenShopModel.GetItem(itemCfg.Id)
	}

	exchangeCfg := gameConfig.GetExchangeCfg(itemCfg.ExchangeId)
	if exchangeCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	costItems := make([]*gameConfig.ItemConfig, 0)
	for _, target := range exchangeCfg.ExchangeId {

		num := target.Num * int64(req.BuyNum)
		if discount > 0 {
			num = num * discount
			num = num / 10000
		}

		costItems = append(costItems, &gameConfig.ItemConfig{
			ID:  target.ID,
			Num: num,
		})
	}
	ok, err := itemService.CheckItemsCount(player, costItems)
	if err != nil || !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	addItems := make([]*gameConfig.ItemConfig, 0)
	for _, target := range exchangeCfg.TargetId {
		addItems = append(addItems, &gameConfig.ItemConfig{
			ID:  target.ID,
			Num: target.Num * int64(req.BuyNum),
		})
	}
	err = itemService.ExchangeItem(player, costItems, addItems, enum.ITEM_CHANGE_REASON_EXCHANGE_ITEM)
	if err != nil {
		// 补丁代码，英雄背包满了提示背包已满
		if err.Error() == "hero bag full" {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_LOTTERY_HERO_BAGE_IS_FULL)
			return
		}
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	if shopItem != nil {
		player.PlayerTokenShopModel.BuyItem(shopItem, req.BuyNum)
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_TOKEN_SHOP_ITEM_RESP, &pb.BuyTokenShopItemResp{})
}

func GetShopHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetShopInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SHOP_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	allShopItem := gameConfig.GetAllCfgWithShopType(req.ShopType)
	if allShopItem == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SHOP_INFO_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	allShopInfo := make(map[int32]*pb.ShopInfo)
	passInfos := make([]*pb.PassItemInfo, 0)

	// 刷新
	for _, shopItem := range player.PlayerShopModel.ItemEntities {
		itemCfg := gameConfig.GetStillShopCfg(shopItem.ShopItemId)
		if itemCfg == nil {
			continue
		}
		if itemCfg.ShopType != req.ShopType {
			continue
		}
		if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_DAY) {
			if tool.GetNatureDayDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
				player.PlayerShopModel.UpdateShopItemCount(shopItem.ShopItemId, 0)
			}
			if tool.GetNatureDayDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
				player.PlayerShopModel.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
			}
			continue
		}
		if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_WEEK) {
			if tool.GetNatureWeekDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
				player.PlayerShopModel.UpdateShopItemCount(shopItem.ShopItemId, 0)
			}
			if tool.GetNatureWeekDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
				player.PlayerShopModel.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
			}
			continue
		}
		if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_MONTH) {
			if tool.GetNatureMonthDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
				player.PlayerShopModel.UpdateShopItemCount(shopItem.ShopItemId, 0)
			}
			if tool.GetNatureMonthDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
				player.PlayerShopModel.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
			}
			continue
		}
	}

	for _, item := range allShopItem {

		if item.ActId != 0 {
			if settled, _ := player.PlayerActivityModel.CheckActivitySettled(item.ActId); settled {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_SHOP_INFO_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
				return
			}
		}

		// 检测是否关闭
		result := true
		if len(item.UnlockStop) > 0 {
			unlockStop := true
			for _, unlock := range item.UnlockStop {
				if !unlockService.CheckUnlock(unlock, player) {
					unlockStop = false
					break
				}
			}
			result = !unlockStop
		}
		if !result {
			continue
		}

		status := int32(0)

		currentItemInfo := player.PlayerShopModel.GetShopItemInfo(item.Id)
		if currentItemInfo == nil {
			// 检测是否显示
			result = true
			for _, unlock := range item.UnlockShow {
				if !unlockService.CheckUnlock(unlock, player) {
					result = false
					break
				}
			}
			if !result {
				continue
			}
			// 检测是否可购买
			for _, unlock := range item.UnlockBuy {
				if !unlockService.CheckUnlock(unlock, player) {
					result = false
					break
				}
			}
		}

		// 链式礼包检测
		if result && item.TypeId == int32(enum.ShopItemTypeChain) {
			if item.PrePackage != 0 && player.PlayerShopModel.GetShopItemInfo(item.PrePackage) == nil {
				result = false
			}
		}

		if result {
			// 限时道具需要先解锁，已保存解锁时间
			if (player.PlayerShopModel.GetShopItemInfo(item.Id) == nil || player.PlayerShopModel.GetShopItemInfo(item.Id).UnlockTime == 0) && item.Duration > 0 {
				err := player.PlayerShopModel.UnlockShopItem(item.Id, int64(item.Duration*1000))
				if err != nil {
					platformLogger.ErrorWithUser("unlock shop item error", player, err)
					continue
				}
			}
			status = 1
		}

		// 周卡月卡
		if item.TypeId == int32(enum.ShopItemTypeWeekly) || item.TypeId == int32(enum.ShopItemTypeFirstCharge) {
			passInfo := player.PlayerShopModel.GetShopPassInfo(item.Id)
			passCfg := gameConfig.GetWeeklyPassCfg(item.Id)
			if passInfo != nil && passCfg != nil && passInfo.AcceptCount < passCfg.ValidityPeriod {
				passInfos = append(passInfos, &pb.PassItemInfo{
					ItemId:    item.Id,
					CanAccept: tool.GetNatureDayDistance(tool.UnixNowMilli(), passInfo.LastAcceptTime) >= 1,
					LeftTimes: passCfg.ValidityPeriod - passInfo.AcceptCount,
				})
			}
		}

		itemInfo := &pb.ShopItemInfo{
			ItemId: item.Id,
			Status: status,
		}
		// 获取购买次数和剩余时间
		currentItemInfo = player.PlayerShopModel.GetShopItemInfo(item.Id)
		if currentItemInfo != nil {
			itemInfo.UnlockTime = currentItemInfo.UnlockTime
			itemInfo.BuyCount = currentItemInfo.BuyCount
			if item.Duration > 0 {
				leftTime := int64(item.Duration)*1000 - (tool.UnixNowMilli() - currentItemInfo.UnlockTime)
				if leftTime <= 0 {
					continue
				}
				itemInfo.LeftTime = leftTime
				itemInfo.IsFirstBuy = currentItemInfo.LastBuyTime == 0
			}
		} else {
			itemInfo.BuyCount = 0
		}

		if allShopInfo[item.TypeId] == nil {
			allShopInfo[item.TypeId] = &pb.ShopInfo{
				ShopType:  item.TypeId,
				ItemInfos: make([]*pb.ShopItemInfo, 0),
			}
		}
		allShopInfo[item.TypeId].ItemInfos = append(allShopInfo[item.TypeId].ItemInfos, itemInfo)
	}

	resp := &pb.GetShopInfoResp{
		ShopInfos: allShopInfo,
		PassInfo:  passInfos,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_SHOP_INFO_RESP, resp)
}

func BuyShopItemHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.BuyShopItemReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	itemCfg := gameConfig.GetStillShopCfg(req.Id)
	if itemCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	productCfg := gameConfig.GetProductCfg(itemCfg.ProductId)
	if productCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	if itemCfg.ActId != 0 {
		if settled, _ := player.PlayerActivityModel.CheckActivitySettled(itemCfg.ActId); settled {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
	}

	if len(itemCfg.UnlockStop) > 0 {
		unlockStop := true
		for _, unlock := range itemCfg.UnlockStop {
			if !unlockService.CheckUnlock(unlock, player) {
				unlockStop = false
				break
			}
		}
		if unlockStop {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_CLOSED)
			return
		}
	}

	shopItem := player.PlayerShopModel.GetShopItemInfo(itemCfg.Id)
	if shopItem == nil {
		for _, unlock := range itemCfg.UnlockShow {
			if !unlockService.CheckUnlock(unlock, player) {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_LOCKED)
				return
			}
		}

		for _, unlock := range itemCfg.UnlockBuy {
			if !unlockService.CheckUnlock(unlock, player) {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_LOCKED)
				return
			}
		}
	}

	// 链式礼包检测
	if itemCfg.TypeId == int32(enum.ShopItemTypeChain) {
		if itemCfg.PrePackage != 0 && player.PlayerShopModel.GetShopItemInfo(itemCfg.PrePackage) == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_NOT_BUY_PRE_ITEM)
			return
		}
	}

	if len(itemCfg.DropId) != len(req.ItemIds) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	selfPickItem := make([]*gameConfig.ItemConfig, 0)
	for dropId, itemId := range req.ItemIds {
		find := false
		dropCfg := gameConfig.GetDropCfg(dropId)
		if dropCfg == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		for _, item := range dropCfg.FixedItem {
			if item.ID == itemId {
				find = true
				selfPickItem = append(selfPickItem, item)
				break
			}
		}
		if !find {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
	}

	shopItem = player.PlayerShopModel.GetShopItemInfo(itemCfg.Id)
	if shopItem != nil {
		// 限时道具和购买限制
		if itemCfg.Duration > 0 && tool.UnixNowMilli()-shopItem.UnlockTime >= int64(itemCfg.Duration)*1000 {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_LOCKED)
			return
		}
		if itemCfg.LimitBuy != 0 && shopItem.BuyCount >= itemCfg.LimitBuy {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SHOP_ITEM_BUY_LIMIT)
			return
		}
	} else {
		err := player.PlayerShopModel.UnlockShopItem(itemCfg.Id, int64(itemCfg.Duration*1000))
		if err != nil {
			platformLogger.ErrorWithUser("unlock shop item error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		shopItem = player.PlayerShopModel.GetShopItemInfo(itemCfg.Id)
	}

	var orderId int64 = 0
	// 免费
	if productCfg.ReflectID == 0 {
		var err error = nil
		orderId, err = payOrderService.BuildFreeOrder(player, itemCfg.Id, productCfg.Id, productCfg.Dollar, selfPickItem)
		if err != nil {
			platformLogger.ErrorWithUser("build order error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		buyItemSuccess(orderId, enum.RECHARGE_TYPE_FREE, player, itemCfg, productCfg, selfPickItem, false, "", "", "", false)
		messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, &pb.BuyShopItemResp{})
		return
	}

	// 使用token
	if req.UseToken == 1 {
		if itemCfg.TypeId == int32(enum.ShopItemTypeCoupon) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
		if productCfg.Value == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		result, err := itemService.CheckItemCount(player, productCfg.Value)
		if !result || err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		err = itemService.RemoveItem(player, productCfg.Value, enum.ITEM_CHANGE_REASON_CASH_SHOP_BUY)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		orderId, err = payOrderService.BuildTokenOrder(player, itemCfg.Id, productCfg.Id, productCfg.Dollar, selfPickItem)
		if err != nil {
			platformLogger.ErrorWithUser("build order error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		buyItemSuccess(orderId, enum.RECHARGE_TYPE_TOKEN, player, itemCfg, productCfg, selfPickItem, true, "", "", "", false)
		messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, &pb.BuyShopItemResp{})
		return
	}

	if nodeConfig.Env == enum.ENV_LOCAL || nodeConfig.Env == enum.ENV_DEVELOP || nodeConfig.Env == enum.ENV_TEST {
		var err error = nil
		orderId, err = payOrderService.BuildTestOrder(player, itemCfg.Id, productCfg.Id, productCfg.Dollar, selfPickItem)
		if err != nil {
			platformLogger.ErrorWithUser("build order error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		buyItemSuccess(orderId, enum.RECHARGE_TYPE_NORMAL, player, itemCfg, productCfg, selfPickItem, false, "", "", "", false)
		messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, &pb.BuyShopItemResp{})
	} else {
		// 本地环境和开发环境不会走订单系统
		var err error = nil
		orderId, err = payOrderService.BuildOrder(player, itemCfg.Id, productCfg.Id, productCfg.Dollar, selfPickItem)
		if err != nil {
			platformLogger.ErrorWithUser("build order error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		operationLogService.OnUserBeginPay(player.GetUserId(), itemCfg.Id, productCfg.Dollar)
		messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_SHOP_ITEM_RESP, &pb.BuyShopItemResp{
			OrderId: orderId,
		})
		return
	}
}

func buyItemSuccess(orderId int64, payType enum.RechargeType, player *model.PlayerModel, itemCfg *gameConfig.StillShopCfg, productCfg *gameConfig.ProductCfg, selfPickItem []*gameConfig.ItemConfig, isCoupon bool, payPlatform string, gpOrderId string, payToken string, isSandBox bool) {
	// 整理实际掉落
	dropItems := make([]*gameConfig.ItemConfig, 0)
	// 自选
	dropItems = append(dropItems, selfPickItem...)
	if itemCfg.Diamond != 0 {
		productReflectCfg := gameConfig.GetProductReflectCfg(productCfg.ReflectID)
		if productReflectCfg != nil {
			dropItems = append(dropItems, gameConfig.Drop(productReflectCfg.Diamond)...)
		}
	}

	shopItem := player.PlayerShopModel.GetShopItemInfo(itemCfg.Id)
	if shopItem.LastBuyTime == 0 && itemCfg.FirstBuy != 0 {
		dropItems = append(dropItems, gameConfig.Drop(itemCfg.FirstBuy)...)
	}
	if productCfg.DropId != 0 {
		dropItems = append(dropItems, gameConfig.Drop(productCfg.DropId)...)
	}

	player.PlayerShopModel.UpdateShopItemCount(itemCfg.Id, shopItem.BuyCount+1)
	// 掉落
	err := itemService.AddItemsWithPayType(player, dropItems, enum.ITEM_CHANGE_REASON_CASH_SHOP_BUY, payType)
	if err != nil {
		logger.ErrorBySprintf("recharge add items error", player, err)
	}

	if itemCfg.TypeId == int32(enum.ShopItemTypeWeekly) || itemCfg.TypeId == int32(enum.ShopItemTypeFirstCharge) {
		passCfg := gameConfig.GetWeeklyPassCfg(itemCfg.Id)
		if passCfg != nil {
			player.PlayerShopModel.ResetPassInfo(itemCfg.Id)
		} else {
			platformLogger.ErrorWithUser("pass config is nil", player, nil)
		}
	}

	if payType == enum.RECHARGE_TYPE_NORMAL {
		player.User.UpdateLastChargeTime(tool.UnixNowMilli())
		player.User.UpdateChargeCount(player.User.GetChargeCount() + productCfg.Dollar)
	}
	operationLogService.OnUserPaySuccess(player.GetUserId(), itemCfg.Id, productCfg.Dollar)
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ORDER_INFO, &pb.PushOrderInfo{
		Id:          itemCfg.Id,
		OrderId:     orderId,
		IsCoupon:    isCoupon,
		PayPlatform: payPlatform,
		GpOrderId:   gpOrderId,
		Token:       payToken,
		IsSandbox:   isSandBox,
	})
}

func RetryDeliverFieldItemHandler(player *model.PlayerModel) {
	if player == nil {
		return
	}
	orderEntities, err := easyDB.GetServerEntitiesByWhere[model.RechargeOrderEntity](map[string]interface{}{
		"user_id":   player.GetUserId(),
		"account":   player.GetUserAccount(),
		"server_id": player.GetUserServerId(),
		"status":    int32(enum.RECHARGE_ORDER_STATUS_FAILED),
	})
	if err != nil {
		platformLogger.ErrorWithUser("[recharge] query failed orders error", player, err)
		return
	}
	for _, orderEntity := range orderEntities {
		if orderEntity == nil {
			continue
		}
		itemCfg := gameConfig.GetStillShopCfg(orderEntity.ShopItemId)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("[recharge] item config is nil", player, nil)
			continue
		}
		productCfg := gameConfig.GetProductCfg(orderEntity.ProductId)
		if productCfg == nil {
			platformLogger.ErrorWithUser("[recharge] product config is nil", player, nil)
			continue
		}
		pickItems := make([]*gameConfig.ItemConfig, 0)
		if orderEntity.PickedItems != "" {
			err = json.Unmarshal([]byte(orderEntity.PickedItems), &pickItems)
			if err != nil {
				platformLogger.ErrorWithUser("[recharge] json unmarshal error", player, err)
				continue
			}
		}
		buyItemSuccess(orderEntity.OrderId, enum.RechargeType(orderEntity.PayType), player, itemCfg, productCfg, pickItems, false, orderEntity.PayPlatform, orderEntity.PlatformOrderId, orderEntity.PayToken, orderEntity.IsSandBox == 1)
		orderEntity.Status = int32(enum.RECHARGE_ORDER_STATUS_DELIVERED)
		orderEntity.DeliverTime = tool.UnixNowMilli()
		err = easyDB.UpdateServerEntity[model.RechargeOrderEntity](orderEntity, map[string]interface{}{
			"status":       orderEntity.Status,
			"deliver_time": orderEntity.DeliverTime,
		})
		if err != nil {
			platformLogger.ErrorWithUser("[recharge] update recharge order error", player, err)
			continue
		}
	}
}

func DeliverRechargeItemHandler(task serviceInterface.InnerTaskInterface) (any, error) {
	innerTask, ok := task.(*dispatcherService.InnerTask)
	if !ok {
		return nil, fmt.Errorf("invalid task type")
	}
	req, ok := innerTask.ReqParameter.(*rpcPb.DeliverRechargeItemReq)
	if !ok {
		return nil, fmt.Errorf("invalid task type")
	}
	p := sessionManager.GetPlayerBasicInfoByUserId(req.UserId)
	if p == nil {
		return nil, fmt.Errorf("player not exist")
	}
	player := p.(*model.PlayerModel)
	orderEntity, err := easyDB.GetServerEntityByWhere[model.RechargeOrderEntity](map[string]interface{}{"order_id": req.OrderId})
	if err != nil {
		logger.ErrorBySprintf("[recharge] db error: %v", err)
		return false, fmt.Errorf("[recharge] db error: %v", err)
	}
	if orderEntity.Status != int32(enum.RECHARGE_ORDER_STATUS_PAYED) {
		logger.ErrorBySprintf("[recharge] recharge order status error: %v", req)
		return false, fmt.Errorf("[recharge] recharge order status error: %v", req)
	}
	itemCfg := gameConfig.GetStillShopCfg(orderEntity.ShopItemId)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("item config is nil", player, nil)
		return false, fmt.Errorf("[recharge] item config is nil")
	}
	productCfg := gameConfig.GetProductCfg(orderEntity.ProductId)
	if productCfg == nil {
		platformLogger.ErrorWithUser("product config is nil", player, nil)
		return false, fmt.Errorf("[recharge] product config is nil")
	}
	pickItems := make([]*gameConfig.ItemConfig, 0)
	if orderEntity.PickedItems != "" {
		err = json.Unmarshal([]byte(orderEntity.PickedItems), &pickItems)
		if err != nil {
			platformLogger.ErrorWithUser("json unmarshal error", player, err)
			return nil, err
		}
	}
	buyItemSuccess(req.OrderId, enum.RechargeType(orderEntity.PayType), player, itemCfg, productCfg, pickItems, false, orderEntity.PayPlatform, orderEntity.PlatformOrderId, orderEntity.PayToken, orderEntity.IsSandBox == 1)
	orderEntity.Status = int32(enum.RECHARGE_ORDER_STATUS_DELIVERED)
	orderEntity.DeliverTime = tool.UnixNowMilli()
	err = easyDB.UpdateServerEntity[model.RechargeOrderEntity](orderEntity, map[string]interface{}{"status": orderEntity.Status, "deliver_time": orderEntity.DeliverTime})
	if err != nil {
		platformLogger.ErrorWithUser("db error", player, err)
		return nil, err
	}
	return true, nil
}

func GetPassAwardHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.GetPassAwardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	shopInfo := player.PlayerShopModel.GetShopItemInfo(req.Id)
	if shopInfo == nil {
		platformLogger.ErrorWithUser("pass not buy", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PASS_NOT_BUY)
		return
	}
	shopItemCfg := gameConfig.GetStillShopCfg(req.Id)
	if shopItemCfg == nil {
		platformLogger.ErrorWithUser("pass not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	for _, unlock := range shopItemCfg.UnlockBuy {
		if !unlockService.CheckUnlock(unlock, player) {
			platformLogger.ErrorWithUser("pass is unlock", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PASS_IS_LOCK)
			return
		}
	}
	passCfg := gameConfig.GetWeeklyPassCfg(req.Id)
	if passCfg == nil {
		platformLogger.ErrorWithUser("pass not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	passInfo := player.PlayerShopModel.GetShopPassInfo(req.Id)
	if passInfo == nil {
		platformLogger.ErrorWithUser("pass not buy", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PASS_NOT_BUY)
		return
	}
	if passInfo.AcceptCount >= passCfg.ValidityPeriod {
		platformLogger.ErrorWithUser("pass already accept", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PASS_ALREADY_ACCEPT)
		return
	}
	canAccept := tool.GetNatureDayDistance(tool.UnixNowMilli(), passInfo.LastAcceptTime)
	if canAccept < 1 {
		platformLogger.ErrorWithUser("pass already accept", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, pb.ERROR_CODE_PASS_ALREADY_ACCEPT)
		return
	}
	if int32(len(passCfg.DropId)) >= passInfo.AcceptCount {
		dropItems := gameConfig.Drop(passCfg.DropId[passInfo.AcceptCount])
		_ = itemService.AddItems(player, dropItems, enum.ITEM_CHANGE_REASON_WEEK_PASS)
	}

	player.PlayerShopModel.UpdateShopPassData(passInfo, tool.UnixNowMilli(), passInfo.AcceptCount+1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_PASS_AWARD_RESP, &pb.GetPassAwardResp{})
}
