package model

import (
	"encoding/json"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

type ExpeditionEntity struct {
	UserId                   int64  `gorm:"column:user_id;primaryKey"` // 玩家id
	LastRecoveryStaminaTime  int64  `gorm:"column:last_recovery_stamina_time"`
	DailyFreeStaminaTimes    int32  `gorm:"column:daily_free_stamina_times"`
	LastDailyFreeStaminaTime int64  `gorm:"column:last_daily_free_stamina_time"`
	MonsterRefreshCount      string `gorm:"column:monster_refresh_count"`

	monsterCount map[int32]int32 `gorm:"-"`
}

func (u *ExpeditionEntity) TableName() string {
	return "player_expedition_data"
}

type ExpeditionBattlefieldEntity struct {
	UserId           int64  `gorm:"column:user_id;primaryKey"`        // 玩家id
	BattlefieldId    int32  `gorm:"column:battlefield_id;primaryKey"` // 战场id
	BattlefieldLevel int32  `gorm:"column:battlefield_level"`         // 战场等级
	PointInfos       string `gorm:"column:battle_point_infos"`        // 战场点信息

	PointMonsterInfos map[int32]*PointInfo `gorm:"-"` // 战场点信息
}

func (u *ExpeditionBattlefieldEntity) TableName() string {
	return "player_expedition_battlefield_data"
}

type PointInfo struct {
	PointId         int32                    `json:"pId"`             // 点id
	MonsterId       int32                    `json:"mId"`             // 怪物id
	Status          int32                    `json:"status"`          // 状态（0:一般，1:被出征，2:领奖）
	Level           int32                    `json:"level"`           // 随机出来时的战场等级
	NextRefreshTime int64                    `json:"nextRefreshTime"` // 下次刷新时间
	RewardItem      []*gameConfig.ItemConfig `json:"RewardItem"`      // 奖励的道具
	IsWin           int32                    `json:"isWin"`           // 是否胜利
}

type ExpeditionSlotEntity struct {
	UserId        int64 `gorm:"column:user_id;primaryKey"` // 玩家id
	SlotId        int32 `gorm:"column:slot_id;primaryKey"` // 槽id
	BattlefieldId int32 `gorm:"column:battlefield_id"`     // 目标战场id
	PointId       int32 `gorm:"column:point_id"`           // 目标点id
	StartTime     int64 `gorm:"column:start_time"`         // 开始时间
	EndTime       int64 `gorm:"column:end_time"`           // 结束时间
}

func (u *ExpeditionSlotEntity) TableName() string {
	return "player_expedition_slot_data"
}

type ExpeditionModel struct {
	UserId int64
	Player *PlayerModel

	ExpeditionData      *ExpeditionEntity
	BattlefieldEntities map[int32]*ExpeditionBattlefieldEntity
	SlotEntities        map[int32]*ExpeditionSlotEntity

	ExpeditionChanged  map[string]interface{}
	BattlefieldChanged map[int32]map[string]interface{}
	ActiveChanged      map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*ExpeditionModel)(nil)

func CreateExpeditionModel(player *PlayerModel) (*ExpeditionModel, error) {
	model := newExpeditionModel(player.GetUserId(), player, nil, nil, nil)
	return model, nil
}

func newExpeditionModel(userId int64, player *PlayerModel, expeditionData *ExpeditionEntity, battlefieldEntities map[int32]*ExpeditionBattlefieldEntity, activeEntities map[int32]*ExpeditionSlotEntity) *ExpeditionModel {
	if expeditionData == nil {
		expeditionData = &ExpeditionEntity{
			UserId:                   userId,
			LastRecoveryStaminaTime:  0,
			DailyFreeStaminaTimes:    0,
			LastDailyFreeStaminaTime: 0,
			MonsterRefreshCount:      "",
			monsterCount:             make(map[int32]int32),
		}
		if err := easyDB.CreatePlayerEntity(expeditionData); err != nil {
			logger.ErrorBySprintf("create expeditionData error: %v", err)
		}
	}
	if battlefieldEntities == nil {
		battlefieldEntities = make(map[int32]*ExpeditionBattlefieldEntity)
	}
	if activeEntities == nil {
		activeEntities = make(map[int32]*ExpeditionSlotEntity)
	}
	model := &ExpeditionModel{
		UserId:              userId,
		Player:              player,
		ExpeditionData:      expeditionData,
		BattlefieldEntities: battlefieldEntities,
		SlotEntities:        activeEntities,
		ExpeditionChanged:   make(map[string]interface{}),
		BattlefieldChanged:  make(map[int32]map[string]interface{}),
		ActiveChanged:       make(map[int32]map[string]interface{}),
	}
	for _, bf := range model.BattlefieldEntities {
		bf.PointMonsterInfos = make(map[int32]*PointInfo)
		if bf.PointInfos != "" {
			err := json.Unmarshal([]byte(bf.PointInfos), &bf.PointMonsterInfos)
			if err != nil {
				return nil
			}
		}
	}
	expeditionData.monsterCount = make(map[int32]int32)
	if expeditionData.MonsterRefreshCount != "" {
		err := json.Unmarshal([]byte(expeditionData.MonsterRefreshCount), &expeditionData.monsterCount)
		if err != nil {
			return nil
		}
	}
	return model
}

func LoadExpeditionModel(player *PlayerModel) (*ExpeditionModel, error) {
	expeditionData, err := easyDB.GetPlayerEntityByWhere[ExpeditionEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		logger.ErrorBySprintf("load expeditionData error: %v", err)
		expeditionData = nil
	}

	battlefields, err := easyDB.GetPlayerEntitiesByWhere[ExpeditionBattlefieldEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		battlefields = nil
		logger.ErrorBySprintf("load battlefields error: %v", err)
	}
	battlefieldMap := make(map[int32]*ExpeditionBattlefieldEntity)
	for _, entity := range battlefields {
		if entity == nil {
			continue
		}
		battlefieldMap[entity.BattlefieldId] = entity
	}

	actives, err := easyDB.GetPlayerEntitiesByWhere[ExpeditionSlotEntity](map[string]interface{}{"user_id": player.GetUserId()})
	if err != nil {
		actives = nil
		logger.ErrorBySprintf("load actives error: %v", err)
	}
	activeMap := make(map[int32]*ExpeditionSlotEntity)
	for _, entity := range actives {
		if entity == nil {
			continue
		}
		activeMap[entity.SlotId] = entity
	}

	model := newExpeditionModel(player.GetUserId(), player, expeditionData, battlefieldMap, activeMap)
	return model, nil
}

func (d *ExpeditionModel) SaveModelToDB() {
	if len(d.ExpeditionChanged) > 0 {
		easyDB.UpdatePlayerEntity(d.ExpeditionData, d.ExpeditionChanged, d.UserId)
		d.ExpeditionChanged = make(map[string]interface{})
	}

	for battlefieldId, changes := range d.BattlefieldChanged {
		if entity := d.BattlefieldEntities[battlefieldId]; entity != nil && len(changes) > 0 {
			easyDB.UpdatePlayerEntity(entity, changes, d.UserId)
		}
	}
	d.BattlefieldChanged = make(map[int32]map[string]interface{})

	for slotId, changes := range d.ActiveChanged {
		if entity := d.SlotEntities[slotId]; entity != nil && len(changes) > 0 {
			easyDB.UpdatePlayerEntity(entity, changes, d.UserId)
		}
	}
	d.ActiveChanged = make(map[int32]map[string]interface{})
}

func (d *ExpeditionModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		d.ExpeditionData.DailyFreeStaminaTimes = 0
		d.ExpeditionData.LastDailyFreeStaminaTime = 0
		d.ExpeditionData.MonsterRefreshCount = ""
		d.ExpeditionData.monsterCount = make(map[int32]int32)

		d.ExpeditionChanged["daily_free_stamina_times"] = d.ExpeditionData.DailyFreeStaminaTimes
		d.ExpeditionChanged["last_daily_free_stamina_time"] = d.ExpeditionData.LastDailyFreeStaminaTime
		d.ExpeditionChanged["monster_refresh_count"] = d.ExpeditionData.MonsterRefreshCount
	}
	d.checkAllStatus()
}

