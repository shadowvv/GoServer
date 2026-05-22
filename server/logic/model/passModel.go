// File: passModel.go
// Description: 通行证系统数据模型定义
// Author: 木村凉太
// Create Time: 2026.02

package model

import (
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

var _ logicCommon.PlayerModelInterface = (*PassModel)(nil)

// PassProgressEntity 通行证进度实体
type PassProgressEntity struct {
	UserId       int64     `gorm:"column:user_id;primaryKey"` // 玩家ID
	PassId       int32     `gorm:"column:pass_id;primaryKey"` // 通行证ID
	Progress     int32     `gorm:"column:progress"`           // 当前进度
	Level        int32     `gorm:"column:level"`              // 当前等级
	LoopProgress int32     `gorm:"column:loop_progress"`      // 循环积分（满级后多出的积分）
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (p *PassProgressEntity) TableName() string {
	return "player_pass_progress"
}

// PassVipEntity 通行证VIP等级实体
type PassVipEntity struct {
	UserId    int64     `gorm:"column:user_id;primaryKey"` // 玩家ID
	PassId    int32     `gorm:"column:pass_id;primaryKey"` // 通行证ID
	VipLevel  int32     `gorm:"column:vip_level"`          // VIP等级（位运算）：0=免费, 1=档位1, 2=档位2, 3=档位1+2, 4=档位3, ...
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (p *PassVipEntity) TableName() string {
	return "player_pass_vip"
}

// PassRewardEntity 通行证奖励领取记录实体
type PassRewardEntity struct {
	UserId      int64     `gorm:"column:user_id;primaryKey"`      // 玩家ID
	PassId      int32     `gorm:"column:pass_id;primaryKey"`      // 通行证ID
	Level       int32     `gorm:"column:level;primaryKey"`        // 等级
	RewardLevel int32     `gorm:"column:reward_level;primaryKey"` // 奖励档位：0=免费, 1=付费档位1, 2=付费档位2
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (p *PassRewardEntity) TableName() string {
	return "player_pass_reward"
}

// PassDropChoiceEntity 通行证掉落选择记录实体
type PassDropChoiceEntity struct {
	UserId       int64     `gorm:"column:user_id;primaryKey"`      // 玩家ID
	PassId       int32     `gorm:"column:pass_id;primaryKey"`      // 通行证ID
	Level        int32     `gorm:"column:level;primaryKey"`        // 等级
	RewardLevel  int32     `gorm:"column:reward_level;primaryKey"` // 奖励档位
	DropId       int32     `gorm:"column:drop_id;primaryKey"`      // 掉落ID
	ChosenItemId int32     `gorm:"column:chosen_item_id"`          // 选择的道具ID
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (p *PassDropChoiceEntity) TableName() string {
	return "player_pass_drop_choice"
}

// PassModel 通行证模型
type PassModel struct {
	UserId            int64
	ProgressMap       map[int32]*PassProgressEntity     // key: passId
	VipMap            map[int32]*PassVipEntity          // key: passId
	RewardMap         map[string]*PassRewardEntity      // key: "passId_level_rewardLevel"
	DropChoiceMap     map[string]*PassDropChoiceEntity  // key: "passId_level_rewardLevel_dropId"
	ProgressChanged   map[int32]map[string]interface{}  // key: passId
	VipChanged        map[int32]map[string]interface{}  // key: passId
	RewardChanged     map[string]map[string]interface{} // key: "passId_level_rewardLevel"
	DropChoiceChanged map[string]map[string]interface{} // key: "passId_level_rewardLevel_dropId"

	reqPassId int32
}

// NewPassModel 创建通行证模型
func NewPassModel(userId int64) *PassModel {
	return &PassModel{
		UserId:            userId,
		ProgressMap:       make(map[int32]*PassProgressEntity),
		VipMap:            make(map[int32]*PassVipEntity),
		RewardMap:         make(map[string]*PassRewardEntity),
		DropChoiceMap:     make(map[string]*PassDropChoiceEntity),
		ProgressChanged:   make(map[int32]map[string]interface{}),
		VipChanged:        make(map[int32]map[string]interface{}),
		RewardChanged:     make(map[string]map[string]interface{}),
		DropChoiceChanged: make(map[string]map[string]interface{}),

		reqPassId: 0,
	}
}

// LoadPassModel 从数据库加载通行证模型
func LoadPassModel(userId int64) (*PassModel, error) {
	m := NewPassModel(userId)

	// 加载进度数据
	progressRows, err := easyDB.GetPlayerEntitiesByWhere[PassProgressEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, ent := range progressRows {
		if ent != nil {
			m.ProgressMap[ent.PassId] = ent
		}
	}

	// 加载VIP等级数据
	vipRows, err := easyDB.GetPlayerEntitiesByWhere[PassVipEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, ent := range vipRows {
		if ent != nil {
			m.VipMap[ent.PassId] = ent
		}
	}

	// 加载奖励领取记录
	rewardRows, err := easyDB.GetPlayerEntitiesByWhere[PassRewardEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, ent := range rewardRows {
		if ent != nil {
			key := getRewardKey(ent.PassId, ent.Level, ent.RewardLevel)
			m.RewardMap[key] = ent
		}
	}

	// 加载掉落选择记录
	dropChoiceRows, err := easyDB.GetPlayerEntitiesByWhere[PassDropChoiceEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, ent := range dropChoiceRows {
		if ent != nil {
			key := getDropChoiceKey(ent.PassId, ent.Level, ent.RewardLevel, ent.DropId)
			m.DropChoiceMap[key] = ent
		}
	}

	return m, nil
}

// SaveModelToDB 保存模型到数据库
func (p *PassModel) SaveModelToDB() {
	// 保存进度数据
	if len(p.ProgressChanged) > 0 {
		for passId, changes := range p.ProgressChanged {
			entity := p.ProgressMap[passId]
			if entity == nil {
				continue
			}
			// 检查是否是新建（通过检查是否有主键字段）
			if _, hasUserId := changes["user_id"]; hasUserId {
				// 新建
				if err := easyDB.CreatePlayerEntity(entity); err == nil {
					// 创建成功后清除变更记录
					delete(p.ProgressChanged, passId)
				}
			} else {
				// 更新
				easyDB.UpdatePlayerEntity(entity, changes, p.UserId)
			}
		}
		p.ProgressChanged = make(map[int32]map[string]interface{})
	}

	// 保存VIP数据
	if len(p.VipChanged) > 0 {
		for passId, changes := range p.VipChanged {
			entity := p.VipMap[passId]
			if entity == nil {
				continue
			}
			// 检查是否是新建
			if _, hasUserId := changes["user_id"]; hasUserId {
				// 新建
				if err := easyDB.CreatePlayerEntity(entity); err == nil {
					delete(p.VipChanged, passId)
				}
			} else {
				// 更新
				easyDB.UpdatePlayerEntity(entity, changes, p.UserId)
			}
		}
		p.VipChanged = make(map[int32]map[string]interface{})
	}

	// 保存奖励记录（奖励记录只有新建，没有更新）
	if len(p.RewardChanged) > 0 {
		for key := range p.RewardChanged {
			entity := p.RewardMap[key]
			if entity == nil {
				continue
			}
			// 奖励记录总是新建
			if err := easyDB.CreatePlayerEntity(entity); err == nil {
				delete(p.RewardChanged, key)
			}
		}
		p.RewardChanged = make(map[string]map[string]interface{})
	}

	// 保存掉落选择记录（掉落选择记录只有新建，没有更新）
	if len(p.DropChoiceChanged) > 0 {
		for key := range p.DropChoiceChanged {
			entity := p.DropChoiceMap[key]
			if entity == nil {
				continue
			}
			// 掉落选择记录总是新建
			if err := easyDB.CreatePlayerEntity(entity); err == nil {
				delete(p.DropChoiceChanged, key)
			}
		}
		p.DropChoiceChanged = make(map[string]map[string]interface{})
	}
}

// GetOrCreateProgress 获取或创建进度 是否是新的pass
func (p *PassModel) GetOrCreateProgress(passId int32) (*PassProgressEntity, bool) {
	if entity, ok := p.ProgressMap[passId]; ok {
		return entity, false
	}
	entity := &PassProgressEntity{
		UserId:       p.UserId,
		PassId:       passId,
		Progress:     0,
		Level:        0,
		LoopProgress: 0,
	}
	p.ProgressMap[passId] = entity
	p.markProgressChanged(passId, "user_id", p.UserId)
	p.markProgressChanged(passId, "pass_id", passId)

	return entity, true
}

// GetOrCreateVip 获取或创建VIP等级
func (p *PassModel) GetOrCreateVip(passId int32) *PassVipEntity {
	if entity, ok := p.VipMap[passId]; ok {
		return entity
	}
	entity := &PassVipEntity{
		UserId:   p.UserId,
		PassId:   passId,
		VipLevel: 0, // 默认免费档位
	}
	p.VipMap[passId] = entity
	p.markVipChanged(passId, "user_id", p.UserId)
	p.markVipChanged(passId, "pass_id", passId)
	return entity
}

// AddProgress 添加进度
func (p *PassModel) AddProgress(passId int32, progress int32) {
	entity, _ := p.GetOrCreateProgress(passId)
	oldProgress := entity.Progress
	entity.Progress += progress
	if entity.Progress != oldProgress {
		p.markProgressChanged(passId, "progress", entity.Progress)
	}
}

// SetVipLevel 设置VIP等级（直接设置，用于兼容旧代码）
func (p *PassModel) SetVipLevel(passId int32, vipLevel int32) {
	entity := p.GetOrCreateVip(passId)
	entity.VipLevel = vipLevel
	p.markVipChanged(passId, "vip_level", vipLevel)
}

// AddVipLevel 添加VIP档位（位运算）
// level: 档位编号（1=档位1, 2=档位2, 3=档位3, ...）
func (p *PassModel) AddVipLevel(passId int32, level int32) {
	if level <= 0 {
		return
	}
	entity := p.GetOrCreateVip(passId)
	// 位运算：设置对应位为1
	// level=1 -> bit=1 (001)
	// level=2 -> bit=2 (010)
	// level=3 -> bit=4 (100)
	bit := int32(1) << (level - 1)
	oldVipLevel := entity.VipLevel
	entity.VipLevel |= bit
	if entity.VipLevel != oldVipLevel {
		p.markVipChanged(passId, "vip_level", entity.VipLevel)
	}
}

// HasVipLevel 检查是否有某个VIP档位（位运算）
// level: 档位编号（1=档位1, 2=档位2, 3=档位3, ...）
func (p *PassModel) HasVipLevel(passId int32, level int32) bool {
	if level <= 0 {
		return false
	}
	entity := p.GetOrCreateVip(passId)
	// 位运算：检查对应位是否为1
	bit := int32(1) << (level - 1)
	return (entity.VipLevel & bit) != 0
}

// HasReceivedReward 检查是否已领取奖励
func (p *PassModel) HasReceivedReward(passId int32, level int32, rewardLevel int32) bool {
	key := getRewardKey(passId, level, rewardLevel)
	_, ok := p.RewardMap[key]
	return ok
}

// AddRewardRecord 添加奖励领取记录
func (p *PassModel) AddRewardRecord(passId int32, level int32, rewardLevel int32) {
	key := getRewardKey(passId, level, rewardLevel)
	entity := &PassRewardEntity{
		UserId:      p.UserId,
		PassId:      passId,
		Level:       level,
		RewardLevel: rewardLevel,
	}
	p.RewardMap[key] = entity
	p.markRewardChanged(key, "user_id", p.UserId)
	p.markRewardChanged(key, "pass_id", passId)
	p.markRewardChanged(key, "level", level)
	p.markRewardChanged(key, "reward_level", rewardLevel)
}

// GetDropChoice 获取掉落选择
func (p *PassModel) GetDropChoice(passId int32, level int32, rewardLevel int32, dropId int32) *PassDropChoiceEntity {
	key := getDropChoiceKey(passId, level, rewardLevel, dropId)
	return p.DropChoiceMap[key]
}

// SetDropChoice 设置掉落选择
func (p *PassModel) SetDropChoice(passId int32, level int32, rewardLevel int32, dropId int32, chosenItemId int32) {
	key := getDropChoiceKey(passId, level, rewardLevel, dropId)
	entity := &PassDropChoiceEntity{
		UserId:       p.UserId,
		PassId:       passId,
		Level:        level,
		RewardLevel:  rewardLevel,
		DropId:       dropId,
		ChosenItemId: chosenItemId,
	}
	p.DropChoiceMap[key] = entity
	p.markDropChoiceChanged(key, "user_id", p.UserId)
	p.markDropChoiceChanged(key, "pass_id", passId)
	p.markDropChoiceChanged(key, "level", level)
	p.markDropChoiceChanged(key, "reward_level", rewardLevel)
	p.markDropChoiceChanged(key, "drop_id", dropId)
	p.markDropChoiceChanged(key, "chosen_item_id", chosenItemId)
}

// UpdateLevel 更新等级（根据进度计算）
func (p *PassModel) UpdateLevel(passId int32, newLevel int32) {
	entity, _ := p.GetOrCreateProgress(passId)
	entity.Level = newLevel
	p.markProgressChanged(passId, "level", newLevel)
}

// GetLoopProgress 获取循环积分
func (p *PassModel) GetLoopProgress(passId int32) int32 {
	entity, _ := p.GetOrCreateProgress(passId)
	return entity.LoopProgress
}

// AddLoopProgress 添加循环积分
func (p *PassModel) AddLoopProgress(passId int32, progress int32) {
	entity, _ := p.GetOrCreateProgress(passId)
	entity.LoopProgress += progress
	p.markProgressChanged(passId, "loop_progress", entity.LoopProgress)
}

// ConsumeLoopProgress 消耗循环积分
func (p *PassModel) ConsumeLoopProgress(passId int32, progress int32) bool {
	entity, _ := p.GetOrCreateProgress(passId)
	if entity.LoopProgress < progress {
		return false
	}
	entity.LoopProgress -= progress
	p.markProgressChanged(passId, "loop_progress", entity.LoopProgress)
	return true
}

// Heartbeat 心跳更新（通行证暂时不需要特殊处理）
func (p *PassModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 通行证暂时不需要心跳处理
}

// markProgressChanged 标记进度字段变更
func (p *PassModel) markProgressChanged(passId int32, field string, value interface{}) {
	if p.ProgressChanged[passId] == nil {
		p.ProgressChanged[passId] = make(map[string]interface{})
	}
	p.ProgressChanged[passId][field] = value
}

// markVipChanged 标记VIP字段变更
func (p *PassModel) markVipChanged(passId int32, field string, value interface{}) {
	if p.VipChanged[passId] == nil {
		p.VipChanged[passId] = make(map[string]interface{})
	}
	p.VipChanged[passId][field] = value
}

// markRewardChanged 标记奖励字段变更
func (p *PassModel) markRewardChanged(key string, field string, value interface{}) {
	if p.RewardChanged[key] == nil {
		p.RewardChanged[key] = make(map[string]interface{})
	}
	p.RewardChanged[key][field] = value
}

// markDropChoiceChanged 标记掉落选择字段变更
func (p *PassModel) markDropChoiceChanged(key string, field string, value interface{}) {
	if p.DropChoiceChanged[key] == nil {
		p.DropChoiceChanged[key] = make(map[string]interface{})
	}
	p.DropChoiceChanged[key][field] = value
}

func (p *PassModel) SetReqPassId(reqPassId int32) {
	p.reqPassId = reqPassId
}

func (p *PassModel) GetReqPassId() int32 {
	return p.reqPassId
}

// 辅助函数
func getRewardKey(passId int32, level int32, rewardLevel int32) string {
	return fmt.Sprintf("reward_%d_%d_%d", passId, level, rewardLevel)
}

func getDropChoiceKey(passId int32, level int32, rewardLevel int32, dropId int32) string {
	return fmt.Sprintf("drop_%d_%d_%d_%d", passId, level, rewardLevel, dropId)
}
