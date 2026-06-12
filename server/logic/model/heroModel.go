package model

import (
	"errors"
	"log"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"

	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AlbumRewardScoreEntity 图鉴奖励积分实体
type AlbumRewardScoreEntity struct {
	UserID        int64 `gorm:"column:user_id;primaryKey"` // 用户ID
	ClaimedReward int32 `gorm:"column:claimed_reward"`     // 已领取积分档位
	AllScore      int32 `gorm:"column:all_score"`          // 总积分
}

func (u *AlbumRewardScoreEntity) TableName() string {
	return "album_reward_score"
}

// AlbumRewardModel 图鉴奖励模型（单记录）
type AlbumRewardModel struct {
	UserId  int64
	Entity  *AlbumRewardScoreEntity
	Changed map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*AlbumRewardModel)(nil)

func NewAlbumRewardModel(userId int64, entity *AlbumRewardScoreEntity) *AlbumRewardModel {
	return &AlbumRewardModel{
		UserId:  userId,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func (a *AlbumRewardModel) UpdateClaimedReward(claimedReward int32) {
	a.Entity.ClaimedReward = claimedReward
	a.Changed["claimed_reward"] = claimedReward
}

func (a *AlbumRewardModel) UpdateAllScore(allScore int32) {
	a.Entity.AllScore = allScore
	a.Changed["all_score"] = allScore
}
func (a *AlbumRewardModel) SaveModelToDB() {
	if a.Changed == nil || len(a.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(a.Entity, a.Changed, a.UserId)
	a.Changed = make(map[string]interface{})
}

func (a *AlbumRewardModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	//nothing to do
}

// HeroAlbumEntity 英雄图鉴实体
type HeroAlbumEntity struct {
	UserID          int64 `gorm:"column:user_id;primaryKey"` // 用户ID
	HeroID          int64 `gorm:"column:hero_id;primaryKey"` // 英雄ID
	HistoryMaxStar  int32 `gorm:"column:history_max_star"`   // 历史最高星级
	ClaimedStar     int32 `gorm:"column:claimed_star"`       // 已领取星级档位
	HistoryMaxLevel int32 `gorm:"column:history_max_level"`  // 历史最高等级
}

func (u *HeroAlbumEntity) TableName() string {
	return "hero_album"
}

// HeroAlbumCollectionModel 英雄图鉴集合模型
type HeroAlbumCollectionModel struct {
	UserId   int64
	Entities map[int64]*HeroAlbumEntity       // heroID -> 英雄图鉴实体
	Changed  map[int64]map[string]interface{} // heroID -> 字段 -> 新值
}

var _ logicCommon.PlayerModelInterface = (*HeroAlbumCollectionModel)(nil)

func NewHeroAlbumCollectionModel(userId int64, entities map[int64]*HeroAlbumEntity) *HeroAlbumCollectionModel {
	return &HeroAlbumCollectionModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int64]map[string]interface{}),
	}
}

func (h *HeroAlbumCollectionModel) UpdateHistoryMaxStar(heroID int64, historyMaxStar int32) {
	h.Entities[heroID].HistoryMaxStar = historyMaxStar
	h.getChangedMap(heroID)["history_max_star"] = historyMaxStar
}

func (h *HeroAlbumCollectionModel) UpdateHistoryMaxLevel(heroID int64, historyMaxLevel int32) {
	h.Entities[heroID].HistoryMaxLevel = historyMaxLevel
	h.getChangedMap(heroID)["history_max_level"] = historyMaxLevel
}

func (h *HeroAlbumCollectionModel) UpdateClaimedStar(heroID int64, claimedStar int32) {
	h.Entities[heroID].ClaimedStar = claimedStar
	h.getChangedMap(heroID)["claimed_star"] = claimedStar
}

func (h *HeroAlbumCollectionModel) GetAlbum(heroID int64) *HeroAlbumEntity {
	return h.Entities[heroID]
}

func (h *HeroAlbumCollectionModel) AddAlbum(album *HeroAlbumEntity) {
	h.Entities[album.HeroID] = album
}

func (h *HeroAlbumCollectionModel) getChangedMap(heroID int64) map[string]interface{} {
	if h.Changed[heroID] == nil {
		h.Changed[heroID] = make(map[string]interface{})
	}
	return h.Changed[heroID]
}

func (h *HeroAlbumCollectionModel) SaveModelToDB() {
	if h.Changed == nil || len(h.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerBatchEntities(h.Entities, h.Changed, h.UserId)
	h.Changed = make(map[int64]map[string]interface{})
}

func (a *HeroAlbumCollectionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	//nothing to do
}

// HeroDetailsEntity 英雄详情实体
type HeroDetailsEntity struct {
	HeroOwnID           int64               `gorm:"column:hero_own_id;primaryKey"` // 英雄唯一ID
	UserID              int64               `gorm:"column:user_id"`                // 用户ID
	HeroID              int64               `gorm:"column:hero_id"`                // 英雄ID
	Level               int32               `gorm:"column:level"`                  // 等级
	StarLevel           int32               `gorm:"column:star_level"`             // 星级
	EvolutionPath       int32               `gorm:"column:evolution_path"`         // 转职方向
	EvolutionUpdateTime int64               `gorm:"column:evolution_update_time"`  // 转职更新时间
	BreakNum            int32               `gorm:"column:break_num"`              // 进阶次数
	EquipmentId         tool.JSONInt64Slice `gorm:"column:equipment_id;type:json"` // 装备ID 0-5表示装备部位
	IsDeleted           bool                `gorm:"column:is_deleted"`             // 是否删除
	Power               int64               `gorm:"column:power"`                  // 战力
	// 英雄属性缓存
	heroAttrMap map[int32]int64 `gorm:"-"` // 英雄属性缓存，key=attrId, value=attrValue
	isDirty     bool            `gorm:"-"` // 脏标记，true表示属性缓存需要重建
}

func (h *HeroDetailsEntity) TableName() string {
	return "hero_details"
}

// Top5HeroLevelItem Top5英雄等级项
type Top5HeroLevelItem struct {
	HeroOwnID int64
	Level     int32
}
type HeroDetailsTree struct {
	Root   bool                                  // 全局是否脏
	First  map[int32]bool                        // heroBaseClass ->isDirty
	Second map[int32]map[int32]bool              // heroBaseClass -> heroModelId -> isDirty
	Third  map[int32]map[int32]map[int32]bool    // heroBaseClass -> heroModelId -> heroOwnId%10 -> isDirty
	Fourth map[int32]map[int32]map[int32][]int64 // heroBaseClass -> heroModelId -> heroOwnId%10 -> heroOwnId -> *HeroDetailsEntity
}

// HeroDetailsCollectionModel 英雄详情集合模型
type HeroDetailsCollectionModel struct {
	UserId   int64
	Entities map[int64]*HeroDetailsEntity     // heroOwnID -> 英雄详情实体
	Changed  map[int64]map[string]interface{} // heroOwnID -> 字段 -> 新值
	player   *PlayerModel

	HeroChangeCache map[int64]*pb.HeroBagInfo

	PushAddHeroDetail []int64

	// 运行时缓存：key=heroID, value=该heroID的最高等级和星级
	HeroMaxLevelCache map[int64]int32 // heroID -> 最高等级
	HeroMaxStarCache  map[int64]int32 // heroID -> 最高星级

	// 反向索引：heroID -> set of heroOwnID，用于O(1)删除时查找同heroID的其他实例
	HeroIDToOwnIDs map[int64]map[int64]bool // heroID -> heroOwnID集合

	// Top5英雄等级缓存，按等级降序排列
	Top5HeroLevels []Top5HeroLevelItem
	top5DirtyFlag  bool // 脏标记，true时GetTop5HeroLevels会懒重建

	// 临时等级替换（非主线阵容计算属性时使用）
	overrideLevel int32

	// 属性缓存树
	heroAttrTree *HeroDetailsTree
}

var _ logicCommon.PlayerModelInterface = (*HeroDetailsCollectionModel)(nil)
var _ logicCommon.HeroAttrInterface = (*HeroDetailsCollectionModel)(nil)

func NewHeroDetailsCollectionModel(userId int64, entities map[int64]*HeroDetailsEntity, levelCache map[int64]int32, starCache map[int64]int32, heroIDToOwnIDs map[int64]map[int64]bool, player *PlayerModel) *HeroDetailsCollectionModel {
	model := &HeroDetailsCollectionModel{
		UserId:            userId,
		Entities:          entities,
		Changed:           make(map[int64]map[string]interface{}),
		player:            player,
		HeroChangeCache:   make(map[int64]*pb.HeroBagInfo),
		HeroMaxLevelCache: levelCache,
		HeroMaxStarCache:  starCache,
		HeroIDToOwnIDs:    heroIDToOwnIDs,
		Top5HeroLevels:    make([]Top5HeroLevelItem, 0, 5),
		top5DirtyFlag:     true, // 懒初始化，首次GetTop5HeroLevels时构建
		PushAddHeroDetail: make([]int64, 0),

		heroAttrTree: buildHeroAttrsTree(entities),
	}
	return model
}

func (h *HeroDetailsCollectionModel) AddHeroChangeCache(heroOwnId int64) {
	heroBagInfo := h.GetHeroInfoByOwnID(h.player, heroOwnId)
	h.HeroChangeCache[heroOwnId] = heroBagInfo
}

func (h *HeroDetailsCollectionModel) AddHeroForMemory(heroOwnId int64) {
	h.PushAddHeroDetail = append(h.PushAddHeroDetail, heroOwnId)
}

func (h *HeroDetailsCollectionModel) GetUintsId(heroOwnId int64) int32 {
	heroDetail := h.Entities[heroOwnId]
	heroStarEffectCfg := gameConfig.GetStarEffectCfg(int32(heroDetail.HeroID), heroDetail.StarLevel)
	if heroStarEffectCfg == nil {
		return 0
	}
	for id, v := range heroStarEffectCfg.ChangeClass {
		if v == heroDetail.EvolutionPath {
			return int32(id)
		}
	}
	return heroStarEffectCfg.UintsId[0]
}

func (h *HeroDetailsCollectionModel) UpdateLevel(heroOwnID int64, level int32) {
	hero := h.Entities[heroOwnID]
	hero.Level = level
	h.getChangedMap(heroOwnID)["level"] = level

	// 简单维护缓存：只更新更大的值
	if !hero.IsDeleted && level > h.HeroMaxLevelCache[hero.HeroID] {
		h.HeroMaxLevelCache[hero.HeroID] = level
	}

	// 增量维护Top5
	h.updateTop5HeroLevels(heroOwnID, level)
}

// DeleteHero 删除英雄并更新缓存
func (h *HeroDetailsCollectionModel) DeleteHero(heroOwnID int64) {
	hero, ok := h.Entities[heroOwnID]
	if !ok || hero == nil {
		return
	}

	heroID := hero.HeroID

	// 从反向索引中移除
	if h.HeroIDToOwnIDs[heroID] != nil {
		delete(h.HeroIDToOwnIDs[heroID], heroOwnID)
	}

	// 标记为已删除并同步到数据库
	//hero.IsDeleted = true
	//h.getChangedMap(heroOwnID)["is_deleted"] = true
	//h.UpdateIsDeleted(heroOwnID, true)
	changed := make(map[string]interface{})
	changed["is_deleted"] = true
	easyDB.UpdatePlayerEntity(h.Entities[heroOwnID], changed, h.UserId)
	// 从Entities中删除
	delete(h.Entities, heroOwnID)

	// 重新计算该heroID的缓存（只遍历同heroID的其他实例，数量通常很少）
	h.rebuildCacheForHeroID(heroID)

	if len(h.HeroIDToOwnIDs[heroID]) == 0 {
		delete(h.HeroIDToOwnIDs, heroID)
	}

	// 标记Top5脏，等查询时懒重建
	h.top5DirtyFlag = true

	h.deleteHeroFormAttrTree(heroOwnID)
}

// rebuildCacheForHeroID 重新计算指定heroID的缓存
func (h *HeroDetailsCollectionModel) rebuildCacheForHeroID(heroID int64) {
	maxLevel := int32(0)
	maxStar := int32(0)

	// 只遍历同heroID的实例，不是全部1200个英雄
	if ownIDs, ok := h.HeroIDToOwnIDs[heroID]; ok {
		for ownID := range ownIDs {
			if otherHero, exists := h.Entities[ownID]; exists && otherHero != nil && !otherHero.IsDeleted {
				if otherHero.Level > maxLevel {
					maxLevel = otherHero.Level
				}
				if otherHero.StarLevel > maxStar {
					maxStar = otherHero.StarLevel
				}
			}
		}
	}

	h.HeroMaxLevelCache[heroID] = maxLevel
	h.HeroMaxStarCache[heroID] = maxStar
}

func (h *HeroDetailsCollectionModel) UpdatePower(heroOwnID int64, power int64) {
	h.Entities[heroOwnID].Power = power
	h.getChangedMap(heroOwnID)["power"] = power
}

func (h *HeroDetailsCollectionModel) UpdateStarLevel(heroOwnID int64, starLevel int32) {
	hero := h.Entities[heroOwnID]
	hero.StarLevel = starLevel
	hero.isDirty = true
	h.getChangedMap(heroOwnID)["star_level"] = starLevel

	// 简单维护缓存：只更新更大的值
	if !hero.IsDeleted && starLevel > h.HeroMaxStarCache[hero.HeroID] {
		h.HeroMaxStarCache[hero.HeroID] = starLevel
	}
}

func (h *HeroDetailsCollectionModel) UpdateEvolutionPath(heroOwnID int64, evolutionPath int32) {
	hero := h.Entities[heroOwnID]
	hero.EvolutionPath = evolutionPath
	h.getChangedMap(heroOwnID)["evolution_path"] = evolutionPath
}

func (h *HeroDetailsCollectionModel) UpdateBreakNum(heroOwnID int64, breakNum int32) {
	hero := h.Entities[heroOwnID]
	hero.BreakNum = breakNum
	h.getChangedMap(heroOwnID)["break_num"] = breakNum
}

func (h *HeroDetailsCollectionModel) UpdateEvolutionUpdateTime(heroOwnID int64, evolutionUpdateTime int64) {
	h.Entities[heroOwnID].EvolutionUpdateTime = evolutionUpdateTime
	h.getChangedMap(heroOwnID)["evolution_update_time"] = evolutionUpdateTime
}

func (h *HeroDetailsCollectionModel) UpdateEquipmentId(heroOwnID int64, equipmentId int64, slot int32) {
	if slot < 1 || slot > 6 {
		log.Printf("UpdateEquipmentId: invalid slot %d, must be 1-6", slot)
		return
	}
	hero := h.Entities[heroOwnID]
	hero.EquipmentId[slot-1] = equipmentId
	h.getChangedMap(heroOwnID)["equipment_id"] = hero.EquipmentId
}

func (h *HeroDetailsCollectionModel) UpdateIsDeleted(heroOwnID int64, isDeleted bool) {
	h.Entities[heroOwnID].IsDeleted = isDeleted
	h.getChangedMap(heroOwnID)["is_deleted"] = isDeleted
}

func (h *HeroDetailsCollectionModel) UpdateIsDirty(heroOwnID int64, isDirty bool) {
	h.Entities[heroOwnID].isDirty = isDirty
}

func (h *HeroDetailsCollectionModel) GetHero(heroOwnID int64) *HeroDetailsEntity {
	return h.Entities[heroOwnID]
}

func (h *HeroDetailsCollectionModel) getChangedMap(heroOwnID int64) map[string]interface{} {
	if h.Changed[heroOwnID] == nil {
		h.Changed[heroOwnID] = make(map[string]interface{})
	}
	return h.Changed[heroOwnID]
}

func (h *HeroDetailsCollectionModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	var attr int64 = 0
	attr += gameConfig.GetHeroBaseAttr(int32(h.Entities[heroId].HeroID), attrId)

	detail := h.Entities[heroId]
	heroBase := gameConfig.GetHeroBaseCfg(int32(detail.HeroID))
	potential := heroBase.HeroPotential
	class := heroBase.HeroClass

	// 非主线阵容时使用替换等级
	level := detail.Level
	if h.overrideLevel > 0 {
		level = h.overrideLevel
	}

	attr += gameConfig.GetSecondAttr(potential, class, level, detail.BreakNum, detail.StarLevel, attrId)

	return attr
}

// SetOverrideLevel 设置临时等级替换（非主线阵容计算属性前调用）
func (h *HeroDetailsCollectionModel) SetOverrideLevel(level int32) {
	h.overrideLevel = level
}

// ClearOverrideLevel 清除临时等级替换（计算完成后调用）
func (h *HeroDetailsCollectionModel) ClearOverrideLevel() {
	h.overrideLevel = 0
}

func (a *HeroDetailsCollectionModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	heroDetail := a.Entities[heroId]
	cfg := gameConfig.GetStarEffectCfg(int32(heroDetail.HeroID), heroDetail.StarLevel)
	if cfg != nil {
		var buffCfg *gameConfig.AttrBuffCfg
		if cfg.SkillType1 == 2 {
			buffCfg = gameConfig.GetAttrBuffCfg(cfg.PassiveSkill1)
		}
		if cfg.SkillType2 == 2 {
			buffCfg = gameConfig.GetAttrBuffCfg(cfg.PassiveSkill2)
		}
		if buffCfg != nil {
			for i, v := range buffCfg.Attr {
				if v == attrId {
					return int64(buffCfg.AttrNum[i])
				}
			}
		}
	}
	return 0
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表
// 英雄自身变化只影响自己，allDirty=false
func (h *HeroDetailsCollectionModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(h.Changed) == 0 {
		return []int64{}, false
	}
	res := make([]int64, 0, len(h.Changed))
	for ownID := range h.Changed {
		res = append(res, ownID)
	}
	return res, false
}

func (h *HeroDetailsCollectionModel) SaveModelToDB() {
	for key, v := range h.Changed {
		if entity, ok := h.Entities[key]; ok {
			easyDB.UpdatePlayerEntity(entity, v, h.UserId)
		}
	}
	h.Changed = make(map[int64]map[string]interface{})
}

func (a *HeroDetailsCollectionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	pushResp := make([]*pb.HeroBagInfo, 0)
	if a.PushAddHeroDetail == nil || len(a.PushAddHeroDetail) == 0 {
		return
	}
	for _, v := range a.PushAddHeroDetail {
		heroDetail := a.GetHeroInfoByOwnID(a.player, v)
		pushResp = append(pushResp, heroDetail)
	}
	messageSender.SendMessage(a.player, pb.MESSAGE_ID_PUSH_ADD_HERO_DETAIL, &pb.PushAddHeroDetail{
		HeroInfo: pushResp,
	})
	a.PushAddHeroDetail = make([]int64, 0)
}

func (h *HeroDetailsEntity) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("hero_own_id", h.HeroOwnID)
	enc.AddInt64("user_id", h.UserID)
	enc.AddInt64("hero_id", h.HeroID)
	enc.AddInt32("level", h.Level)
	enc.AddInt32("star_level", h.StarLevel)
	enc.AddBool("is_deleted", h.IsDeleted)
	enc.AddInt32("evolution_path", h.EvolutionPath)
	enc.AddInt32("break_num", h.BreakNum)
	enc.AddInt64("evolution_update_time", h.EvolutionUpdateTime)
	return nil
}

func (h *HeroDetailsEntity) GetUnits() int32 {
	//TODO:
	return 0
}

func buildHeroAttrsTree(entities map[int64]*HeroDetailsEntity) *HeroDetailsTree {
	tree := &HeroDetailsTree{
		Root:   true,
		First:  make(map[int32]bool),
		Second: make(map[int32]map[int32]bool),
		Third:  make(map[int32]map[int32]map[int32]bool),
		Fourth: make(map[int32]map[int32]map[int32][]int64),
	}
	for _, entity := range entities {
		heroBase := gameConfig.GetHeroBaseCfg(int32(entity.HeroID))
		if heroBase == nil {
			continue
		}
		heroClass := heroBase.HeroClass
		heroModelId := heroBase.HeroId
		// 初始化 Second 子 map
		tree.First[heroClass] = true
		if tree.Second[heroClass] == nil {
			tree.Second[heroClass] = make(map[int32]bool)
		}
		tree.Second[heroClass][heroModelId] = true
		// 初始化 Third 子 map
		if tree.Third[heroClass] == nil {
			tree.Third[heroClass] = make(map[int32]map[int32]bool)
		}
		if tree.Third[heroClass][heroModelId] == nil {
			tree.Third[heroClass][heroModelId] = make(map[int32]bool)
		}
		tree.Third[heroClass][heroModelId][int32(entity.HeroOwnID%10)] = true
		// 初始化 Fourth 子 map
		if tree.Fourth[heroClass] == nil {
			tree.Fourth[heroClass] = make(map[int32]map[int32][]int64)
		}
		if tree.Fourth[heroClass][heroModelId] == nil {
			tree.Fourth[heroClass][heroModelId] = make(map[int32][]int64)
		}
		tail := int32(entity.HeroOwnID % 10)
		tree.Fourth[heroClass][heroModelId][tail] = append(tree.Fourth[heroClass][heroModelId][tail], entity.HeroOwnID)
	}
	return tree
}

func (h *HeroDetailsCollectionModel) FindHeroAttrByHeroId(heroOwnId int64) map[int32]int64 {
	heroAttrTree := h.heroAttrTree
	heroDetail := h.GetHero(heroOwnId)
	if heroDetail == nil {
		return nil
	}
	heroBase := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
	if heroBase == nil {
		return nil
	}
	heroClass := heroBase.HeroClass
	heroModelId := heroBase.HeroId

	// 从 Root 向下传播脏标记
	if heroAttrTree.Root {
		// 全局刷新：把所有 First 设为 true
		for key := range heroAttrTree.Fourth {
			heroAttrTree.First[key] = true
		}
		heroAttrTree.Root = false
	}
	if heroAttrTree.First[heroClass] {
		// 该职业需要刷新：把该职业下所有 Second 设为 true
		for key, _ := range heroAttrTree.Second[heroClass] {
			heroAttrTree.Second[heroClass][key] = true
		}
		heroAttrTree.First[heroClass] = false
	}
	if heroAttrTree.Second[heroClass][heroModelId] {
		// 该英雄模型需要刷新：把该模型下所有 Third 设为 true
		for key, _ := range heroAttrTree.Third[heroClass][heroModelId] {
			heroAttrTree.Third[heroClass][heroModelId][key] = true
		}
		heroAttrTree.Second[heroClass][heroModelId] = false
	}
	if heroAttrTree.Third[heroClass][heroModelId][int32(heroDetail.HeroOwnID%10)] {
		// 该分组需要刷新：标记该分组下所有英雄为脏
		heroAttrTree.Third[heroClass][heroModelId][int32(heroDetail.HeroOwnID%10)] = false
		for _, ownId := range heroAttrTree.Fourth[heroClass][heroModelId][int32(heroDetail.HeroOwnID%10)] {
			if hero := h.GetHero(ownId); hero != nil {
				hero.isDirty = true
			}
		}
	}

	// 重建缓存
	if heroDetail.isDirty {
		heroDetail.isDirty = false
		heroDetail.heroAttrMap = h.player.getHeroAttr(heroOwnId)
	}
	return heroDetail.heroAttrMap
}

func (h *HeroDetailsCollectionModel) refreshHeroAttrTree() {
	h.heroAttrTree.Root = true
}

func (h *HeroDetailsCollectionModel) addHeroFormAttrTree(heroOwnId int64) {
	heroDetail := h.GetHero(heroOwnId)
	if heroDetail == nil {
		return
	}
	heroBase := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
	if heroBase == nil {
		return
	}
	heroClass := heroBase.HeroClass
	heroModelId := heroBase.HeroId

	// 初始化 Second 子 map
	if h.heroAttrTree.Second[heroClass] == nil {
		h.heroAttrTree.Second[heroClass] = make(map[int32]bool)
	}
	if _, ok := h.heroAttrTree.Second[heroClass][heroModelId]; !ok {
		h.heroAttrTree.Second[heroClass][heroModelId] = false
	}
	// 初始化 Third 子 map
	if h.heroAttrTree.Third[heroClass] == nil {
		h.heroAttrTree.Third[heroClass] = make(map[int32]map[int32]bool)
	}
	if h.heroAttrTree.Third[heroClass][heroModelId] == nil {
		h.heroAttrTree.Third[heroClass][heroModelId] = make(map[int32]bool)
	}
	if _, ok := h.heroAttrTree.Third[heroClass][heroModelId][int32(heroDetail.HeroOwnID%10)]; !ok {
		h.heroAttrTree.Third[heroClass][heroModelId][int32(heroDetail.HeroOwnID%10)] = false
	}
	// 初始化 Fourth 子 map
	if h.heroAttrTree.Fourth[heroClass] == nil {
		h.heroAttrTree.Fourth[heroClass] = make(map[int32]map[int32][]int64)
	}
	if h.heroAttrTree.Fourth[heroClass][heroModelId] == nil {
		h.heroAttrTree.Fourth[heroClass][heroModelId] = make(map[int32][]int64)
	}
	tail := int32(heroDetail.HeroOwnID % 10)
	h.heroAttrTree.Fourth[heroClass][heroModelId][tail] = append(h.heroAttrTree.Fourth[heroClass][heroModelId][tail], heroOwnId)
}

func (h *HeroDetailsCollectionModel) deleteHeroFormAttrTree(heroId int64) {
	heroDetail := h.GetHero(heroId)
	if heroDetail == nil {
		return
	}
	heroBase := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
	if heroBase == nil {
		return
	}
	heroClass := heroBase.HeroClass
	heroModelId := heroBase.HeroId
	tail := int32(heroDetail.HeroOwnID % 10)

	// 清理 Fourth 层：从切片中删除 heroId
	if h.heroAttrTree.Fourth[heroClass] == nil || h.heroAttrTree.Fourth[heroClass][heroModelId] == nil {
		return
	}
	if ids := h.heroAttrTree.Fourth[heroClass][heroModelId][tail]; len(ids) > 0 {
		newIds := make([]int64, 0, len(ids)-1)
		for _, id := range ids {
			if id != heroId {
				newIds = append(newIds, id)
			}
		}
		if len(newIds) == 0 {
			delete(h.heroAttrTree.Fourth[heroClass][heroModelId], tail)
			delete(h.heroAttrTree.Third[heroClass][heroModelId], tail)
		} else {
			h.heroAttrTree.Fourth[heroClass][heroModelId][tail] = newIds
		}
	}

	// Fourth[heroClass][heroModelId] 为空时，清理该 heroModelId
	if len(h.heroAttrTree.Fourth[heroClass][heroModelId]) == 0 {
		delete(h.heroAttrTree.Fourth[heroClass], heroModelId)
		delete(h.heroAttrTree.Third[heroClass], heroModelId)
		delete(h.heroAttrTree.Second[heroClass], heroModelId)
	}

	// Fourth[heroClass] 为空时，清理该 heroClass
	if len(h.heroAttrTree.Fourth[heroClass]) == 0 {
		delete(h.heroAttrTree.Fourth, heroClass)
		delete(h.heroAttrTree.Third, heroClass)
		delete(h.heroAttrTree.Second, heroClass)
		delete(h.heroAttrTree.First, heroClass)
	}
}

// HeroFormationEntity 英雄阵型实体
type HeroFormationEntity struct {
	UserID        int64               `gorm:"column:user_id;primaryKey"`         // 用户ID
	FormationID   int32               `gorm:"column:formation_id;primaryKey"`    // 阵型ID
	HeroOwnIDList tool.JSONInt64Slice `gorm:"column:hero_own_id_list;type:json"` // 阵型中的英雄唯一ID列表
	FormationType int32               `gorm:"column:formation_type;primaryKey"`  // 阵型类型
	IsActive      bool                `gorm:"column:is_active"`                  // 是否激活
}

func (u *HeroFormationEntity) TableName() string {
	return "hero_formation"
}

// HeroFormationCollectionModel 英雄阵型集合模型（Changed 使用 formationID int32 作为 key）
type HeroFormationCollectionModel struct {
	UserId   int64
	Entities map[int32]map[int32]*HeroFormationEntity // formationType -> formationId -> 英雄阵型实体
	Changed  map[int32]map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*HeroFormationCollectionModel)(nil)

func NewHeroFormationCollectionModel(userId int64, entities map[int32]map[int32]*HeroFormationEntity) *HeroFormationCollectionModel {
	return &HeroFormationCollectionModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int32]map[int32]map[string]interface{}),
	}
}

