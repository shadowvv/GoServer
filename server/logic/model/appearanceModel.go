package model

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type AppearanceEntity struct {
	UserId       int64 `gorm:"primary_key;column:user_id"`
	AppearanceId int32 `gorm:"column:appearance_id;primary_key"`
	EndTime      int64 `gorm:"column:end_time"`
	IsWear       bool  `gorm:"column:is_wear"`
}

func (a *AppearanceEntity) TableName() string {
	return "appearance"
}

type AppearanceModel struct {
	AppearanceEntities map[int32]*AppearanceEntity
	UserId             int64
	Changed            map[int32]map[string]interface{}
	player             *PlayerModel

	wearingAppearanceId map[enum.AvatarType]int32
}

var _ logicCommon.PlayerModelInterface = &AppearanceModel{}
var _ logicCommon.HeroAttrInterface = &AppearanceModel{}

func NewAppearanceModel(userId int64, entities map[int32]*AppearanceEntity, player *PlayerModel, wearingAppearanceId map[enum.AvatarType]int32) *AppearanceModel {
	return &AppearanceModel{
		AppearanceEntities: entities,
		UserId:             userId,
		Changed:            make(map[int32]map[string]interface{}),

		player:              player,
		wearingAppearanceId: wearingAppearanceId,
	}
}

