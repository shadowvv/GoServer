package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

type PlayerActivityEntity struct {
	UserId     int64  `gorm:"column:user_id;primaryKey"`
	ActivityId int32  `gorm:"column:activity_id;primaryKey"`
	Version    string `gorm:"column:version;primaryKey"`
	IsSettled  int32  `gorm:"column:is_settled"`
}

func (u *PlayerActivityEntity) TableName() string {
	return "player_activity_data"
}

type PlayerActivityModel struct {
	Player           *PlayerModel
	ActivityEntities map[int32]*PlayerActivityEntity
	Changed          map[int32]map[string]interface{}
	LastChangeTime   int64
	OpenActivity     map[int32]bool
}

var _ logicCommon.PlayerModelInterface = (*PlayerActivityModel)(nil)

func (p *PlayerActivityModel) SaveModelToDB() {
	if p.Changed == nil || len(p.Changed) == 0 {
		return
	}
	for id, changes := range p.Changed {
		easyDB.UpdatePlayerEntity[PlayerActivityEntity](p.ActivityEntities[id], changes, p.Player.GetUserId())
	}
	p.Changed = make(map[int32]map[string]interface{})
}

func (p *PlayerActivityModel) CheckNewActivity() {
	for _, activity := range p.ActivityEntities {
		if activity.IsSettled == 0 {
			if activityService.IsActivitySettled(p.Player.GetUserServerId(), activity.ActivityId, activity.Version) {
				activity.IsSettled = 1
				if p.Changed[activity.ActivityId] == nil {
					p.Changed[activity.ActivityId] = make(map[string]interface{})
				}
				p.Changed[activity.ActivityId] = map[string]interface{}{"is_settled": 1}
			}
		}
	}
	resp := &pb.PushActivityInfo{}
	resp.ActivityInfos = make([]*pb.ActivityInfo, 0)
	allActivity := activityService.GetAllOpenActivityByServerId(p.Player.GetUserServerId())
	for _, activity := range allActivity {
		attend := p.checkAttendActivity(activity)
		if _, ok := p.OpenActivity[activity.GetActivityId()]; ok {
			currentStatus := p.OpenActivity[activity.GetActivityId()]
			if attend != currentStatus {
				p.OpenActivity[activity.GetActivityId()] = attend
				resp.ActivityInfos = append(resp.ActivityInfos, &pb.ActivityInfo{
					ActivityId: activity.GetActivityId(),
					SettleTime: activity.GetSettleTime(),
					EndTime:    activity.GetEndTime(),
					StartTime:  activity.GetOpenTime(),
					Locked:     attend,
				})
			}
		} else {
			if attend {
				p.OpenActivity[activity.GetActivityId()] = attend
				resp.ActivityInfos = append(resp.ActivityInfos, &pb.ActivityInfo{
					ActivityId: activity.GetActivityId(),
					SettleTime: activity.GetSettleTime(),
					EndTime:    activity.GetEndTime(),
					StartTime:  activity.GetOpenTime(),
					Locked:     attend,
				})
			}
		}
	}
	if len(resp.ActivityInfos) > 0 {
		messageSender.SendMessage(p.Player, pb.MESSAGE_ID_PUSH_ACTIVITY_INFO, resp)
	}
}

func (p *PlayerActivityModel) checkAttendActivity(activity logicCommon.GameActivityInterface) bool {
	cfg := activityService.GetActivityConfig(activity.GetActivityId())
	if cfg == nil {
		logger.ErrorBySprintf("activity config not found: %d", activity.GetActivityId())
		return false
	}
	for _, unlock := range cfg.GetAttendUnlockId() {
		if !unlockService.CheckUnlock(unlock, p.Player) {
			return false
		}
	}
	if p.ActivityEntities[activity.GetActivityId()] == nil {
		p.ActivityEntities[activity.GetActivityId()] = &PlayerActivityEntity{
			UserId:     p.Player.GetUserId(),
			Version:    activity.GetVersion(),
			ActivityId: activity.GetActivityId(),
			IsSettled:  0,
		}
		err := easyDB.CreatePlayerEntity[PlayerActivityEntity](p.ActivityEntities[activity.GetActivityId()])
		if err != nil {
			logger.InfoWithSprintf("create player activity error: %v", err)
		}
	}
	return true
}

