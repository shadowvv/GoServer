package model

import (
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
	"github.com/google/uuid"
)

// AdChestEntity 广告宝箱实例
type AdChestEntity struct {
	UserId     int64  `gorm:"column:user_id"`
	UniqueId   string `gorm:"column:unique_id;primaryKey"`
	ItemId     int32  `gorm:"column:item_id"`
	CfgIndex   int32  `gorm:"column:cfg_index"`   // targetId，指向 limitedAdChest 配置
	CreateTime int64  `gorm:"column:create_time"` // 创建时间戳(毫秒)
}

func (a *AdChestEntity) TableName() string {
	return "player_ad_chest"
}

// AdChestDailyEntity 广告宝箱每日开启计数
type AdChestDailyEntity struct {
	UserId    int64 `gorm:"column:user_id;primaryKey"`
	OpenDate  int32 `gorm:"column:open_date;primaryKey"` // YYYYMMDD
	OpenCount int32 `gorm:"column:open_count"`
}

func (a *AdChestDailyEntity) TableName() string {
	return "player_ad_chest_daily"
}

var _ logicCommon.PlayerModelInterface = (*AdChestModel)(nil)

type AdChestModel struct {
	UserId      int64
	Chests      map[string]*AdChestEntity // key: uniqueId
	DailyEntity *AdChestDailyEntity       // 当日开启计数
	Changed     map[string]interface{}
}

func NewAdChestModel(userId int64) *AdChestModel {
	return &AdChestModel{
		UserId:      userId,
		Chests:      make(map[string]*AdChestEntity),
		DailyEntity: nil,
		Changed:     make(map[string]interface{}),
	}
}

// LoadAdChestModel 加载广告宝箱模型
func LoadAdChestModel(userId int64) (*AdChestModel, error) {
	chestRows, err := easyDB.GetPlayerEntitiesByWhere[AdChestEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}

	today := tool.GetTodayDataIntByTimeStamp(tool.UnixNowMilli())
	dailyEntity, err := easyDB.GetPlayerEntityByWhere[AdChestDailyEntity](map[string]interface{}{
		"user_id":   userId,
		"open_date": today,
	})
	var daily *AdChestDailyEntity
	if err == nil && dailyEntity != nil {
		daily = dailyEntity
	}

	m := NewAdChestModel(userId)
	for _, e := range chestRows {
		if e != nil {
			m.Chests[e.UniqueId] = e
		}
	}
	m.DailyEntity = daily
	return m, nil
}

// CreateAdChestModel 创建空模型（新玩家）
func CreateAdChestModel(userId int64) *AdChestModel {
	return NewAdChestModel(userId)
}

func (a *AdChestModel) SaveModelToDB() {
	// 每日计数在 IncrementTodayOpenCount 中直接持久化
}

func (a *AdChestModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 每日0点刷新由拉取/开启时检查 open_date 实现，无需心跳
}

// AddChest 添加广告宝箱，返回唯一ID
func (a *AdChestModel) AddChest(itemId int32, cfgIndex int32, createTime int64) string {
	uniqueId := uuid.New().String()
	entity := &AdChestEntity{
		UserId:     a.UserId,
		UniqueId:   uniqueId,
		ItemId:     itemId,
		CfgIndex:   cfgIndex,
		CreateTime: createTime,
	}
	a.Chests[uniqueId] = entity
	_ = easyDB.CreatePlayerEntity(entity)
	return uniqueId
}

// GetChest 获取宝箱
func (a *AdChestModel) GetChest(uniqueId string) *AdChestEntity {
	return a.Chests[uniqueId]
}

// RemoveChest 移除宝箱（开启后消耗）
func (a *AdChestModel) RemoveChest(uniqueId string) {
	delete(a.Chests, uniqueId)
	_ = easyDB.DeletePlayerEntityByWhere[AdChestEntity](map[string]interface{}{
		"user_id":   a.UserId,
		"unique_id": uniqueId,
	}, a.UserId)
}

// GetTodayOpenCount 获取今日已开启次数
func (a *AdChestModel) GetTodayOpenCount(currentTime int64) int32 {
	today := tool.GetTodayDataIntByTimeStamp(currentTime)
	if a.DailyEntity == nil || a.DailyEntity.OpenDate != today {
		return 0
	}
	return a.DailyEntity.OpenCount
}

// IncrementTodayOpenCount 增加今日开启次数
func (a *AdChestModel) IncrementTodayOpenCount(currentTime int64) {
	today := tool.GetTodayDataIntByTimeStamp(currentTime)
	if a.DailyEntity == nil || a.DailyEntity.OpenDate != today {
		a.DailyEntity = &AdChestDailyEntity{
			UserId:    a.UserId,
			OpenDate:  today,
			OpenCount: 1,
		}
		_ = easyDB.CreatePlayerEntity(a.DailyEntity)
	} else {
		a.DailyEntity.OpenCount++
		easyDB.UpdatePlayerEntity(a.DailyEntity, map[string]interface{}{"open_count": a.DailyEntity.OpenCount}, a.UserId)
	}
}

// GetAllChests 获取所有宝箱列表（用于拉取）
func (a *AdChestModel) GetAllChests() []*AdChestEntity {
	list := make([]*AdChestEntity, 0, len(a.Chests))
	for _, e := range a.Chests {
		list = append(list, e)
	}
	return list
}

// GenerateAdChestUniqueId 生成唯一ID
func GenerateAdChestUniqueId() string {
	return uuid.New().String()
}