func (f *HeroFormationCollectionModel) AddHeroFormation(formationType int32, formationId int32, entity *HeroFormationEntity) error {
	if f.Entities[formationType] == nil {
		f.Entities[formationType] = make(map[int32]*HeroFormationEntity)
	}
	f.Entities[formationType][formationId] = entity
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return err
	}
	return nil
}

func (f *HeroFormationCollectionModel) GetHeroFormation(formationType int32, formationId int32) *HeroFormationEntity {
	if f.Entities[formationType] == nil {
		return nil
	}
	return f.Entities[formationType][formationId]
}

func (f *HeroFormationCollectionModel) UpdateHeroOwnIDListByTypeAndId(formationType int32, formationId int32, list []int64) {
	if f.Entities[formationType] == nil {
		f.Entities[formationType] = make(map[int32]*HeroFormationEntity)
	}
	if f.Entities[formationType][formationId] == nil {
		f.Entities[formationType][formationId] = &HeroFormationEntity{
			FormationID:   formationId,
			FormationType: formationType,
		}
	}
	f.Entities[formationType][formationId].HeroOwnIDList = tool.JSONInt64Slice(list)
	f.getChangedMap(formationType, formationId)[formationType][formationId]["hero_own_id_list"] = tool.JSONInt64Slice(list)
}

