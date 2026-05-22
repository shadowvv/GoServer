package model

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type PetAffinityBookEntity struct {
	UserID  int64 `gorm:"column:user_id;primaryKey"`
	PetID   int32 `gorm:"column:pet_id;primaryKey"`
	MaxStar int32 `gorm:"column:max_star"`
}

func (e *PetAffinityBookEntity) TableName() string { return "pet_affinity_book" }

func LoadPetAffinityBookEntities(userId int64) (map[int32]*PetAffinityBookEntity, error) {
	rows, err := easyDB.GetPlayerEntitiesByWhere[PetAffinityBookEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	m := make(map[int32]*PetAffinityBookEntity)
	for _, r := range rows {
		if r == nil {
			continue
		}
		m[r.PetID] = r
	}
	return m, nil
}

func EnsurePetAffinityBookEntity(userId int64, petID int32) *PetAffinityBookEntity {
	ent := &PetAffinityBookEntity{
		UserID:  userId,
		PetID:   petID,
		MaxStar: 0,
	}
	_ = easyDB.CreatePlayerEntity(ent)
	return ent
}

// PetEntity 玩家宠物实体
type PetEntity struct {
	PetOwnID      int64               `gorm:"column:pet_own_id;primaryKey"`    // 宠物唯一ID
	UserID        int64               `gorm:"column:user_id;index"`            // 玩家ID
	PetID         int32               `gorm:"column:pet_id;"`                  // 宠物配置ID
	Level         int32               `gorm:"column:level;"`                   // 等级
	Star          int32               `gorm:"column:star;"`                    // 星级
	HeroOwnId     int64               `gorm:"column:hero_own_id"`              // 绑定的英雄，0 表示在宠物背包中未装备
	PassiveSkills tool.JSONInt32Slice `gorm:"column:passive_skills;type:json"` // 已解锁的被动技能ID列表
	IsDeleted     bool                `gorm:"column:is_deleted"`               // 是否删除（软删除）
}

func (p *PetEntity) TableName() string {
	return "pet"
}

// PetModel 玩家宠物集合模型
// 层级与 AccessoryModel 一致，通过 HeroOwnId 绑定到具体英雄
type PetModel struct {
	UserId   int64
	Entities map[int64]*PetEntity // key: petOwnId
	Changed  map[int64]map[string]interface{}

	// 运行时缓存：key=petId, value=该petId的最高等级和星级
	PetMaxLevelCache map[int32]int32 // petId -> 最高等级
	PetMaxStarCache  map[int32]int32 // petId -> 最高星级

	// 反向索引：petId -> set of petOwnId，用于O(1)删除时查找同petId的其他实例
	PetIDToOwnIDs map[int32]map[int64]bool // petId -> petOwnId集合

	// 装备索引：heroOwnId -> petOwnId（每个英雄最多一只宠物）
	HeroToPetOwnID map[int64]int64

	// 登录后由 PetAffinityModel.BindPetModel 回填，用于 AddPet/UpdateStar 写图鉴 JSON
	AffinityModel *PetAffinityModel
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记。
// 规则与 AccessoryModel 保持一致：
// - 若能定位到具体英雄（宠物穿戴/卸下/升级/升星且当前有绑定英雄），返回这些 heroOwnId，allDirty=false
// - 若无法定位（例如卸下导致 hero_own_id 变为 0），返回空列表，allDirty=true（推主阵容全量刷新）
func (m *PetModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if m == nil || len(m.Changed) == 0 {
		return []int64{}, false
	}

	heroOwnIDs := make(map[int64]bool)
	needFullRefresh := false
	for petOwnID, changedFields := range m.Changed {
		ent := m.Entities[petOwnID]
		if ent == nil || ent.IsDeleted {
			if _, ok := changedFields["hero_own_id"]; ok {
				needFullRefresh = true
			}
			continue
		}
		if ent.HeroOwnId != 0 {
			heroOwnIDs[ent.HeroOwnId] = true
			continue
		}
		if _, ok := changedFields["hero_own_id"]; ok {
			needFullRefresh = true
		}
	}

	// 如果无法定位到具体英雄（常见：卸下导致 hero_own_id=0），则视为全局脏。
	if len(heroOwnIDs) == 0 && needFullRefresh {
		return []int64{}, true
	}

	res := make([]int64, 0, len(heroOwnIDs))
	for ownID := range heroOwnIDs {
		res = append(res, ownID)
	}
	return res, false
}

var _ logicCommon.PlayerModelInterface = (*PetModel)(nil)
var _ logicCommon.HeroAttrInterface = (*PetModel)(nil)

func NewPetModel(
	userId int64,
	entities map[int64]*PetEntity,
	levelCache map[int32]int32,
	starCache map[int32]int32,
	petIDToOwnIDs map[int32]map[int64]bool,
	heroToPetOwnID map[int64]int64,
) *PetModel {
	return &PetModel{
		UserId:           userId,
		Entities:         entities,
		Changed:          make(map[int64]map[string]interface{}),
		PetMaxLevelCache: levelCache,
		PetMaxStarCache:  starCache,
		PetIDToOwnIDs:    petIDToOwnIDs,
		HeroToPetOwnID:   heroToPetOwnID,
	}
}

// LoadPetModel 从数据库加载玩家全部宠物
func LoadPetModel(userId int64, isLogin bool) (*PetModel, error) {
	entities := make(map[int64]*PetEntity)
	levelCache := make(map[int32]int32)
	starCache := make(map[int32]int32)
	petIDToOwnIDs := make(map[int32]map[int64]bool)
	heroToPetOwnID := make(map[int64]int64)
	rows, err := easyDB.GetPlayerEntitiesByWhere[PetEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewPetModel(userId, entities, levelCache, starCache, petIDToOwnIDs, heroToPetOwnID), err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		if row.IsDeleted {
			continue
		}
		entities[row.PetOwnID] = row

		// 反向索引
		if petIDToOwnIDs[row.PetID] == nil {
			petIDToOwnIDs[row.PetID] = make(map[int64]bool)
		}
		petIDToOwnIDs[row.PetID][row.PetOwnID] = true

		// 缓存：load 时直接构建最大值
		if row.Level > levelCache[row.PetID] {
			levelCache[row.PetID] = row.Level
		}
		if row.Star > starCache[row.PetID] {
			starCache[row.PetID] = row.Star
		}

		// 装备索引
		if row.HeroOwnId != 0 {
			heroToPetOwnID[row.HeroOwnId] = row.PetOwnID
		}
	}
	// 物理清理：把已标记删除的数据从 DB 中删掉（与 hero 的做法一致）
	if isLogin {
		_ = DeleteDeletedPets(userId)
	}
	return NewPetModel(userId, entities, levelCache, starCache, petIDToOwnIDs, heroToPetOwnID), nil
}

// CreatePetModel 为新玩家创建空的宠物模型
func CreatePetModel(userId int64) (*PetModel, error) {
	entities := make(map[int64]*PetEntity)
	levelCache := make(map[int32]int32)
	starCache := make(map[int32]int32)
	petIDToOwnIDs := make(map[int32]map[int64]bool)
	heroToPetOwnID := make(map[int64]int64)
	return NewPetModel(userId, entities, levelCache, starCache, petIDToOwnIDs, heroToPetOwnID), nil
}

func (m *PetModel) getChangedMap(petOwnId int64) map[string]interface{} {
	if m.Changed[petOwnId] == nil {
		m.Changed[petOwnId] = make(map[string]interface{})
	}
	return m.Changed[petOwnId]
}

func (m *PetModel) GetPet(petOwnId int64) *PetEntity {
	if m == nil || m.Entities == nil {
		return nil
	}
	return m.Entities[petOwnId]
}

// HasPetId 当前是否仍持有该 petId 的任意未删除实例（不关心星级/是否上阵）。
func (m *PetModel) HasPetId(petID int32) bool {
	if m == nil || m.Entities == nil {
		return false
	}
	// 优先走反向索引（存在即代表至少有一只同 petId 宠物）
	if m.PetIDToOwnIDs != nil {
		if s := m.PetIDToOwnIDs[petID]; len(s) > 0 {
			// 二次确认避免索引极端情况下残留已删除 ownId
			for ownID := range s {
				if p := m.Entities[ownID]; p != nil && !p.IsDeleted && p.PetID == petID {
					return true
				}
			}
		}
	}
	// 兜底：全量扫描
	for _, p := range m.Entities {
		if p == nil || p.IsDeleted {
			continue
		}
		if p.PetID == petID {
			return true
		}
	}
	return false
}

func (m *PetModel) GetEquippedPetByHero(heroOwnId int64) *PetEntity {
	if m == nil || m.Entities == nil {
		return nil
	}
	if m.HeroToPetOwnID == nil {
		return nil
	}
	petOwnID := m.HeroToPetOwnID[heroOwnId]
	if petOwnID == 0 {
		return nil
	}
	p := m.Entities[petOwnID]
	if p == nil || p.IsDeleted || p.HeroOwnId != heroOwnId {
		return nil
	}
	return p
}

// GetBagPets 返回背包中的宠物（HeroOwnId==0）
func (m *PetModel) GetBagPets() []*PetEntity {
	if m == nil || m.Entities == nil {
		return nil
	}
	res := make([]*PetEntity, 0)
	for _, p := range m.Entities {
		if p == nil {
			continue
		}
		if p.IsDeleted {
			continue
		}
		if p.HeroOwnId == 0 {
			res = append(res, p)
		}
	}
	return res
}

// WearPet 给英雄穿戴宠物。
// 约束：同一英雄最多只能挂 1 只宠物；调用该方法会自动卸下原有宠物。
func (m *PetModel) WearPet(petOwnId int64, heroOwnId int64) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		return
	}
	if heroOwnId <= 0 {
		return
	}

	// 先卸下该英雄当前已绑定的其他宠物（O(1)）
	if m.HeroToPetOwnID != nil {
		if oldPetOwnID := m.HeroToPetOwnID[heroOwnId]; oldPetOwnID != 0 && oldPetOwnID != petOwnId {
			if old := m.Entities[oldPetOwnID]; old != nil && !old.IsDeleted && old.HeroOwnId == heroOwnId {
				old.HeroOwnId = 0
				m.getChangedMap(old.PetOwnID)["hero_own_id"] = int64(0)
			}
		}
	}

	// 再把当前宠物挂到该英雄
	pet.HeroOwnId = heroOwnId
	m.getChangedMap(petOwnId)["hero_own_id"] = heroOwnId
	if m.HeroToPetOwnID != nil {
		m.HeroToPetOwnID[heroOwnId] = petOwnId
	}
}

