package model

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

type LotteryEntity struct {
	UserId                  int64 `gorm:"column:user_id;primaryKey"`         // 用户ID
	Id                      int32 `gorm:"column:id;primaryKey"`              // 卡池ID
	AllCount                int32 `gorm:"column:all_count"`                  // 总抽取次数
	LastBasicGuaranteeNum   int32 `gorm:"column:last_basic_guarantee_num"`   // 上次基础保底计数
	LastSpecialGuaranteeNum int32 `gorm:"column:last_special_guarantee_num"` // 上次循环保底计数
	SpecialGuaranteeCount   int32 `gorm:"column:special_guarantee_count"`    // 循环保底生效档位
	LastChangeTime          int64 `gorm:"column:last_change_time"`           // 上次更新时间
	LastOnceEffectiveNum    int32 `gorm:"column:last_once_effective_num"`    // 上次单次保底计数
	OnceEffectiveCount      int32 `gorm:"column:once_effective_count"`       // 单次保底生效档位
	FirstDropFree           int32 `gorm:"column:first_drop_free"`            // 首次免费次数
	LastFirstDropFreeTime   int64 `gorm:"column:last_once_effective_time"`   // 上次免费次数更新时间
	IsDirty                 bool  `gorm:"column:is_dirty"`                   // 是否歪了

	ActVersion string `gorm:"column:act_version"` // 活动版本号
}

func (LotteryEntity) TableName() string {
	return "lottery"
}

type LotteryLuckyEventEntity struct {
	UserId           int64 `gorm:"column:user_id;primaryKey"`
	LuckyNum         int32 `gorm:"column:lucky_num"`
	CreateTime       int64 `gorm:"column:create_time"`
	NewLuckyCount    int32 `gorm:"column:new_lucky_count"`
	IsNewLuckyReward bool  `gorm:"column:is_new_lucky_reward"`
}

func (LotteryLuckyEventEntity) TableName() string {
	return "lottery_lucky_event"
}

var _ logicCommon.PlayerModelInterface = (*LotteryModel)(nil)

type LotteryModel struct {
	UserId               int64
	LotteryEntities      map[int32]*LotteryEntity // key: 卡池ID
	LotteryHistoryDetail map[int32][]*LotteryLog
	Changed              map[int32]map[string]interface{}

	LotteryLuckyEventEntity *LotteryLuckyEventEntity
	LuckyEventChanged       map[string]interface{}

	// 运行时缓存字段（不存数据库，仅在内存中维护）
	LotterySet        map[int32]map[int32]bool // 已抽到的英雄集合 itemId -> true
	LotteryQualitySet map[int32]map[int32]bool // 已抽到的品质集合 quality -> true
}

func (l *LotteryModel) GetLotterySystemId(lotterType int32) enum.FunctionIdEnum {
	if lotterType == 1 {
		return enum.FUNCTION_ID_DRAW_HERO_CARD
	}
	if lotterType == 2 {
		return enum.FUNCTION_ID_COLLECTION
	}
	return 0
}

func (l *LotteryModel) SaveModelToDB() {
	if l.Changed == nil || len(l.Changed) == 0 {
	} else {
		for id, changes := range l.Changed {
			easyDB.UpdatePlayerEntity[LotteryEntity](l.LotteryEntities[id], changes, l.UserId)
		}
		l.Changed = make(map[int32]map[string]interface{})
	}

	if l.LotteryLuckyEventEntity != nil && (l.LuckyEventChanged == nil || len(l.LuckyEventChanged) == 0) {
		return
	}
	easyDB.UpdatePlayerEntity[LotteryLuckyEventEntity](l.LotteryLuckyEventEntity, l.LuckyEventChanged, l.UserId)
	l.LuckyEventChanged = make(map[string]interface{})
}

