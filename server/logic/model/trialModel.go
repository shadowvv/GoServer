package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

var _ logicCommon.PlayerModelInterface = (*TrialModel)(nil)

// TrialEntity 七日试炼玩家数据（对应 trial 表），按 (user_id, act_id) 唯一。
type TrialEntity struct {
	UserId         int64 `gorm:"column:user_id;primaryKey"`
	ActId          int32 `gorm:"column:act_id;primaryKey"` // 活动/试炼配置 id，与 foremost 等活动 id 一致
	InitializedDay int32 `gorm:"column:initialized_day"`   // 已初始化任务到哪一「天」（配置天数序号）
	ClaimId        int64 `gorm:"column:claimed_progress"`  // 已领取的进度奖励 id
	CreateTime     int64 `gorm:"column:create_time"`
}

func (e *TrialEntity) TableName() string { return "trial" }

// TrialModel 七日试炼数据模型，持久化 trial 表；每个活动一条 TrialEntity。
type TrialModel struct {
	UserId   int64
	Entities map[int32]*TrialEntity // actID -> 该活动试炼存档
	Changed  map[int32]map[string]interface{}
}

// NewTrialModel 创建空模型（内存用）。
func NewTrialModel(userId int64) *TrialModel {
	return &TrialModel{
		UserId:   userId,
		Entities: make(map[int32]*TrialEntity),
		Changed:  make(map[int32]map[string]interface{}),
	}
}

// LoadTrialModel 从库加载玩家全部试炼行；失败时仍返回空模型避免阻塞登录。
func LoadTrialModel(userId int64) (*TrialModel, error) {
	if userId == 0 {
		return NewTrialModel(0), nil
	}
	rows, err := easyDB.GetPlayerEntitiesByWhere[TrialEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewTrialModel(userId), err
	}
	m := NewTrialModel(userId)
	for _, row := range rows {
		if row == nil {
			continue
		}
		m.Entities[row.ActId] = row
	}
	return m, nil
}

// GetInitializedDay 返回该活动已生成任务所覆盖到的最大「天」序号（0 表示尚未初始化试炼任务）。
func (m *TrialModel) GetInitializedDay(actID int32) int32 {
	if e, ok := m.Entities[actID]; ok {
		return e.InitializedDay
	}
	return 0
}

// SetInitializedDay 更新已初始化天数并标记脏数据。
func (m *TrialModel) SetInitializedDay(actID int32, day int32) {
	e := m.getOrCreateEntity(actID)
	e.InitializedDay = day
	m.markChanged(actID, "initialized_day", day)
}

// GetClaimId 返回进度奖励已领到的配置 id（用于顺序领取判定）。
func (m *TrialModel) GetClaimId(actID int32) int64 {
	if e, ok := m.Entities[actID]; ok {
		return e.ClaimId
	}
	return 0
}

// SetClaimId 更新已领取进度奖励 id 并标记脏数据。
func (m *TrialModel) SetClaimId(actID int32, claimId int64) {
	e := m.getOrCreateEntity(actID)
	e.ClaimId = claimId
	m.markChanged(actID, "claimed_progress", claimId)
}

// getOrCreateEntity 获取或创建某活动的实体行（无则 insert）。
func (m *TrialModel) getOrCreateEntity(actID int32) *TrialEntity {
	if e, ok := m.Entities[actID]; ok {
		return e
	}
	e := &TrialEntity{
		UserId:     m.UserId,
		ActId:      actID,
		CreateTime: tool.UnixNowMilli(),
	}
	m.Entities[actID] = e
	_ = easyDB.CreatePlayerEntity(e)
	return e
}

// markChanged 记录待回写字段。
func (m *TrialModel) markChanged(actID int32, field string, value interface{}) {
	if m.Changed[actID] == nil {
		m.Changed[actID] = make(map[string]interface{})
	}
	m.Changed[actID][field] = value
}

// RemoveAct 删除内存与库中该活动试炼存档。
func (m *TrialModel) RemoveAct(actID int32) {
	delete(m.Entities, actID)
	delete(m.Changed, actID)
	_ = easyDB.DeletePlayerEntityByWhere[TrialEntity](map[string]interface{}{
		"user_id": m.UserId,
		"act_id":  actID,
	}, m.UserId)
}

// SaveModelToDB 将 Changed 中的 trial 字段更新落库。
func (m *TrialModel) SaveModelToDB() {
	for actID, fields := range m.Changed {
		if e, ok := m.Entities[actID]; ok {
			easyDB.UpdatePlayerEntity(e, fields, m.UserId)
		}
	}
	m.Changed = make(map[int32]map[string]interface{})
}

// Heartbeat 试炼无定时逻辑，占位实现 PlayerModelInterface。
func (m *TrialModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}
