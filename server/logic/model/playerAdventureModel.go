package model

import (
	"errors"
	"sort"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ADVENTURE_ENTRY_STATUS_WAIT     = 1
	ADVENTURE_ENTRY_STATUS_FIGHTING = 2
	ADVENTURE_ENTRY_STATUS_SETTLED  = 3
	ADVENTURE_ENTRY_STATUS_EXPIRED  = 4
)

type PlayerAdventureEntity struct {
	UserId            int64 `gorm:"column:user_id;primaryKey"`
	Progress          int32 `gorm:"column:progress;default:0"`
	DailyTriggerCount int32 `gorm:"column:daily_trigger_count;default:0"`
	DailySettleCount  int32 `gorm:"column:daily_settle_count;default:0"`
	LastResetTime     int64 `gorm:"column:last_reset_time;default:0"`
}

func (p *PlayerAdventureEntity) TableName() string {
	return "player_adventure"
}

type PlayerAdventureSettleTypeEntity struct {
	UserId        int64 `gorm:"column:user_id;primaryKey"`
	AdventureType int32 `gorm:"column:adventure_type;primaryKey"`
	SettleCount   int32 `gorm:"column:settle_count;default:0"`
	LastResetTime int64 `gorm:"column:last_reset_time;default:0"`
}

func (p *PlayerAdventureSettleTypeEntity) TableName() string {
	return "player_adventure_settle_type"
}

type PlayerAdventureEntryEntity struct {
	UniqueId    string `gorm:"column:unique_id;primaryKey" json:"uniqueId"`
	UserId      int64  `gorm:"column:user_id" json:"userId"`
	AdventureId int32  `gorm:"column:adventure_id" json:"adventureId"`
	DungeonId   int32  `gorm:"column:dungeon_id" json:"dungeonId"`
	CreateTime  int64  `gorm:"column:create_time" json:"createTime"`
	ExpireTime  int64  `gorm:"column:expire_time" json:"expireTime"`
	Status      int32  `gorm:"column:status" json:"status"`
}

func (p *PlayerAdventureEntryEntity) TableName() string {
	return "player_adventure_entry"
}

type AdventureEntry = PlayerAdventureEntryEntity

type PlayerAdventureModel struct {
	UserId             int64
	Entity             *PlayerAdventureEntity
	SettleTypeEntities map[int32]*PlayerAdventureSettleTypeEntity
	EntryEntities      map[string]*PlayerAdventureEntryEntity
	Changed            map[string]interface{}
	SettleTypeChanged  map[int32]map[string]interface{}
	EntryChanged       map[string]map[string]interface{}
	player             *PlayerModel
}

var _ logicCommon.PlayerModelInterface = (*PlayerAdventureModel)(nil)

func NewPlayerAdventureModel(userId int64, entity *PlayerAdventureEntity, player *PlayerModel) *PlayerAdventureModel {
	return &PlayerAdventureModel{
		UserId:             userId,
		Entity:             entity,
		SettleTypeEntities: make(map[int32]*PlayerAdventureSettleTypeEntity),
		EntryEntities:      make(map[string]*PlayerAdventureEntryEntity),
		Changed:            make(map[string]interface{}),
		SettleTypeChanged:  make(map[int32]map[string]interface{}),
		EntryChanged:       make(map[string]map[string]interface{}),
		player:             player,
	}
}