func (l *LotteryModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {

}

func NewLotteryModel(userId int64, entity map[int32]*LotteryEntity, luckyEventEntity *LotteryLuckyEventEntity, detail map[int32][]*LotteryLog, set map[int32]map[int32]bool, qualitySet map[int32]map[int32]bool) *LotteryModel {
	return &LotteryModel{
		UserId:                  userId,
		LotteryEntities:         entity,
		LotteryHistoryDetail:    detail,
		Changed:                 make(map[int32]map[string]interface{}),
		LotterySet:              set,
		LotteryQualitySet:       qualitySet,
		LotteryLuckyEventEntity: luckyEventEntity,
		LuckyEventChanged:       make(map[string]interface{}),
	}
}

func (l *LotteryModel) UpdateAllCount(lotterId int32, allCount int32) {
	l.LotteryEntities[lotterId].AllCount = allCount
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["all_count"] = allCount
}

func (l *LotteryModel) UpdateLastBasicGuaranteeNum(lotterId int32, lastBasicGuaranteeNum int32) {
	l.LotteryEntities[lotterId].LastBasicGuaranteeNum = lastBasicGuaranteeNum
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["last_basic_guarantee_num"] = lastBasicGuaranteeNum
}

func (l *LotteryModel) UpdateLastSpecialGuaranteeNum(lotterId int32, lastSpecialGuaranteeNum int32) {
	l.LotteryEntities[lotterId].LastSpecialGuaranteeNum = lastSpecialGuaranteeNum
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["last_special_guarantee_num"] = lastSpecialGuaranteeNum
}

func (l *LotteryModel) UpdateSpecialGuaranteeCount(lotterId int32, specialGuaranteeCount int32) {
	l.LotteryEntities[lotterId].SpecialGuaranteeCount = specialGuaranteeCount
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["special_guarantee_count"] = specialGuaranteeCount
}

func (l *LotteryModel) UpdateLastChengeTime(lotterId int32, lastChengeTime int64) {
	l.LotteryEntities[lotterId].LastChangeTime = lastChengeTime
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["last_change_time"] = lastChengeTime
}

func (l *LotteryModel) UpdateLastOnceEffectiveNum(lotterId int32, lastOnceEffectiveNum int32) {
	l.LotteryEntities[lotterId].LastOnceEffectiveNum = lastOnceEffectiveNum
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["last_once_effective_num"] = lastOnceEffectiveNum
}

func (l *LotteryModel) UpdateOnceEffectiveCount(lotterId int32, onceEffectiveCount int32) {
	l.LotteryEntities[lotterId].OnceEffectiveCount = onceEffectiveCount
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["once_effective_count"] = onceEffectiveCount
}

func (l *LotteryModel) UpdateFirstDropFree(lotterId int32, num int32) {
	l.LotteryEntities[lotterId].FirstDropFree = num
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["first_drop_free"] = num
}

func (l *LotteryModel) UpdateLastFirstDropFreeTime(lotterId int32, lastFirstDropFreeTime int64) {
	l.LotteryEntities[lotterId].LastFirstDropFreeTime = lastFirstDropFreeTime
	if l.Changed[lotterId] == nil {
		l.Changed[lotterId] = make(map[string]interface{})
	}
	l.Changed[lotterId]["last_once_effective_time"] = lastFirstDropFreeTime
}

func (l *LotteryModel) GetLotteryEntityById(lotteryId int32) *LotteryEntity {
	if l.LotteryEntities[lotteryId] == nil {
		return nil
	}
	return l.LotteryEntities[lotteryId]
}

// GetSharedLotteryEntityId 获取共享保底数据的实体ID
// 如果 ActModID 存在（不为0），则所有有 ActModID 的卡池共用一个保底数据（取最小卡池ID作为共享key）
// 如果 ActModID 不存在，则使用自身卡池ID
func (l *LotteryModel) GetSharedLotteryEntityId(lotteryId int32) int32 {
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil || cfg.ActModID == 0 {
		return lotteryId
	}
	sharedId := gameConfig.GetActModSharedLotteryId()
	if sharedId == 0 {
		return lotteryId
	}
	return sharedId
}

func (l *LotteryModel) AddLotteryEntity(entity *LotteryEntity) error {
	l.LotteryEntities[entity.Id] = entity
	if err := easyDB.CreatePlayerEntity[LotteryEntity](entity); err != nil {
		return err
	}
	return nil
}

type LotteryLog struct {
	UserId  int64 `gorm:"column:user_id;"` // 用户ID
	Id      int32 `gorm:"column:id;"`      // 日志ID
	ItemId  int32 `gorm:"column:item_id;"` // 物品ID
	Count   int32 `gorm:"column:count"`    // 抽取次数
	AddTime int64 `gorm:"column:add_time"` // 添加时间
}

func (LotteryLog) TableName() string {
	return "lottery_log"
}

func (l *LotteryModel) AddLotteryLog(entity *LotteryLog) {
	l.LotteryHistoryDetail[entity.Id] = append(l.LotteryHistoryDetail[entity.Id], entity)
	if err := easyDB.CreatePlayerEntity[LotteryLog](entity); err != nil {
		logger.ErrorWithZapFields("AddLotteryLog error")
	}

	// 维护运行时缓存
	if l.LotterySet[entity.Id] == nil {
		l.LotterySet[entity.Id] = make(map[int32]bool)
	}
	l.LotterySet[entity.Id][entity.ItemId] = true
	if itemCfg := gameConfig.GetItemCfg(entity.ItemId); itemCfg != nil {
		if l.LotteryQualitySet[entity.Id] == nil {
			l.LotteryQualitySet[entity.Id] = make(map[int32]bool)
		}
		l.LotteryQualitySet[entity.Id][itemCfg.Quality] = true
	}
}

func CreateLotteryLuckyEvent(userId int64) (error, *LotteryLuckyEventEntity) {
	entity := &LotteryLuckyEventEntity{
		UserId:           userId,
		LuckyNum:         0,
		CreateTime:       0,
		NewLuckyCount:    0,
		IsNewLuckyReward: false,
	}
	return easyDB.CreatePlayerEntity[LotteryLuckyEventEntity](entity), entity
}

func LoadLotteryModel(userId int64) (*LotteryModel, error) {
	LotteryEntities := make(map[int32]*LotteryEntity)
	lotteryHistoryDetail := make(map[int32][]*LotteryLog)
	set := make(map[int32]map[int32]bool)
	qualitySet := make(map[int32]map[int32]bool)
	lotteryLuckyEventEntity := &LotteryLuckyEventEntity{}

	row, err := easyDB.GetPlayerEntitiesByWhere[LotteryEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewLotteryModel(userId, LotteryEntities, lotteryLuckyEventEntity, lotteryHistoryDetail, set, qualitySet), err
	}
	row1, err1 := easyDB.GetPlayerEntitiesByWhere[LotteryLog](map[string]interface{}{"user_id": userId})
	if err1 != nil {
		return NewLotteryModel(userId, LotteryEntities, lotteryLuckyEventEntity, lotteryHistoryDetail, set, qualitySet), err1
	}
	row2, err2 := easyDB.GetPlayerEntityByWhere[LotteryLuckyEventEntity](map[string]interface{}{"user_id": userId})
	if err2 != nil {
		if !errors.Is(err2, gorm.ErrRecordNotFound) {
			return NewLotteryModel(userId, LotteryEntities, lotteryLuckyEventEntity, lotteryHistoryDetail, set, qualitySet), err2
		} else {
			err, entity := CreateLotteryLuckyEvent(userId)
			if err != nil {
				logger.ErrorWithZapFields("CreateLotteryLuckyEvent error")
			} else {
				lotteryLuckyEventEntity = entity
			}
			return NewLotteryModel(userId, LotteryEntities, lotteryLuckyEventEntity, lotteryHistoryDetail, set, qualitySet), err2
		}
	}
	for _, v := range row {
		LotteryEntities[v.Id] = v
	}
	for _, v := range row1 {
		lotteryHistoryDetail[v.Id] = append(lotteryHistoryDetail[v.Id], v)
		// 在load遍历时直接构建缓存
		if set[v.Id] == nil {
			set[v.Id] = make(map[int32]bool)
		}
		set[v.Id][v.ItemId] = true
		if itemCfg := gameConfig.GetItemCfg(v.ItemId); itemCfg != nil {
			if qualitySet[v.Id] == nil {
				qualitySet[v.Id] = make(map[int32]bool)
			}
			qualitySet[v.Id][itemCfg.Quality] = true
		}
	}
	lotteryLuckyEventEntity = row2
	return NewLotteryModel(userId, LotteryEntities, lotteryLuckyEventEntity, lotteryHistoryDetail, set, qualitySet), nil
}

// 普通抽取
func (l *LotteryModel) BasicLottery(cfg *gameConfig.SummonPoolCfg, res *[]*pb.ItemBasicInfo, onceFlag, specialFlag bool, LotteryId int32) {
	itemIdList := make([]*gameConfig.ItemConfig, 0)
	var dropGroupId int32
	lotterDetail := l.LotteryEntities[LotteryId]
	if cfg.DrawNum > 0 && lotterDetail.AllCount-lotterDetail.LastBasicGuaranteeNum >= cfg.DrawNum {
		dropGroupId = gameConfig.WeightedRandomChoice(cfg.DropGroupId2, cfg.Weight2)
		itemIdList = gameConfig.DropGroupItems(dropGroupId, nil)
	} else {
		dropGroupId = gameConfig.WeightedRandomChoice(cfg.DropGroupId1, cfg.Weight1)
		itemIdList = gameConfig.DropGroupItems(dropGroupId, nil)
	}
	for _, itemInfo := range itemIdList {
		*res = append(*res, &pb.ItemBasicInfo{
			ItemId: itemInfo.ID,
			Count:  itemInfo.Num,
		})
	}
	// 原有逻辑：判断抽取到的是否为特殊物品
	flag := gameConfig.CheckLotterIdIsGuarantess(LotteryId, dropGroupId, onceFlag, specialFlag)
	if flag == 1 {
		l.UpdateOnceEffectiveCount(LotteryId, lotterDetail.OnceEffectiveCount+1)
		l.UpdateLastOnceEffectiveNum(LotteryId, lotterDetail.AllCount)
		l.UpdateLastSpecialGuaranteeNum(LotteryId, lotterDetail.AllCount)
		l.UpdateLastBasicGuaranteeNum(LotteryId, lotterDetail.AllCount)
		for _, v := range itemIdList {
			itemCfg := gameConfig.GetItemCfg(v.ID)
			if itemCfg != nil {
				if gameConfig.CheckItemIsGuarantess(itemCfg.ShowGroup) {
					l.AddLotteryLog(&LotteryLog{UserId: lotterDetail.UserId, Id: LotteryId, ItemId: v.ID, Count: lotterDetail.AllCount, AddTime: tool.UnixNowMilli()})
				}
			}
		}
	} else if flag == 2 {
		l.CheckAndUpSpecialGuaranteeNum(cfg, LotteryId)
		l.UpdateLastSpecialGuaranteeNum(LotteryId, lotterDetail.AllCount)
		l.UpdateLastBasicGuaranteeNum(LotteryId, lotterDetail.AllCount)
		for _, v := range itemIdList {
			itemCfg := gameConfig.GetItemCfg(v.ID)
			if itemCfg != nil {
				if gameConfig.CheckItemIsGuarantess(itemCfg.ShowGroup) {
					l.AddLotteryLog(&LotteryLog{UserId: lotterDetail.UserId, Id: LotteryId, ItemId: v.ID, Count: lotterDetail.AllCount, AddTime: tool.UnixNowMilli()})
				}
			}
		}
	} else if flag == 3 {
		l.UpdateLastBasicGuaranteeNum(LotteryId, lotterDetail.AllCount)
		// 判断是否歪了（抽到的是否在LuckyGuarantees奖池里）
		isDirty := true
		for _, v := range itemIdList {
			if !l.IsLotteryDirty(cfg, v.ID) {
				isDirty = false
				break
			}
		}
		if isDirty {
			l.UpdateIsDirty(LotteryId, true)
		} else {
			l.UpdateIsDirty(LotteryId, false)
		}
		for _, v := range itemIdList {
			itemCfg := gameConfig.GetItemCfg(v.ID)
			if itemCfg != nil {
				if gameConfig.CheckItemIsGuarantess(itemCfg.ShowGroup) {
					l.AddLotteryLog(&LotteryLog{UserId: lotterDetail.UserId, Id: LotteryId, ItemId: v.ID, Count: lotterDetail.AllCount, AddTime: tool.UnixNowMilli()})
				}
			}
		}
	}
	l.UpdateLastChengeTime(LotteryId, tool.UnixNowMilli())
}

// 普通保底
func (l *LotteryModel) BasicLotteryGuarantee(cfg *gameConfig.SummonPoolCfg, res *[]*pb.ItemBasicInfo, onceFlag, specialFlag bool, LotteryId int32) {
	dropGroupId := gameConfig.WeightedRandomChoice(cfg.Guarantees.DropGroupIdList, cfg.GuaranteesWeight)
	itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)
	lotterDetail := l.LotteryEntities[LotteryId]
	l.UpdateLastBasicGuaranteeNum(LotteryId, lotterDetail.AllCount)
	for _, v := range itemIdList {
		itemCfg := gameConfig.GetItemCfg(v.ID)
		if itemCfg != nil {
			if gameConfig.CheckItemIsGuarantess(itemCfg.ShowGroup) {
				l.AddLotteryLog(&LotteryLog{UserId: lotterDetail.UserId, Id: LotteryId, ItemId: v.ID, Count: lotterDetail.AllCount, AddTime: tool.UnixNowMilli()})
			}
		}
	}
	l.UpdateLastChengeTime(LotteryId, tool.UnixNowMilli())
	for _, itemInfo := range itemIdList {
		*res = append(*res, &pb.ItemBasicInfo{
			ItemId: itemInfo.ID,
			Count:  itemInfo.Num,
		})
	}
	// 奖励式保底机制：判断保底结果是否歪了
	if cfg.LuckyGuarantees != nil && len(cfg.LuckyGuarantees) > 0 {
		isDirty := true
		for _, v := range itemIdList {
			if !l.IsLotteryDirty(cfg, v.ID) {
				isDirty = false
				break
			}
		}
		if isDirty {
			l.UpdateIsDirty(LotteryId, true)
		} else {
			l.UpdateIsDirty(LotteryId, false)
		}
	}
	flag := gameConfig.CheckLotterIdIsGuarantess(LotteryId, dropGroupId, onceFlag, specialFlag)
	if flag == 1 {
		l.UpdateOnceEffectiveCount(LotteryId, lotterDetail.OnceEffectiveCount+1)
		l.UpdateLastOnceEffectiveNum(LotteryId, lotterDetail.AllCount)
		l.UpdateLastSpecialGuaranteeNum(LotteryId, lotterDetail.AllCount)
	} else if flag == 2 {
		l.CheckAndUpSpecialGuaranteeNum(cfg, LotteryId)
		l.UpdateLastSpecialGuaranteeNum(LotteryId, lotterDetail.AllCount)
	}
}

