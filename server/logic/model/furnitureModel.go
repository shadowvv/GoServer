package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

// FurnitureEntity 家具等级数据（对应 city_furniture 表）
// 每个建筑类型+家具类型一条记录
type FurnitureEntity struct {
	UserId        int64 `gorm:"column:user_id;primaryKey"`        // 玩家ID
	BuildingType  int32 `gorm:"column:building_type;primaryKey"`  // 建筑类型（对应 ArchitectureType）
	FurnitureType int32 `gorm:"column:furniture_type;primaryKey"` // 家具类型
	Level         int32 `gorm:"column:level"`                     // 家具等级
}

func (e *FurnitureEntity) TableName() string { return "city_furniture" }

// FurnitureModel 家具数据模型
// 二级索引：building_type -> furniture_type -> 实体
type FurnitureModel struct {
	UserId   int64                                // 玩家ID
	Entities map[int32]map[int32]*FurnitureEntity // 建筑类型 -> 家具类型 -> 实体
	Changed  map[int64]map[string]interface{}     // 复合主键 -> 变更字段
}

var _ logicCommon.PlayerModelInterface = (*FurnitureModel)(nil)

// furnitureCompositeKey 将建筑类型和家具类型编码为一个int64作为复合主键
func furnitureCompositeKey(buildingType, furnitureType int32) int64 {
	return int64(buildingType)<<32 | int64(furnitureType)
}

// NewFurnitureModel 创建家具数据模型实例
func NewFurnitureModel(entities map[int32]map[int32]*FurnitureEntity, userId int64) *FurnitureModel {
	return &FurnitureModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int64]map[string]interface{}),
	}
}

// LoadFurnitureModel 从数据库加载指定玩家的所有家具等级数据
func LoadFurnitureModel(userId int64) (*FurnitureModel, error) {
	if userId == 0 {
		return nil, errors.New("userId is null")
	}
	rows, err := easyDB.GetPlayerEntitiesByWhere[FurnitureEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	entities := make(map[int32]map[int32]*FurnitureEntity)
	for _, v := range rows {
		if entities[v.BuildingType] == nil {
			entities[v.BuildingType] = make(map[int32]*FurnitureEntity)
		}
		entities[v.BuildingType][v.FurnitureType] = v
	}
	return NewFurnitureModel(entities, userId), nil
}

// SaveModelToDB 将所有脏数据写入数据库
func (m *FurnitureModel) SaveModelToDB() {
	for compositeKey, fields := range m.Changed {
		bt := int32(compositeKey >> 32)
		ft := int32(compositeKey & 0xFFFFFFFF)
		if m.Entities[bt] != nil && m.Entities[bt][ft] != nil {
			easyDB.UpdatePlayerEntity(m.Entities[bt][ft], fields, m.UserId)
		}
	}
	m.Changed = make(map[int64]map[string]interface{})
}

// Heartbeat 家具数据不需要心跳结算
func (m *FurnitureModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}

// GetOrCreate 获取指定建筑和家具类型的实体，不存在则创建默认记录（0级）
func (m *FurnitureModel) GetOrCreate(buildingType, furnitureType int32) (*FurnitureEntity, error) {
	if m.Entities[buildingType] != nil {
		if entity, ok := m.Entities[buildingType][furnitureType]; ok {
			return entity, nil
		}
	}
	entity := &FurnitureEntity{
		UserId:        m.UserId,
		BuildingType:  buildingType,
		FurnitureType: furnitureType,
		Level:         0,
	}
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return nil, err
	}
	if m.Entities[buildingType] == nil {
		m.Entities[buildingType] = make(map[int32]*FurnitureEntity)
	}
	m.Entities[buildingType][furnitureType] = entity
	return entity, nil
}

// GetLevel 获取指定家具的当前等级，不存在返回0
func (m *FurnitureModel) GetLevel(buildingType, furnitureType int32) int32 {
	if m == nil || m.Entities[buildingType] == nil {
		return 0
	}
	if entity, ok := m.Entities[buildingType][furnitureType]; ok {
		return entity.Level
	}
	return 0
}

// UpdateLevel 更新指定家具的等级（不存在时自动创建）
func (m *FurnitureModel) UpdateLevel(buildingType, furnitureType int32, level int32) error {
	entity, err := m.GetOrCreate(buildingType, furnitureType)
	if err != nil {
		return err
	}
	entity.Level = level
	key := furnitureCompositeKey(buildingType, furnitureType)
	if m.Changed[key] == nil {
		m.Changed[key] = make(map[string]interface{})
	}
	m.Changed[key]["level"] = level
	return nil
}