// UnwearPet 将宠物从英雄身上卸下，回到宠物背包（HeroOwnId = 0）
func (m *PetModel) UnwearPet(petOwnId int64) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		return
	}
	oldHeroOwnId := pet.HeroOwnId
	pet.HeroOwnId = 0
	m.getChangedMap(petOwnId)["hero_own_id"] = int64(0)
	if oldHeroOwnId != 0 && m.HeroToPetOwnID != nil {
		if cur := m.HeroToPetOwnID[oldHeroOwnId]; cur == petOwnId {
			delete(m.HeroToPetOwnID, oldHeroOwnId)
		}
	}
}

func (m *PetModel) UpdateLevel(petOwnId int64, level int32) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		return
	}
	pet.Level = level
	m.getChangedMap(petOwnId)["level"] = level

	// 简单维护缓存：只更新更大的值
	if m.PetMaxLevelCache != nil && level > m.PetMaxLevelCache[pet.PetID] {
		m.PetMaxLevelCache[pet.PetID] = level
	}
}

func (m *PetModel) UpdateStar(petOwnId int64, star int32) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		return
	}
	pet.Star = star
	m.getChangedMap(petOwnId)["star"] = star

	// 简单维护缓存：只更新更大的值
	if m.PetMaxStarCache != nil && star > m.PetMaxStarCache[pet.PetID] {
		m.PetMaxStarCache[pet.PetID] = star
	}
	if m.AffinityModel != nil {
		m.AffinityModel.RecordPetMaxStar(pet.PetID, star)
	}
}

