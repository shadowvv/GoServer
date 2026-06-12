package model

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

var _ logicCommon.PlayerModelInterface = (*TurnTableModel)(nil)

type TurnTableEntity struct {
	UserId          int64 `gorm:"column:user_id;primaryKey"`
	ModId           int32 `gorm:"column:mod_id;primaryKey"`
	TaskRefreshTime int64 `gorm:"column:task_refresh_time"`
}

func (e *TurnTableEntity) TableName() string {
	return "turn_table"
}

const (
	// 累计奖励状态：StateId=act_usuallyuse.type，Progress=累计进度，Count=已领取次数。
	TurnTableStateTypeUsuallyProgress int32 = 1
	// 限制奖励状态：StateId=act_turntable.id，Progress不用，Count=已命中次数。
	TurnTableStateTypeRewardLimit int32 = 2
	// 保底状态：StateId=act_turntable.id，Progress不用，Count=已触发保底次数。
	TurnTableStateTypeGuarantee           int32 = 3
	TurnTableStateTypeUsuallySingleReward int32 = 4
)

type TurnTableStateEntity struct {
	UserId int64 `gorm:"column:user_id;primaryKey"`
	ModId  int32 `gorm:"column:mod_id;primaryKey"`
	// StateType区分状态用途：1累计奖励，2限制奖励命中，3保底触发次数。
	StateType int32 `gorm:"column:state_type;primaryKey"`
	// StateId随StateType变化：累计奖励用act_usuallyuse.type，限制奖励和保底用act_turntable.id。
	StateId int32 `gorm:"column:state_id;primaryKey"`
	// Progress目前只给累计奖励用，记录抽数、消耗数等累计值。
	Progress int64 `gorm:"column:progress"`
	// Count随StateType变化：累计奖励是已领取次数，限制奖励是已命中次数，保底是已触发次数。
	Count int32 `gorm:"column:count"`
}

func (e *TurnTableStateEntity) TableName() string {
	return "turn_table_state"
}

type TurnTableStateKey struct {
	ModId     int32
	StateType int32
	StateId   int32
}

type TurnTableModel struct {
	UserId        int64
	Player        *PlayerModel
	Entities      map[int32]*TurnTableEntity
	StateEntities map[TurnTableStateKey]*TurnTableStateEntity
	Changed       map[int32]map[string]interface{}
	StateChanged  map[TurnTableStateKey]map[string]interface{}
}

func TurnTableTaskSlot(taskId int32) int32 {
	return enum.TaskAffiliationAct*100000 + taskId
}

func NewTurnTableModel(entities map[int32]*TurnTableEntity, stateEntities map[TurnTableStateKey]*TurnTableStateEntity, userId int64, player *PlayerModel) *TurnTableModel {
	return &TurnTableModel{
		UserId:        userId,
		Player:        player,
		Entities:      entities,
		StateEntities: stateEntities,
		Changed:       make(map[int32]map[string]interface{}),
		StateChanged:  make(map[TurnTableStateKey]map[string]interface{}),
	}
}

func LoadTurnTableModel(player *PlayerModel) (*TurnTableModel, error) {
	rows, err := easyDB.GetPlayerEntitiesByWhere[TurnTableEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		return NewTurnTableModel(make(map[int32]*TurnTableEntity), make(map[TurnTableStateKey]*TurnTableStateEntity), player.GetUserId(), player), err
	}
	entities := make(map[int32]*TurnTableEntity)
	for _, row := range rows {
		entities[row.ModId] = row
	}
	stateRows, err := easyDB.GetPlayerEntitiesByWhere[TurnTableStateEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		return NewTurnTableModel(entities, make(map[TurnTableStateKey]*TurnTableStateEntity), player.GetUserId(), player), err
	}
	stateEntities := make(map[TurnTableStateKey]*TurnTableStateEntity)
	for _, row := range stateRows {
		stateEntities[TurnTableStateKey{ModId: row.ModId, StateType: row.StateType, StateId: row.StateId}] = row
	}
	return NewTurnTableModel(entities, stateEntities, player.GetUserId(), player), nil
}

func (m *TurnTableModel) SaveModelToDB() {
	for key, changes := range m.Changed {
		easyDB.UpdatePlayerEntity(m.Entities[key], changes, m.UserId)
	}
	m.Changed = make(map[int32]map[string]interface{})
	for key, changes := range m.StateChanged {
		easyDB.UpdatePlayerEntity(m.StateEntities[key], changes, m.UserId)
	}
	m.StateChanged = make(map[TurnTableStateKey]map[string]interface{})
}

