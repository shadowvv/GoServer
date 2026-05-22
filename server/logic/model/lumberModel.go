package model

import (
	"encoding/json"
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

// LumberEntity 伐木场生产运行时数据（对应 lumber 表）
// 每个建筑类型一条记录，存储当前暂存产物、上次结算时间和派驻英雄列表
type LumberEntity struct {
	UserId       int64               `gorm:"column:user_id;primaryKey"`       // 玩家ID
	BuildingType int32               `gorm:"column:building_type;primaryKey"` // 建筑类型（对应 ArchitectureType）
	Stored       string              `gorm:"column:stored;type:json"`         // 暂存产物JSON，格式 [{itemId,count}]
	LastCalcTime int64               `gorm:"column:last_calc_time"`           // 上次结算时间（毫秒时间戳）
	HeroOwnIds   tool.JSONInt64Slice `gorm:"column:hero_own_ids;type:json"`   // 派驻英雄OwnID列表
}

func (e *LumberEntity) TableName() string { return "lumber" }

// LumberModel 伐木场生产数据模型
// 管理所有生产建筑的运行时数据，支持按建筑类型索引和脏标记
type LumberModel struct {
	UserId   int64                            // 玩家ID
	Entities map[int32]*LumberEntity          // 建筑类型 -> 生产实体
	Changed  map[int32]map[string]interface{} // 建筑类型 -> 变更字段（脏标记，用于落库）
}

var _ logicCommon.PlayerModelInterface = (*LumberModel)(nil)

// NewLumberModel 创建伐木场数据模型实例
func NewLumberModel(entities map[int32]*LumberEntity, userId int64) *LumberModel {
	return &LumberModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int32]map[string]interface{}),
	}
}

// LoadLumberModel 从数据库加载指定玩家的所有伐木场生产数据
func LoadLumberModel(userId int64) (*LumberModel, error) {
	if userId == 0 {
		return nil, errors.New("userId is null")
	}
	rows, err := easyDB.GetPlayerEntitiesByWhere[LumberEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	entities := make(map[int32]*LumberEntity)
	for _, v := range rows {
		entities[v.BuildingType] = v
	}
	return NewLumberModel(entities, userId), nil
}

// SaveModelToDB 将所有脏数据写入数据库，然后清空脏标记
func (m *LumberModel) SaveModelToDB() {
	for key, v := range m.Changed {
		easyDB.UpdatePlayerEntity(m.Entities[key], v, m.UserId)
	}
	m.Changed = make(map[int32]map[string]interface{})
}

// Heartbeat 心跳接口（生产结算不依赖心跳，此处为空实现）
func (m *LumberModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}

// GetOrCreate 获取指定建筑类型的生产实体，不存在则创建默认记录
func (m *LumberModel) GetOrCreate(buildingType int32) (*LumberEntity, error) {
	if entity, ok := m.Entities[buildingType]; ok {
		return entity, nil
	}
	entity := &LumberEntity{
		UserId:       m.UserId,
		BuildingType: buildingType,
		Stored:       "[]",
		LastCalcTime: 0,
		HeroOwnIds:   tool.JSONInt64Slice{},
	}
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return nil, err
	}
	m.Entities[buildingType] = entity
	return entity, nil
}

// markChanged 标记指定建筑类型的某个字段为脏（等待落库）
func (m *LumberModel) markChanged(buildingType int32, field string, value interface{}) {
	if m.Changed[buildingType] == nil {
		m.Changed[buildingType] = make(map[string]interface{})
	}
	m.Changed[buildingType][field] = value
}

// UpdateStored 更新暂存产物JSON
func (m *LumberModel) UpdateStored(buildingType int32, stored string) {
	m.Entities[buildingType].Stored = stored
	m.markChanged(buildingType, "stored", stored)
}

// UpdateLastCalcTime 更新上次结算时间
func (m *LumberModel) UpdateLastCalcTime(buildingType int32, t int64) {
	m.Entities[buildingType].LastCalcTime = t
	m.markChanged(buildingType, "last_calc_time", t)
}

// UpdateHeroOwnIds 更新派驻英雄列表
func (m *LumberModel) UpdateHeroOwnIds(buildingType int32, ids tool.JSONInt64Slice) {
	m.Entities[buildingType].HeroOwnIds = ids
	m.markChanged(buildingType, "hero_own_ids", ids)
}

// ===================== 暂存序列化 =====================

// StoredItem 暂存产物的单条记录，用于JSON序列化/反序列化
type StoredItem struct {
	ItemId int32 `json:"itemId"` // 道具ID
	Count  int64 `json:"count"`  // 数量
}

// DecodeStored 将暂存JSON字符串反序列化为 map[道具ID]数量
// 空字符串或 "[]" 返回空map
func DecodeStored(s string) (map[int32]int64, error) {
	if s == "" || s == "[]" {
		return make(map[int32]int64), nil
	}
	var arr []StoredItem
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return make(map[int32]int64), err
	}
	m := make(map[int32]int64, len(arr))
	for _, it := range arr {
		if it.ItemId > 0 && it.Count > 0 {
			m[it.ItemId] += it.Count
		}
	}
	return m, nil
}

// EncodeStored 将 map[道具ID]数量 序列化为暂存JSON字符串
// 过滤掉ID或数量<=0的无效项
func EncodeStored(m map[int32]int64) string {
	arr := make([]StoredItem, 0, len(m))
	for id, cnt := range m {
		if id > 0 && cnt > 0 {
			arr = append(arr, StoredItem{ItemId: id, Count: cnt})
		}
	}
	b, _ := json.Marshal(arr)
	return string(b)
}
