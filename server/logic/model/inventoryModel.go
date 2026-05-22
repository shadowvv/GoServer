package model

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
)

// PlayerInventoryEntity 玩家背包实体（主背包，按 item_id 聚合）
type PlayerInventoryEntity struct {
	ID      int64 `gorm:"primaryKey;column:id"`
	UserId  int64 `gorm:"primaryKey;column:user_id;index:idx_user_inventory"`
	ItemId  int32 `gorm:"column:item_id"`
	ItemNum int64 `gorm:"column:item_num;default:1"`
}

func (PlayerInventoryEntity) TableName() string {
	return "player_inventory"
}

type InventoryItemStack struct {
	ItemId  int32
	ItemNum int64
}

// InventoryModel 挂载到 PlayerModel 的背包模型
type InventoryModel struct {
	UserId       int64
	Items        map[int32]*PlayerInventoryEntity
	ChangedItems map[int32]*PlayerInventoryEntity
	NewItems     map[int32]bool
	DeletedItems map[int32]bool
}

var _ logicCommon.PlayerModelInterface = (*InventoryModel)(nil)

func NewInventoryModel(userId int64, items map[int32]*PlayerInventoryEntity) *InventoryModel {
	if items == nil {
		items = make(map[int32]*PlayerInventoryEntity)
	}
	return &InventoryModel{
		UserId:       userId,
		Items:        items,
		ChangedItems: make(map[int32]*PlayerInventoryEntity),
		NewItems:     make(map[int32]bool),
		DeletedItems: make(map[int32]bool),
	}
}

func LoadInventoryModel(userId int64) (*InventoryModel, error) {
	rows, err := easyDB.GetPlayerEntitiesByWhere[PlayerInventoryEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}

	items := make(map[int32]*PlayerInventoryEntity, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		items[row.ItemId] = row
	}

	return NewInventoryModel(userId, items), nil
}

func CreateInventoryModel(userId int64) *InventoryModel {
	return NewInventoryModel(userId, make(map[int32]*PlayerInventoryEntity))
}

func (m *InventoryModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 背包无按心跳结算逻辑
}

func (m *InventoryModel) SaveModelToDB() {
	if len(m.ChangedItems) == 0 {
		return
	}

	for itemId, item := range m.ChangedItems {
		if m.DeletedItems[itemId] {
			_ = easyDB.DeletePlayerEntityByWhere[PlayerInventoryEntity](map[string]interface{}{
				"user_id": m.UserId,
				"item_id": itemId,
			}, m.UserId)
			continue
		}

		if item == nil {
			continue
		}

		if m.NewItems[itemId] {
			copyEntity := *item
			if err := easyDB.CreatePlayerEntity(&copyEntity); err != nil {
				logger.ErrorBySprintf("[inventoryModel] create item failed userId:%d itemId:%d err:%v", m.UserId, itemId, err)
			}
			continue
		}

		easyDB.UpdatePlayerEntity(item, map[string]interface{}{"item_num": item.ItemNum}, m.UserId)
	}

	m.ChangedItems = make(map[int32]*PlayerInventoryEntity)
	m.NewItems = make(map[int32]bool)
	m.DeletedItems = make(map[int32]bool)
}

func (m *InventoryModel) AddItem(itemId int32, itemNum int64) enum.InventoryResult {
	if itemNum <= 0 {
		return enum.INVENTORY_RESULT_INVALID_ITEM
	}

	if existing, exists := m.Items[itemId]; exists {
		existing.ItemNum += itemNum
		m.ChangedItems[itemId] = existing
		return enum.INVENTORY_RESULT_SUCCESS
	}

	newItem := &PlayerInventoryEntity{
		ID:      int64(itemId),
		UserId:  m.UserId,
		ItemId:  itemId,
		ItemNum: itemNum,
	}
	m.Items[itemId] = newItem
	m.ChangedItems[itemId] = newItem

	if m.DeletedItems[itemId] {
		// 同一 tick 内先删后加，视为更新已有记录
		delete(m.DeletedItems, itemId)
		delete(m.NewItems, itemId)
	} else {
		m.NewItems[itemId] = true
	}

	return enum.INVENTORY_RESULT_SUCCESS
}

func (m *InventoryModel) RemoveItem(itemId int32, itemNum int64) enum.InventoryResult {
	if itemNum <= 0 {
		return enum.INVENTORY_RESULT_INVALID_ITEM
	}

	item, exists := m.Items[itemId]
	if !exists {
		return enum.INVENTORY_RESULT_INVALID_ITEM
	}

	if item.ItemNum < itemNum {
		return enum.INVENTORY_RESULT_INSUFFICIENT_ITEMNUM
	}

	item.ItemNum -= itemNum
	if item.ItemNum <= 0 {
		item.ItemNum = 0
		delete(m.Items, itemId)

		if m.NewItems[itemId] {
			// 新增后同 tick 内扣光，不触发任何 DB 写
			delete(m.NewItems, itemId)
			delete(m.ChangedItems, itemId)
			return enum.INVENTORY_RESULT_SUCCESS
		}
		m.DeletedItems[itemId] = true
	}

	m.ChangedItems[itemId] = item
	return enum.INVENTORY_RESULT_SUCCESS
}

func (m *InventoryModel) GetItemList() []*InventoryItemStack {
	result := make([]*InventoryItemStack, 0, len(m.Items))
	for itemId, item := range m.Items {
		result = append(result, &InventoryItemStack{
			ItemId:  itemId,
			ItemNum: item.ItemNum,
		})
	}
	return result
}

func (m *InventoryModel) HasItem(itemId int32, itemNum int64) bool {
	if item, exists := m.Items[itemId]; exists {
		return item.ItemNum >= itemNum
	}
	return false
}

func (m *InventoryModel) GetItemCount(itemId int32) int64 {
	if item, exists := m.Items[itemId]; exists {
		return item.ItemNum
	}
	return 0
}