func (l *LotteryModel) CheckAndUpSpecialGuaranteeNum(cfg *gameConfig.SummonPoolCfg, LotteryId int32) {
	lotterDetail := l.LotteryEntities[LotteryId]
	if cfg.Guarantees2Type == 1 {
		l.UpdateSpecialGuaranteeCount(LotteryId, min(lotterDetail.SpecialGuaranteeCount+1, int32(len(cfg.Guarantees2)-1)))
	} else {
		if lotterDetail.SpecialGuaranteeCount+1 >= int32(len(cfg.Guarantees2)) {
			l.UpdateSpecialGuaranteeCount(LotteryId, 0)
		} else {
			l.UpdateSpecialGuaranteeCount(LotteryId, lotterDetail.SpecialGuaranteeCount+1)
		}
	}
}

func (l *LotteryModel) UpdateNewLuckyCount(num int32) {
	l.LotteryLuckyEventEntity.NewLuckyCount = num
	l.LuckyEventChanged["new_lucky_count"] = num
}

func (l *LotteryModel) UpdateIsNewLuckyReward() {
	l.LotteryLuckyEventEntity.IsNewLuckyReward = true
	l.LuckyEventChanged["is_new_lucky_reward"] = true
}

func (l *LotteryModel) UpdateIsDirty(lotteryId int32, isDirty bool) {
	l.LotteryEntities[lotteryId].IsDirty = isDirty
	if l.Changed[lotteryId] == nil {
		l.Changed[lotteryId] = make(map[string]interface{})
	}
	l.Changed[lotteryId]["is_dirty"] = isDirty
}