func (m *PetModel) UpdatePassiveSkills(petOwnId int64, skills tool.JSONInt32Slice) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		return
	}
	pet.PassiveSkills = skills
	m.getChangedMap(petOwnId)["passive_skills"] = skills
}

// DeletePet 删除宠物（软删除：标记 is_deleted=true；下次 Load 时会物理清理）
func (m *PetModel) DeletePet(petOwnId int64) {
	pet := m.Entities[petOwnId]
	if pet == nil {
		return
	}
	if pet.IsDeleted {
		delete(m.Entities, petOwnId)
		return
	}

	petID := pet.PetID
	oldHeroOwnId := pet.HeroOwnId

	pet.IsDeleted = true
	// 直接落库，避免依赖 SaveModelToDB（删除后实体会从 Entities 中移除）
	easyDB.UpdatePlayerEntity(pet, map[string]interface{}{"is_deleted": true}, m.UserId)
	delete(m.Entities, petOwnId)

	// 从反向索引中移除
	if m.PetIDToOwnIDs != nil && m.PetIDToOwnIDs[petID] != nil {
		delete(m.PetIDToOwnIDs[petID], petOwnId)
		if len(m.PetIDToOwnIDs[petID]) == 0 {
			delete(m.PetIDToOwnIDs, petID)
		}
	}

	// 从装备索引中移除
	if oldHeroOwnId != 0 && m.HeroToPetOwnID != nil {
		if cur := m.HeroToPetOwnID[oldHeroOwnId]; cur == petOwnId {
			delete(m.HeroToPetOwnID, oldHeroOwnId)
		}
	}

	// 重新计算该 petID 的缓存（只遍历同 petID 的其他实例）
	m.rebuildCacheForPetID(petID)
}