func LoadPlayerAdventureModel(userId int64, player *PlayerModel) (*PlayerAdventureModel, error) {
	entity, err := easyDB.GetPlayerEntityByWhere[PlayerAdventureEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CreatePlayerAdventureModel(userId, player)
		}
		return nil, err
	}
	m := NewPlayerAdventureModel(userId, entity, player)
	settleTypeEntities, err := easyDB.GetPlayerEntitiesByWhere[PlayerAdventureSettleTypeEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, entity := range settleTypeEntities {
		if entity != nil {
			m.SettleTypeEntities[entity.AdventureType] = entity
		}
	}
	entryEntities, err := easyDB.GetPlayerEntitiesByWhere[PlayerAdventureEntryEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for _, entity := range entryEntities {
		if entity != nil {
			m.EntryEntities[entity.UniqueId] = entity
		}
	}
	return m, nil
}

func CreatePlayerAdventureModel(userId int64, player *PlayerModel) (*PlayerAdventureModel, error) {
	entity := &PlayerAdventureEntity{
		UserId:        userId,
		LastResetTime: tool.UnixNowMilli(),
	}
	if err := easyDB.CreatePlayerEntity[PlayerAdventureEntity](entity); err != nil {
		return nil, err
	}
	return NewPlayerAdventureModel(userId, entity, player), nil
}

func (p *PlayerAdventureModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}

func (p *PlayerAdventureModel) SaveModelToDB() {
	if p == nil || p.Entity == nil {
		return
	}
	if len(p.Changed) > 0 {
		easyDB.UpdatePlayerEntity(p.Entity, p.Changed, p.UserId)
		p.Changed = make(map[string]interface{})
	}
	for adventureType, changes := range p.SettleTypeChanged {
		entity := p.SettleTypeEntities[adventureType]
		if entity == nil {
			continue
		}
		if _, ok := changes["user_id"]; ok {
			_ = easyDB.CreatePlayerEntity(entity)
		} else {
			easyDB.UpdatePlayerEntity(entity, changes, p.UserId)
		}
	}
	p.SettleTypeChanged = make(map[int32]map[string]interface{})
	for uniqueId, changes := range p.EntryChanged {
		entity := p.EntryEntities[uniqueId]
		if entity == nil {
			continue
		}
		if _, ok := changes["unique_id"]; ok {
			_ = easyDB.CreatePlayerEntity(entity)
		} else {
			easyDB.UpdatePlayerEntity(entity, changes, p.UserId)
		}
	}
	p.EntryChanged = make(map[string]map[string]interface{})
}

func (p *PlayerAdventureModel) getChangedMap() map[string]interface{} {
	if p.Changed == nil {
		p.Changed = make(map[string]interface{})
	}
	return p.Changed
}

func (p *PlayerAdventureModel) getSettleTypeChangedMap(adventureType int32) map[string]interface{} {
	if p.SettleTypeChanged == nil {
		p.SettleTypeChanged = make(map[int32]map[string]interface{})
	}
	if p.SettleTypeChanged[adventureType] == nil {
		p.SettleTypeChanged[adventureType] = make(map[string]interface{})
	}
	return p.SettleTypeChanged[adventureType]
}

func (p *PlayerAdventureModel) getEntryChangedMap(uniqueId string) map[string]interface{} {
	if p.EntryChanged == nil {
		p.EntryChanged = make(map[string]map[string]interface{})
	}
	if p.EntryChanged[uniqueId] == nil {
		p.EntryChanged[uniqueId] = make(map[string]interface{})
	}
	return p.EntryChanged[uniqueId]
}

func (p *PlayerAdventureModel) ResetDaily(now int64) {
	if p == nil || p.Entity == nil {
		return
	}
	if p.Entity.LastResetTime != 0 && tool.IsSameDay(tool.MilliToTime(p.Entity.LastResetTime), tool.MilliToTime(now)) {
		return
	}
	p.Entity.Progress = 0
	p.Entity.DailyTriggerCount = 0
	p.Entity.DailySettleCount = 0
	p.Entity.LastResetTime = now
	changed := p.getChangedMap()
	changed["progress"] = p.Entity.Progress
	changed["daily_trigger_count"] = p.Entity.DailyTriggerCount
	changed["daily_settle_count"] = p.Entity.DailySettleCount
	changed["last_reset_time"] = p.Entity.LastResetTime
	for adventureType, entity := range p.SettleTypeEntities {
		if entity == nil {
			continue
		}
		entity.SettleCount = 0
		entity.LastResetTime = now
		typeChanged := p.getSettleTypeChangedMap(adventureType)
		typeChanged["settle_count"] = entity.SettleCount
		typeChanged["last_reset_time"] = entity.LastResetTime
	}
	p.RemoveWaitEntries()
}

func (p *PlayerAdventureModel) GetEntries() []*AdventureEntry {
	if p == nil {
		return make([]*AdventureEntry, 0)
	}
	res := make([]*AdventureEntry, 0, len(p.EntryEntities))
	for _, entry := range p.EntryEntities {
		if entry != nil {
			res = append(res, entry)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].CreateTime == res[j].CreateTime {
			return res[i].UniqueId < res[j].UniqueId
		}
		return res[i].CreateTime < res[j].CreateTime
	})
	return res
}