func (p *PlayerActivityModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if currentTime-p.LastChangeTime > 1000*10 {
		p.CheckNewActivity()
		p.LastChangeTime = currentTime
	}
}

func (p *PlayerActivityModel) GetAllOpenActivity() []*pb.ActivityInfo {
	allActivity := make([]*pb.ActivityInfo, 0)
	allServerOpenActivity := activityService.GetAllOpenActivityByServerId(p.Player.GetUserServerId())
	for _, activity := range allServerOpenActivity {
		pActivity := p.ActivityEntities[activity.GetActivityId()]
		if pActivity == nil {
			continue
		}
		if tool.UnixNowMilli() == activity.GetEndTime() {
			continue
		}
		allActivity = append(allActivity, &pb.ActivityInfo{
			ActivityId: activity.GetActivityId(),
			SettleTime: activity.GetSettleTime(),
			EndTime:    activity.GetEndTime(),
			StartTime:  activity.GetOpenTime(),
			Locked:     p.checkAttendActivity(activity),
		})
	}
	return allActivity
}

func (p *PlayerActivityModel) CheckActivitySettled(activityId int32) (bool, string) {
	activity := activityService.IsActivityOpen(p.Player.GetUserServerId(), activityId)
	if activity == nil {
		playerActivity := p.ActivityEntities[activityId]
		if playerActivity == nil {
			return true, ""
		}
		playerActivity.IsSettled = 1
		if p.Changed[activityId] == nil {
			p.Changed[activityId] = make(map[string]interface{})
		}
		p.Changed[activityId] = map[string]interface{}{"is_settled": 1}
		return true, ""
	} else {
		playerActivity := p.ActivityEntities[activityId]
		if playerActivity == nil {
			p.checkAttendActivity(activity)
		}
		playerActivity = p.ActivityEntities[activityId]
		if playerActivity == nil {
			return true, ""
		}
		return playerActivity.IsSettled == 1, playerActivity.Version
	}
}

func (p *PlayerActivityModel) CheckActivityOpen(activityId int32) (bool, string) {
	activity := activityService.IsActivityOpen(p.Player.GetUserServerId(), activityId)
	if activity == nil {
		playerActivity := p.ActivityEntities[activityId]
		if playerActivity == nil {
			return false, ""
		}
		playerActivity.IsSettled = 1
		if p.Changed[activityId] == nil {
			p.Changed[activityId] = make(map[string]interface{})
		}
		p.Changed[activityId] = map[string]interface{}{"is_settled": 1}
		return false, ""
	} else {
		playerActivity := p.ActivityEntities[activityId]
		if playerActivity == nil {
			p.checkAttendActivity(activity)
		}
		playerActivity = p.ActivityEntities[activityId]
		if playerActivity == nil {
			return false, ""
		}
		return true, playerActivity.Version
	}
}

func NewPlayerActivityModel(player *PlayerModel) *PlayerActivityModel {
	ActivityModel := &PlayerActivityModel{
		Player:           player,
		ActivityEntities: make(map[int32]*PlayerActivityEntity),
		Changed:          make(map[int32]map[string]interface{}),
		LastChangeTime:   0,
		OpenActivity:     make(map[int32]bool),
	}
	return ActivityModel
}

func LoadPlayerActivityModel(player *PlayerModel) (*PlayerActivityModel, error) {
	entities, err := easyDB.GetPlayerEntitiesByWhere[PlayerActivityEntity](map[string]interface{}{"user_id": player.GetUserId(), "is_settled": 0})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	entitiesMap := make(map[int32]*PlayerActivityEntity)
	for _, entity := range entities {
		entitiesMap[entity.ActivityId] = entity
	}

	model := &PlayerActivityModel{
		Player:           player,
		ActivityEntities: entitiesMap,
		Changed:          make(map[int32]map[string]interface{}),
		OpenActivity:     make(map[int32]bool),
	}
	return model, nil
}
