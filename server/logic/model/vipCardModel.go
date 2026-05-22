// File: vipCardModel.go
// Description: 特权卡系统数据模型定义
// Author: 木村凉太
// Create Time: 2026.02

package model

import (
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

// VipCardEntity 特权卡实体
type VipCardEntity struct {
	UserId     int64 `gorm:"column:user_id;primaryKey"` // 玩家ID
	ItemId     int32 `gorm:"column:item_id;primaryKey"` // 特权卡配置ID（itemId）
	ExpireTime int64 `gorm:"column:expire_time"`        // 过期时间（Unix时间戳，毫秒）；-1 表示永久
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (v *VipCardEntity) TableName() string {
	return "player_vip_card"
}

// VipCardModel 特权卡模型
type VipCardModel struct {
	UserId   int64
	VipCards map[int32]*VipCardEntity // key: itemId
	Changed  map[int32]map[string]interface{}
}

// NewVipCardModel 创建特权卡模型
func NewVipCardModel(userId int64) *VipCardModel {
	return &VipCardModel{
		UserId:   userId,
		VipCards: make(map[int32]*VipCardEntity),
		Changed:  make(map[int32]map[string]interface{}),
	}
}

// LoadVipCardModel 从数据库加载特权卡模型（没有记录会返回空模型）
func LoadVipCardModel(userId int64) (*VipCardModel, error) {
	entities, err := easyDB.GetPlayerEntitiesByWhere[VipCardEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}

	m := NewVipCardModel(userId)
	for _, ent := range entities {
		if ent != nil {
			m.VipCards[ent.ItemId] = ent
		}
	}
	return m, nil
}

// SaveModelToDB 保存模型到数据库
func (v *VipCardModel) SaveModelToDB() {
	if len(v.Changed) == 0 {
		return
	}

	for itemId, changes := range v.Changed {
		// 删除
		if del, ok := changes["__delete"]; ok {
			if b, ok2 := del.(bool); ok2 && b {
				_ = easyDB.DeletePlayerEntityByWhere[VipCardEntity](
					map[string]interface{}{
						"user_id": v.UserId,
						"item_id": itemId,
					},
					v.UserId,
				)
				continue
			}
		}

		entity := v.VipCards[itemId]
		if entity == nil {
			continue
		}

		// 检查是否是新建（通过检查是否有主键字段）
		if _, hasUserId := changes["user_id"]; hasUserId {
			// 新建：对于复合主键，使用 Save 更安全（自动判断插入或更新）
			// 或者使用 Create，但需要确保所有主键字段都有值
			if err := easyDB.DirectSavePlayerEntityByWhere(entity); err == nil {
				// 保存成功后清除变更记录
				delete(v.Changed, itemId)
			}
		} else {
			// 更新：使用 UpdatePlayerEntity
			easyDB.UpdatePlayerEntity(entity, changes, v.UserId)
		}
	}

	v.Changed = make(map[int32]map[string]interface{})
}

// Heartbeat 心跳更新（检查过期特权卡）
func (v *VipCardModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 检查并清理过期特权卡
	// currentTime 与 ExpireTime 均为毫秒时间戳
	for itemId, card := range v.VipCards {
		// 永久卡不检查过期（expireTime < 0）
		if card.ExpireTime > 0 && card.ExpireTime < currentTime {
			// 标记删除，由 vipCardService.SaveVipCardModel 执行 Delete
			v.markChanged(itemId, "__delete", true)
			delete(v.VipCards, itemId)
		}
	}
}

// markChanged 标记字段变更
func (v *VipCardModel) markChanged(itemId int32, field string, value interface{}) {
	if v.Changed[itemId] == nil {
		v.Changed[itemId] = make(map[string]interface{})
	}
	v.Changed[itemId][field] = value
}

// AddVipCardHours 添加/续期特权卡（hours 单位：小时）
// 规则：
// - hours >= VIP_CARD_PERMANENT_HOURS：置为永久（ExpireTime=-1）
// - 已永久：不变
// - 已非永久：在 max(now, oldExpire) 的基础上续期 hours
func (v *VipCardModel) AddVipCardHours(itemId int32, hours int64) *VipCardEntity {
	now := time.Now().UnixMilli()
	if hours <= 0 {
		return v.VipCards[itemId]
	}

	// 永久阈值
	if hours >= enum.VIP_CARD_PERMANENT_HOURS {
		entity := v.VipCards[itemId]
		if entity == nil {
			// 新建
			entity = &VipCardEntity{UserId: v.UserId, ItemId: itemId, ExpireTime: -1}
			v.VipCards[itemId] = entity
			v.markChanged(itemId, "user_id", v.UserId)
			v.markChanged(itemId, "item_id", itemId)
			v.markChanged(itemId, "expire_time", int64(-1))
		} else {
			// 更新已存在的实体
			entity.ExpireTime = -1
			v.markChanged(itemId, "expire_time", int64(-1))
		}
		return entity
	}

	entity := v.VipCards[itemId]
	if entity == nil {
		// 新建
		entity = &VipCardEntity{UserId: v.UserId, ItemId: itemId}
		v.VipCards[itemId] = entity
		v.markChanged(itemId, "user_id", v.UserId)
		v.markChanged(itemId, "item_id", itemId)
	}
	// 已永久
	if entity.ExpireTime < 0 {
		return entity
	}

	base := now
	if entity.ExpireTime > base {
		base = entity.ExpireTime
	}
	expireTime := base + hours*3600*1000
	entity.ExpireTime = expireTime

	// 更新已存在的实体，只标记变更的字段
	v.markChanged(itemId, "expire_time", expireTime)
	return entity
}

// RemoveVipCard 移除特权卡
func (v *VipCardModel) RemoveVipCard(itemId int32) {
	if _, exists := v.VipCards[itemId]; exists {
		v.markChanged(itemId, "__delete", true)
		delete(v.VipCards, itemId)
	}
}

// GetActiveVipCards 获取所有有效的特权卡（未过期）
func (v *VipCardModel) GetActiveVipCards(currentTime int64) []*VipCardEntity {
	result := make([]*VipCardEntity, 0)
	for _, card := range v.VipCards {
		// 永久卡（expireTime < 0）或未过期的卡
		if card.ExpireTime < 0 || card.ExpireTime >= currentTime {
			result = append(result, card)
		}
	}
	return result
}

// GetVipCardByItemId 根据itemId获取特权卡列表
func (v *VipCardModel) GetVipCardByItemId(itemId int32, currentTime int64) []*VipCardEntity {
	result := make([]*VipCardEntity, 0)
	if card := v.VipCards[itemId]; card != nil {
		// 永久卡或未过期的卡
		if card.ExpireTime < 0 || card.ExpireTime >= currentTime {
			result = append(result, card)
		}
	}
	return result
}

func (v *VipCardModel) GetFunctionValue(privilegeType enum.VipPrivilegeType, currentTime int64) int64 {
	activeCards := v.GetActiveVipCards(currentTime)

	var totalValue int64 = 0
	for _, card := range activeCards {
		// 获取特权卡配置
		vipCardCfg := gameConfig.GetVipCardCfg(card.ItemId)
		if vipCardCfg == nil {
			continue
		}

		// 累加该功能的数值
		if value, ok := vipCardCfg.Functions[privilegeType]; ok {
			totalValue += value
		}
	}

	return totalValue
}