func (f *HeroFormationCollectionModel) UpdateIsActiveByTypeAndId(formationType int32, formationId int32, isActive bool) {
	if f.Entities[formationType] == nil {
		f.Entities[formationType] = make(map[int32]*HeroFormationEntity)
	}
	if f.Entities[formationType][formationId] == nil {
		f.Entities[formationType][formationId] = &HeroFormationEntity{
			FormationID:   formationId,
			FormationType: formationType,
		}
	}
	f.Entities[formationType][formationId].IsActive = isActive
	f.getChangedMap(formationType, formationId)[formationType][formationId]["is_active"] = isActive
}
func (f *HeroFormationCollectionModel) GetHeroAttr(heroInfo *HeroDetailsEntity, attrId int32, classSynergy []int32) int64 {
	//for _, v := range classSynergy {
	//	heroClassCfg := gameConfig.GetHeroClassCfg(heroInfo.EvolutionPath)
	//	if heroClassCfg == nil {
	//		continue
	//	}
	//	for _, value := range heroClassCfg.ClassSynergy {
	//		if v == value {
	//			cfg := gameConfig.GetAttrBuffCfg(v)
	//			if cfg != nil {
	//				for i, value := range cfg.Attr {
	//					if value == attrId {
	//						return int64(cfg.AttrNum[i])
	//					}
	//				}
	//			}
	//		}
	//	}
	//}
	return 0
}

