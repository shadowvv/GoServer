// File: privilegeRewardModel.go
// Description: 特权奖励每日领取记录模型
// Author: 木村凉太
// Create Time: 2026.02

package model

import (
	"time"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

// PrivilegeRewardType 特权奖励类型
const (
	PRIVILEGE_REWARD_TYPE_RECRUITMENT int32 = 1 // 招募权益
)

// PrivilegeRewardEntity 特权奖励实体
type PrivilegeRewardEntity struct {
	UserId        int64 `gorm:"column:user_id;primaryKey"`        // 玩家ID
	RewardType    int32 `gorm:"column:reward_type;primaryKey"`    // 奖励类型
	LastClaimTime int64 `gorm:"column:last_claim_time;default:0"` // 上次领取时间（毫秒时间戳）
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (p *PrivilegeRewardEntity) TableName() string {
	return "player_privilege_reward"
}

// PrivilegeRewardModel 特权奖励模型
type PrivilegeRewardModel struct {
	UserId  int64
	Rewards map[int32]*PrivilegeRewardEntity // key: rewardType
	Changed map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*PrivilegeRewardModel)(nil)

// NewPrivilegeRewardModel 创建特权奖励模型
func NewPrivilegeRewardModel(userId int64) *PrivilegeRewardModel {
	return &PrivilegeRewardModel{
		UserId:  userId,
		Rewards: make(map[int32]*PrivilegeRewardEntity),
		Changed: make(map[int32]map[string]interface{}),
	}
}

// LoadPrivilegeRewardModel 从数据库加载特权奖励模型（没有记录会返回空模型）
func LoadPrivilegeRewardModel(userId int64) (*PrivilegeRewardModel, error) {
	db := easyDB.GetPlayerDB()

	var entities []PrivilegeRewardEntity
	if err := db.Where("user_id = ?", userId).Find(&entities).Error; err != nil {
		return nil, err
	}

	m := NewPrivilegeRewardModel(userId)
	for i := range entities {
		ent := entities[i] // copy
		m.Rewards[ent.RewardType] = &ent
	}
	return m, nil
}

// SaveModelToDB 保存模型到数据库
func (p *PrivilegeRewardModel) SaveModelToDB() {
	if len(p.Changed) == 0 {
		return
	}

	db := easyDB.GetPlayerDB()
	tx := db.Begin()
	if tx.Error != nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	for rewardType, changes := range p.Changed {
		// 删除
		if del, ok := changes["__delete"]; ok {
			if b, ok2 := del.(bool); ok2 && b {
				_ = tx.Where("user_id = ? AND reward_type = ?", p.UserId, rewardType).Delete(&PrivilegeRewardEntity{}).Error
				continue
			}
		}
		entity := p.Rewards[rewardType]
		if entity == nil {
			continue
		}
		// upsert：依赖 (user_id, reward_type) 复合主键
		_ = tx.Save(entity).Error
	}

	_ = tx.Commit()
	p.Changed = make(map[int32]map[string]interface{})
}

// Heartbeat 心跳更新（检查每日重置）
func (p *PrivilegeRewardModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 特权奖励模型不需要心跳重置，每日重置逻辑在领取时检查
}

// markChanged 标记字段变更
func (p *PrivilegeRewardModel) markChanged(rewardType int32, field string, value interface{}) {
	if p.Changed[rewardType] == nil {
		p.Changed[rewardType] = make(map[string]interface{})
	}
	p.Changed[rewardType][field] = value
}

// CanClaimReward 检查是否可以领取奖励（每日一次）
func (p *PrivilegeRewardModel) CanClaimReward(rewardType int32, currentTime int64) bool {
	entity := p.Rewards[rewardType]
	if entity == nil {
		// 没有记录，可以领取
		return true
	}
	// 检查是否跨天
	return !isSameDay(entity.LastClaimTime, currentTime)
}

// ClaimReward 领取奖励（更新领取时间）
func (p *PrivilegeRewardModel) ClaimReward(rewardType int32, currentTime int64) {
	entity := p.Rewards[rewardType]
	if entity == nil {
		entity = &PrivilegeRewardEntity{
			UserId:        p.UserId,
			RewardType:    rewardType,
			LastClaimTime: currentTime,
		}
		p.Rewards[rewardType] = entity
	} else {
		entity.LastClaimTime = currentTime
	}
	p.markChanged(rewardType, "user_id", p.UserId)
	p.markChanged(rewardType, "reward_type", rewardType)
	p.markChanged(rewardType, "last_claim_time", currentTime)
}

// isSameDay 检查两个时间戳是否在同一天（使用本地时区）
func isSameDay(timestamp1, timestamp2 int64) bool {
	if timestamp1 == 0 || timestamp2 == 0 {
		return false
	}
	t1 := time.UnixMilli(timestamp1).In(time.Local)
	t2 := time.UnixMilli(timestamp2).In(time.Local)
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}
