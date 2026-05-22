// File: inventoryManager.go
// Description: 背包管理器实现
// Author: 木村凉太
// Create Time: 2025.11

package inventory

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/tool"
)

var InventoryIdGenerator *tool.IdGenerator

func InitInventory() {
	InventoryIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_ITEM))
}

// ItemStack 数据快照,用于client、rpc、tmp data
type ItemStack struct {
	ItemId  int32
	ItemNum int64
}

// InventoryManager 背包管理器
type InventoryManager struct {
	UserId       int64
	Items        map[int32]*PlayerInventoryEntity // key: itemId, value: item (按物品唯一ID存储)
	ChangedItems map[int32]*PlayerInventoryEntity // 变更的物品
}

// 创建背包管理器
func NewInventoryManager(userId int64) *InventoryManager {
	return &InventoryManager{
		UserId:       userId,
		Items:        make(map[int32]*PlayerInventoryEntity),
		ChangedItems: make(map[int32]*PlayerInventoryEntity),
	}
}

// 添加物品到背包
func (im *InventoryManager) AddItem(itemId int32, itemNum int64) enum.InventoryResult {
	// TODO 读取配置看是否需要创建唯一ID
	inventoryId := int64(itemId)
	// 直接按物品ID查找
	if existingItem, exists := im.Items[itemId]; exists {
		// 物品已存在，直接增加数量（无上限）
		existingItem.ItemNum += itemNum
		im.ChangedItems[itemId] = existingItem
	} else {
		// 创建新物品
		// 生成雪花ID
		// TODO:根据道具类型判断
		if 0 == inventoryId {
			inventoryId = InventoryIdGenerator.NextId()
		}
		newItem := &PlayerInventoryEntity{
			ID:      inventoryId,
			UserId:  im.UserId,
			ItemId:  itemId,
			ItemNum: itemNum,
		}
		im.Items[itemId] = newItem
		im.ChangedItems[itemId] = newItem
	}
	return enum.INVENTORY_RESULT_SUCCESS
}

// 移除物品
func (im *InventoryManager) RemoveItem(uid int32, itemNum int64) enum.InventoryResult {
	item, exists := im.Items[uid]
	if !exists {
		return enum.INVENTORY_RESULT_INVALID_ITEM
	}

	if item.ItemNum < itemNum {
		return enum.INVENTORY_RESULT_INSUFFICIENT_ITEMNUM
	}

	item.ItemNum -= itemNum

	if item.ItemNum <= 0 {
		// 标记为删除（数量为0）
		item.ItemNum = 0
		// 删除物品
		delete(im.Items, uid)
	}

	im.ChangedItems[uid] = item
	return enum.INVENTORY_RESULT_SUCCESS
}

// 获取背包物品列表
func (im *InventoryManager) GetItemList() []*ItemStack {
	result := make([]*ItemStack, 0, len(im.Items))
	for itemId, item := range im.Items {
		result = append(result, &ItemStack{
			ItemId:  itemId,
			ItemNum: item.ItemNum,
		})
	}
	return result
}

// 检查是否有足够物品 - 简化版：直接按物品ID检查数量
func (im *InventoryManager) HasItem(itemId int32, itemNum int64) bool {
	if item, exists := im.Items[itemId]; exists {
		return item.ItemNum >= itemNum
	}
	return false
}

// 获取物品总数 - 简化版：直接按物品ID查找
func (im *InventoryManager) GetItemCount(itemId int32) int64 {
	if item, exists := im.Items[itemId]; exists {
		return item.ItemNum
	}
	return 0
}

// 清除变更标记
func (im *InventoryManager) ClearChanged() {
	// 清除各背包的变更记录
	im.ChangedItems = make(map[int32]*PlayerInventoryEntity)
}
