package model

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"gorm.io/gorm"
)

type PlayerFunctionEntity struct {
	UserId         int64 `gorm:"column:user_id;primaryKey"`
	FunctionId     int32 `gorm:"column:function_id;primaryKey"`
	RewardCommited int32 `gorm:"column:reward_commited"`
	UnlockTime     int64 `gorm:"column:unlock_time"`
}

func (m *PlayerFunctionEntity) TableName() string {
	return "player_function_data"
}

type PlayerFunctionModel struct {
	UserId         int64
	Player         *PlayerModel
	Entities       map[int32]*PlayerFunctionEntity
	Changed        map[int32]map[string]interface{}
	FunctionStatus map[enum.FunctionIdEnum]int32 // 功能状态
}

func NewPlayerFunctionModel(userId int64, player *PlayerModel, entities map[int32]*PlayerFunctionEntity) *PlayerFunctionModel {
	return &PlayerFunctionModel{
		UserId:         userId,
		Player:         player,
		Entities:       entities,
		Changed:        make(map[int32]map[string]interface{}),
		FunctionStatus: make(map[enum.FunctionIdEnum]int32),
	}
}

func LoadPlayerFunctionModel(userId int64, player *PlayerModel) (*PlayerFunctionModel, error) {
	entities, err := easyDB.GetPlayerEntitiesByWhere[PlayerFunctionEntity](map[string]interface{}{"user_id": userId})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	entitiesMap := make(map[int32]*PlayerFunctionEntity)
	for _, entity := range entities {
		entitiesMap[entity.FunctionId] = entity
	}

	model := &PlayerFunctionModel{
		UserId:         userId,
		Player:         player,
		Entities:       entitiesMap,
		Changed:        make(map[int32]map[string]interface{}),
		FunctionStatus: make(map[enum.FunctionIdEnum]int32),
	}
	return model, nil
}

func (m *PlayerFunctionModel) SaveModelToDB() {
	if m.Changed == nil || len(m.Changed) == 0 {
		return
	}
	for id, changes := range m.Changed {
		easyDB.UpdatePlayerEntity[PlayerFunctionEntity](m.Entities[id], changes, m.UserId)
	}
	m.Changed = make(map[int32]map[string]interface{})
}

func (m *PlayerFunctionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	funcInfo := make([]*pb.FunctionOpenInfo, 0)
	for funcId, cfg := range gameConfig.GetAllSystemUnlockCfg() {
		status := int32(0)
		isShow := true
		isOpen := true
		for _, show := range cfg.ShowId {
			if !unlockService.CheckUnlock(show, m.Player) {
				isShow = false
				break
			}
		}
		for _, unlock := range cfg.UnlockId {
			if !unlockService.CheckUnlock(unlock, m.Player) {
				isOpen = false
				break
			}
		}
		if isShow {
			status = int32(enum.FUNCTION_STATUS_SHOW)
		}
		if isOpen {
			status = int32(enum.FUNCTION_STATUS_UNLOCK)
		}

		// 这里用 currentTime(毫秒) 初始化 idle 的 LastSettleTime(秒)
		if status == int32(enum.FUNCTION_STATUS_UNLOCK) && funcId == int32(enum.FUNCTION_ID_IDLE) &&
			m.Player != nil && m.Player.IdleModel != nil && m.Player.IdleModel.Entity != nil &&
			m.Player.IdleModel.Entity.LastSettleTime == 0 {
			m.Player.IdleModel.UpdateLastSettleTime(currentTime / 1000)
		}

		rewardCommited := int32(0)
		unlockTime := int64(0)
		var entity *PlayerFunctionEntity
		if entity = m.Get(funcId); entity != nil {
			rewardCommited = entity.RewardCommited
		} else {
			if status == int32(enum.FUNCTION_STATUS_UNLOCK) {
				entity = &PlayerFunctionEntity{
					UserId:         m.UserId,
					FunctionId:     funcId,
					RewardCommited: rewardCommited,
					UnlockTime:     currentTime,
				}
				err := easyDB.CreatePlayerEntity[PlayerFunctionEntity](entity)
				if err != nil {
					continue
				}
				unlockTime = entity.UnlockTime
				m.Entities[funcId] = entity
			}
		}

		if current, ok := m.FunctionStatus[enum.FunctionIdEnum(funcId)]; ok {
			if status == current {
				continue
			}
			m.FunctionStatus[enum.FunctionIdEnum(funcId)] = status
			funcInfo = append(funcInfo, &pb.FunctionOpenInfo{
				FuncId:         funcId,
				Status:         status,
				RewardCommited: rewardCommited,
				UnlockTime:     unlockTime,
			})
		} else {
			if status == 0 {
				continue
			}
			m.FunctionStatus[enum.FunctionIdEnum(funcId)] = status
			funcInfo = append(funcInfo, &pb.FunctionOpenInfo{
				FuncId:         funcId,
				Status:         status,
				RewardCommited: rewardCommited,
				UnlockTime:     unlockTime,
			})
		}
	}
	if len(funcInfo) == 0 {
		return
	}
	if senderMsg {
		messageSender.SendMessage(m.Player, pb.MESSAGE_ID_PUSH_FUNCTION_CHANGE, &pb.PushFunctionChange{
			Infos: funcInfo,
		})
	}
}

func (m *PlayerFunctionModel) Get(functionId int32) *PlayerFunctionEntity {
	return m.Entities[functionId]
}

func (m *PlayerFunctionModel) CommitReward(functionId int32) {
	if entity := m.Get(functionId); entity != nil {
		if entity.RewardCommited == 0 {
			entity.RewardCommited = 1
			if m.Changed[functionId] == nil {
				m.Changed[functionId] = make(map[string]interface{})
			}
			m.Changed[functionId]["reward_commited"] = 1
		}
	}
}