func (f *HeroFormationCollectionModel) getChangedMap(formationType, formationId int32) map[int32]map[int32]map[string]interface{} {
	if f.Changed[formationType] == nil {
		f.Changed[formationType] = make(map[int32]map[string]interface{})
	}
	if f.Changed[formationType][formationId] == nil {
		f.Changed[formationType][formationId] = make(map[string]interface{})
	}
	return f.Changed
}

func (f *HeroFormationCollectionModel) SaveModelToDB() {
	if f.Changed == nil || len(f.Changed) == 0 {
		return
	}
	// 批量更新
	for formationType, formationMap := range f.Changed {
		for formationId, changes := range formationMap {
			easyDB.UpdatePlayerEntity(f.Entities[formationType][formationId], changes, f.UserId)
		}
	}
	f.Changed = make(map[int32]map[int32]map[string]interface{})
}

func (a *HeroFormationCollectionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	//nothing to do
}

func (f *HeroFormationCollectionModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(f.Changed) == 0 {
		return []int64{}, false
	}
	heroOwnIDs := make(map[int64]bool)
	if _, formations := f.Changed[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]; formations {
		for _, formation := range f.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
			if formation.IsActive == true {
				for v := range formation.HeroOwnIDList {
					heroOwnIDs[formation.HeroOwnIDList[v]] = true
				}
				break
			}
		}
	}
	if len(heroOwnIDs) == 0 {
		// 没有特定英雄变化（只有升级/获取），全局脏
		return []int64{}, false
	}
	res := make([]int64, 0, len(heroOwnIDs))
	for ownID := range heroOwnIDs {
		res = append(res, ownID)
	}
	return res, false
}

