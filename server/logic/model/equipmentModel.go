// File: equipmentModel.go
// Description: 装备系统数据模型定义
// Author: 木村凉太
// Create Time: 2025.11

package model

import (
	"encoding/json"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EquipmentEntity 装备实体
type EquipmentEntity struct {
	EquipmentOwnID int64  `gorm:"column:equipment_own_id;primaryKey"` // 装备唯一ID
	UserID         int64  `gorm:"column:user_id;index"`               // 用户ID
	EquipmentID    int32  `gorm:"column:equipment_id"`                // 装备模板ID
	HeroOwnID      int64  `gorm:"column:hero_own_id;default:0"`       // 穿戴的英雄唯一ID（0表示在仓库）
	SlotType       int32  `gorm:"column:slot_type"`                   // 装备部位类型（1武器 2防具）
	SlotIndex      int32  `gorm:"column:slot_index"`                  // 装备槽位索引（武器0，防具1-5）
	Level          int32  `gorm:"column:level"`                       // 装备等级（固定，不可升级）
	StarLevel      int32  `gorm:"column:star_level;default:0"`        // 装备星级（预留）
	ForgeLevel     int32  `gorm:"column:forge_level;default:0"`       // 装备锻造等级（预留）
	AttributeAffix string `gorm:"column:attribute_affix;type:json"`   // 属性词条（JSON格式）
	SkillAffix     string `gorm:"column:skill_affix;type:json"`       // 技能词条（JSON格式）
	SetID          int32  `gorm:"column:set_id;default:0"`            // 套装ID
	IsLocked       bool   `gorm:"column:is_locked;default:false"`     // 是否锁定
	IsDeleted      bool   `gorm:"column:is_deleted;default:false"`    // 是否删除
	StrongLevel    int32  `gorm:"column:strong_level"`
}

func (e *EquipmentEntity) TableName() string {
	return "equipment"
}

// EquipmentCollectionModel 装备集合模型
type EquipmentCollectionModel struct {
	UserId   int64
	Entities map[int64]*EquipmentEntity       // equipmentOwnID -> 装备实体
	Changed  map[int64]map[string]interface{} // equipmentOwnID -> 字段 -> 新值
	player   *PlayerModel

	pushAddEquip []*pb.EquipmentDetailInfo
}

func NewEquipmentCollectionModel(userId int64, entities map[int64]*EquipmentEntity, player *PlayerModel) *EquipmentCollectionModel {
	return &EquipmentCollectionModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int64]map[string]interface{}),
		player:   player,

		pushAddEquip: make([]*pb.EquipmentDetailInfo, 0),
	}
}

var _ logicCommon.PlayerModelInterface = (*EquipmentCollectionModel)(nil)
var _ logicCommon.HeroAttrInterface = (*EquipmentCollectionModel)(nil)

func (e *EquipmentCollectionModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	if e == nil || e.Entities == nil {
		return 0
	}

	var totalAttr int64 = 0

	// 遍历该英雄穿戴的所有装备
	for _, equipment := range e.Entities {
		if equipment == nil || equipment.IsDeleted {
			continue
		}
		if equipment.HeroOwnID != heroId {
			continue
		}

		// 1. 计算基础属性加成（从装备等级属性配置中获取）
		levelAttrCfg := gameConfig.GetEquipmentLevelAttrCfg(equipment.EquipmentID, equipment.Level)
		if levelAttrCfg != nil {
			for _, attr := range levelAttrCfg.Attributes {
				if attr.AttrID == attrId {
					totalAttr += int64(attr.Value)
				}
			}
		}

		// 2. 计算属性词条加成
		if equipment.AttributeAffix != "" {
			var attributeAffix []struct {
				AffixID   int32 `json:"affixId"`
				AttrID    int32 `json:"attrId"`
				StatValue int32 `json:"statValue"`
			}
			if err := json.Unmarshal([]byte(equipment.AttributeAffix), &attributeAffix); err == nil {
				for _, affix := range attributeAffix {
					if affix.AttrID == attrId {
						totalAttr += int64(affix.StatValue)
					}
				}
			}
		}

		// 3. 套装效果暂时不计算属性加成（当前配置只有技能ID）

		// 4. 强化等级加成（预留，当前配置没有强化属性加成）
		equipmentCfg := gameConfig.GetEquipmentBaseCfg(equipment.EquipmentID)
		if equipmentCfg == nil {
			continue
		}
		equipmentStrongAttrCfgId := GetEquipmentStrongId(equipmentCfg)
		equipmentStrongAttrCfg := gameConfig.GetEquipEnhanceCfg(equipmentStrongAttrCfgId)
		if equipmentStrongAttrCfg != nil {
			for id, attrID := range equipmentStrongAttrCfg.Attr {
				if attrID == attrId {
					totalAttr += int64(equipmentStrongAttrCfg.AttrNum[id] * equipment.StrongLevel)
				}
			}
		}
	}

	return totalAttr
}