func LoadAppearanceModel(userId int64, player *PlayerModel) (*AppearanceModel, error) {
	entities := make(map[int32]*AppearanceEntity)
	rows, err := easyDB.GetPlayerEntitiesByWhere[AppearanceEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	wearingAppearanceId := make(map[enum.AvatarType]int32)
	for _, row := range rows {
		// 如果到期以后续费，内存中没有数据会create，但库里有数据，可能err
		//if row.EndTime > 0 && row.EndTime <= tool.UnixNowMilli() {
		//	continue
		//}
		if row.IsWear {
			appearanceCfg := gameConfig.GetAvatarItemCfg(row.AppearanceId)
			if appearanceCfg == nil {
				logger.ErrorBySprintf("appearance cfg is not exist, appearanceId: %d", row.AppearanceId)
				continue
			}
			wearingAppearanceId[appearanceCfg.CfgType] = row.AppearanceId
		}
		entities[row.AppearanceId] = row
	}
	return NewAppearanceModel(userId, entities, player, wearingAppearanceId), nil
}

func (a *AppearanceModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	attr := int64(0)
	for _, v := range a.AppearanceEntities {
		appearanceCfg := gameConfig.GetAvatarItemCfg(v.AppearanceId)
		if appearanceCfg == nil {
			continue
		}
		for id, value := range appearanceCfg.Attr {
			if value == attrId {
				attr += int64(appearanceCfg.AttrNum[id])
			}
		}
	}
	return attr
}

func (a *AppearanceModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

func (a *AppearanceModel) GetChangedHeroOwnIDs() (heroOwnIDs []int64, allDirty bool) {
	if len(a.Changed) == 0 {
		return []int64{}, false
	}
	return []int64{}, true
}

func (a *AppearanceModel) SaveModelToDB() {
	for id, v := range a.Changed {
		if v != nil || len(v) != 0 {
			easyDB.UpdatePlayerEntity[AppearanceEntity](a.AppearanceEntities[id], v, a.UserId)
		}
	}
	a.Changed = make(map[int32]map[string]interface{})
}

func (a *AppearanceModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	allAppearanceCfg := gameConfig.GetAllAvatarItemCfg()
	if allAppearanceCfg == nil {
		return
	}
	for _, v := range allAppearanceCfg {
		if v.ItemId == 0 {
			continue
		}
		if v.UnlockId == 0 {
			continue
		}
		if a.AppearanceEntities[v.Id] != nil && a.AppearanceEntities[v.Id].EndTime != 0 {
			entity := a.AppearanceEntities[v.Id]
			if entity.EndTime > 0 && entity.EndTime <= currentTime {
				if entity.IsWear {
					err := a.UnfixAppearance(v.Id)
					if err != nil {
						platformLogger.ErrorWithUser("unfix appearance error", a.player, err)
						continue
					}
					// todo 到期了需要推送给客户端吗？
					//messageSender.SendMessage(a.player, pb.MESSAGE_ID_PUSH_AVATAR_DETAIL, &pb.PushAvatarDetail{
					//	AvatarDetail: &pb.AvatarDetail{
					//		Id:      entity.AppearanceId,
					//		EndTime: entity.EndTime,
					//		IsWear:  false,
					//	},
					//})

					// 如果到期续费，内存中没有数据会create，但库里有数据，可能err
					//delete(a.AppearanceEntities, v.Id)
				}
			}
			continue
		}
		// TODO:有bug，处理一下
		if unlockService.CheckUnlock(v.UnlockId, a.player) && (a.AppearanceEntities[v.Id] == nil || (a.AppearanceEntities[v.Id] != nil && a.AppearanceEntities[v.Id].EndTime != 0)) {
			a.UseAppearanceItem(v.ItemId, 1)
		}
	}
}

func (a *AppearanceModel) UpdateEndTime(appearanceId int32, time int64) {
	a.AppearanceEntities[appearanceId].EndTime = time
	if a.Changed[appearanceId] == nil {
		a.Changed[appearanceId] = make(map[string]interface{})
	}
	a.Changed[appearanceId]["end_time"] = time
}

func (a *AppearanceModel) UpdateIsWear(appearanceId int32, isWear bool) {
	a.AppearanceEntities[appearanceId].IsWear = isWear
	if a.Changed[appearanceId] == nil {
		a.Changed[appearanceId] = make(map[string]interface{})
	}
	a.Changed[appearanceId]["is_wear"] = isWear
}

func (a *AppearanceModel) creatAppearance(entity *AppearanceEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}

func (a *AppearanceModel) AddAppearance(appearanceId int32, endTime int64) error {
	if a.AppearanceEntities[appearanceId] != nil {
		if endTime == 0 {
			a.UpdateEndTime(appearanceId, 0)
		} else {
			a.UpdateEndTime(appearanceId, a.AppearanceEntities[appearanceId].EndTime+endTime)
		}
		return nil
	}
	entity := &AppearanceEntity{
		AppearanceId: appearanceId,
		EndTime:      endTime,
		IsWear:       false,
		UserId:       a.UserId,
	}
	err := a.creatAppearance(entity)
	if err != nil {
		return err
	}
	a.AppearanceEntities[appearanceId] = entity
	a.player.HeroDetailsModel.refreshHeroAttrTree()
	return nil
}

// 穿戴
func (a *AppearanceModel) WearAppearance(appearanceId int32) error {
	if a.AppearanceEntities[appearanceId] == nil {
		return errors.New("appearance entity is not exist")
	}
	appearanceCfg := gameConfig.GetAvatarItemCfg(appearanceId)
	if appearanceCfg == nil {
		return errors.New("appearance cfg is not exist")
	}
	if a.wearingAppearanceId[appearanceCfg.CfgType] == 0 {
		a.UpdateIsWear(appearanceId, true)
		a.wearingAppearanceId[appearanceCfg.CfgType] = appearanceId
		if a.player != nil {
			a.player.UpdatePlayerBasicInfoToRedis()
		}
		return nil
	}
	a.UpdateIsWear(a.wearingAppearanceId[appearanceCfg.CfgType], false)
	a.UpdateIsWear(appearanceId, true)
	a.wearingAppearanceId[appearanceCfg.CfgType] = appearanceId
	if a.player != nil {
		a.player.UpdatePlayerBasicInfoToRedis()
	}
	return nil
}

// 卸下
func (a *AppearanceModel) UnfixAppearance(appearanceId int32) error {
	if a.AppearanceEntities[appearanceId] == nil {
		return errors.New("appearance entity is not exist")
	}
	appearanceCfg := gameConfig.GetAvatarItemCfg(appearanceId)
	if appearanceCfg == nil {
		return errors.New("appearance cfg is not exist")
	}
	a.UpdateIsWear(appearanceId, false)
	a.wearingAppearanceId[appearanceCfg.CfgType] = 0
	if a.player != nil {
		a.player.UpdatePlayerBasicInfoToRedis()
	}
	return nil
}

// 使用背包中的外观道具(时间为0表永久，否则表正常时间戳)
func (a *AppearanceModel) UseAppearanceItem(itemId int32, num int32) {
	itemCfg := gameConfig.GetItemCfg(itemId)
	if itemCfg == nil || itemCfg.ShowGroup != int32(enum.ITEM_TYPE_APPEARANCE) {
		messageSender.SendErrorMessage(a.player, pb.MESSAGE_ID_USE_ITEMS_REQ, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	timeHourToMilli := int32(60 * 60 * 1000)

	if a.AppearanceEntities[itemCfg.TargetId] != nil {
		// 已永久，兑换
		if a.AppearanceEntities[itemCfg.TargetId].EndTime == 0 {
			err := itemService.AddItem(a.player, &gameConfig.ItemConfig{ID: enum.DIAMOND_ITEM_ID, Num: int64(num * itemCfg.Value)}, enum.ITEM_CHANGE_REASON_EXCHANGE_ITEM)
			if err != nil {
				messageSender.SendErrorMessage(a.player, pb.MESSAGE_ID_USE_ITEMS_REQ, pb.ERROR_CODE_ADD_ITEM_ERROR)
			}
			return
		}
		// 续费
		endTime := int64(0)
		if itemCfg.TargetId2*timeHourToMilli > 0 {
			// 如果当前时间还没有过期，则在当前基础上续费，否则从现在开始续费
			if a.AppearanceEntities[itemCfg.TargetId].EndTime >= tool.UnixNowMilli() {
				endTime = tool.UnixNowMilli() + int64(itemCfg.TargetId2*timeHourToMilli)*int64(num)
			} else {
				endTime = a.AppearanceEntities[itemCfg.TargetId].EndTime + int64(itemCfg.TargetId2*timeHourToMilli)*int64(num)
			}
		}
		a.UpdateEndTime(itemCfg.TargetId, endTime)
	} else {
		// 新增
		endTime := int64(0)
		if itemCfg.TargetId2*timeHourToMilli > 0 {
			endTime = tool.UnixNowMilli() + int64(itemCfg.TargetId2*timeHourToMilli)*int64(num)
		}
		err := a.AddAppearance(itemCfg.TargetId, endTime)
		if err != nil {
			logger.ErrorBySprintf("add appearance error, appearanceId: %d, err: %v", itemCfg.TargetId, err)
			messageSender.SendErrorMessage(a.player, pb.MESSAGE_ID_USE_ITEMS_REQ, pb.ERROR_CODE_APPEARANCE_ADD_FAILED)
			return
		}
	}
	entity := a.AppearanceEntities[itemCfg.TargetId]
	res := make([]*pb.AvatarDetail, 0)
	res = append(res, &pb.AvatarDetail{
		Id:      entity.AppearanceId,
		EndTime: entity.EndTime,
		IsWear:  entity.IsWear,
	})
	messageSender.SendMessage(a.player, pb.MESSAGE_ID_PUSH_AVATAR_DETAIL, &pb.PushAvatarDetail{
		AvatarDetail: res,
	})
}

func (a *AppearanceModel) GetWearAppearance(avatarType enum.AvatarType) int32 {
	return a.wearingAppearanceId[avatarType]
}
