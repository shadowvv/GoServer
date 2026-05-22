package model

import (
	"encoding/json"
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

type PlayerInstanceEntity struct {
	UserId            int64                          `gorm:"column:user_id;primaryKey"`
	InstanceId        int32                          `gorm:"column:instance_id;primaryKey"`
	StageId           int32                          `gorm:"column:stage_id"`
	CurrentSubStageId int32                          `gorm:"column:current_sub_stage_id"`
	StageInfo         string                         `gorm:"column:stage_info;size:512"`
	MaxStageId        int32                          `gorm:"column:max_stage_id"`
	MaxSubStageId     int32                          `gorm:"column:max_sub_stage_id"`
	CommitLevelReward int32                          `gorm:"column:commit_level_reward"`
	Info              *logicCommon.InstanceStageInfo `gorm:"-"`
}

func (u *PlayerInstanceEntity) TableName() string {
	return "player_Instance_data"
}

type PlayerInstanceModel struct {
	player           *PlayerModel
	UserId           int64
	InstanceEntities map[int32]*PlayerInstanceEntity
	Changed          map[int32]map[string]interface{}

	CurrentRaidInfo             *logicCommon.PlayerInstanceRaid // 当前副本数据
	CurrentMainInstanceInfo     *logicCommon.PlayerInstanceRaid // 当前主线关
	NextMainInstanceInfo        *logicCommon.PlayerInstanceRaid // 主线下一关
	LastDeadMainInstanceStageId int32
}

var _ logicCommon.PlayerModelInterface = (*PlayerInstanceModel)(nil)

func (p *PlayerInstanceModel) SaveModelToDB() {
	if p.Changed == nil || len(p.Changed) == 0 {
		return
	}
	for id, changes := range p.Changed {
		easyDB.UpdatePlayerEntity[PlayerInstanceEntity](p.InstanceEntities[id], changes, p.UserId)
	}
	p.Changed = make(map[int32]map[string]interface{})
}

func (p *PlayerInstanceModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		for _, instance := range p.InstanceEntities {
			instanceCfg := gameConfig.GetInstanceCfg(instance.InstanceId)
			if instanceCfg == nil {
				continue
			}
			count := p.player.VipCardModel.GetFunctionValue(enum.VIP_PRIVILEGE_INSTANCE_TICKET, currentTime)
			if count > 0 && !enum.IsResidentInstanceType(instanceCfg.InstanceType) {
				newItems := make([]*gameConfig.ItemConfig, 0)
				for _, item := range instanceCfg.RecoveryTicketID {
					newItems = append(newItems, &gameConfig.ItemConfig{
						ID:  item.ID,
						Num: item.Num + count,
					})
				}
				_ = itemService.ResetItems(p.player, newItems, enum.ITEM_CHANGE_REASON_INSTANCE_RESET_TICKET)
			} else {
				_ = itemService.ResetItems(p.player, instanceCfg.RecoveryTicketID, enum.ITEM_CHANGE_REASON_INSTANCE_RESET_TICKET)
			}
		}
	}
}

func (p *PlayerInstanceModel) OnPassMainInstance(passStageId int32, passSubStageId int32, isCycle int32) {
	instance := p.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instance != nil {
		if instance.MaxStageId < passStageId && isCycle == 0 {
			if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
				p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
			}
			instance.MaxStageId = passStageId
			p.Changed[int32(enum.MAIN_INSTANCE_ID)]["max_stage_id"] = passStageId
			// 注意：通行证进度更新在 controller 层处理，避免循环依赖
		}
		if instance.MaxSubStageId < passSubStageId {
			if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
				p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
			}
			instance.MaxSubStageId = passSubStageId
			p.Changed[int32(enum.MAIN_INSTANCE_ID)]["max_sub_stage_id"] = passSubStageId
		}
		// 更新关卡信息
		if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
			p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
		}
		instance.Info.IsCycle = isCycle
		instance.Info.KillMonsterId = make([]int32, 0)
		data, err := json.Marshal(instance.Info)
		if err != nil {
			return
		}
		p.Changed[int32(enum.MAIN_INSTANCE_ID)]["stage_info"] = string(data)
	}
}

func (p *PlayerInstanceModel) OnKillMonster() {
	instance := p.InstanceEntities[int32(p.CurrentRaidInfo.InstanceID)]
	if instance != nil {
		data, err := json.Marshal(instance.Info)
		if err != nil {
			return
		}
		if p.Changed[int32(p.CurrentRaidInfo.InstanceID)] == nil {
			p.Changed[int32(p.CurrentRaidInfo.InstanceID)] = make(map[string]interface{})
		}
		p.Changed[int32(p.CurrentRaidInfo.InstanceID)]["stage_info"] = string(data)
	}
}

func (p *PlayerInstanceModel) UpdateMainInstanceInfo(currentStageId int32, subStageId int32, isCycle int32) {
	instance := p.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instance != nil {
		if currentStageId != 0 && instance.StageId != currentStageId {
			if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
				p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
			}
			instance.StageId = currentStageId
			p.Changed[int32(enum.MAIN_INSTANCE_ID)]["stage_id"] = currentStageId
		}
		if instance.CurrentSubStageId != subStageId {
			if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
				p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
			}
			instance.CurrentSubStageId = subStageId
			p.Changed[int32(enum.MAIN_INSTANCE_ID)]["current_sub_stage_id"] = subStageId
		}

		// 更新关卡信息
		if p.Changed[int32(enum.MAIN_INSTANCE_ID)] == nil {
			p.Changed[int32(enum.MAIN_INSTANCE_ID)] = make(map[string]interface{})
		}
		instance.Info.IsCycle = isCycle
		instance.Info.KillMonsterId = make([]int32, 0)
		data, err := json.Marshal(instance.Info)
		if err != nil {
			return
		}
		p.Changed[int32(enum.MAIN_INSTANCE_ID)]["stage_info"] = string(data)
	}
}