func (m *TurnTableModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if activityService == nil || m.Player == nil {
		return
	}
	for _, entity := range m.Entities {
		mainCfg := gameConfig.GetTurnTableMainCfg(entity.ModId)
		if mainCfg == nil {
			continue
		}
		act := activityService.IsActivityOpen(m.Player.GetUserServerId(), mainCfg.ActId)
		if act == nil {
			continue
		}
		if tool.UnixNowMilli() >= act.GetSettleTime() {
			m.FreezeActTasks(entity.ModId)
			continue
		}
		if passDay <= 0 {
			continue
		}
		pushes, _ := m.SyncActTasks(entity.ModId, false)
		if senderMsg {
			for _, push := range pushes {
				messageSender.SendMessage(m.Player, pb.MESSAGE_ID_PUSH_TASK_UPDATE, push)
			}
		}
	}
}

func (m *TurnTableModel) GetOrCreate(modId int32) (*TurnTableEntity, bool, error) {
	if entity := m.Entities[modId]; entity != nil {
		return entity, false, nil
	}
	entity := &TurnTableEntity{
		UserId:          m.UserId,
		ModId:           modId,
		TaskRefreshTime: tool.UnixNowMilli(),
	}
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return nil, false, err
	}
	m.Entities[modId] = entity
	return entity, true, nil
}

func (m *TurnTableModel) markChanged(key int32, field string, value interface{}) {
	if m.Changed[key] == nil {
		m.Changed[key] = make(map[string]interface{})
	}
	m.Changed[key][field] = value
}

func (m *TurnTableModel) markStateChanged(key TurnTableStateKey, field string, value interface{}) {
	if m.StateChanged[key] == nil {
		m.StateChanged[key] = make(map[string]interface{})
	}
	m.StateChanged[key][field] = value
}

func (m *TurnTableModel) SetTaskRefreshTime(entity *TurnTableEntity, ts int64) {
	entity.TaskRefreshTime = ts
	m.markChanged(entity.ModId, "task_refresh_time", ts)
}

func (m *TurnTableModel) GetState(modId int32, stateType int32, stateId int32) *TurnTableStateEntity {
	return m.StateEntities[TurnTableStateKey{ModId: modId, StateType: stateType, StateId: stateId}]
}

func (m *TurnTableModel) GetOrCreateState(modId int32, stateType int32, stateId int32) (*TurnTableStateEntity, error) {
	key := TurnTableStateKey{ModId: modId, StateType: stateType, StateId: stateId}
	if entity := m.StateEntities[key]; entity != nil {
		return entity, nil
	}
	entity := &TurnTableStateEntity{
		UserId:    m.UserId,
		ModId:     modId,
		StateType: stateType,
		StateId:   stateId,
	}
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return nil, err
	}
	m.StateEntities[key] = entity
	return entity, nil
}

func (m *TurnTableModel) AddStateProgress(entity *TurnTableStateEntity, count int64) {
	key := TurnTableStateKey{ModId: entity.ModId, StateType: entity.StateType, StateId: entity.StateId}
	entity.Progress += count
	m.markStateChanged(key, "progress", entity.Progress)
}

func (m *TurnTableModel) SetStateCount(entity *TurnTableStateEntity, count int32) {
	key := TurnTableStateKey{ModId: entity.ModId, StateType: entity.StateType, StateId: entity.StateId}
	entity.Count = count
	m.markStateChanged(key, "count", entity.Count)
}

func (m *TurnTableModel) Reset(modId int32) {
	entity := m.Entities[modId]
	if entity == nil {
		return
	}
	m.SetTaskRefreshTime(entity, tool.UnixNowMilli())
	for key := range m.StateEntities {
		if key.ModId != modId {
			continue
		}
		stateEntity := m.StateEntities[key]
		stateEntity.Progress = 0
		stateEntity.Count = 0
		m.markStateChanged(key, "progress", int64(0))
		m.markStateChanged(key, "count", int32(0))
	}
}