func (l *LotteryModel) RewardNewLucky(itemId int32) bool {
	newLuckyCfg := gameConfig.GetConstantCfg(gameConfig.CONSTANT_beginnerBenefits)
	if newLuckyCfg == nil || len(newLuckyCfg.Value) < 0 {
		return false
	}
	for _, v := range newLuckyCfg.Value {
		items := gameConfig.Drop(v)
		for _, v := range items {
			if v.ID == itemId {
				l.UpdateIsNewLuckyReward()
				return true
			}
		}
	}
	return false
}

func (l *LotteryModel) GetLotteryLuckyEvent() *LotteryLuckyEventEntity {
	return l.LotteryLuckyEventEntity
}

func (l *LotteryModel) CreateLotteryLuckyEvent() error {
	entity := &LotteryLuckyEventEntity{UserId: l.UserId}
	cfgWeight := gameConfig.GetConstantCfg(gameConfig.CONSTANT_limitedGachaLuckyEventWeight)
	if cfgWeight == nil || len(cfgWeight.Value) < 0 {
		return errors.New("constant cfg error")
	}
	cfgValue := gameConfig.GetConstantCfg(gameConfig.CONSTANT_limitedGachaLuckyEventDiscountRate)
	if cfgValue == nil || len(cfgValue.Value) < 0 {
		return errors.New("constant cfg error")
	}
	if len(cfgWeight.Value) != len(cfgValue.Value) {
		return errors.New("constant cfg error")
	}
	luckyEventNum := gameConfig.WeightedRandomChoice(cfgValue.Value, cfgWeight.Value)
	entity.LuckyNum = luckyEventNum
	entity.CreateTime = tool.UnixNowMilli()
	l.LotteryLuckyEventEntity = entity
	l.LuckyEventChanged["LuckyNum"] = entity.LuckyNum
	l.LuckyEventChanged["CreateTime"] = entity.CreateTime
	return nil
}

