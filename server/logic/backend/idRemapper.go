package backend

import (
	"github.com/drop/GoServer/server/tool"
)

// IdRemapper 负责维护 oldId → newId 的映射，并按需生成新的雪花 ID
type IdRemapper struct {
	heroOwnIdMap  map[int64]int64 // oldHeroOwnId → newHeroOwnId
	equipOwnIdMap map[int64]int64 // oldEquipOwnId → newEquipOwnId
	petOwnIdMap   map[int64]int64 // oldPetOwnId → newPetOwnId
	NewUserId     int64           // 新生成的 user_id

	userIdGen  *tool.IdGenerator
	heroIdGen  *tool.IdGenerator
	equipIdGen *tool.IdGenerator
	petIdGen   *tool.IdGenerator
}

// NewIdRemapper 创建新的 ID 重映射器，并立即生成新的 user_id
func NewIdRemapper() *IdRemapper {
	return &IdRemapper{
		heroOwnIdMap:  make(map[int64]int64),
		equipOwnIdMap: make(map[int64]int64),
		petOwnIdMap:   make(map[int64]int64),
		NewUserId:     backendUserIdGenerator.NextId(),
		userIdGen:     backendUserIdGenerator,
		heroIdGen:     backendHeroIdGenerator,
		equipIdGen:    backendEquipIdGenerator,
		petIdGen:      backendPetIdGenerator,
	}
}

// RemapHeroOwnId 映射英雄唯一ID，0值不映射
func (r *IdRemapper) RemapHeroOwnId(oldId int64) int64 {
	if oldId == 0 {
		return 0
	}
	if newId, ok := r.heroOwnIdMap[oldId]; ok {
		return newId
	}
	newId := r.heroIdGen.NextId()
	r.heroOwnIdMap[oldId] = newId
	return newId
}

// RemapEquipOwnId 映射装备唯一ID，0值不映射
func (r *IdRemapper) RemapEquipOwnId(oldId int64) int64 {
	if oldId == 0 {
		return 0
	}
	if newId, ok := r.equipOwnIdMap[oldId]; ok {
		return newId
	}
	newId := r.equipIdGen.NextId()
	r.equipOwnIdMap[oldId] = newId
	return newId
}

// RemapPetOwnId 映射宠物唯一ID，0值不映射
func (r *IdRemapper) RemapPetOwnId(oldId int64) int64 {
	if oldId == 0 {
		return 0
	}
	if newId, ok := r.petOwnIdMap[oldId]; ok {
		return newId
	}
	newId := r.petIdGen.NextId()
	r.petOwnIdMap[oldId] = newId
	return newId
}

// RemapHeroOwnIdList 映射英雄唯一ID列表（用于 HeroFormation 的 hero_own_id_list）
func (r *IdRemapper) RemapHeroOwnIdList(oldList []int64) []int64 {
	if len(oldList) == 0 {
		return oldList
	}
	newList := make([]int64, len(oldList))
	for i, oldId := range oldList {
		newList[i] = r.RemapHeroOwnId(oldId)
	}
	return newList
}

// RemapEquipOwnIdList 映射装备唯一ID列表（用于 HeroDetails 的 equipment_id JSON数组）
func (r *IdRemapper) RemapEquipOwnIdList(oldList []int64) []int64 {
	if len(oldList) == 0 {
		return oldList
	}
	newList := make([]int64, len(oldList))
	for i, oldId := range oldList {
		newList[i] = r.RemapEquipOwnId(oldId)
	}
	return newList
}