func (m *TurnTableModel) SyncActTasks(modId int32, forceReset bool) ([]*pb.PushTaskUpdate, error) {
	mainCfg := gameConfig.GetTurnTableMainCfg(modId)
	if mainCfg == nil {
		return nil, nil
	}
	entity, _, err := m.GetOrCreate(modId)
	if err != nil {
		return nil, err
	}
	pushes := make([]*pb.PushTaskUpdate, 0)
	for _, taskCfg := range gameConfig.GetActTaskCfgsByActID(mainCfg.ActId) {
		taskEntity := NewTaskEntity(m.Player.GetUserId(), TurnTableTaskSlot(taskCfg.Id), taskCfg.Id, enum.TaskAffiliationAct, 0, enum.TaskStatusUnFinish, tool.UnixNowMilli(), 0)
		oldEntity := m.Player.TaskModel.TaskEntityBySlot[taskEntity.SlotId]
		if forceReset || oldEntity == nil || oldEntity.TaskAttribution != enum.TaskAffiliationAct || oldEntity.TaskID != taskCfg.Id {
			if oldEntity != nil {
				m.Player.TaskModel.UpdateChangedWithNewData(oldEntity, taskEntity)
				m.Player.TaskModel.DeteleTaskEntityFormMemory(oldEntity)
				m.Player.TaskModel.AddTaskEntityToMemory(taskEntity)
			} else {
				m.Player.TaskModel.AddTaskEntityToMemory(taskEntity)
				if err := easyDB.CreatePlayerEntity(taskEntity); err != nil {
					return nil, err
				}
			}
			pushes = append(pushes, &pb.PushTaskUpdate{
				Attribution: enum.TaskAffiliationAct,
				TaskId:      taskEntity.TaskID,
				TaskState:   taskEntity.Status,
				Progress:    taskEntity.ProgressData,
			})
			continue
		}
		if oldEntity.Status == enum.TaskStatusUnFinish {
			m.Player.TaskModel.NeedCheckTaskList = append(m.Player.TaskModel.NeedCheckTaskList, oldEntity)
		} else if oldEntity.Status == enum.TaskStatusFinishAndReward {
			m.Player.TaskModel.DeteleTaskEntityFormMemory(oldEntity)
			for i := len(m.Player.TaskModel.NeedCheckTaskList) - 1; i >= 0; i-- {
				if m.Player.TaskModel.NeedCheckTaskList[i] != oldEntity {
					continue
				}
				m.Player.TaskModel.NeedCheckTaskList = append(m.Player.TaskModel.NeedCheckTaskList[:i], m.Player.TaskModel.NeedCheckTaskList[i+1:]...)
			}
		}
	}
	if entity.TaskRefreshTime > 0 && !tool.IsSameDayByMilli(entity.TaskRefreshTime, tool.UnixNowMilli()) {
		for _, taskCfg := range gameConfig.GetActTaskCfgsByActID(mainCfg.ActId) {
			if taskCfg.Reflect != 1 {
				continue
			}
			taskEntity := m.Player.TaskModel.TaskEntityBySlot[TurnTableTaskSlot(taskCfg.Id)]
			if taskEntity == nil {
				continue
			}
			coreCfg := gameConfig.GetCoreCfg(taskCfg.TaskId)
			if coreCfg == nil {
				continue
			}
			taskEntity.ProgressData = 0
			taskEntity.Status = enum.TaskStatusUnFinish
			taskEntity.UpdateTime = tool.UnixNowMilli()
			if m.Player.TaskModel.TaskEntity[enum.TaskAffiliationAct] == nil ||
				m.Player.TaskModel.TaskEntity[enum.TaskAffiliationAct][coreCfg.TaskType] == nil ||
				m.Player.TaskModel.TaskEntity[enum.TaskAffiliationAct][coreCfg.TaskType][taskCfg.Id] == nil {
				m.Player.TaskModel.AddTaskEntityToMemory(taskEntity)
			}
			if m.Player.TaskModel.Changed[enum.TaskAffiliationAct] == nil {
				m.Player.TaskModel.Changed[enum.TaskAffiliationAct] = make(map[int32]map[string]interface{})
			}
			m.Player.TaskModel.Changed[enum.TaskAffiliationAct][taskEntity.SlotId] = map[string]interface{}{
				"progress_data": taskEntity.ProgressData,
				"status":        taskEntity.Status,
				"update_time":   taskEntity.UpdateTime,
			}
			pushes = append(pushes, &pb.PushTaskUpdate{
				Attribution: enum.TaskAffiliationAct,
				TaskId:      taskEntity.TaskID,
				TaskState:   taskEntity.Status,
				Progress:    taskEntity.ProgressData,
			})
		}
	}
	if entity.TaskRefreshTime == 0 || !tool.IsSameDayByMilli(entity.TaskRefreshTime, tool.UnixNowMilli()) {
		m.SetTaskRefreshTime(entity, tool.UnixNowMilli())
	}
	return pushes, nil
}

func (m *TurnTableModel) FreezeActTasks(modId int32) {
	mainCfg := gameConfig.GetTurnTableMainCfg(modId)
	if mainCfg == nil {
		return
	}
	for _, taskCfg := range gameConfig.GetActTaskCfgsByActID(mainCfg.ActId) {
		taskEntity := m.Player.TaskModel.TaskEntityBySlot[TurnTableTaskSlot(taskCfg.Id)]
		if taskEntity == nil || taskEntity.Status != enum.TaskStatusUnFinish {
			continue
		}
		m.Player.TaskModel.DeteleTaskEntityFormMemory(taskEntity)
		for i := len(m.Player.TaskModel.NeedCheckTaskList) - 1; i >= 0; i-- {
			if m.Player.TaskModel.NeedCheckTaskList[i] != taskEntity {
				continue
			}
			m.Player.TaskModel.NeedCheckTaskList = append(m.Player.TaskModel.NeedCheckTaskList[:i], m.Player.TaskModel.NeedCheckTaskList[i+1:]...)
			break
		}
	}
}