func (d *ExpeditionModel) CheckExpeditionUnlock(slotNum int32) error {
	for _, cfg := range gameConfig.GetAllCityDispatchCfg() {
		if cfg.Level != 1 {
			continue
		}
		if d.BattlefieldEntities[cfg.Area] != nil {
			continue
		}
		if !unlockService.CheckUnlock(cfg.Unlock, d.Player) {
			continue
		}
		entity := &ExpeditionBattlefieldEntity{
			UserId:            d.UserId,
			BattlefieldId:     cfg.Area,
			BattlefieldLevel:  1,
			PointMonsterInfos: make(map[int32]*PointInfo),
		}
		emptyPoint := make([]int32, len(cfg.MonsterPoint))
		copy(emptyPoint, cfg.MonsterPoint)
		infos := randomMonster(cfg, cfg.AllMonsterNum, emptyPoint, d.ExpeditionData.monsterCount)
		for _, info := range infos {
			entity.PointMonsterInfos[info.PointId] = info
		}
		marshal, err := json.Marshal(entity.PointMonsterInfos)
		if err != nil {
			logger.ErrorBySprintf("json marshal battlefieldPointInfos error: %v", err)
			continue
		}
		entity.PointInfos = string(marshal)
		if err := easyDB.CreatePlayerEntity(entity); err != nil {
			return err
		}
		d.BattlefieldEntities[cfg.Area] = entity
	}

	for i := int32(1); i <= slotNum; i++ {
		if d.SlotEntities[i] != nil {
			continue
		}
		entity := &ExpeditionSlotEntity{
			UserId: d.UserId,
			SlotId: i,
		}
		if err := easyDB.CreatePlayerEntity(entity); err != nil {
			return err
		}
		d.SlotEntities[i] = entity
	}
	return nil
}

