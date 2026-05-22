package model

import (
	"errors"
	"fmt"
	"slices"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

type PlayerSignEntity struct {
	UserId        int64 `gorm:"column:user_id;primaryKey"`
	SignId        int32 `gorm:"column:sign_id;primaryKey"`
	ActivityId    int32 `gorm:"column:activity_id"`
	SignedDay     int32 `gorm:"column:signed_day"`
	ClaimedIndex  int32 `gorm:"column:claimed_index"`
	LastSignDay   int32 `gorm:"column:last_sign_day"`
	CycleStartDay int32 `gorm:"column:cycle_start_day"`
	WasCovered    int32 `gorm:"column:was_covered"`
}

func (entity *PlayerSignEntity) TableName() string {
	return "player_sign_data"
}

type PlayerSignModel struct {
	Player      *PlayerModel
	Entities    map[int32]*PlayerSignEntity
	Changed     map[int32]map[string]interface{}
	NewEntities map[int32]bool
}

var _ logicCommon.PlayerModelInterface = (*PlayerSignModel)(nil)

const (
	signLoopEnabled int32 = 1
)

func NewPlayerSignModel(player *PlayerModel) *PlayerSignModel {
	return &PlayerSignModel{
		Player:      player,
		Entities:    make(map[int32]*PlayerSignEntity),
		Changed:     make(map[int32]map[string]interface{}),
		NewEntities: make(map[int32]bool),
	}
}

func LoadPlayerSignModel(player *PlayerModel) (*PlayerSignModel, error) {
	rows, err := easyDB.GetPlayerEntitiesByWhere[PlayerSignEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	model := NewPlayerSignModel(player)
	for _, row := range rows {
		model.Entities[row.SignId] = row
	}
	return model, nil
}

func (m *PlayerSignModel) SaveModelToDB() {
	for signID := range m.NewEntities {
		entity := m.Entities[signID]
		if err := easyDB.CreatePlayerEntity(entity); err != nil {
			logger.ErrorBySprintf("[PlayerSignModel] create sign row user:%d sign:%d err:%v", m.Player.GetUserId(), signID, err)
			continue
		}
		delete(m.Changed, signID)
	}
	m.NewEntities = make(map[int32]bool)

	for signID, changes := range m.Changed {
		if len(changes) == 0 {
			continue
		}
		entity := m.Entities[signID]
		easyDB.UpdatePlayerEntity(entity, changes, m.Player.GetUserId())
	}
	m.Changed = make(map[int32]map[string]interface{})
}

func (m *PlayerSignModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}

func (m *PlayerSignModel) SyncAllDaySignOnLogin(nowMilli int64) {
	cfgByID := gameConfig.GetAllDaySignCfg()
	seenActivityIDs := make(map[int32]struct{}, len(cfgByID))
	activityIDs := make([]int32, 0, len(cfgByID))
	for _, cfg := range cfgByID {
		if _, exists := seenActivityIDs[cfg.ActID]; exists {
			continue
		}
		seenActivityIDs[cfg.ActID] = struct{}{}
		activityIDs = append(activityIDs, cfg.ActID)
	}

	slices.Sort(activityIDs)

	syncActivityIDs := make([]int32, 0, len(activityIDs))
	for _, activityID := range activityIDs {
		if activityService.IsActivityOpen(m.Player.GetUserServerId(), activityID) == nil {
			for _, signID := range gameConfig.GetDaySignIdsByActID(activityID) {
				m.processExpiredSignRewardMail(signID)
				entity := m.Entities[signID]
				if entity == nil {
					continue
				}
				m.resetSignProgress(signID, entity, 0, true)
			}
			continue
		}

		cfg := activityService.GetActivityConfig(activityID)
		if cfg == nil {
			logger.ErrorBySprintf("activity config not found: %d", activityID)
			continue
		}
		activityCfg, ok := cfg.(*ServerActivityConfigEntity)
		if !ok {
			logger.ErrorBySprintf("activity config type assert failed: %d", activityID)
			continue
		}
		canSync := true
		for _, unlockID := range activityCfg.UnlockIds {
			if !unlockService.CheckUnlock(unlockID, m.Player) {
				canSync = false
				break
			}
		}
		if !canSync {
			continue
		}
		syncActivityIDs = append(syncActivityIDs, activityID)
	}

	m.SyncAndGetVisibleSigns(syncActivityIDs, nowMilli)
}

func (m *PlayerSignModel) GetOrCreateSign(signID int32) *PlayerSignEntity {
	if entity, exists := m.Entities[signID]; exists {
		return entity
	}

	entity := &PlayerSignEntity{
		UserId: m.Player.GetUserId(),
		SignId: signID,
	}
	m.Entities[signID] = entity
	m.NewEntities[signID] = true
	return entity
}

func (m *PlayerSignModel) markChanged(signID int32, field string, value interface{}) {
	if m.Changed[signID] == nil {
		m.Changed[signID] = make(map[string]interface{})
	}
	m.Changed[signID][field] = value
}

func (m *PlayerSignModel) UpdateActivityId(signID int32, activityID int32) {
	m.Entities[signID].ActivityId = activityID
	m.markChanged(signID, "activity_id", activityID)
}

func (m *PlayerSignModel) UpdateSignedDay(signID int32, signedDay int32) {
	m.Entities[signID].SignedDay = signedDay
	m.markChanged(signID, "signed_day", signedDay)
}

func (m *PlayerSignModel) UpdateClaimedIndex(signID int32, claimedIndex int32) {
	m.Entities[signID].ClaimedIndex = claimedIndex
	m.markChanged(signID, "claimed_index", claimedIndex)
}

func (m *PlayerSignModel) UpdateLastSignDay(signID int32, lastSignDay int32) {
	m.Entities[signID].LastSignDay = lastSignDay
	m.markChanged(signID, "last_sign_day", lastSignDay)
}

func (m *PlayerSignModel) UpdateCycleStartDay(signID int32, cycleStartDay int32) {
	m.Entities[signID].CycleStartDay = cycleStartDay
	m.markChanged(signID, "cycle_start_day", cycleStartDay)
}

func (m *PlayerSignModel) UpdateWasCovered(signID int32, wasCovered int32) {
	m.Entities[signID].WasCovered = wasCovered
	m.markChanged(signID, "was_covered", wasCovered)
}

func (m *PlayerSignModel) resetSignProgress(signID int32, entity *PlayerSignEntity, cycleStartDay int32, resetWasCovered bool) {
	m.UpdateActivityId(signID, 0)
	m.UpdateSignedDay(signID, 0)
	m.UpdateClaimedIndex(signID, 0)
	m.UpdateLastSignDay(signID, 0)
	m.UpdateCycleStartDay(signID, cycleStartDay)
	if resetWasCovered {
		m.UpdateWasCovered(signID, 0)
	}
}

func isSignRoundReadyToSettle(signID int32, entity *PlayerSignEntity, today int32) bool {
	if entity == nil || entity.LastSignDay == today {
		return false
	}
	cfg := gameConfig.GetDaySignCfg(signID)
	rewardDayCount := int32(len(cfg.DropID))
	if rewardDayCount > 0 && entity.SignedDay >= rewardDayCount {
		return true
	}
	if cfg.Duration > 0 && entity.CycleStartDay > 0 {
		return tool.GetNatureDayDistanceByDateInt(today, entity.CycleStartDay) >= cfg.Duration
	}
	return false
}

func BuildSignClaimItems(signID int32, claimFrom int32, claimTo int32) ([]*gameConfig.ItemConfig, error) {
	if claimFrom <= 0 || claimTo < claimFrom {
		return nil, nil
	}

	cfg := gameConfig.GetDaySignCfg(signID)
	items := make([]*gameConfig.ItemConfig, 0)
	for day := claimFrom; day <= claimTo; day++ {
		dropIdx := day - 1
		if dropIdx < 0 || int(dropIdx) >= len(cfg.DropID) {
			return nil, fmt.Errorf("sign reward drop index out of range sign:%d day:%d", cfg.Id, day)
		}
		dropID := cfg.DropID[int(dropIdx)]
		dropItems := gameConfig.Drop(dropID)
		if len(dropItems) == 0 {
			return nil, fmt.Errorf("sign reward drop empty sign:%d day:%d drop:%d", cfg.Id, day, dropID)
		}
		items = append(items, dropItems...)
	}
	return items, nil
}

func (m *PlayerSignModel) processExpiredSignRewardMail(signID int32) bool {
	cfg := gameConfig.GetDaySignCfg(signID)
	mailConstantName := gameConfig.CONSTANT_sevenSignMail
	if cfg.Permanent > 0 {
		mailConstantName = gameConfig.CONSTANT_actSignMail
	}
	templateID := int32(0)
	mailConstant := gameConfig.GetConstantCfg(mailConstantName)
	if mailConstant != nil && len(mailConstant.Value) > 0 {
		templateID = mailConstant.Value[0]
	}

	entity := m.Entities[signID]
	claimTo := int32(0)
	if entity != nil {
		rewardDayCount := int32(len(cfg.DropID))
		if rewardDayCount > 0 {
			claimTo = min(entity.SignedDay, rewardDayCount)
		}
	}
	if entity == nil || claimTo <= entity.ClaimedIndex {
		return true
	}

	mailSent := true
	items, err := BuildSignClaimItems(signID, entity.ClaimedIndex+1, claimTo)
	if err != nil {
		logger.ErrorBySprintf("[PlayerSignModel] build sign expire rewards failed user:%d sign:%d err:%v", m.Player.GetUserId(), signID, err)
		mailSent = false
	} else if templateID <= 0 {
		logger.ErrorBySprintf("[PlayerSignModel] sign expire mail template invalid user:%d sign:%d template:%d", m.Player.GetUserId(), signID, templateID)
		mailSent = false
	} else if _, err = mailServer.SendRewardMailByTemplateID(m.Player.GetUserId(), templateID, items, nil, nil); err != nil {
		logger.ErrorBySprintf("[PlayerSignModel] send sign expire mail failed user:%d sign:%d template:%d err:%v", m.Player.GetUserId(), signID, templateID, err)
		mailSent = false
	}

	m.MergeClaimed(signID, claimTo)
	return mailSent
}

func (m *PlayerSignModel) SyncAndGetVisibleSigns(activityIDs []int32, nowMilli int64) []int32 {
	today := tool.GetTodayDataIntByTimeStamp(nowMilli)
	signIDs := make([]int32, 0)
	for _, activityID := range activityIDs {
		for _, signID := range gameConfig.GetDaySignIdsByActID(activityID) {
			cfg := gameConfig.GetDaySignCfg(signID)
			if cfg.Loop != signLoopEnabled && isSignRoundReadyToSettle(signID, m.Entities[signID], today) {
				m.processExpiredSignRewardMail(signID)
				continue
			}
			signIDs = append(signIDs, signID)
		}
	}

	visibleCfgByCase := make(map[int32]*gameConfig.DaySignCfg)
	for _, signID := range signIDs {
		cfg := gameConfig.GetDaySignCfg(signID)
		currentVisibleCfg := visibleCfgByCase[cfg.Case]
		if currentVisibleCfg == nil ||
			cfg.Sort > currentVisibleCfg.Sort ||
			(cfg.Sort == currentVisibleCfg.Sort && cfg.Id < currentVisibleCfg.Id) {
			visibleCfgByCase[cfg.Case] = cfg
		}
	}

	coveredByID := make(map[int32]bool, len(signIDs))
	for _, signID := range signIDs {
		cfg := gameConfig.GetDaySignCfg(signID)
		visibleCfg := visibleCfgByCase[cfg.Case]
		if cfg.Id == visibleCfg.Id {
			continue
		}
		coveredByID[signID] = true
	}

	visibleSignIDs := make([]int32, 0, len(signIDs))
	for _, signID := range signIDs {
		m.TryAutoSignToday(signID, nowMilli, coveredByID[signID])
		if coveredByID[signID] {
			continue
		}
		visibleSignIDs = append(visibleSignIDs, signID)
	}
	return visibleSignIDs
}

func (m *PlayerSignModel) MergeClaimed(signID int32, claimedIndex int32) {
	m.UpdateClaimedIndex(signID, claimedIndex)
}

func (m *PlayerSignModel) TryAutoSignToday(signID int32, nowMilli int64, covered bool) {
	cfg := gameConfig.GetDaySignCfg(signID)
	entity := m.GetOrCreateSign(signID)
	if covered {
		if entity.WasCovered != 1 {
			m.UpdateWasCovered(signID, 1)
		}
		return
	}

	today := tool.GetTodayDataIntByTimeStamp(nowMilli)
	if entity.LastSignDay == today {
		return
	}
	if entity.WasCovered == 1 {
		m.resetSignProgress(signID, entity, 0, true)
	}

	if entity.CycleStartDay <= 0 {
		m.UpdateCycleStartDay(signID, today)
	}
	if cfg.Loop == signLoopEnabled {
		if isSignRoundReadyToSettle(signID, entity, today) {
			m.processExpiredSignRewardMail(signID)
			m.resetSignProgress(signID, entity, today, false)
		}
	} else {
		rewardDayCount := int32(len(cfg.DropID))
		if rewardDayCount > 0 && entity.SignedDay >= rewardDayCount {
			return
		}
	}

	signedDay := entity.SignedDay + 1
	rewardDayCount := int32(len(cfg.DropID))
	if signedDay > rewardDayCount {
		signedDay = rewardDayCount
	}
	if signedDay < entity.ClaimedIndex {
		signedDay = entity.ClaimedIndex
	}
	if signedDay <= entity.SignedDay {
		m.UpdateLastSignDay(signID, today)
		m.UpdateWasCovered(signID, 0)
		return
	}

	m.UpdateSignedDay(signID, signedDay)
	m.UpdateLastSignDay(signID, today)
	m.UpdateWasCovered(signID, 0)
}