func LoadHeroBags(userId int64, player *PlayerModel, isLogin bool) (*HeroDetailsCollectionModel, error) {
	// 查询原始数据（maps）
	detailMap := make(map[int64]*HeroDetailsEntity)
	levelCache := make(map[int64]int32)
	starCache := make(map[int64]int32)
	heroIDToOwnIDs := make(map[int64]map[int64]bool) // 反向索引

	rows, err := easyDB.GetPlayerEntitiesByWhere[HeroDetailsEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	for i, d := range rows {
		if d == nil {
			continue
		}
		if d.IsDeleted {
			logger.InfoWithZapFields("LoadHeroBags found deleted hero", zap.Object("heroDetail", d))
			rows[i] = nil
			continue
		}
		detailMap[d.HeroOwnID] = d

		// 初始化缓存
		d.heroAttrMap = make(map[int32]int64)
		d.isDirty = true

		// 构建反向索引
		if heroIDToOwnIDs[d.HeroID] == nil {
			heroIDToOwnIDs[d.HeroID] = make(map[int64]bool)
		}
		heroIDToOwnIDs[d.HeroID][d.HeroOwnID] = true

		// 在load遍历时直接构建缓存
		if d.Level > levelCache[d.HeroID] {
			levelCache[d.HeroID] = d.Level
		}
		if d.StarLevel > starCache[d.HeroID] {
			starCache[d.HeroID] = d.StarLevel
		}
	}
	if isLogin {
		err = DeleteDeletedHeroes(userId)
		if err != nil {
			return nil, err
		}
	}
	return NewHeroDetailsCollectionModel(userId, detailMap, levelCache, starCache, heroIDToOwnIDs, player), nil
}

func CreateHeroModel(userId int64, player *PlayerModel) (*HeroDetailsCollectionModel, error) {
	detailMap := make(map[int64]*HeroDetailsEntity)
	levelCache := make(map[int64]int32)
	starCache := make(map[int64]int32)
	heroIDToOwnIDs := make(map[int64]map[int64]bool)
	return NewHeroDetailsCollectionModel(userId, detailMap, levelCache, starCache, heroIDToOwnIDs, player), nil
}

func CreateAlbumRewardModel(userId int64) (*AlbumRewardModel, error) {
	AlbumRewardEntity := &AlbumRewardScoreEntity{
		UserID:        userId,
		ClaimedReward: 0,
		AllScore:      0,
	}
	err := easyDB.CreatePlayerEntity(AlbumRewardEntity)
	if err != nil {
		return nil, err
	}
	return NewAlbumRewardModel(userId, AlbumRewardEntity), nil
}

// 定时删除玩家已删除的英雄数据
func DeleteDeletedHeroes(userId int64) error {

	err := easyDB.DeletePlayerEntityByWhere[HeroDetailsEntity](map[string]interface{}{"user_id": userId, "is_deleted": true}, userId)
	if err != nil {
		return err
	}
	return nil
}

func (h *HeroDetailsCollectionModel) AddHero(player *PlayerModel, heroId int64, heroOwnId int64) (bool, error) {
	if player == nil {
		return false, errors.New("player not found")
	}

	// 插入英雄数据
	heroDetail := &HeroDetailsEntity{
		HeroOwnID:           heroOwnId,
		UserID:              player.GetUserId(),
		HeroID:              heroId,
		Level:               1,
		StarLevel:           gameConfig.GetHeroBaseCfg(int32(heroId)).HeroStar,
		EvolutionPath:       gameConfig.GetHeroBaseCfg(int32(heroId)).HeroClass,
		EvolutionUpdateTime: 0,
		BreakNum:            0,
		EquipmentId:         tool.JSONInt64Slice{0, 0, 0, 0, 0, 0},
		IsDeleted:           false,
		isDirty:             true,
	}
	err := easyDB.CreatePlayerEntity[HeroDetailsEntity](heroDetail)
	player.HeroDetailsModel.Entities[heroOwnId] = heroDetail
	if err != nil {
		return false, err
	}

	// 维护反向索引
	if h.HeroIDToOwnIDs[heroId] == nil {
		h.HeroIDToOwnIDs[heroId] = make(map[int64]bool)
	}
	h.HeroIDToOwnIDs[heroId][heroOwnId] = true

	// 更新缓存（新英雄等级/星级通常不会比现有高，但保险起见检查）
	if heroDetail.Level > h.HeroMaxLevelCache[heroId] {
		h.HeroMaxLevelCache[heroId] = heroDetail.Level
	}
	if heroDetail.StarLevel > h.HeroMaxStarCache[heroId] {
		h.HeroMaxStarCache[heroId] = heroDetail.StarLevel
	}

	// 插入图鉴数据
	if player.HeroAlbumModel.Entities[heroId] == nil {
		heroAlbum := &HeroAlbumEntity{
			UserID:          player.GetUserId(),
			HeroID:          heroId,
			HistoryMaxStar:  gameConfig.GetHeroBaseCfg(int32(heroId)).HeroStar,
			ClaimedStar:     gameConfig.GetHeroBaseCfg(int32(heroId)).HeroStar - 1,
			HistoryMaxLevel: 1,
		}
		err := easyDB.CreatePlayerEntity[HeroAlbumEntity](heroAlbum)
		if err != nil {
			return false, err
		}
		player.HeroAlbumModel.Entities[heroId] = heroAlbum
		eventServer.SubmitAddHeroAlbumEvent(player.GetUserId(), int32(heroId))
	}

	// 维护属性缓存树
	h.addHeroFormAttrTree(heroOwnId)

	return true, nil
}

func GetHeroSkills(heroDetail *HeroDetailsEntity, equipmentDetail *EquipmentCollectionModel) *pb.SkillsInfo {
	skills := &pb.SkillsInfo{
		SkillList: make([]int32, 0),
	}
	if heroDetail == nil {
		return skills
	}
	cfg := gameConfig.GetStarEffectCfg(int32(heroDetail.HeroID), heroDetail.StarLevel)
	if cfg != nil {
		skills.BasicSkill = cfg.BasicSkill
		if cfg.ActiveSkill != 0 {
			skills.SkillList = append(skills.SkillList, cfg.ActiveSkill)
		}
		if cfg.SkillType1 != 2 && cfg.PassiveSkill1 != 0 {
			skills.SkillList = append(skills.SkillList, cfg.PassiveSkill1)
		}
		if cfg.SkillType2 != 2 && cfg.PassiveSkill2 != 0 {
			skills.SkillList = append(skills.SkillList, cfg.PassiveSkill2)
		}
	}
	for i := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID)).HeroStar; i <= heroDetail.StarLevel; i++ {
		starCfg := gameConfig.GetStarEffectCfg(int32(heroDetail.HeroID), i)
		if starCfg == nil || starCfg.ChangeClass == nil {
			continue
		}
		flag := false
		for id, v := range starCfg.ChangeClass {
			if v == heroDetail.EvolutionPath {
				if starCfg.ClassSkill[id] != 0 {
					flag = true
					skills.SkillList = append(skills.SkillList, starCfg.ClassSkill[id])
					break
				}
			}
		}
		if flag {
			break
		}
	}
	if equipmentDetail != nil {
		for _, v := range heroDetail.EquipmentId {
			if v == 0 {
				continue
			}
			if equipmentDetail.Entities[v] == nil {
				continue
			}
			equipmentId := equipmentDetail.Entities[v].EquipmentID
			if gameConfig.GetEquipmentBaseCfgByEquipmentID(equipmentId).SkillID != 0 {
				skills.SkillList = append(skills.SkillList, gameConfig.GetEquipmentBaseCfgByEquipmentID(equipmentId).SkillID)
			}
		}
	}
	return skills
}

