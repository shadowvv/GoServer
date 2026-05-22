// File: idleModel.go
// Description: 挂机奖励系统数据模型定义
// Author: 木村凉太
// Create Time: 2026.02

package model

import (
	"encoding/json"
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"gorm.io/gorm"
)

// IdleEntity 挂机实体
type IdleEntity struct {
	UserID              int64 `gorm:"column:user_id;primaryKey"`               // 用户ID
	IdleLevel           int32 `gorm:"column:idle_level;default:1"`             // 挂机等级
	AccumulatedTime     int32 `gorm:"column:accumulated_time;default:0"`       // 累计挂机时间（秒）
	LastSettleTime      int64 `gorm:"column:last_settle_time;default:0"`       // 上次结算时间（秒时间戳）
	LastClaimTime       int64 `gorm:"column:last_claim_time;default:0"`        // 上次领取时间（秒时间戳）
	QuickClaimCount     int32 `gorm:"column:quick_claim_count;default:0"`      // 今日快速领取次数
	QuickADClaimCount   int32 `gorm:"column:quick_ad_claim_count;default:0"`   // 今日广告快速领取次数
	QuickClaimResetTime int64 `gorm:"column:quick_claim_reset_time;default:0"` // 快速领取次数重置时间（秒时间戳）

	// PendingRewards 待领取奖励（普通领取），用于“预览后不可变更”
	// JSON: [{"itemId":1001,"count":123}, ...]
	PendingRewards string `gorm:"column:pending_rewards;type:json"` // 待领取奖励（JSON）

	// QuickClaimPreviewRewards 快速领取预览奖励，用于“预览后不可变更”
	QuickClaimPreviewRewards string `gorm:"column:quick_claim_preview_rewards;type:json"` // 预览奖励（JSON）
}

func (i *IdleEntity) TableName() string {
	return "idle"
}

// IdleModel 挂机模型
type IdleModel struct {
	UserId  int64
	Entity  *IdleEntity
	Changed map[string]interface{}
}

