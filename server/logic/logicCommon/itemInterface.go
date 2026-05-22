package logicCommon

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
)

type ItemService interface {
	AddItem(player PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	AddItems(player PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	CheckItemCount(player PlayerInterface, item *gameConfig.ItemConfig) (bool, error)
	CheckItemsCountWithMap(player PlayerInterface, items map[int32]int64) (bool, error)
	CheckItemsCount(player PlayerInterface, items []*gameConfig.ItemConfig) (bool, error)
	RemoveItem(player PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	RemoveItems(player PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	RemoveItemsWithMap(player PlayerInterface, items map[int32]int64, reason enum.ItemChangeReason) error
	ExchangeItem(player PlayerInterface, costItems []*gameConfig.ItemConfig, addItems []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	ResetItems(player PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	ResetItem(player PlayerInterface, items *gameConfig.ItemConfig, reason enum.ItemChangeReason) error
	GetItemCount(player PlayerInterface, itemId int32) int64
}