func randomMonster(cfg *gameConfig.CityDispatchCfg, monsterNum int32, emptyPointIds []int32, monsterCount map[int32]int32) []*PointInfo {
	currentTime := tool.UnixNowMilli()
	if monsterNum > int32(len(emptyPointIds)) {
		monsterNum = int32(len(emptyPointIds))
	}
	pointIds := tool.Shuffle(emptyPointIds)
	points := make([]*PointInfo, 0)
	for i := int32(0); i < monsterNum; i++ {
		pointId := pointIds[i]
		monsterId := cfg.RandomMonsterId(monsterCount)
		if monsterId == 0 {
			logger.ErrorBySprintf("cityDispatchCfg.RandomMonsterId() monsterId == 0")
			continue
		}
		if _, ok := monsterCount[monsterId]; !ok {
			monsterCount[monsterId] = 1
		} else {
			monsterCount[monsterId] += 1
		}
		points = append(points, &PointInfo{
			PointId:         pointId,
			MonsterId:       monsterId,
			Status:          enum.ExpeditionPointStatusIdle,
			Level:           cfg.Level,
			NextRefreshTime: currentTime + int64(cfg.Cd*1000),
			RewardItem:      make([]*gameConfig.ItemConfig, 0),
		})
	}
	return points
}

func (d *ExpeditionModel) checkAllStatus() {
	currentTime := tool.UnixNowMilli()
	interval := currentTime - d.ExpeditionData.LastRecoveryStaminaTime

	// 体力回复
	if interval > int64(gameConfig.GetStaminaRecoveryTime()*1000) {
		num := interval / int64(gameConfig.GetStaminaRecoveryTime()*1000)
		d.ExpeditionData.LastRecoveryStaminaTime = d.ExpeditionData.LastRecoveryStaminaTime + num*int64(gameConfig.GetStaminaRecoveryTime())*1000
		d.ExpeditionChanged["last_recovery_stamina_time"] = d.ExpeditionData.LastRecoveryStaminaTime

		count := itemService.GetItemCount(d.Player, enum.STAMINA_ITEM_ID)
		if count < int64(gameConfig.GetMaximumStamina()) {
			if count+num > int64(gameConfig.GetMaximumStamina()) {
				num = int64(gameConfig.GetMaximumStamina()) - count
			}
			_ = itemService.AddItem(d.Player, &gameConfig.ItemConfig{
				ID:  enum.STAMINA_ITEM_ID,
				Num: num,
			}, enum.ITEM_CHANGE_REASON_RECOVERY_STAMINA)
		}
	}

	slotChange := make([]*pb.ExpeditionSlotInfo, 0)
	pointChange := make([]*pb.ExpeditionPointInfo, 0)
	battleFieldChange := make(map[int32]bool)
	// 出征状态更新
	for slotId, active := range d.SlotEntities {
		if active == nil {
			continue
		}
		if active.StartTime > 0 && active.EndTime <= currentTime {
			_, point, battlefieldId := d.completeSlot(slotId)
			if point != nil {
				pointChange = append(pointChange, &pb.ExpeditionPointInfo{
					PointId:   point.PointId,
					MonsterId: point.MonsterId,
					IsReward:  1,
				})
				eventServer.SubmitDispatchKillMonsterEvent(d.Player.GetUserId(), d.Player.GetUserServerId())
				battleFieldChange[battlefieldId] = true
			}

			slotChange = append(slotChange, &pb.ExpeditionSlotInfo{
				SlotId: slotId,
				Status: 0,
			})
		}
	}

	// 战场怪物状态更新
	for _, bf := range d.BattlefieldEntities {
		cfg := gameConfig.GetCityDispatchCfg(bf.BattlefieldId, bf.BattlefieldLevel)
		if cfg == nil {
			continue
		}
		removeIndexes := make([]int32, 0)
		emptyPoints := make(map[int32]bool)
		for _, point := range cfg.MonsterPoint {
			emptyPoints[point] = true
		}
		monsterNum := cfg.AllMonsterNum
		for index, info := range bf.PointMonsterInfos {
			if info.Status == enum.ExpeditionPointStatusIdle && currentTime > info.NextRefreshTime {
				removeIndexes = append(removeIndexes, index)
			} else {
				delete(emptyPoints, info.PointId)
			}
		}
		for _, index := range removeIndexes {
			pointChange = append(pointChange, &pb.ExpeditionPointInfo{
				PointId: bf.PointMonsterInfos[index].PointId,
			})
			delete(bf.PointMonsterInfos, index)
			battleFieldChange[bf.BattlefieldId] = true
		}
		currentMonsterNum := int32(len(bf.PointMonsterInfos))
		monsterNum = monsterNum - currentMonsterNum
		if monsterNum > 0 {
			temp := make([]int32, 0)
			for pointId := range emptyPoints {
				temp = append(temp, pointId)
			}
			newPoints := randomMonster(cfg, monsterNum, temp, d.ExpeditionData.monsterCount)
			if len(newPoints) > 0 {
				battleFieldChange[bf.BattlefieldId] = true
				for _, point := range newPoints {
					bf.PointMonsterInfos[point.PointId] = point
					pointChange = append(pointChange, &pb.ExpeditionPointInfo{
						PointId:         point.PointId,
						MonsterId:       point.MonsterId,
						NextRefreshTime: point.NextRefreshTime,
					})
				}
			}
		}
	}
	for bfId := range battleFieldChange {
		d.markBattlefieldPointInfosChanged(bfId)
	}

	if len(pointChange) > 0 || len(slotChange) > 0 {
		messageSender.SendMessage(d.Player, pb.MESSAGE_ID_EXPEDITION_CHANGE_PUSH, &pb.ExpeditionChangePush{
			Slots:  slotChange,
			Points: pointChange,
		})
	}
}

