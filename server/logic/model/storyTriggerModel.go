package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

// StoryTriggerEntity 记录玩家每个剧情/引导 id 的触发次数
type StoryTriggerEntity struct {
	UserId  int64 `gorm:"column:user_id;primaryKey"`
	StoryId int32 `gorm:"column:story_id;primaryKey"`
	Count   int32 `gorm:"column:count"`
}

func (e *StoryTriggerEntity) TableName() string {
	return "player_story_trigger"
}

// StoryTriggerModel 管理 player_story_trigger 表
type StoryTriggerModel struct {
	UserId   int64
	Entities map[int32]*StoryTriggerEntity    // storyId -> entity
	Changed  map[int32]map[string]interface{} // storyId -> changed fields
}

var _ logicCommon.PlayerModelInterface = (*StoryTriggerModel)(nil)

// NewStoryTriggerModel 创建特定玩家的空剧情触发模型（用于新玩家）
func NewStoryTriggerModel(userId int64) *StoryTriggerModel {
	return &StoryTriggerModel{
		UserId:   userId,
		Entities: make(map[int32]*StoryTriggerEntity),
		Changed:  make(map[int32]map[string]interface{}),
	}
}

// LoadStoryTriggerModel 从 DB 加载玩家的所有剧情触发记录
func LoadStoryTriggerModel(userId int64) (*StoryTriggerModel, error) {
	entities, err := easyDB.GetPlayerEntitiesByWhere[StoryTriggerEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	m := NewStoryTriggerModel(userId)
	for _, e := range entities {
		if e == nil {
			continue
		}
		m.Entities[e.StoryId] = e
	}
	return m, nil
}

// markChanged 标记某个剧情记录的字段变更
func (m *StoryTriggerModel) markChanged(storyId int32, field string, value interface{}) {
	if m.Changed[storyId] == nil {
		m.Changed[storyId] = make(map[string]interface{})
	}
	m.Changed[storyId][field] = value
}

// AddStoryTrigger 增加某剧情触发次数（可重复触发）
func (m *StoryTriggerModel) AddStoryTrigger(storyId int32) {
	if m == nil {
		return
	}
	entity := m.Entities[storyId]
	if entity == nil {
		entity = &StoryTriggerEntity{
			UserId:  m.UserId,
			StoryId: storyId,
			Count:   0,
		}
		m.Entities[storyId] = entity
		// 新纪录立即插入，后续只更新 count 字段
		_ = easyDB.CreatePlayerEntity(entity)
	}
	entity.Count++
	m.markChanged(storyId, "count", entity.Count)
}

// GetStoryTriggerCount 获取某个剧情的触发次数
func (m *StoryTriggerModel) GetStoryTriggerCount(storyId int32) int32 {
	if m == nil {
		return 0
	}
	if e := m.Entities[storyId]; e != nil {
		return e.Count
	}
	return 0
}

// GetAllStoryTriggers 返回 storyId -> count 的拷贝，用于下发给客户端
func (m *StoryTriggerModel) GetAllStoryTriggers() map[int32]int32 {
	res := make(map[int32]int32)
	if m == nil {
		return res
	}
	for id, e := range m.Entities {
		if e != nil {
			res[id] = e.Count
		}
	}
	return res
}

// SaveModelToDB 保存模型到数据库（依赖 (user_id, story_id) 复合主键做 upsert）
func (m *StoryTriggerModel) SaveModelToDB() {
	if m == nil || len(m.Changed) == 0 {
		return
	}
	for storyId, changes := range m.Changed {
		entity := m.Entities[storyId]
		if entity == nil || len(changes) == 0 {
			continue
		}
		// 依赖 (user_id, story_id) 复合主键，使用 UpdatePlayerEntity 做更新
		easyDB.UpdatePlayerEntity(entity, changes, m.UserId)
	}
	m.Changed = make(map[int32]map[string]interface{})
}

// Heartbeat 这里无需特殊处理
func (m *StoryTriggerModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}