// AddPet 向玩家添加一只新宠物（仅数据插入，不负责消耗道具）
func (m *PetModel) AddPet(entity *PetEntity) error {
	if entity == nil {
		return nil
	}
	// 初始化默认值：避免外部漏填导致脏数据
	if entity.Level <= 0 {
		entity.Level = 1
	}
	if entity.Star < 0 {
		entity.Star = 0
	}
	// 新宠物默认在背包中
	// 若外部明确指定了绑定英雄，则不覆盖
	if entity.HeroOwnId < 0 {
		entity.HeroOwnId = 0
	}
	entity.IsDeleted = false

	m.Entities[entity.PetOwnID] = entity

	// 维护反向索引
	if m.PetIDToOwnIDs == nil {
		m.PetIDToOwnIDs = make(map[int32]map[int64]bool)
	}
	if m.PetIDToOwnIDs[entity.PetID] == nil {
		m.PetIDToOwnIDs[entity.PetID] = make(map[int64]bool)
	}
	m.PetIDToOwnIDs[entity.PetID][entity.PetOwnID] = true

	// 更新缓存
	if m.PetMaxLevelCache != nil && entity.Level > m.PetMaxLevelCache[entity.PetID] {
		m.PetMaxLevelCache[entity.PetID] = entity.Level
	}
	if m.PetMaxStarCache != nil && entity.Star > m.PetMaxStarCache[entity.PetID] {
		m.PetMaxStarCache[entity.PetID] = entity.Star
	}

	// 更新装备索引
	if entity.HeroOwnId != 0 {
		if m.HeroToPetOwnID == nil {
			m.HeroToPetOwnID = make(map[int64]int64)
		}
		m.HeroToPetOwnID[entity.HeroOwnId] = entity.PetOwnID
	}

	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return err
	}
	if m.AffinityModel != nil {
		m.AffinityModel.RecordPetMaxStar(entity.PetID, entity.Star)
	}
	return nil
}

// DeleteDeletedPets 定时/加载时清理玩家已删除的宠物数据
func DeleteDeletedPets(userId int64) error {
	return easyDB.DeletePlayerEntityByWhere[PetEntity](map[string]interface{}{"user_id": userId, "is_deleted": true}, userId)
}

// SaveModelToDB 按 Changed 批量落库
func (m *PetModel) SaveModelToDB() {
	if len(m.Changed) == 0 {
		return
	}
	for id, fields := range m.Changed {
		if ent := m.Entities[id]; ent != nil {
			easyDB.UpdatePlayerEntity(ent, fields, m.UserId)
		}
	}
	m.Changed = make(map[int64]map[string]interface{})
}

func (m *PetModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 目前宠物没有定时逻辑，先留空壳
}

