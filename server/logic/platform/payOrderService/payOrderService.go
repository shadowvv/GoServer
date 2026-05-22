package payOrderService

import (
	"encoding/json"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

var payOrderIdGenerator *tool.IdGenerator

func InitService(nodeId int32) {
	payOrderIdGenerator = tool.NewIdGenerator(int64(nodeId), int64(enum.ID_GENERATOR_RECHARGE_ORDER))
}

func BuildOrder(player *model.PlayerModel, shopItemId int32, productId int32, price int32, items []*gameConfig.ItemConfig) (int64, error) {
	pickItems := ""
	if len(items) > 0 {
		data, err := json.Marshal(items)
		if err != nil {
			return 0, err
		}
		pickItems = string(data)
	}
	orderId := payOrderIdGenerator.NextId()
	rechargeOrder := &model.RechargeOrderEntity{
		OrderId:     orderId,
		Account:     player.GetUserAccount(),
		UserId:      player.GetUserId(),
		ServerId:    player.GetUserServerId(),
		ShopItemId:  shopItemId,
		ProductId:   productId,
		Price:       price,
		PickedItems: pickItems,
		Status:      int32(enum.RECHARGE_ORDER_STATUS_CREATED),
		CreateTime:  tool.UnixNowMilli(),
		DeliverTime: 0,
		PayTime:     0,
		PayType:     int32(enum.RECHARGE_TYPE_NORMAL),
		ExtraInfo:   "",
	}
	err := easyDB.CreateServerEntity[model.RechargeOrderEntity](rechargeOrder)
	if err != nil {
		return 0, err
	}
	return orderId, nil
}

func BuildTestOrder(player *model.PlayerModel, shopItemId int32, productId int32, price int32, items []*gameConfig.ItemConfig) (int64, error) {
	pickItems := ""
	if len(items) > 0 {
		data, err := json.Marshal(items)
		if err != nil {
			return 0, err
		}
		pickItems = string(data)
	}
	orderId := payOrderIdGenerator.NextId()
	rechargeOrder := &model.RechargeOrderEntity{
		OrderId:     orderId,
		Account:     player.GetUserAccount(),
		UserId:      player.GetUserId(),
		ServerId:    player.GetUserServerId(),
		ShopItemId:  shopItemId,
		ProductId:   productId,
		Price:       price,
		PickedItems: pickItems,
		Status:      int32(enum.RECHARGE_ORDER_STATUS_DELIVERED),
		CreateTime:  tool.UnixNowMilli(),
		DeliverTime: tool.UnixNowMilli(),
		PayTime:     tool.UnixNowMilli(),
		PayType:     int32(enum.RECHARGE_TYPE_NORMAL),
		ExtraInfo:   "",
	}
	err := easyDB.CreateServerEntity[model.RechargeOrderEntity](rechargeOrder)
	if err != nil {
		return 0, err
	}
	return orderId, nil
}

func BuildTokenOrder(player *model.PlayerModel, shopItemId int32, productId int32, price int32, items []*gameConfig.ItemConfig) (int64, error) {
	pickItems := ""
	if len(items) > 0 {
		data, err := json.Marshal(items)
		if err != nil {
			return 0, err
		}
		pickItems = string(data)
	}
	orderId := payOrderIdGenerator.NextId()
	rechargeOrder := &model.RechargeOrderEntity{
		OrderId:     orderId,
		Account:     player.GetUserAccount(),
		UserId:      player.GetUserId(),
		ServerId:    player.GetUserServerId(),
		ShopItemId:  shopItemId,
		ProductId:   productId,
		Price:       price,
		PickedItems: pickItems,
		Status:      int32(enum.RECHARGE_ORDER_STATUS_DELIVERED),
		CreateTime:  tool.UnixNowMilli(),
		DeliverTime: tool.UnixNowMilli(),
		PayTime:     tool.UnixNowMilli(),
		PayType:     int32(enum.RECHARGE_TYPE_TOKEN),
		ExtraInfo:   "",
	}
	err := easyDB.CreateServerEntity[model.RechargeOrderEntity](rechargeOrder)
	if err != nil {
		return 0, err
	}
	return orderId, nil
}

func BuildFreeOrder(player *model.PlayerModel, shopItemId int32, productId int32, price int32, items []*gameConfig.ItemConfig) (int64, error) {
	pickItems := ""
	if len(items) > 0 {
		data, err := json.Marshal(items)
		if err != nil {
			return 0, err
		}
		pickItems = string(data)
	}
	orderId := payOrderIdGenerator.NextId()
	rechargeOrder := &model.RechargeOrderEntity{
		OrderId:     orderId,
		Account:     player.GetUserAccount(),
		UserId:      player.GetUserId(),
		ServerId:    player.GetUserServerId(),
		ShopItemId:  shopItemId,
		ProductId:   productId,
		Price:       price,
		PickedItems: pickItems,
		Status:      int32(enum.RECHARGE_ORDER_STATUS_DELIVERED),
		CreateTime:  tool.UnixNowMilli(),
		DeliverTime: tool.UnixNowMilli(),
		PayTime:     tool.UnixNowMilli(),
		PayType:     int32(enum.RECHARGE_TYPE_FREE),
		ExtraInfo:   "",
	}
	err := easyDB.CreateServerEntity[model.RechargeOrderEntity](rechargeOrder)
	if err != nil {
		return 0, err
	}
	return orderId, nil
}