func GetEquipmentStrongId(equipmentCfg *gameConfig.EquipmentBaseCfg) int32 {
	//阶数*1000000+品质*100000+星级*10000+类型*10+部位
	return equipmentCfg.Tier*1000000 + equipmentCfg.EquipmentQuality*100000 + equipmentCfg.Star*10000 + equipmentCfg.Type*10 + equipmentCfg.EquipmentSlot
}

func (a *EquipmentCollectionModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记
// 装备变化只影响穿戴该装备的英雄，allDirty=false
func (e *EquipmentCollectionModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(e.Changed) == 0 {
		return []int64{}, false
	}
	res := make(map[int64]bool)
	for equipOwnID := range e.Changed {
		if ent := e.Entities[equipOwnID]; ent != nil && ent.HeroOwnID != 0 {
			res[ent.HeroOwnID] = true
		}
	}
	if len(res) == 0 {
		return []int64{}, false
	}
	ownIDs := make([]int64, 0, len(res))
	for ownID := range res {
		ownIDs = append(ownIDs, ownID)
	}
	return ownIDs, false
}

func (e *EquipmentCollectionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if e.pushAddEquip == nil || len(e.pushAddEquip) == 0 {
		return
	}
	messageSender.SendMessage(e.player, pb.MESSAGE_ID_PUSH_EQUIPMENT_DETAIL, &pb.PushEquipmentDetail{
		Info: e.pushAddEquip,
	})
	e.pushAddEquip = make([]*pb.EquipmentDetailInfo, 0)
}

func (e *EquipmentCollectionModel) GetEquipment(equipmentOwnID int64) *EquipmentEntity {
	return e.Entities[equipmentOwnID]
}

func (e *EquipmentCollectionModel) AddEquipment(equipment *EquipmentEntity) {
	e.Entities[equipment.EquipmentOwnID] = equipment
}

func (e *EquipmentCollectionModel) AddPushEquipInfoForMemory(equipment *pb.EquipmentDetailInfo) {
	e.pushAddEquip = append(e.pushAddEquip, equipment)
}

func (e *EquipmentCollectionModel) getChangedMap(equipmentOwnID int64) map[string]interface{} {
	// 确保 Changed map 已初始化
	if e.Changed == nil {
		e.Changed = make(map[int64]map[string]interface{})
	}
	// 确保该装备的变更 map 已初始化
	if e.Changed[equipmentOwnID] == nil {
		e.Changed[equipmentOwnID] = make(map[string]interface{})
	}
	return e.Changed[equipmentOwnID]
}

// UpdateHeroOwnID 更新装备穿戴的英雄ID
func (e *EquipmentCollectionModel) UpdateHeroOwnID(equipmentOwnID int64, heroOwnID int64) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.HeroOwnID = heroOwnID
		e.getChangedMap(equipmentOwnID)["hero_own_id"] = heroOwnID
	}
}

func (e *EquipmentCollectionModel) UpdateStrongLevel(equipmentOwnID int64, strongLevel int32) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.StrongLevel = strongLevel
		e.getChangedMap(equipmentOwnID)["strong_level"] = strongLevel
	}
}

