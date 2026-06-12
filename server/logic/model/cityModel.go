package model

import (
	"errors"
	"slices"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/service/logger"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"

	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

type ArchitectureEntity struct {
	UserId      int64 `gorm:"column:user_id;primaryKey"`
	Type        int32 `gorm:"column:type;primaryKey"`
	Level       int32 `gorm:"column:level"`
	Status      int32 `gorm:"column:status"`
	UpStartTime int64 `gorm:"column:up_start_time"`
}

func (a *ArchitectureEntity) TableName() string { return "architecture" }

const cityCenterEffectTypeUnlockArchitecture int32 = 1

// ArchitectureUpgradeCallback 建筑升级完成回调函数类型
// 用于在建筑升级完成时通知外部模块（如伐木场）执行相应的结算和重置逻辑
// 通过回调机制避免 model 包与具体业务包之间的循环依赖
type ArchitectureUpgradeCallback func(player *PlayerModel, archType int32, oldLevel int32, senderMsg bool)

type ArchitectureModel struct {
	UserId            int64
	Entities          map[int32]*ArchitectureEntity
	Changed           map[int32]map[string]interface{}
	Player            *PlayerModel
	OnUpgradeCallback ArchitectureUpgradeCallback // 建筑升级完成回调（伐木场等生产建筑使用）
}

func (a *ArchitectureModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	attrNum := int64(0)
	cityCenterCfg := gameConfig.GetCityCenterCfg(a.Entities[int32(enum.ARCHITECTURE_TYPE_MAIN)].Level)
	if cityCenterCfg != nil {
		for id, v := range cityCenterCfg.Attr {
			if v == attrId {
				attrNum = int64(cityCenterCfg.AttrNum[id])
				break
			}
		}
	}
	collectionBuildDetail := a.Entities[int32(enum.ARCHITECTURE_TYPE_COLLECTION)]
	if collectionBuildDetail != nil {
		collectionBuildCfg := gameConfig.GetCollectionCfg(collectionBuildDetail.Level)
		if collectionBuildCfg != nil {
			for id, v := range collectionBuildCfg.AttrId {
				if v == attrId {
					attrNum = int64(collectionBuildCfg.AttrNum[id])
					break
				}
			}
		}
	}
	return attrNum
}

func (a *ArchitectureModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记
// 建筑等级影响全部英雄属性，allDirty=true
func (a *ArchitectureModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(a.Changed) == 0 {
		return []int64{}, false
	}
	return []int64{}, true
}

func NewArchitectureModel(entities map[int32]*ArchitectureEntity, userid int64, player *PlayerModel) *ArchitectureModel {
	return &ArchitectureModel{
		UserId:   userid,
		Entities: entities,
		Changed:  make(map[int32]map[string]interface{}),
		Player:   player,
	}
}

var _ logicCommon.PlayerModelInterface = (*ArchitectureModel)(nil)
var _ logicCommon.HeroAttrInterface = (*ArchitectureModel)(nil)

func (a *ArchitectureModel) SaveModelToDB() {
	for key, v := range a.Changed {
		easyDB.UpdatePlayerEntity(a.Entities[key], v, a.UserId)
	}
	a.Changed = make(map[int32]map[string]interface{})
}

func (a *ArchitectureModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	for key, v := range a.Entities {
		cfg := gameConfig.GetCityLevelCfg(key, v.Level+1)
		if v.Status == 3 || v.Status == 2 {
			if currentTime-v.UpStartTime >= int64(cfg.GetTime())*1000 {
				a.OnUpgradeComplete(v.Type, v.Level, senderMsg)
			}
		}
	}

	// 自动解锁建筑
	allSystemUnlockCfg := gameConfig.GetAllSystemUnlockCfg()
	if allSystemUnlockCfg == nil {
		return
	}
	for key, v := range allSystemUnlockCfg {
		if v.ParentFunction == int32(enum.FUNCTION_ID_CAPITAL) {
			architectureId := enum.GetArchitectureIdFormFunctionId(key)
			if architectureId == 0 {
				continue
			}
			if a.Entities[architectureId] != nil {
				continue
			}
			flag := unlockService.CheckSystemUnlock(key, a.Player)
			if flag {
				if err := a.AddArchitectureEntity(architectureId, 0); err != nil {
					logger.ErrorBySprintf("add architecture error")
				}
				if senderMsg {
					messageSender.SendMessage(a.Player, pb.MESSAGE_ID_PUSH_ARCHITECTURE_INFO, &pb.PushArchitectureInfo{
						ArInfo: &pb.ArchitectureInfo{
							Type:   architectureId,
							Level:  0,
							Status: 0,
						},
					})
				}
			}
		}
	}
}

func LoadArchitecture(userId int64, player *PlayerModel) (*ArchitectureModel, error) {
	if userId == 0 {
		return nil, errors.New("userId is null")
	}
	row, err := easyDB.GetPlayerEntitiesByWhere[ArchitectureEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	entities := make(map[int32]*ArchitectureEntity)
	for _, v := range row {
		entities[v.Type] = v
	}
	return NewArchitectureModel(entities, userId, player), nil
}

func (a *ArchitectureModel) creatArchitectureEntity(entity *ArchitectureEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (a *ArchitectureModel) AddArchitectureEntity(ttype, level int32) error {
	entity := &ArchitectureEntity{
		UserId:      a.UserId,
		Type:        ttype,
		Level:       level,
		Status:      0,
		UpStartTime: 0,
	}
	a.Entities[ttype] = entity
	return a.creatArchitectureEntity(entity)
}

func (a *ArchitectureModel) EnsureUnlockedArchitectureEntities() {
	mainEntity := a.Entities[int32(enum.ARCHITECTURE_TYPE_MAIN)]
	if mainEntity == nil {
		return
	}
	for level := int32(1); level <= mainEntity.Level; level++ {
		for _, archType := range getUnlockedArchitectureTypes(level) {
			if !a.canEnsureArchitectureEntity(archType) {
				continue
			}
			if err := a.AddArchitectureEntity(archType, 0); err != nil {
				logger.ErrorBySprintf("add architecture error")
			}
		}
	}
}

func getUnlockedArchitectureTypes(mainLevel int32) []int32 {
	cfg := gameConfig.GetCityCenterCfg(mainLevel)
	if cfg == nil {
		return nil
	}
	if slices.Contains(cfg.EffectType, cityCenterEffectTypeUnlockArchitecture) {
		return cfg.EffectPara
	}
	return nil
}

func (a *ArchitectureModel) canEnsureArchitectureEntity(archType int32) bool {
	if archType == int32(enum.ARCHITECTURE_TYPE_MAIN) {
		return false
	}
	_, exists := a.Entities[archType]
	return !exists
}

func (a *ArchitectureModel) UpdateLevel(ttype int32, level int32) {
	a.Entities[ttype].Level = level
	if a.Changed[ttype] == nil {
		a.Changed[ttype] = make(map[string]interface{})
	}
	a.Changed[ttype]["level"] = level
}

func (a *ArchitectureModel) UpdateStatus(ttype int32, status int32) {
	a.Entities[ttype].Status = status
	if a.Changed[ttype] == nil {
		a.Changed[ttype] = make(map[string]interface{})
	}
	a.Changed[ttype]["status"] = status
}

func (a *ArchitectureModel) UpdateUpStartTime(ttype int32, upStartTime int64) {
	a.Entities[ttype].UpStartTime = upStartTime
	if a.Changed[ttype] == nil {
		a.Changed[ttype] = make(map[string]interface{})
	}
	a.Changed[ttype]["up_start_time"] = upStartTime
}

// OnUpgradeComplete 建筑升级完成时的处理逻辑
// 包含：更新状态、升级等级、主城特殊处理（解锁石像）、推送英雄战力变化
func (a *ArchitectureModel) OnUpgradeComplete(archType int32, oldLevel int32, senderMsg bool) {
	a.UpdateStatus(archType, 1)
	a.UpdateLevel(archType, oldLevel+1)
	// 上报建筑升级日志
	operationLogService.OnUserArchitecture(a.Player.GetUserId(), archType, oldLevel, a.Entities[archType].Level)
	eventServer.SubmitBuildLevelUpEvent(a.Player.GetUserId(), a.Player.GetUserServerId(), enum.ArchitectureType(archType), a.Entities[archType].Level)
	// 主城升级特殊处理：可能解锁石像建筑
	if archType == int32(enum.ARCHITECTURE_TYPE_MAIN) {
		a.Player.User.UpdateLevel(a.GetMainLevel())
		if senderMsg {
			messageSender.SendMessage(a.Player, pb.MESSAGE_ID_PUSH_PLAYER_BASIC_INFO, &pb.PushPlayerBasicInfo{
				BasicInfo: &pb.PlayerBasicInfo{
					Level: a.GetMainLevel(),
				},
			})
		}

		// 推送英雄战力变化
		heroInfos := make([]*pb.HeroBagInfo, 0)
		for _, formation := range a.Player.HeroFormationModel.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
			if formation.IsActive != true {
				continue
			}
			for _, heroOwnID := range formation.HeroOwnIDList {
				heroInfos = append(heroInfos, a.Player.HeroDetailsModel.GetHeroInfoByOwnID(a.Player, heroOwnID))
			}
		}
		if senderMsg {
			messageSender.SendMessage(a.Player, pb.MESSAGE_ID_PUSH_HERO_POWER_CHANGE, &pb.PushHeroPowerChange{
				HeroInfos: heroInfos,
				PushType:  0,
			})
		}
	}

	// 触发建筑升级完成回调，通知伐木场等生产建筑按旧等级结算并重置
	if a.OnUpgradeCallback != nil {
		a.OnUpgradeCallback(a.Player, archType, oldLevel, senderMsg)
	}
}

func (a *ArchitectureModel) GetMainLevel() int32 {
	return a.Entities[int32(enum.ARCHITECTURE_TYPE_MAIN)].Level
}

type StoneEntity struct {
	UserId    int64               `gorm:"primaryKey;column:user_id"`
	Class     int32               `gorm:"primaryKey;column:class"`
	AttrLevel tool.JSONInt32Slice `gorm:"column:attr_level;type:json"`
}

func (s *StoneEntity) TableName() string { return "stone" }

type StoneModel struct {
	UserId   int64
	Entities map[int32]*StoneEntity
	Changed  map[int32]map[string]interface{}

	player *PlayerModel
}

func (s *StoneModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	heroDetail := s.player.HeroDetailsModel.Entities[heroId]
	heroCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))

	if heroCfg != nil {
		if v, ok := gameConfig.GetStatueAttrIndexMap()[heroCfg.HeroClass][attrId]; ok {
			if s.Entities[heroCfg.HeroClass] == nil {
				return 0
			}
			if v <= int32(len(s.Entities[heroCfg.HeroClass].AttrLevel)) {
				classLevel := s.Entities[heroCfg.HeroClass].AttrLevel[v]
				return gameConfig.GetStatueAttrInfoByLevel()[heroCfg.HeroClass][attrId][classLevel]
			}
		}
	}
	return 0
}

func (s *StoneModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记
// 石像属性影响全部英雄，allDirty=true
func (s *StoneModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(s.Changed) == 0 {
		return []int64{}, false
	}
	return []int64{}, true
}

func NewStoneModel(entities map[int32]*StoneEntity, userid int64, player *PlayerModel) *StoneModel {
	return &StoneModel{
		UserId:   userid,
		Entities: entities,
		Changed:  make(map[int32]map[string]interface{}),
		player:   player,
	}
}

var _ logicCommon.PlayerModelInterface = (*StoneModel)(nil)
var _ logicCommon.HeroAttrInterface = (*StoneModel)(nil)

func (s *StoneModel) SaveModelToDB() {
	for key, v := range s.Changed {
		easyDB.UpdatePlayerEntity(s.Entities[key], v, s.UserId)
	}
	s.Changed = make(map[int32]map[string]interface{})
}

func (s *StoneModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {

}

func LoadStone(userId int64, player *PlayerModel) (*StoneModel, error) {
	if userId == 0 {
		return nil, errors.New("userId is null")
	}
	row, err := easyDB.GetPlayerEntitiesByWhere[StoneEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	entities := make(map[int32]*StoneEntity)
	for _, v := range row {
		entities[v.Class] = v
	}
	return NewStoneModel(entities, userId, player), nil
}

func (s *StoneModel) creatStoneEntity(entity *StoneEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (s *StoneModel) AddStoneEntity(class int32, attrLevel []int32) error {
	entity := &StoneEntity{
		UserId:    s.UserId,
		Class:     class,
		AttrLevel: tool.JSONInt32Slice(attrLevel),
	}
	s.Entities[class] = entity
	return s.creatStoneEntity(entity)
}

func (s *StoneModel) UpdateAttrLevel(class int32, attrLevel []int32) {
	s.Entities[class].AttrLevel = tool.JSONInt32Slice(attrLevel)
	if s.Changed[class] == nil {
		s.Changed[class] = make(map[string]interface{})
	}
	s.Changed[class]["attr_level"] = tool.JSONInt32Slice(attrLevel)
}

type CollectionEntity struct {
	UserId                int64 `gorm:"primaryKey;column:user_id"`
	CollectionAttribution int32 `gorm:"primaryKey;column:collection_attribution"`
	CollectId             int32 `gorm:"column:collect_id"`
	CollectLevel          int32 `gorm:"column:collect_level"`
}

func (c *CollectionEntity) TableName() string { return "collection" }

type CollectionEntryEntity struct {
	UserId     int64 `gorm:"column:user_id;primary_key"`
	EntryId    int32 `gorm:"column:entry_id;primary_key"` // 词条Id
	EntryLevel int32 `gorm:"column:entry_level"`
}

func (c *CollectionEntryEntity) TableName() string { return "collection_entry" }

type CollectionModel struct {
	userId           int64
	CollectionEntity map[int32]*CollectionEntity

	EntryEntity map[int32]*CollectionEntryEntity

	ItemsChanged map[int32]map[string]interface{}
	EntryChanged map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*CollectionModel)(nil)
var _ logicCommon.HeroAttrInterface = (*CollectionModel)(nil)

func NewCollectionModel(userId int64, itemsEntity map[int32]*CollectionEntity, entryEntity map[int32]*CollectionEntryEntity, cbcid map[int32]*CollectionEntity) *CollectionModel {
	return &CollectionModel{
		userId:           userId,
		CollectionEntity: itemsEntity,
		EntryEntity:      entryEntity,

		ItemsChanged: make(map[int32]map[string]interface{}),
		EntryChanged: make(map[int32]map[string]interface{}),
	}
}

func (c *CollectionModel) SaveModelToDB() {
	for key, v := range c.ItemsChanged {
		easyDB.UpdatePlayerEntity(c.CollectionEntity[key], v, c.userId)
	}
	c.ItemsChanged = make(map[int32]map[string]interface{})
	for key, v := range c.EntryChanged {
		easyDB.UpdatePlayerEntity(c.EntryEntity[key], v, c.userId)
	}
	c.EntryChanged = make(map[int32]map[string]interface{})
}

func (c *CollectionModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	attrNum := int64(0)
	for _, v := range c.CollectionEntity {
		entityCfg := gameConfig.GetCollectionEntityCfg(v.CollectId)
		if entityCfg == nil {
			continue
		}
		for id, attr := range entityCfg.Attrid {
			if attr == attrId {
				attrNum += int64(entityCfg.AttrNum[id])
			}
		}
	}
	for _, v := range c.EntryEntity {
		attrNum += gameConfig.GetEntryAttrInfoByEntryIdAndLevel(v.EntryId, v.EntryLevel, attrId)
	}
	return attrNum
}

func (c *CollectionModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

func (c *CollectionModel) GetChangedHeroOwnIDs() (heroOwnIDs []int64, allDirty bool) {
	if len(c.ItemsChanged) == 0 && len(c.EntryChanged) == 0 {
		return []int64{}, false
	}
	return []int64{}, true
}

func (c *CollectionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {

}

func LoadCollectionModel(userId int64) (*CollectionModel, error) {
	row1, err := easyDB.GetPlayerEntitiesByWhere[CollectionEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	entities1 := make(map[int32]*CollectionEntity)
	cbcid := make(map[int32]*CollectionEntity)
	for _, v := range row1 {
		entities1[v.CollectionAttribution] = v
		cbcid[v.CollectId] = v
	}
	row2, err := easyDB.GetPlayerEntitiesByWhere[CollectionEntryEntity](map[string]interface{}{"user_id": userId})
	entities2 := make(map[int32]*CollectionEntryEntity)
	if err != nil {
		return nil, err
	}
	for _, entity := range row2 {
		entities2[entity.EntryId] = entity
	}
	return NewCollectionModel(userId, entities1, entities2, cbcid), nil
}

func (c *CollectionModel) creatCollectionEntity(entity *CollectionEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (c *CollectionModel) AddCollection(entity *CollectionEntity) error {
	c.CollectionEntity[entity.CollectionAttribution] = entity
	err := c.creatCollectionEntity(entity)
	if err != nil {
		return err
	}
	c.UpdateCollectionLevel(entity.CollectionAttribution, entity.CollectLevel)
	return nil
}

func (c *CollectionModel) creatCollectionEntryEntity(entity *CollectionEntryEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (c *CollectionModel) AddCollectionEntry(entity *CollectionEntryEntity) error {
	c.EntryEntity[entity.EntryId] = entity
	err := c.creatCollectionEntryEntity(entity)
	if err != nil {
		return err
	}
	c.UpdateEntryLevel(entity.EntryId, entity.EntryLevel)
	return nil
}

func (c *CollectionModel) UpdateCollectionLevel(collectionAttribution, collectLevel int32) {
	c.CollectionEntity[collectionAttribution].CollectLevel = collectLevel
	if c.ItemsChanged[collectionAttribution] == nil {
		c.ItemsChanged[collectionAttribution] = make(map[string]interface{})
	}
	c.ItemsChanged[collectionAttribution]["collect_level"] = collectLevel
}

func (c *CollectionModel) UpdateCollectionAttribution(collectionAttribution, collectId int32) {
	c.CollectionEntity[collectionAttribution].CollectId = collectId
	if c.ItemsChanged[collectionAttribution] == nil {
		c.ItemsChanged[collectionAttribution] = make(map[string]interface{})
	}
	c.ItemsChanged[collectionAttribution]["collect_id"] = collectId
}

func (c *CollectionModel) UpdateEntryLevel(entryId, entryLevel int32) {
	c.EntryEntity[entryId].EntryLevel = entryLevel
	if c.EntryChanged[entryId] == nil {
		c.EntryChanged[entryId] = make(map[string]interface{})
	}
	c.EntryChanged[entryId]["entry_level"] = entryLevel
}

func (c *CollectionModel) CollectionLevelUp(collectionAttribution int32, player *PlayerModel, useItem []*pb.ItemBasicInfo) (*CollectionEntity, error) {
	collectionEntity := c.CollectionEntity[collectionAttribution]
	if collectionEntity == nil {
		collectionEntity = &CollectionEntity{
			UserId:                c.userId,
			CollectLevel:          0,
			CollectionAttribution: collectionAttribution,
			CollectId:             0,
		}
	}
	collectionCfg := gameConfig.GetCollectionMainCfgByAtrAndLevel(collectionAttribution, collectionEntity.CollectLevel+1)
	if collectionCfg == nil {
		return nil, errors.New("collection level up is max")
	}
	if collectionCfg.Unlock != 0 {
		if !unlockService.CheckUnlock(collectionCfg.Unlock, player) {
			return nil, errors.New("collection level up is not unlock")
		}
	}
	qualityForNum := make(map[int32]int32) // 质量对应的数量
	itemInfos := make([]*gameConfig.ItemConfig, 0)
	// 检查升级条件
	for _, v := range useItem {
		itemCfg := gameConfig.GetItemCfg(v.ItemId)
		if itemCfg == nil {
			return nil, errors.New("item id is not exist")
		}
		itemInfos = append(itemInfos, &gameConfig.ItemConfig{ID: v.ItemId, Num: v.Count})
		if itemCfg.TargetId != 0 && v.ItemId != collectionCfg.Spid {
			needCheckEntity := c.CollectionEntity[itemCfg.TargetId]
			if needCheckEntity == nil {
				return nil, errors.New("item target entity is not exist")
			}
			if needCheckEntity.CollectLevel < 4 {
				return nil, errors.New("item target entity level is not enough")
			}
		}
		qualityForNum[itemCfg.Quality] += int32(v.Count)
	}
	flag, err := itemService.CheckItemsCount(player, itemInfos)
	if !flag || err != nil {
		return nil, errors.New("item count is not enough")
	}
	if collectionCfg.Upgrade1 != 0 {
		flag, err = itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: collectionCfg.Spid, Num: int64(collectionCfg.Upgrade1)})
		if !flag || err != nil {
			return nil, errors.New("item count is not enough")
		}
	}
	for id, v := range collectionCfg.Upgrade2 {
		if qualityForNum[v] != collectionCfg.Upgrade2Num[id] {
			return nil, errors.New("item count is not enough")
		}
	}
	err = itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: collectionCfg.Spid, Num: int64(collectionCfg.Upgrade1)}, enum.ITEM_CHANGE_REASON_COLLECTION_LEVEL_UP)
	if err != nil {
		return nil, err
	}
	err = itemService.RemoveItems(player, itemInfos, enum.ITEM_CHANGE_REASON_COLLECTION_LEVEL_UP)
	if err != nil {
		return nil, err
	}
	if c.CollectionEntity[collectionAttribution] == nil {
		collectionEntity.CollectLevel = 1
		collectionEntity.CollectId = collectionCfg.Id
		err = c.AddCollection(collectionEntity)
		if err != nil {
			return nil, errors.New("add collection failed")
		}
	} else {
		c.UpdateCollectionLevel(collectionAttribution, collectionEntity.CollectLevel+1)
		c.UpdateCollectionAttribution(collectionAttribution, collectionCfg.Id)
	}
	return collectionEntity, nil
}

func (c *CollectionModel) EntryLevelUp(entryId int32, player *PlayerModel) (*CollectionEntryEntity, error) {
	entryEntity := c.EntryEntity[entryId]
	if entryEntity == nil {
		entryEntity = &CollectionEntryEntity{
			EntryId:    entryId,
			UserId:     c.userId,
			EntryLevel: 0,
		}
	}
	entryCfg := gameConfig.GetEntryCfg(entryId)
	entryLevelCfg := gameConfig.GetEntryConsumeCfg(entryEntity.EntryLevel + 1)
	if entryCfg == nil {
		return nil, errors.New("entry cfg no exist")
	}
	if entryLevelCfg == nil {
		return nil, errors.New("entry level is max")
	}
	if entryLevelCfg.Item != nil {
		flag, err := itemService.CheckItemsCount(player, entryLevelCfg.Item)
		if !flag || err != nil {
			return nil, errors.New("item count is not enough")
		}
	}
	if c.EntryEntity[entryId] == nil {
		for _, v := range entryCfg.MainId {
			if c.CollectionEntity[v] == nil {
				return nil, errors.New("entry main id is not exist")
			}
		}
	} else {
		if entryEntity.EntryLevel+1 > entryCfg.Lvcap[len(entryCfg.Lvcap)-1][1] {
			return nil, errors.New("entry level is max")
		}
	}
	err := itemService.RemoveItems(player, entryLevelCfg.Item, enum.ITEM_CHANGE_REASON_ENTRY_LEVEL_UP)
	if err != nil {
		return nil, err
	}
	if c.EntryEntity[entryId] == nil {
		entryEntity.EntryLevel = 1
		err = c.AddCollectionEntry(entryEntity)
		if err != nil {
			return nil, errors.New("add entry failed")
		}
	} else {
		c.UpdateEntryLevel(entryId, entryEntity.EntryLevel+1)
	}
	return entryEntity, nil
}