func GetHeroPetBattleInfo(heroOwnID int64, petModel *PetModel) *pb.PetBattleInfo {
	if heroOwnID == 0 || petModel == nil {
		return nil
	}
	pet := petModel.GetEquippedPetByHero(heroOwnID)
	if pet == nil {
		return nil
	}
	info := &pb.PetBattleInfo{
		PetOwnId:  pet.PetOwnID,
		PetId:     pet.PetID,
		Level:     pet.Level,
		Star:      pet.Star,
		SkillList: make([]int32, 0),
	}
	if starCfg := gameConfig.GetPetStarCfgByPetIdStar(pet.PetID, pet.Star); starCfg != nil && starCfg.ActiveSkill != 0 {
		info.SkillList = append(info.SkillList, starCfg.ActiveSkill)
	}
	if baseCfg := gameConfig.GetPetBaseCfg(pet.PetID); baseCfg != nil && baseCfg.UniqueSkill != 0 {
		info.SkillList = append(info.SkillList, baseCfg.UniqueSkill)
	}
	return info
}

func LoadHeroFormations(id int64) map[int32]map[int32]*HeroFormationEntity {
	res := make(map[int32]map[int32]*HeroFormationEntity)
	if id == 0 {
		return res
	}

	rows, err := easyDB.GetPlayerEntitiesByWhere[HeroFormationEntity](map[string]interface{}{"user_id": id})
	if err != nil {
		platformLogger.ErrorWithUser(" QueryHeroFormations is fail ", nil, err)
		return res
	}
	if rows == nil {
		return res
	}

	for _, r := range rows {
		if r == nil {
			continue
		}
		formId := r.FormationID
		formType := r.FormationType
		if _, ok := res[formType]; !ok {
			res[formType] = make(map[int32]*HeroFormationEntity)
		}
		res[formType][formId] = r
	}

	return res
}