// GetHeroAttr 实现 HeroAttrInterface，用于参与英雄属性聚合。
// 当前实现：基础属性 + 等级属性 + 星级属性 + 缘分（全局加成）。
// 不涉及战斗与技能释放逻辑。
func (m *PetModel) GetHeroAttr(heroOwnId int64, attrId int32) int64 {
	total := int64(0)

	// 1) 英雄装配的宠物属性（O(1)索引）
	if pet := m.GetEquippedPetByHero(heroOwnId); pet != nil {
		total += gameConfig.CalcPetAttr(pet.PetID, pet.Level, pet.Star, attrId)
	}

	// 2) 缘分属性（对所有英雄生效）
	// 已迁移到 PetAffinityModel（需要激活/升级），这里不再自动按“满足即生效”计算。

	return total
}

// GetBuffAttr 预留：如果宠物也有 Buff 类属性，可以在这里实现。
func (m *PetModel) GetBuffAttr(heroOwnId int64, attrId int32) int64 {
	if m == nil || heroOwnId == 0 || attrId <= 0 {
		return 0
	}
	pet := m.GetEquippedPetByHero(heroOwnId)
	if pet == nil {
		return 0
	}

	total := int64(0)
	for _, skillID := range pet.PassiveSkills {
		if skillID <= 0 {
			continue
		}
		buffCfg := gameConfig.GetAttrBuffCfg(skillID)
		if buffCfg == nil {
			continue
		}
		for i, id := range buffCfg.Attr {
			if id != attrId || i < 0 || i >= len(buffCfg.AttrNum) {
				continue
			}
			total += int64(buffCfg.AttrNum[i])
		}
	}
	return total
}

// --------------------------
// PetAffinity（缘分）模型
// 放在 petModel.go 内，便于集中维护
// --------------------------

// PetAffinityEntity 宠物缘分状态表：记录玩家对某个缘分组合的激活与等级。
type PetAffinityEntity struct {
	UserID     int64 `gorm:"column:user_id;primaryKey"`
	AffinityID int32 `gorm:"column:affinity_id;primaryKey"`
	Level      int32 `gorm:"column:level"` // 0=未激活；>=1 表示已激活且档位（与配置 AttrNum 行对应：Level=1 -> 第0行）
}

func (e *PetAffinityEntity) TableName() string {
	return "pet_affinity"
}

type PetAffinityModel struct {
	UserId   int64
	Entities map[int32]*PetAffinityEntity // key: affinityId
	Changed  map[int32]map[string]interface{}

	// 运行时依赖：用于校验组合是否满足（不落库）
	PetModel *PetModel

	// 图鉴簿（pet_id -> 历史最高星），落库到 pet_affinity_book（行式结构）
	BookEntities map[int32]*PetAffinityBookEntity // key: petId
	BookChanged  map[int32]bool                   // key: petId
}

// GetChangedHeroOwnIDs implements [logicCommon.HeroAttrInterface].
func (m *PetAffinityModel) GetChangedHeroOwnIDs() (heroOwnIDs []int64, allDirty bool) {
	// 缘分是全局属性：任意缘分激活/升级/图鉴变更都可能影响上阵英雄属性
	if m == nil || len(m.Changed) == 0 {
		return []int64{}, false
	}
	return []int64{}, true
}

var _ logicCommon.PlayerModelInterface = (*PetAffinityModel)(nil)
var _ logicCommon.HeroAttrInterface = (*PetAffinityModel)(nil)

func NewPetAffinityModel(userId int64, entities map[int32]*PetAffinityEntity) *PetAffinityModel {
	return &PetAffinityModel{
		UserId:       userId,
		Entities:     entities,
		Changed:      make(map[int32]map[string]interface{}),
		PetModel:     nil,
		BookEntities: make(map[int32]*PetAffinityBookEntity),
		BookChanged:  make(map[int32]bool),
	}
}

