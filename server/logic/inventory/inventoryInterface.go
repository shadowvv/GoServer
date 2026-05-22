package inventory

import (
	"github.com/drop/GoServer/server/enum"
)

// InventoryServiceInterface 背包服务接口
type InventoryServiceInterface interface {
	// AddItem 添加物品到指定背包
	// userId: 用户ID, itemId: 物品ID, quantity: 数量, invType: 背包类型
	// 返回操作结果和错误信息
	AddItem(userId int64, itemId int32, quantity int64) (enum.InventoryResult, error)

	// RemoveItem 移除指定物品
	// userId: 用户ID, itemId: 物品ID, quantity: 移除数量, invType: 背包类型
	// 返回操作结果和错误信息
	RemoveItem(userId int64, itemId int32, quantity int64) (enum.InventoryResult, error)

	// 查询操作
	// GetItemList 获取指定背包类型的物品列表
	// userId: 用户ID, invType: 背包类型
	// 返回物品堆叠信息列表和错误信息
	GetItemList(userId int64) ([]*ItemStack, error)

	// GetItemCount 获取指定物品在背包中的总数量
	// userId: 用户ID, itemId: 物品ID, invType: 背包类型
	// 返回物品总数量和错误信息
	GetItemCount(userId int64, itemId int32) (int64, error)

	// HasItem 检查背包中是否有足够数量的指定物品
	// userId: 用户ID, itemId: 物品ID, quantity: 需要检查的数量, invType: 背包类型
	// 返回是否拥有足够数量和错误信息
	HasItem(userId int64, itemId int32, quantity int64) (bool, error)

	// 批量操作
	// AddItems 批量添加多个物品到背包
	// userId: 用户ID, items: 物品ID到数量的映射, invType: 背包类型
	// 返回每个物品ID对应的操作结果和错误信息
	AddItems(userId int64, items map[int32]int64) (map[int32]enum.InventoryResult, error)

	// RemoveItems 批量移除多个物品
	// userId: 用户ID, items: 物品ID到数量的映射, invType: 背包类型
	// 返回每个物品ID对应的操作结果和错误信息
	RemoveItems(userId int64, items map[int32]int64) (map[int32]enum.InventoryResult, error)
}
