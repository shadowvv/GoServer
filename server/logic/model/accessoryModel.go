package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

type AccessoryEntity struct {
	UserId         int64 `gorm:"column:user_id;primaryKey"`
	AccessoryId    int32 `gorm:"column:accessory_id;primaryKey"`
	AccessoryLevel int32 `gorm:"column:accessory_level;not null"`
	Num            int32 `gorm:"column:num;not null"`
	HeroOwnId      int64 `gorm:"column:hero_own_id"`
}

func (a *AccessoryEntity) TableName() string {
	return "accessory"
}

type AccessoryModel struct {
	UserId   int64
	Entities map[int32]*AccessoryEntity
	Changed  map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*AccessoryModel)(nil)
var _ logicCommon.HeroAttrInterface = (*AccessoryModel)(nil)

func NewAccessoryModel(userId int64, entities map[int32]*AccessoryEntity) *AccessoryModel {
	return &AccessoryModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int32]map[string]interface{}),
	}
}

func (a *AccessoryModel) SaveModelToDB() {
	for id, v := range a.Changed {
		if v != nil || len(v) != 0 {
			easyDB.UpdatePlayerEntity[AccessoryEntity](a.Entities[id], v, a.UserId)
		}
	}
	a.Changed = make(map[int32]map[string]interface{})
}