func (p *PlayerAdventureModel) GetDailySettleTypeCount() map[int32]int32 {
	res := make(map[int32]int32)
	if p == nil {
		return res
	}
	for adventureType, entity := range p.SettleTypeEntities {
		if entity != nil {
			res[adventureType] = entity.SettleCount
		}
	}
	return res
}

func (p *PlayerAdventureModel) GetActiveEntries(now int64) []*AdventureEntry {
	entries := p.GetEntries()
	res := make([]*AdventureEntry, 0)
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.Status == ADVENTURE_ENTRY_STATUS_WAIT && entry.ExpireTime > now && p.CanSettle(entry.AdventureId) {
			res = append(res, entry)
		}
	}
	return res
}

func (p *PlayerAdventureModel) GetEntry(uniqueId string, now int64) *AdventureEntry {
	for _, entry := range p.GetActiveEntries(now) {
		if entry.UniqueId == uniqueId {
			return entry
		}
	}
	return nil
}

func (p *PlayerAdventureModel) AddEntry(adventureId int32, dungeonId int32, createTime int64, expireTime int64) string {
	if p == nil || p.Entity == nil {
		return ""
	}
	uniqueId := uuid.New().String()
	entry := &PlayerAdventureEntryEntity{
		UniqueId:    uniqueId,
		UserId:      p.UserId,
		AdventureId: adventureId,
		DungeonId:   dungeonId,
		CreateTime:  createTime,
		ExpireTime:  expireTime,
		Status:      ADVENTURE_ENTRY_STATUS_WAIT,
	}
	p.EntryEntities[uniqueId] = entry
	changed := p.getEntryChangedMap(uniqueId)
	changed["unique_id"] = entry.UniqueId
	changed["user_id"] = entry.UserId
	changed["adventure_id"] = entry.AdventureId
	changed["dungeon_id"] = entry.DungeonId
	changed["create_time"] = entry.CreateTime
	changed["expire_time"] = entry.ExpireTime
	changed["status"] = entry.Status
	return uniqueId
}

func (p *PlayerAdventureModel) MarkEntryFighting(uniqueId string, now int64) bool {
	if p == nil {
		return false
	}
	entry := p.EntryEntities[uniqueId]
	if entry == nil || entry.Status != ADVENTURE_ENTRY_STATUS_WAIT || entry.ExpireTime <= now {
		return false
	}
	entry.Status = ADVENTURE_ENTRY_STATUS_FIGHTING
	p.getEntryChangedMap(uniqueId)["status"] = entry.Status
	return true
}

func (p *PlayerAdventureModel) MarkEntryWait(uniqueId string) bool {
	if p == nil {
		return false
	}
	entry := p.EntryEntities[uniqueId]
	if entry == nil || entry.Status != ADVENTURE_ENTRY_STATUS_FIGHTING {
		return false
	}
	entry.Status = ADVENTURE_ENTRY_STATUS_WAIT
	p.getEntryChangedMap(uniqueId)["status"] = entry.Status
	return true
}

func (p *PlayerAdventureModel) MarkEntrySettled(uniqueId string) bool {
	if p == nil {
		return false
	}
	entry := p.EntryEntities[uniqueId]
	if entry == nil {
		return false
	}
	entry.Status = ADVENTURE_ENTRY_STATUS_SETTLED
	p.getEntryChangedMap(uniqueId)["status"] = entry.Status
	return true
}

func (p *PlayerAdventureModel) RemoveExpiredEntries(now int64) int32 {
	if p == nil {
		return 0
	}
	changed := int32(0)
	for uniqueId, entry := range p.EntryEntities {
		if entry == nil {
			continue
		}
		if entry.Status == ADVENTURE_ENTRY_STATUS_WAIT && entry.ExpireTime <= now {
			entry.Status = ADVENTURE_ENTRY_STATUS_EXPIRED
			p.getEntryChangedMap(uniqueId)["status"] = entry.Status
			changed++
		}
	}
	return changed
}

func (p *PlayerAdventureModel) RemoveWaitEntries() int32 {
	if p == nil {
		return 0
	}
	changed := int32(0)
	for uniqueId, entry := range p.EntryEntities {
		if entry == nil {
			continue
		}
		if entry.Status == ADVENTURE_ENTRY_STATUS_WAIT {
			entry.Status = ADVENTURE_ENTRY_STATUS_EXPIRED
			p.getEntryChangedMap(uniqueId)["status"] = entry.Status
			changed++
		}
	}
	return changed
}