func (d *ExpeditionModel) getBattlefieldChangedMap(battlefieldId int32) map[string]interface{} {
	if d.BattlefieldChanged[battlefieldId] == nil {
		d.BattlefieldChanged[battlefieldId] = make(map[string]interface{})
	}
	return d.BattlefieldChanged[battlefieldId]
}

func (d *ExpeditionModel) getSlotChangedMap(slotId int32) map[string]interface{} {
	if d.ActiveChanged[slotId] == nil {
		d.ActiveChanged[slotId] = make(map[string]interface{})
	}
	return d.ActiveChanged[slotId]
}

func (d *ExpeditionModel) GetSlotById(slotId int32) *ExpeditionSlotEntity {
	return d.SlotEntities[slotId]
}

func (d *ExpeditionModel) UnlockSlot(id int32) *ExpeditionSlotEntity {
	entity := d.SlotEntities[id]
	if entity == nil {
		entity = &ExpeditionSlotEntity{
			UserId:        d.UserId,
			SlotId:        id,
			BattlefieldId: 0,
			StartTime:     0,
			EndTime:       0,
			PointId:       0,
		}
		d.SlotEntities[id] = entity
		err := easyDB.CreatePlayerEntity(entity)
		if err != nil {
			logger.ErrorBySprintf("UnlockSlot CreatePlayerEntity error: %v", err)
			return nil
		}
	}
	return entity
}

func (d *ExpeditionModel) GetPointInfo(battleId, pointId int32) *PointInfo {
	if d.BattlefieldEntities[battleId] == nil {
		return nil
	}
	return d.BattlefieldEntities[battleId].PointMonsterInfos[pointId]
}

func (d *ExpeditionModel) StartExpedition(slot *ExpeditionSlotEntity, battleFileId int32, info *PointInfo, reward []*gameConfig.ItemConfig, win bool) {
	slot.PointId = info.PointId
	slot.BattlefieldId = battleFileId
	slot.StartTime = tool.UnixNowMilli()
	cfg := gameConfig.GetCityMonsterCfg(info.MonsterId)
	if cfg == nil {
		slot.EndTime = slot.StartTime + tool.HOUR_MILLI
	} else {
		slot.EndTime = slot.StartTime + int64(cfg.Time)*1000
	}

	changed := d.getSlotChangedMap(slot.SlotId)
	changed["point_id"] = slot.PointId
	changed["battlefield_id"] = slot.BattlefieldId
	changed["start_time"] = slot.StartTime
	changed["end_time"] = slot.EndTime

	info.Status = enum.ExpeditionPointStatusBusy
	info.IsWin = 1
	if !win {
		info.IsWin = 0
	}
	info.RewardItem = reward

	d.markBattlefieldPointInfosChanged(battleFileId)
}