// 查询图鉴积分
func LoadAlbumRewardScore(id int64) *AlbumRewardScoreEntity {
	if id == 0 {
		return nil
	}

	ent, err := easyDB.GetPlayerEntityByWhere[AlbumRewardScoreEntity](map[string]interface{}{"user_id": id})
	if err != nil {
		platformLogger.ErrorWithUser("QueryAlbumRewardScore is fail ", nil, err)
		return nil
	}
	return ent
}

func LoadAlbum(uid int64) map[int64]*HeroAlbumEntity {
	res := make(map[int64]*HeroAlbumEntity)
	if uid == 0 {
		return res
	}

	rows, err := easyDB.GetPlayerEntitiesByWhere[HeroAlbumEntity](map[string]interface{}{"user_id": uid})
	if err != nil {
		platformLogger.ErrorWithUser(" QueryAlbum is fail ", nil, err)
		return res
	}
	if rows == nil {
		return res
	}

	for _, r := range rows {
		if r == nil {
			continue
		}
		key := r.HeroID
		if key == 0 {
			continue
		}
		res[key] = r
	}

	return res
}

func (h *HeroDetailsCollectionModel) GetHeroInfoByOwnID(player *PlayerModel, heroOwnID int64) *pb.HeroBagInfo {

	if player == nil || heroOwnID == 0 {
		return nil
	}
	detail, ok := h.Entities[heroOwnID]
	if !ok || detail == nil {
		return nil
	}
	var isCultivated bool = false
	if detail.Level > 1 || detail.StarLevel > gameConfig.GetHeroBaseCfg(int32(detail.HeroID)).HeroStar {
		isCultivated = true
	}
	if !isCultivated {
		for _, v := range detail.EquipmentId {
			if v != 0 {
				isCultivated = true
				break
			}
		}
	}
	heroBagInfo := &pb.HeroBagInfo{
		HeroOwnId:     detail.HeroOwnID,
		HeroId:        detail.HeroID,
		Level:         detail.Level,
		Star:          detail.StarLevel,
		EvolutionPath: detail.EvolutionPath,
		EvolutionTime: detail.EvolutionUpdateTime,
		Attributes:    player.GetHeroAttr(detail.HeroOwnID),
		BreakNum:      detail.BreakNum,
		IsCultivated:  isCultivated,
	}
	h.GetHeroSkillInfo(heroBagInfo)
	if detail.Power != heroBagInfo.Attributes[enum.AttributeBasicCombatPower] {
		player.HeroDetailsModel.UpdatePower(heroOwnID, heroBagInfo.Attributes[enum.AttributeBasicCombatPower])
	}
	return heroBagInfo
}