func (l *LotteryModel) IsLotteryDirty(cfg *gameConfig.SummonPoolCfg, itemId int32) bool {
	if cfg == nil {
		return true
	}
	dropIdCfgs := cfg.LuckyGuarantees
	for _, v := range dropIdCfgs {
		for _, value := range v.DropGroupIdList {
			items := gameConfig.DropGroupItems(value, nil)
			for _, item := range items {
				if item.ID == itemId {
					return false
				}
			}
		}
	}
	return true
}

// 奖励式保底抽取（强制从奖励式保底池中抽取）
func (l *LotteryModel) LuckyGuaranteeLottery(cfg *gameConfig.SummonPoolCfg, res *[]*pb.ItemBasicInfo, LotteryId int32) {
	lotterDetail := l.LotteryEntities[LotteryId]
	// 从奖励式保底池中随机抽取
	dropGroupId := gameConfig.WeightedRandomChoice(cfg.LuckyGuarantees[0].DropGroupIdList, cfg.LuckyWeight[0])
	itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)

	for _, v := range itemIdList {
		itemCfg := gameConfig.GetItemCfg(v.ID)
		if itemCfg != nil {
			if gameConfig.CheckItemIsGuarantess(itemCfg.ShowGroup) {
				l.AddLotteryLog(&LotteryLog{UserId: lotterDetail.UserId, Id: LotteryId, ItemId: v.ID, Count: lotterDetail.AllCount, AddTime: tool.UnixNowMilli()})
			}
		}
	}

	for _, itemInfo := range itemIdList {
		*res = append(*res, &pb.ItemBasicInfo{
			ItemId: itemInfo.ID,
			Count:  itemInfo.Num,
		})
	}

	// 强制保底成功，重置保底计数，标记未歪
	l.UpdateLastBasicGuaranteeNum(LotteryId, lotterDetail.AllCount)
	l.UpdateIsDirty(LotteryId, false)
	l.UpdateLastChengeTime(LotteryId, tool.UnixNowMilli())
}