func (d *ExpeditionModel) FinishSlot(slotId int32) *ExpeditionSlotEntity {
	slot, point, battlefieldId := d.completeSlot(slotId)
	if point != nil {
		d.markBattlefieldPointInfosChanged(battlefieldId)
	}
	d.checkAllStatus()
	return slot
}

func (d *ExpeditionModel) CancelSlot(slotId int32) *ExpeditionSlotEntity {
	slot := d.SlotEntities[slotId]
	if slot != nil {
		bf := d.BattlefieldEntities[slot.BattlefieldId]
		if bf != nil {
			point := bf.PointMonsterInfos[slot.PointId]
			if point != nil {
				point.Status = enum.ExpeditionPointStatusIdle
				point.IsWin = 0
				point.RewardItem = make([]*gameConfig.ItemConfig, 0)
				d.markBattlefieldPointInfosChanged(slot.BattlefieldId)
			}
		}

		slot.BattlefieldId = 0
		slot.EndTime = 0
		slot.PointId = 0
		slot.StartTime = 0

		changed := d.getSlotChangedMap(slotId)
		changed["battlefield_id"] = slot.BattlefieldId
		changed["point_id"] = slot.PointId
		changed["start_time"] = slot.StartTime
		changed["end_time"] = slot.EndTime
	}
	return slot
}

func (d *ExpeditionModel) completeSlot(slotId int32) (*ExpeditionSlotEntity, *PointInfo, int32) {
	slot := d.SlotEntities[slotId]
	if slot == nil {
		return nil, nil, 0
	}

	battlefieldId := slot.BattlefieldId
	pointId := slot.PointId
	var point *PointInfo
	if bf := d.BattlefieldEntities[battlefieldId]; bf != nil {
		point = bf.PointMonsterInfos[pointId]
		if point != nil {
			point.Status = enum.ExpeditionPointStatusReward
		}
	}

	slot.BattlefieldId = 0
	slot.EndTime = 0
	slot.PointId = 0
	slot.StartTime = 0

	changed := d.getSlotChangedMap(slotId)
	changed["battlefield_id"] = slot.BattlefieldId
	changed["point_id"] = slot.PointId
	changed["start_time"] = slot.StartTime
	changed["end_time"] = slot.EndTime

	d.Player.StaticData.AddExpeditionNum(1)
	return slot, point, battlefieldId
}

func (d *ExpeditionModel) markBattlefieldPointInfosChanged(battlefieldId int32) {
	bf := d.BattlefieldEntities[battlefieldId]
	if bf == nil {
		return
	}

	marshal, err := json.Marshal(bf.PointMonsterInfos)
	if err != nil {
		logger.ErrorBySprintf("expeditionModel json.Marshal point infos error: %v", err)
		return
	}
	bf.PointInfos = string(marshal)
	bfChanged := d.getBattlefieldChangedMap(battlefieldId)
	bfChanged["battle_point_infos"] = bf.PointInfos
}

func (d *ExpeditionModel) ChangeLevel(battlefieldId int32, level int32) {
	bf := d.BattlefieldEntities[battlefieldId]
	if bf != nil {
		bf.BattlefieldLevel = level
		changed := d.getBattlefieldChangedMap(battlefieldId)
		changed["battlefield_level"] = level
	}
}

func (d *ExpeditionModel) SpeedUpSlot(slotId int32, speedTime int64) *ExpeditionSlotEntity {
	slot := d.SlotEntities[slotId]
	if slot != nil {
		slot.EndTime -= speedTime

		changed := d.getSlotChangedMap(slotId)
		changed["end_time"] = slot.EndTime
	}
	return slot
}

func (d *ExpeditionModel) ClaimFreeStamina() {
	d.ExpeditionData.DailyFreeStaminaTimes += 1
	d.ExpeditionData.LastDailyFreeStaminaTime = tool.UnixNowMilli()

	d.ExpeditionChanged["daily_free_stamina_times"] = d.ExpeditionData.DailyFreeStaminaTimes
	d.ExpeditionChanged["last_daily_free_stamina_time"] = d.ExpeditionData.LastDailyFreeStaminaTime
}

func (d *ExpeditionModel) ClaimPointReward(battleId, pointId int32) {
	bf := d.BattlefieldEntities[battleId]
	if bf != nil {
		delete(bf.PointMonsterInfos, pointId)
	}
}