func (p *PlayerInstanceModel) UpdateInstanceInfo(id int32, currentStageId int32, currentSubStageId int32) error {

	instance := p.InstanceEntities[id]
	if instance != nil {
		if instance.StageId != currentStageId {
			if p.Changed[id] == nil {
				p.Changed[id] = make(map[string]interface{})
			}
			instance.StageId = currentStageId
			p.Changed[id]["stage_id"] = currentStageId
		}
		if instance.CurrentSubStageId != currentSubStageId {
			if p.Changed[id] == nil {
				p.Changed[id] = make(map[string]interface{})
			}
			instance.CurrentSubStageId = currentSubStageId
			p.Changed[id]["current_sub_stage_id"] = currentSubStageId
		}
		if instance.MaxStageId < currentStageId {
			if p.Changed[id] == nil {
				p.Changed[id] = make(map[string]interface{})
			}
			instance.MaxStageId = currentStageId
			p.Changed[id]["max_stage_id"] = currentStageId
		}
		if instance.MaxSubStageId < currentSubStageId {
			if p.Changed[id] == nil {
				p.Changed[id] = make(map[string]interface{})
			}
			instance.MaxSubStageId = currentSubStageId
			p.Changed[id]["max_sub_stage_id"] = currentSubStageId
		}
	} else {
		instance := &PlayerInstanceEntity{
			UserId:            p.UserId,
			InstanceId:        id,
			StageId:           currentStageId,
			CurrentSubStageId: currentSubStageId,
			StageInfo:         "",
			Info:              logicCommon.NewInstanceStageInfo(),
			MaxStageId:        currentStageId,
			MaxSubStageId:     currentSubStageId,
		}
		err := easyDB.CreatePlayerEntity[PlayerInstanceEntity](instance)
		if err != nil {
			return errors.New("create instance data error")
		}
		p.InstanceEntities[id] = instance
	}
	return nil
}

func (p *PlayerInstanceModel) CommitedLevelReward(instanceId int32) {
	instance := p.InstanceEntities[instanceId]
	if instance != nil {
		instance.CommitLevelReward = instance.StageId
		if p.Changed[instanceId] == nil {
			p.Changed[instanceId] = make(map[string]interface{})
		}
		p.Changed[instanceId]["commit_level_reward"] = instance.StageId
	}
}

func (p *PlayerInstanceModel) GetLastDeadMainInstanceStageId() int32 {
	return p.LastDeadMainInstanceStageId
}

func (p *PlayerInstanceModel) UpdateLastDeadMainInstanceStageId(stageId int32) {
	p.LastDeadMainInstanceStageId = stageId
}

func (p *PlayerInstanceModel) GetMainInstanceMaxStageId() int32 {
	instance := p.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
	if instance != nil {
		return instance.MaxStageId
	}
	return 0
}

func CreatePlayerInstanceModel(player *PlayerModel) *PlayerInstanceModel {
	cfg := gameConfig.GetBaseCfg()
	if cfg == nil {
		return nil
	}
	mainStageCfg := gameConfig.GetMainStageCfg(cfg.Stage)
	if mainStageCfg == nil {
		return nil
	}
	entity := &PlayerInstanceEntity{
		UserId:            player.GetUserId(),
		InstanceId:        int32(enum.MAIN_INSTANCE_ID),
		StageId:           cfg.Stage,
		CurrentSubStageId: mainStageCfg.SubStageId[0],
		StageInfo:         "",
		Info:              logicCommon.NewInstanceStageInfo(),
		MaxStageId:        0,
		MaxSubStageId:     0,
	}
	err := easyDB.CreatePlayerEntity[PlayerInstanceEntity](entity)
	if err != nil {
		return nil
	}
	entities := make(map[int32]*PlayerInstanceEntity)
	entities[entity.InstanceId] = entity
	instanceModel := &PlayerInstanceModel{
		player:           player,
		UserId:           player.GetUserId(),
		InstanceEntities: entities,
		Changed:          make(map[int32]map[string]interface{}),
	}
	return instanceModel
}

func LoadPlayerInstanceModel(player *PlayerModel) (*PlayerInstanceModel, error) {
	entities, err := easyDB.GetPlayerEntitiesByWhere[PlayerInstanceEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		return nil, err
	}
	entitiesMap := make(map[int32]*PlayerInstanceEntity)
	for _, entity := range entities {
		entitiesMap[entity.InstanceId] = entity
		info := logicCommon.NewInstanceStageInfo()
		err = json.Unmarshal([]byte(entity.StageInfo), info)
		if err != nil {
			entity.Info = logicCommon.NewInstanceStageInfo()
			continue
		}
		entity.Info = info
		for _, id := range entity.Info.KillMonsterId {
			entity.Info.KillMonsterMap[id] = true
		}
	}

	model := &PlayerInstanceModel{
		player:           player,
		UserId:           player.GetUserId(),
		InstanceEntities: entitiesMap,
		Changed:          make(map[int32]map[string]interface{}),
	}
	return model, nil
}
