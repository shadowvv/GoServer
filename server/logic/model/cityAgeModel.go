package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const CityAgeGroupCount = 5

var _ logicCommon.PlayerModelInterface = (*CityAgeModel)(nil)
var _ logicCommon.HeroAttrInterface = (*CityAgeModel)(nil)

type CityAgeEntity struct {
	UserId            int64               `gorm:"column:user_id;primaryKey"`
	AgeId             int32               `gorm:"column:age_id"`
	GroupRewardStatus tool.JSONInt32Slice `gorm:"column:group_reward_status;type:json"`
	DailyRewardTime   int64               `gorm:"column:daily_reward_time"`
	CreateTime        int64               `gorm:"column:create_time"`
}

func (e *CityAgeEntity) TableName() string {
	return "city_age"
}

type CityAgeFirstReachEntity struct {
	ServerId  int32 `gorm:"column:server_id;primaryKey"`
	CityAgeId int32 `gorm:"column:city_age_id;primaryKey"`
	UserId    int64 `gorm:"column:user_id"`
	ReachTime int64 `gorm:"column:reach_time"`
}

func (e *CityAgeFirstReachEntity) TableName() string {
	return "city_age_first_reach"
}

type CityAgeModel struct {
	UserId   int64
	Entity   *CityAgeEntity
	Changed  map[string]interface{}
	initDone bool
}

func NewCityAgeModel(userId int64, entity *CityAgeEntity) *CityAgeModel {
	if entity == nil {
		entity = &CityAgeEntity{
			UserId:            userId,
			GroupRewardStatus: defaultCityAgeGroupRewardStatus(),
			CreateTime:        tool.UnixNowMilli(),
		}
	}
	entity.GroupRewardStatus = normalizeCityAgeGroupRewardStatus(entity.GroupRewardStatus)
	return &CityAgeModel{
		UserId:  userId,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func LoadCityAgeModel(userId int64) (*CityAgeModel, error) {
	entity, err := easyDB.GetPlayerEntityByWhere[CityAgeEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewCityAgeModel(userId, nil), nil
		}
		return NewCityAgeModel(userId, nil), err
	}
	return NewCityAgeModel(userId, entity), nil
}

func TryCreateCityAgeFirstReach(entity *CityAgeFirstReachEntity) (bool, error) {
	if entity == nil {
		return false, errors.New("city age first reach entity is nil")
	}
	result := easyDB.GetPlayerDB().Clauses(clause.OnConflict{DoNothing: true}).Create(entity)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func LoadCityAgeFirstReachList(serverId int32) ([]*CityAgeFirstReachEntity, error) {
	var entities []*CityAgeFirstReachEntity
	err := easyDB.GetPlayerDB().Where("server_id = ?", serverId).Order("city_age_id ASC").Find(&entities).Error
	return entities, err
}

func defaultCityAgeGroupRewardStatus() tool.JSONInt32Slice {
	return tool.JSONInt32Slice{0, 0, 0, 0, 0}
}

func normalizeCityAgeGroupRewardStatus(status tool.JSONInt32Slice) tool.JSONInt32Slice {
	if len(status) >= CityAgeGroupCount {
		return status[:CityAgeGroupCount]
	}
	res := make(tool.JSONInt32Slice, CityAgeGroupCount)
	copy(res, status)
	return res
}

func (m *CityAgeModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
}

func (m *CityAgeModel) GetHeroAttr(heroId int64, attrId int32) int64 {
	cfg := m.GetCurrentCfg()
	if cfg == nil {
		return 0
	}
	return cfg.AttrAge[attrId]
}

func (m *CityAgeModel) GetBuffAttr(heroId int64, attrId int32) int64 {
	return 0
}

func (m *CityAgeModel) GetChangedHeroOwnIDs() ([]int64, bool) {
	if _, ok := m.Changed["age_id"]; !ok {
		return []int64{}, false
	}
	return []int64{}, true
}

func (m *CityAgeModel) SaveModelToDB() {
	if len(m.Changed) == 0 || m.Entity == nil || m.Entity.UserId == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(m.Entity, m.Changed, m.UserId)
	m.Changed = make(map[string]interface{})
}

func (m *CityAgeModel) markChanged(field string, value interface{}) {
	m.Changed[field] = value
}

func (m *CityAgeModel) EnsureInit() error {
	if m.initDone {
		return nil
	}
	m.initDone = true

	if m.Entity == nil {
		m.Entity = &CityAgeEntity{
			UserId:            m.UserId,
			GroupRewardStatus: defaultCityAgeGroupRewardStatus(),
			CreateTime:        tool.UnixNowMilli(),
		}
	}
	m.Entity.GroupRewardStatus = normalizeCityAgeGroupRewardStatus(m.Entity.GroupRewardStatus)
	if m.Entity.AgeId == 0 {
		firstCfg := gameConfig.GetFirstCityAgeUpCfg()
		if firstCfg == nil {
			return errors.New("city age cfg not found")
		}
		m.Entity = &CityAgeEntity{
			UserId:            m.UserId,
			AgeId:             firstCfg.Id,
			GroupRewardStatus: defaultCityAgeGroupRewardStatus(),
			DailyRewardTime:   0,
			CreateTime:        tool.UnixNowMilli(),
		}
		if err := easyDB.CreatePlayerEntity(m.Entity); err != nil {
			return err
		}
	}
	return nil
}

func (m *CityAgeModel) GetCurrentCfg() *gameConfig.CityAgeUpCfg {
	if m.Entity == nil {
		return nil
	}
	return gameConfig.GetCityAgeUpCfg(m.Entity.AgeId)
}

func (m *CityAgeModel) GetGroupRewardStatus(groupIndex int32) int32 {
	if groupIndex < 1 || groupIndex > CityAgeGroupCount {
		return 0
	}
	m.Entity.GroupRewardStatus = normalizeCityAgeGroupRewardStatus(m.Entity.GroupRewardStatus)
	return m.Entity.GroupRewardStatus[groupIndex-1]
}

func (m *CityAgeModel) GetGroupRewardStatuses() tool.JSONInt32Slice {
	m.Entity.GroupRewardStatus = normalizeCityAgeGroupRewardStatus(m.Entity.GroupRewardStatus)
	res := make(tool.JSONInt32Slice, len(m.Entity.GroupRewardStatus))
	copy(res, m.Entity.GroupRewardStatus)
	return res
}

func (m *CityAgeModel) SetGroupRewardStatus(status tool.JSONInt32Slice) {
	m.Entity.GroupRewardStatus = normalizeCityAgeGroupRewardStatus(status)
	m.markChanged("group_reward_status", m.Entity.GroupRewardStatus)
}

func (m *CityAgeModel) ResetGroupRewardStatus() {
	m.SetGroupRewardStatus(defaultCityAgeGroupRewardStatus())
}

func (m *CityAgeModel) SetAgeId(ageId int32) {
	m.Entity.AgeId = ageId
	m.markChanged("age_id", ageId)
}

func (m *CityAgeModel) SetDailyRewardTime(ts int64) {
	m.Entity.DailyRewardTime = ts
	m.markChanged("daily_reward_time", ts)
}