func (p *PlayerAdventureModel) RemoveWaitEntriesByAdventureId(adventureId int32) int32 {
	if p == nil {
		return 0
	}
	changed := int32(0)
	for uniqueId, entry := range p.EntryEntities {
		if entry == nil {
			continue
		}
		if entry.Status == ADVENTURE_ENTRY_STATUS_WAIT && entry.AdventureId == adventureId {
			entry.Status = ADVENTURE_ENTRY_STATUS_EXPIRED
			p.getEntryChangedMap(uniqueId)["status"] = entry.Status
			changed++
		}
	}
	return changed
}

func (p *PlayerAdventureModel) CanSettle(adventureId int32) bool {
	if p == nil || p.Entity == nil {
		return false
	}
	cfg := gameConfig.GetAdventureCfg(adventureId)
	if cfg == nil {
		return false
	}
	if p.getSettleTypeCount(adventureId) >= cfg.Limit {
		return false
	}
	return p.Entity.DailySettleCount < gameConfig.GetDailyAdventureLimit()
}

func (p *PlayerAdventureModel) AddSettleCount(adventureId int32) {
	if p == nil || p.Entity == nil {
		return
	}
	p.Entity.DailySettleCount++
	p.getChangedMap()["daily_settle_count"] = p.Entity.DailySettleCount
	entity := p.getOrCreateSettleTypeEntity(adventureId)
	entity.SettleCount++
	changed := p.getSettleTypeChangedMap(adventureId)
	changed["settle_count"] = entity.SettleCount
	changed["last_reset_time"] = entity.LastResetTime
}

func (p *PlayerAdventureModel) SetProgress(progress int32) {
	if p == nil || p.Entity == nil {
		return
	}
	p.Entity.Progress = progress
	p.getChangedMap()["progress"] = progress
}

func (p *PlayerAdventureModel) AddProgress(progress int32) {
	if p == nil || p.Entity == nil {
		return
	}
	p.SetProgress(p.Entity.Progress + progress)
}

func (p *PlayerAdventureModel) AddDailyTriggerCount() {
	if p == nil || p.Entity == nil {
		return
	}
	p.Entity.DailyTriggerCount++
	p.getChangedMap()["daily_trigger_count"] = p.Entity.DailyTriggerCount
}

func (p *PlayerAdventureModel) getSettleTypeCount(adventureId int32) int32 {
	if p == nil || p.Entity == nil {
		return 0
	}
	entity := p.SettleTypeEntities[adventureId]
	if entity == nil {
		return 0
	}
	if entity.LastResetTime == 0 || tool.IsSameDay(tool.MilliToTime(entity.LastResetTime), tool.MilliToTime(p.Entity.LastResetTime)) {
		return entity.SettleCount
	}
	entity.SettleCount = 0
	entity.LastResetTime = p.Entity.LastResetTime
	changed := p.getSettleTypeChangedMap(adventureId)
	changed["settle_count"] = entity.SettleCount
	changed["last_reset_time"] = entity.LastResetTime
	return 0
}

func (p *PlayerAdventureModel) getOrCreateSettleTypeEntity(adventureId int32) *PlayerAdventureSettleTypeEntity {
	if p.SettleTypeEntities == nil {
		p.SettleTypeEntities = make(map[int32]*PlayerAdventureSettleTypeEntity)
	}
	if entity := p.SettleTypeEntities[adventureId]; entity != nil {
		return entity
	}
	lastResetTime := int64(0)
	if p.Entity != nil {
		lastResetTime = p.Entity.LastResetTime
	}
	if lastResetTime == 0 {
		lastResetTime = tool.UnixNowMilli()
	}
	entity := &PlayerAdventureSettleTypeEntity{
		UserId:        p.UserId,
		AdventureType: adventureId,
		SettleCount:   0,
		LastResetTime: lastResetTime,
	}
	p.SettleTypeEntities[adventureId] = entity
	changed := p.getSettleTypeChangedMap(adventureId)
	changed["user_id"] = entity.UserId
	changed["adventure_type"] = entity.AdventureType
	changed["settle_count"] = entity.SettleCount
	changed["last_reset_time"] = entity.LastResetTime
	return entity
}