func (h *HeroDetailsCollectionModel) GetHeroSkillInfo(detail *pb.HeroBagInfo) {
	cfg := gameConfig.GetStarEffectCfg(int32(detail.HeroId), detail.Star)
	if cfg != nil {
		detail.BasicSkill = cfg.BasicSkill
		detail.ActiveSkill = cfg.ActiveSkill
		if cfg.SkillType1 == 1 && cfg.PassiveSkill1 != 0 {
			detail.PassiveSkill1 = cfg.PassiveSkill1
		}
		if cfg.SkillType2 == 1 && cfg.PassiveSkill2 != 0 {
			detail.PassiveSkill2 = cfg.PassiveSkill2
		}
	} else {
		return
	}
	for i := gameConfig.GetHeroBaseCfg(int32(detail.HeroId)).HeroStar; i <= detail.Star; i++ {
		starCfg := gameConfig.GetStarEffectCfg(int32(detail.HeroId), i)
		if starCfg == nil || starCfg.ChangeClass == nil {
			continue
		}
		for id, v := range starCfg.ChangeClass {
			if v == detail.EvolutionPath {
				detail.ClassSkill = starCfg.ClassSkill[id]
				break
			}
		}
	}
}

// GetTop5HeroLevels 获取Top5英雄等级
// 正常情况O(1)；脏标记触发时O(n)重建一次，后续仍O(1)
func (h *HeroDetailsCollectionModel) GetTop5HeroLevels() []int32 {
	if h.top5DirtyFlag {
		h.rebuildTop5HeroLevels()
		h.top5DirtyFlag = false
	}
	resp := make([]int32, 0)
	for _, v := range h.Top5HeroLevels {
		resp = append(resp, v.Level)
	}
	return resp
}

// updateTop5HeroLevels 增量更新Top5英雄等级
func (h *HeroDetailsCollectionModel) updateTop5HeroLevels(heroOwnID int64, newLevel int32) {
	// 查找是否已在Top5中
	foundIdx := -1
	for i, item := range h.Top5HeroLevels {
		if item.HeroOwnID == heroOwnID {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		// 已在Top5中，更新等级
		h.Top5HeroLevels[foundIdx].Level = newLevel
	} else {
		// 不在Top5中
		if len(h.Top5HeroLevels) < 5 {
			// Top5未满，直接添加
			h.Top5HeroLevels = append(h.Top5HeroLevels, Top5HeroLevelItem{HeroOwnID: heroOwnID, Level: newLevel})
		} else {
			// Top5已满，检查是否比最小的大
			minIdx := len(h.Top5HeroLevels) - 1
			if newLevel <= h.Top5HeroLevels[minIdx].Level {
				return // 不够资格进入Top5
			}
			// 替换最小的
			h.Top5HeroLevels[minIdx] = Top5HeroLevelItem{HeroOwnID: heroOwnID, Level: newLevel}
		}
	}

	// 按等级降序排序（冒泡，最多5个元素）
	h.sortTop5Desc()
}

// rebuildTop5HeroLevels 重建Top5（初始化或补充时使用）
func (h *HeroDetailsCollectionModel) rebuildTop5HeroLevels() {
	// 收集所有有效英雄
	items := make([]Top5HeroLevelItem, 0, len(h.Entities))
	for ownID, entity := range h.Entities {
		if entity != nil && !entity.IsDeleted {
			items = append(items, Top5HeroLevelItem{HeroOwnID: ownID, Level: entity.Level})
		}
	}

	// 按等级降序排序
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Level > items[i].Level {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// 取Top5
	if len(items) > 5 {
		items = items[:5]
	}
	h.Top5HeroLevels = items
}

// sortTop5Desc 按等级降序排序Top5
func (h *HeroDetailsCollectionModel) sortTop5Desc() {
	// 冒泡排序，最多5个元素，简单高效
	for i := 0; i < len(h.Top5HeroLevels); i++ {
		for j := i + 1; j < len(h.Top5HeroLevels); j++ {
			if h.Top5HeroLevels[j].Level > h.Top5HeroLevels[i].Level {
				h.Top5HeroLevels[i], h.Top5HeroLevels[j] = h.Top5HeroLevels[j], h.Top5HeroLevels[i]
			}
		}
	}
}