// UpdateSlotIndex 更新装备槽位索引
func (e *EquipmentCollectionModel) UpdateSlotIndex(equipmentOwnID int64, slotIndex int32) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.SlotIndex = slotIndex
		e.getChangedMap(equipmentOwnID)["slot_index"] = slotIndex
	}
}

// UpdateStarLevel 更新装备星级
func (e *EquipmentCollectionModel) UpdateStarLevel(equipmentOwnID int64, starLevel int32) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.StarLevel = starLevel
		e.getChangedMap(equipmentOwnID)["star_level"] = starLevel
	}
}

// UpdateForgeLevel 更新装备锻造等级
func (e *EquipmentCollectionModel) UpdateForgeLevel(equipmentOwnID int64, forgeLevel int32) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.ForgeLevel = forgeLevel
		e.getChangedMap(equipmentOwnID)["forge_level"] = forgeLevel
	}
}

// UpdateAttributeAffix 更新属性词条
func (e *EquipmentCollectionModel) UpdateAttributeAffix(equipmentOwnID int64, affix string) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.AttributeAffix = affix
		e.getChangedMap(equipmentOwnID)["attribute_affix"] = affix
	}
}

// UpdateSkillAffix 更新技能词条
func (e *EquipmentCollectionModel) UpdateSkillAffix(equipmentOwnID int64, affix string) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.SkillAffix = affix
		e.getChangedMap(equipmentOwnID)["skill_affix"] = affix
	}
}

// UpdateIsLocked 更新锁定状态
func (e *EquipmentCollectionModel) UpdateIsLocked(equipmentOwnID int64, isLocked bool) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.IsLocked = isLocked
		e.getChangedMap(equipmentOwnID)["is_locked"] = isLocked
	}
}

// UpdateIsDeleted 更新删除状态
func (e *EquipmentCollectionModel) UpdateIsDeleted(equipmentOwnID int64, isDeleted bool) {
	if ent := e.Entities[equipmentOwnID]; ent != nil {
		ent.IsDeleted = isDeleted
		e.getChangedMap(equipmentOwnID)["is_deleted"] = isDeleted
	}
}

func (e *EquipmentCollectionModel) SaveModelToDB() {
	if len(e.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerBatchEntities(e.Entities, e.Changed, e.UserId)
	e.Changed = make(map[int64]map[string]interface{})
}

func (e *EquipmentEntity) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("equipment_own_id", e.EquipmentOwnID)
	enc.AddInt64("user_id", e.UserID)
	enc.AddInt32("equipment_id", e.EquipmentID)
	enc.AddInt64("hero_own_id", e.HeroOwnID)
	enc.AddInt32("slot_type", e.SlotType)
	enc.AddInt32("slot_index", e.SlotIndex)
	enc.AddInt32("level", e.Level)
	enc.AddInt32("star_level", e.StarLevel)
	enc.AddInt32("forge_level", e.ForgeLevel)
	enc.AddInt32("set_id", e.SetID)
	enc.AddBool("is_locked", e.IsLocked)
	enc.AddBool("is_deleted", e.IsDeleted)
	return nil
}

// LoadEquipmentBags 从数据库加载装备数据
func LoadEquipmentBags(userId int64, player *PlayerModel) (*EquipmentCollectionModel, error) {
	equipmentMap := make(map[int64]*EquipmentEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[EquipmentEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for i, d := range rows {
		if d == nil {
			continue
		}
		// 软删除：只加载未删除的装备，已删除的装备保留在数据库中但不加载到内存
		if d.IsDeleted {
			logger.InfoWithZapFields("LoadEquipmentBags found deleted equipment (soft delete)", zap.Object("equipment", d))
			rows[i] = nil
			continue
		}
		equipmentMap[d.EquipmentOwnID] = d
	}
	return NewEquipmentCollectionModel(userId, equipmentMap, player), nil
}

// CreateEquipmentModel 创建空的装备模型
func CreateEquipmentModel(userId int64, player *PlayerModel) (*EquipmentCollectionModel, error) {
	equipmentMap := make(map[int64]*EquipmentEntity)
	return NewEquipmentCollectionModel(userId, equipmentMap, player), nil
}