func LoadPetAffinityModel(userId int64) (*PetAffinityModel, error) {
	entities := make(map[int32]*PetAffinityEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[PetAffinityEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewPetAffinityModel(userId, entities), err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		// 兼容旧数据：历史上 affinity_id=0 被用作图鉴簿（现已迁移到 pet_affinity_book）
		if row.AffinityID == 0 {
			continue
		}
		entities[row.AffinityID] = row
	}
	m := NewPetAffinityModel(userId, entities)

	// 加载图鉴簿（行式结构）
	if bookMap, err := LoadPetAffinityBookEntities(userId); err == nil && bookMap != nil {
		m.BookEntities = bookMap
	}

	return m, nil
}

func CreatePetAffinityModel(userId int64) (*PetAffinityModel, error) {
	return NewPetAffinityModel(userId, make(map[int32]*PetAffinityEntity)), nil
}

func (m *PetAffinityModel) BindPetModel(petModel *PetModel) {
	m.PetModel = petModel
	if petModel != nil {
		petModel.AffinityModel = m
		// 老号补图鉴：按当前持有宠物把 pet_id->最高星 写入 JSON（与历史合并只升不降）
		for _, p := range petModel.Entities {
			if p == nil || p.IsDeleted {
				continue
			}
			m.RecordPetMaxStar(p.PetID, p.Star)
		}
	}
}

func (m *PetAffinityModel) getChangedMap(affinityId int32) map[string]interface{} {
	if m.Changed[affinityId] == nil {
		m.Changed[affinityId] = make(map[string]interface{})
	}
	return m.Changed[affinityId]
}

func (m *PetAffinityModel) SaveModelToDB() {
	if len(m.Changed) == 0 && len(m.BookChanged) == 0 {
		return
	}
	for id, fields := range m.Changed {
		if ent := m.Entities[id]; ent != nil {
			easyDB.UpdatePlayerEntity(ent, fields, m.UserId)
		}
	}
	m.Changed = make(map[int32]map[string]interface{})

	for petID := range m.BookChanged {
		if ent := m.BookEntities[petID]; ent != nil {
			easyDB.UpdatePlayerEntity(ent, map[string]interface{}{"max_star": ent.MaxStar}, m.UserId)
		}
	}
	m.BookChanged = make(map[int32]bool)
}

func (m *PetAffinityModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	_ = currentTime
	_ = passDay
	_ = senderMsg
}

// GetHeroAttr 缘分属性（对所有英雄生效）
func (m *PetAffinityModel) GetHeroAttr(heroOwnId int64, attrId int32) int64 {
	_ = heroOwnId
	if m == nil {
		return 0
	}
	allAff := gameConfig.GetAllPetAffinityCfg()
	if allAff == nil {
		return 0
	}

	total := int64(0)
	for _, affCfg := range allAff {
		if affCfg == nil {
			continue
		}
		ent := m.Entities[affCfg.Id]
		if ent == nil || ent.Level <= 0 {
			continue
		}
		// 可选：激活后仍要求组合条件满足才生效（默认开启，便于和界面一致）
		if !m.MeetAffinityRequirementAtLevel(affCfg, ent.Level) {
			continue
		}

		row := ent.Level - 1
		if row < 0 || int(row) >= len(affCfg.AttrNum) {
			continue
		}
		if affCfg.Attr == attrId {
			total += int64(affCfg.AttrNum[row])
		}
	}
	return total
}

// starBookMap 图鉴 pet_id -> 历史最高星
func (m *PetAffinityModel) starBookMap() map[int32]int32 {
	if m == nil {
		return nil
	}
	out := make(map[int32]int32)
	for petID, ent := range m.BookEntities {
		if ent == nil {
			continue
		}
		out[petID] = ent.MaxStar
	}
	return out
}

// RecordPetMaxStar 获得/升星时写入图鉴（只增不减）
func (m *PetAffinityModel) RecordPetMaxStar(petID int32, star int32) {
	if m == nil {
		return
	}
	if petID <= 0 {
		return
	}

	ent := m.BookEntities[petID]
	if ent == nil {
		ent = EnsurePetAffinityBookEntity(m.UserId, petID)
		m.BookEntities[petID] = ent
	}
	if star <= ent.MaxStar {
		return
	}
	ent.MaxStar = star
	m.BookChanged[petID] = true
}

// MeetAffinityRequirementAtLevel 缘分条件（按目标缘分等级档位判断）：
// - 配置里的 petId 是组合清单；petStar 是每个等级档位的统一星级门槛（例如 0 表示激活档，2 表示升到 2 级的门槛）
// - 必须“拥有过/当前拥有”该组合中的每个 pet（图鉴 JSON 有记录，或当前背包仍有该 pet）
// - 且每个 pet 的历史最高星（或当前最高星） >= 该等级档位要求的门槛
func (m *PetAffinityModel) MeetAffinityRequirementAtLevel(aff *gameConfig.PetAffinityCfg, level int32) bool {
	if m == nil || aff == nil {
		return false
	}
	if level <= 0 {
		return false
	}
	book := m.starBookMap()

	if len(aff.PetId) == 0 {
		return false
	}

	idx := level - 1
	if idx < 0 || int(idx) >= len(aff.PetStar) {
		return false
	}
	starReq := aff.PetStar[idx]

	for _, petIdReq := range aff.PetId {
		maxStar := int32(0)
		hasEverOrNow := false

		if book != nil {
			if v, ok := book[petIdReq]; ok {
				hasEverOrNow = true
				maxStar = v
			}
		}

		// 图鉴无记录时，退化到当前背包（用于老号未补全图鉴/初始化异常等情况）。
		if !hasEverOrNow && m.PetModel != nil {
			curMax := petIdCurrentMaxStar(m.PetModel, petIdReq)
			if curMax > 0 || m.PetModel.HasPetId(petIdReq) {
				hasEverOrNow = true
				maxStar = max(maxStar, curMax)
			}
		}

		if !hasEverOrNow || maxStar < starReq {
			return false
		}
	}
	return true
}

func petIdCurrentMaxStar(petModel *PetModel, petID int32) int32 {
	if petModel == nil || petModel.Entities == nil {
		return 0
	}
	maxS := int32(0)
	for _, p := range petModel.Entities {
		if p == nil || p.IsDeleted || p.PetID != petID {
			continue
		}
		if p.Star > maxS {
			maxS = p.Star
		}
	}
	return maxS
}

func (m *PetAffinityModel) GetBuffAttr(heroOwnId int64, attrId int32) int64 {
	_ = heroOwnId
	_ = attrId
	return 0
}

func (m *PetAffinityModel) EnsureEntity(affinityId int32) *PetAffinityEntity {
	if m.Entities[affinityId] != nil {
		return m.Entities[affinityId]
	}
	ent := &PetAffinityEntity{
		UserID:     m.UserId,
		AffinityID: affinityId,
		Level:      0,
	}
	m.Entities[affinityId] = ent
	_ = easyDB.CreatePlayerEntity(ent)
	return ent
}

func (m *PetAffinityModel) UpdateLevel(affinityId int32, level int32) {
	ent := m.Entities[affinityId]
	if ent == nil {
		ent = m.EnsureEntity(affinityId)
	}
	ent.Level = level
	m.getChangedMap(affinityId)["level"] = level
}

// rebuildCacheForPetID 重新计算指定petId的缓存
func (m *PetModel) rebuildCacheForPetID(petID int32) {
	if m == nil {
		return
	}
	if m.PetMaxLevelCache == nil {
		m.PetMaxLevelCache = make(map[int32]int32)
	}
	if m.PetMaxStarCache == nil {
		m.PetMaxStarCache = make(map[int32]int32)
	}

	maxLevel := int32(0)
	maxStar := int32(0)

	if m.PetIDToOwnIDs != nil && m.PetIDToOwnIDs[petID] != nil {
		for ownID := range m.PetIDToOwnIDs[petID] {
			p := m.Entities[ownID]
			if p == nil || p.IsDeleted {
				continue
			}
			if p.Level > maxLevel {
				maxLevel = p.Level
			}
			if p.Star > maxStar {
				maxStar = p.Star
			}
		}
	} else {
		// 兜底：索引缺失时退化为全量遍历（理论上不会发生）
		for _, p := range m.Entities {
			if p == nil || p.IsDeleted || p.PetID != petID {
				continue
			}
			if p.Level > maxLevel {
				maxLevel = p.Level
			}
			if p.Star > maxStar {
				maxStar = p.Star
			}
		}
	}

	m.PetMaxLevelCache[petID] = maxLevel
	m.PetMaxStarCache[petID] = maxStar
}