func (a *AccessoryModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {

}

func (a *AccessoryModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	var attr int64 = 0
	for _, v := range a.Entities {
		if v.HeroOwnId == heroId {
			attr += gameConfig.GetAccessoryAttr2(v.AccessoryId, v.AccessoryLevel, attrId)
		}
		attr += gameConfig.GetAccessoryAttr1(v.AccessoryId, v.AccessoryLevel, attrId)
	}
	return attr
}

func (a *AccessoryModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

// GetChangedHeroOwnIDs 返回本次有变化的英雄OwnID列表和全局脏标记
// - 如果变化涉及 hero_own_id（穿戴/卸下/替换），返回对应的 heroOwnId 列表，allDirty=false
// - 如果只有 level/num 变化（升级/获取），返回空列表，allDirty=true
func (a *AccessoryModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if len(a.Changed) == 0 {
		return []int64{}, false
	}
	// 检查是否有 hero_own_id 变化（穿戴/卸下/替换）
	heroOwnIDs := make(map[int64]bool)
	for accessoryId, changes := range a.Changed {
		if _, hasHeroOwnID := changes["hero_own_id"]; hasHeroOwnID {
			if ent := a.Entities[accessoryId]; ent != nil && ent.HeroOwnId != 0 {
				heroOwnIDs[ent.HeroOwnId] = true
			}
		}
	}
	if len(heroOwnIDs) == 0 {
		// 没有特定英雄变化（只有升级/获取），全局脏
		return []int64{}, true
	}
	res := make([]int64, 0, len(heroOwnIDs))
	for ownID := range heroOwnIDs {
		res = append(res, ownID)
	}
	return res, false
}

func (a *AccessoryModel) creatUserAccessory(entity *AccessoryEntity) error {
	a.Entities[entity.AccessoryId] = entity
	return easyDB.CreatePlayerEntity(entity)
}

func (a *AccessoryModel) AddAccessory(accessoryId int32, num int32) error {
	if a.Entities[accessoryId] == nil {
		entity := &AccessoryEntity{
			UserId:         a.UserId,
			AccessoryId:    accessoryId,
			AccessoryLevel: 1,
			HeroOwnId:      0,
			Num:            num,
		}
		err := a.creatUserAccessory(entity)
		if err != nil {
			return err
		}
		a.UpdateHeroOwnId(accessoryId, entity.HeroOwnId)
		return nil
	} else {
		a.UpdateAccessoryNum(accessoryId, a.Entities[accessoryId].Num+num)
		return nil
	}
}

func LoadAccessory(userId int64) (*AccessoryModel, error) {
	entities := make(map[int32]*AccessoryEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[AccessoryEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewAccessoryModel(userId, entities), err
	}
	for _, v := range rows {
		entities[v.AccessoryId] = v
	}
	return NewAccessoryModel(userId, entities), nil
}

func (a *AccessoryModel) UpdateAccessoryLevel(accessoryId int32, level int32) {
	a.Entities[accessoryId].AccessoryLevel = level
	if a.Changed[accessoryId] == nil {
		a.Changed[accessoryId] = make(map[string]interface{})
	}
	a.Changed[accessoryId]["accessory_level"] = level
}

func (a *AccessoryModel) UpdateAccessoryNum(accessoryId int32, num int32) {
	a.Entities[accessoryId].Num = num
	if a.Changed[accessoryId] == nil {
		a.Changed[accessoryId] = make(map[string]interface{})
	}
	a.Changed[accessoryId]["num"] = num
}

func (a *AccessoryModel) UpdateHeroOwnId(accessoryId int32, heroOwnId int64) {
	a.Entities[accessoryId].HeroOwnId = heroOwnId
	if a.Changed[accessoryId] == nil {
		a.Changed[accessoryId] = make(map[string]interface{})
	}
	a.Changed[accessoryId]["hero_own_id"] = heroOwnId
}

func (a *AccessoryModel) LevelUp(accessoryId int32, userLevel int32) []*pb.AccessoryDetail {
	res := make([]*pb.AccessoryDetail, 0)
	if _, ok := a.Entities[accessoryId]; !ok {
		return nil
	}
	detail := a.Entities[accessoryId]
	level := detail.AccessoryLevel
	num := detail.Num
	cfg := gameConfig.GetAccessoryLevelUpCfg(level + 1)
	for cfg != nil && num >= cfg.Sum {
		level++
		cfg = gameConfig.GetAccessoryLevelUpCfg(level + 1)
	}

	if cfg == nil {
		CfgBase := gameConfig.GetAccessoryBaseCfg(accessoryId)
		if CfgBase == nil {
			return nil
		}
		CfgLevel := gameConfig.GetAccessoryLevelUpCfg(level)
		if CfgLevel == nil {
			return nil
		}

		// 修改兑换道具
		if entity := a.Entities[CfgBase.NextId]; entity != nil {
			changedNum := (num - CfgLevel.Sum) / CfgBase.Rate
			if changedNum > 0 {
				res = append(res, &pb.AccessoryDetail{
					AccessoryId:    entity.AccessoryId,
					AccessoryLevel: entity.AccessoryLevel,
					AccessoryNum:   entity.Num + changedNum - gameConfig.GetAccessoryLevelUpCfg(entity.AccessoryLevel).Sum,
					HeroOwnId:      entity.HeroOwnId,
					Power:          gameConfig.GetAccessoryPower(entity.AccessoryId, entity.AccessoryLevel, userLevel),
				})
				a.UpdateAccessoryNum(entity.AccessoryId, entity.Num+changedNum)
				num -= CfgBase.Rate * changedNum
			}
		}
		// 修改升级道具
		res = append(res, &pb.AccessoryDetail{
			AccessoryId:    accessoryId,
			AccessoryLevel: level,
			AccessoryNum:   num - gameConfig.GetAccessoryLevelUpCfg(level).Sum,
			HeroOwnId:      detail.HeroOwnId,
			Power:          gameConfig.GetAccessoryPower(accessoryId, level, userLevel),
		})
		a.UpdateAccessoryNum(accessoryId, num)
		a.UpdateAccessoryLevel(accessoryId, level)
	} else if level != detail.AccessoryLevel {
		a.UpdateAccessoryLevel(accessoryId, level)
		res = append(res, &pb.AccessoryDetail{
			AccessoryId:    accessoryId,
			AccessoryLevel: level,
			AccessoryNum:   num - gameConfig.GetAccessoryLevelUpCfg(level).Sum,
			HeroOwnId:      detail.HeroOwnId,
			Power:          gameConfig.GetAccessoryPower(accessoryId, level, userLevel),
		})
	}

	return res
}

func (a *AccessoryModel) WearAccessory(accessoryId int32, heroOwnId int64, userLevel int32) []*pb.AccessoryDetail {
	a.UpdateHeroOwnId(accessoryId, heroOwnId)
	detail := a.Entities[accessoryId]
	res := make([]*pb.AccessoryDetail, 0)
	for _, entity := range a.Entities {
		if heroOwnId == entity.HeroOwnId && accessoryId != entity.AccessoryId {
			for _, v := range a.UnloadAccessory(entity.AccessoryId, entity.HeroOwnId, userLevel) {
				res = append(res, v)
			}
		}
	}
	res = append(res, &pb.AccessoryDetail{
		AccessoryId:    accessoryId,
		AccessoryLevel: detail.AccessoryLevel,
		AccessoryNum:   detail.Num - gameConfig.GetAccessoryLevelUpCfg(detail.AccessoryLevel).Sum,
		HeroOwnId:      detail.HeroOwnId,
		Power:          gameConfig.GetAccessoryPower(accessoryId, detail.AccessoryLevel, userLevel),
	})
	return res
}

func (a *AccessoryModel) UnloadAccessory(accessoryId int32, heroOwnId int64, userLevel int32) []*pb.AccessoryDetail {
	a.UpdateHeroOwnId(accessoryId, int64(0))
	detail := a.Entities[accessoryId]
	res := make([]*pb.AccessoryDetail, 0)
	res = append(res, &pb.AccessoryDetail{
		AccessoryId:    accessoryId,
		AccessoryLevel: detail.AccessoryLevel,
		AccessoryNum:   detail.Num - gameConfig.GetAccessoryLevelUpCfg(detail.AccessoryLevel).Sum,
		HeroOwnId:      detail.HeroOwnId,
		Power:          gameConfig.GetAccessoryPower(accessoryId, detail.AccessoryLevel, userLevel),
	})
	return res
}

type AccessoryLuckyEntity struct {
	UserId         int64 `gorm:"column:user_id;primaryKey" json:"user_id"`
	LuckyLevel     int32 `gorm:"column:lucky_level;" json:"lucky_level"`
	LuckyId        int32 `gorm:"column:lucky_id;primaryKey" json:"lucky_id"`
	LuckyNum       int32 `gorm:"column:lucky_num;" json:"lucky_num"`
	FreeNum        int32 `gorm:"column:free_num;" json:"free_num"`
	FreeUpdateTime int64 `gorm:"column:free_update_time;" json:"free_update_time"`
	FreeUsedNum    int32 `gorm:"column:free_used_num;" json:"free_used_num"`
}

// TableName 指定表名
func (a *AccessoryLuckyEntity) TableName() string {
	return "accessory_lucky"
}

type AccessoryLuckyModel struct {
	UserId   int64
	Entities map[int32]*AccessoryLuckyEntity
	Changed  map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*AccessoryLuckyModel)(nil)

func (a *AccessoryLuckyModel) SaveModelToDB() {
	for id, v := range a.Changed {
		if v != nil || len(v) != 0 {
			easyDB.UpdatePlayerEntity[AccessoryLuckyEntity](a.Entities[id], v, a.UserId)
		}
	}
	a.Changed = make(map[int32]map[string]interface{})
}

func (a *AccessoryLuckyModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	//TODO implement me
}

func NewAccessoryLuckyModel(entities map[int32]*AccessoryLuckyEntity, userId int64) *AccessoryLuckyModel {
	return &AccessoryLuckyModel{
		UserId:   userId,
		Entities: entities,
		Changed:  make(map[int32]map[string]interface{}),
	}
}

func LoadAccessoryLucky(userId int64) (*AccessoryLuckyModel, error) {
	// todo 根据活动 ， 只加载活动时间内卡池
	entities := make(map[int32]*AccessoryLuckyEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[AccessoryLuckyEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewAccessoryLuckyModel(entities, userId), err
	}
	for _, v := range rows {
		entities[v.LuckyId] = v
	}
	return NewAccessoryLuckyModel(entities, userId), nil
}

func (a *AccessoryLuckyModel) creatAccessoryLucky(entity *AccessoryLuckyEntity) error {
	a.Entities[entity.LuckyId] = entity
	return easyDB.CreatePlayerEntity(entity)
}

func (a *AccessoryLuckyModel) AddAccessoryLucky(luckyId int32, luckyNum int32) (*AccessoryLuckyEntity, error) {
	if a.Entities[luckyId] == nil {
		entity := &AccessoryLuckyEntity{
			UserId:         a.UserId,
			LuckyId:        luckyId,
			LuckyLevel:     1,
			LuckyNum:       luckyNum,
			FreeNum:        0,
			FreeUpdateTime: 0,
			FreeUsedNum:    0,
		}
		return entity, a.creatAccessoryLucky(entity)
	}
	return nil, errors.New("luckyId is exist")
}

func (a *AccessoryLuckyModel) UpdateLuckyLevel(LuckyId int32, level int32) {
	a.Entities[LuckyId].LuckyLevel = level
	if a.Changed[LuckyId] == nil {
		a.Changed[LuckyId] = make(map[string]interface{})
	}
	a.Changed[LuckyId]["lucky_level"] = level
}

func (a *AccessoryLuckyModel) UpdateLuckyNum(luckyId int32, luckyNum int32) {
	a.Entities[luckyId].LuckyNum = luckyNum
	if a.Changed[luckyId] == nil {
		a.Changed[luckyId] = make(map[string]interface{})
	}
	a.Changed[luckyId]["lucky_num"] = luckyNum
}

func (a *AccessoryLuckyModel) UpdateFreeNum(luckyId int32, freeNum int32) {
	a.Entities[luckyId].FreeNum = freeNum
	if a.Changed[luckyId] == nil {
		a.Changed[luckyId] = make(map[string]interface{})
	}
	a.Changed[luckyId]["free_num"] = freeNum
}

func (a *AccessoryLuckyModel) UpdateFreeUpdateTime(luckyId int32, freeUpdateTime int64) {
	a.Entities[luckyId].FreeUpdateTime = freeUpdateTime
	if a.Changed[luckyId] == nil {
		a.Changed[luckyId] = make(map[string]interface{})
	}
	a.Changed[luckyId]["free_update_time"] = freeUpdateTime
}

func (a *AccessoryLuckyModel) UpdateFreeUsedNum(luckyId, freeUsedNum int32) {
	a.Entities[luckyId].FreeUsedNum = freeUsedNum
	if a.Changed[luckyId] == nil {
		a.Changed[luckyId] = make(map[string]interface{})
	}
	a.Changed[luckyId]["free_used_num"] = freeUsedNum
}