func NewIdleModel(userId int64, entity *IdleEntity) *IdleModel {
	return &IdleModel{
		UserId:  userId,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

var _ logicCommon.PlayerModelInterface = (*IdleModel)(nil)

func (i *IdleModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 挂机奖励结算逻辑在服务层处理
}

func (i *IdleModel) SaveModelToDB() {
	if len(i.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(i.Entity, i.Changed, i.UserId)
	i.Changed = make(map[string]interface{})
}

func (i *IdleModel) getChangedMap() map[string]interface{} {
	if i.Changed == nil {
		i.Changed = make(map[string]interface{})
	}
	return i.Changed
}

// UpdateIdleLevel 更新挂机等级
func (i *IdleModel) UpdateIdleLevel(level int32) {
	if i.Entity == nil {
		return
	}
	i.Entity.IdleLevel = level
	i.getChangedMap()["idle_level"] = level
}

// UpdateAccumulatedTime 更新累计挂机时间
func (i *IdleModel) UpdateAccumulatedTime(time int32) {
	if i.Entity == nil {
		return
	}
	i.Entity.AccumulatedTime = time
	i.getChangedMap()["accumulated_time"] = time
}

// UpdateLastSettleTime 更新上次结算时间
func (i *IdleModel) UpdateLastSettleTime(time int64) {
	if i.Entity == nil {
		return
	}
	i.Entity.LastSettleTime = time
	i.getChangedMap()["last_settle_time"] = time
}

// UpdateLastClaimTime 更新上次领取时间
func (i *IdleModel) UpdateLastClaimTime(time int64) {
	if i.Entity == nil {
		return
	}
	i.Entity.LastClaimTime = time
	i.getChangedMap()["last_claim_time"] = time
}

// UpdateQuickClaimCount 更新快速领取次数
func (i *IdleModel) UpdateQuickClaimCount(count int32) {
	if i.Entity == nil {
		return
	}
	i.Entity.QuickClaimCount = count
	i.getChangedMap()["quick_claim_count"] = count
}

// UpdateQuickADClaimCount 更新广告快速领取剩余次数
func (i *IdleModel) UpdateQuickADClaimCount(count int32) {
	if i.Entity == nil {
		return
	}
	i.Entity.QuickADClaimCount = count
	i.getChangedMap()["quick_ad_claim_count"] = count
}

// UpdateQuickClaimResetTime 更新快速领取次数重置时间
func (i *IdleModel) UpdateQuickClaimResetTime(time int64) {
	if i.Entity == nil {
		return
	}
	i.Entity.QuickClaimResetTime = time
	i.getChangedMap()["quick_claim_reset_time"] = time
}

// UpdatePendingRewards 更新待领取奖励（JSON）
func (i *IdleModel) UpdatePendingRewards(jsonStr string) {
	if i.Entity == nil {
		return
	}
	i.Entity.PendingRewards = jsonStr
	i.getChangedMap()["pending_rewards"] = jsonStr
}

// ClearPendingRewards 清空待领取奖励
func (i *IdleModel) ClearPendingRewards() {
	i.UpdatePendingRewards("[]")
}

// UpdateQuickClaimPreview 更新快速领取预览
func (i *IdleModel) UpdateQuickClaimPreview(rewardsJSON string) {
	if i.Entity == nil {
		return
	}
	i.Entity.QuickClaimPreviewRewards = rewardsJSON
	i.getChangedMap()["quick_claim_preview_rewards"] = rewardsJSON
}

// ClearQuickClaimPreview 清空快速领取预览
func (i *IdleModel) ClearQuickClaimPreview() {
	i.UpdateQuickClaimPreview("[]")
}

// LoadIdleModel 从数据库加载挂机数据
func LoadIdleModel(userId int64) (*IdleModel, error) {
	entity, err := easyDB.GetPlayerEntityByWhere[IdleEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		// 如果是记录不存在的错误，创建新记录
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity = &IdleEntity{
				UserID:                   userId,
				IdleLevel:                1,
				AccumulatedTime:          0,
				LastSettleTime:           0,
				LastClaimTime:            0,
				QuickClaimCount:          0,
				QuickADClaimCount:        1,
				QuickClaimResetTime:      0,
				PendingRewards:           "[]",
				QuickClaimPreviewRewards: "[]",
			}
			if err := easyDB.CreatePlayerEntity[IdleEntity](entity); err != nil {
				return nil, err
			}
		} else {
			// 其他错误直接返回
			return nil, err
		}
	} else {
		// 兼容旧数据：空字符串时补默认 []
		if entity.PendingRewards == "" {
			entity.PendingRewards = "[]"
		}
		if entity.QuickClaimPreviewRewards == "" {
			entity.QuickClaimPreviewRewards = "[]"
		}
	}
	return NewIdleModel(userId, entity), nil
}

// CreateIdleModel 创建空的挂机模型
func CreateIdleModel(userId int64) (*IdleModel, error) {
	entity := &IdleEntity{
		UserID:                   userId,
		IdleLevel:                1,
		AccumulatedTime:          0,
		LastSettleTime:           0,
		LastClaimTime:            0,
		QuickClaimCount:          0,
		QuickADClaimCount:        1,
		QuickClaimResetTime:      0,
		PendingRewards:           "[]",
		QuickClaimPreviewRewards: "[]",
	}
	if err := easyDB.CreatePlayerEntity[IdleEntity](entity); err != nil {
		return nil, err
	}
	return NewIdleModel(userId, entity), nil
}

// DecodeItemBasicInfoJSON 解析 ItemBasicInfo JSON（用于服务层/迁移）
func DecodeItemBasicInfoJSON(jsonStr string) ([]struct {
	ItemId int32 `json:"itemId"`
	Count  int64 `json:"count"`
}, error) {
	if jsonStr == "" {
		jsonStr = "[]"
	}
	var arr []struct {
		ItemId int32 `json:"itemId"`
		Count  int64 `json:"count"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &arr); err != nil {
		return nil, err
	}
	return arr, nil
}
