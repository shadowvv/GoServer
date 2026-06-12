package model

import (
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
	"gorm.io/gorm"
)

type AllianceEntity struct {
	AllianceId          int64  `gorm:"column:alliance_id;primaryKey"`
	ServerId            int32  `gorm:"column:server_id;index:idx_server_name,priority:1"`
	Name                string `gorm:"column:name;size:64;index:idx_server_name,priority:2"`
	Announce            string `gorm:"column:announce;size:512"`
	BadgeId             int32  `gorm:"column:badge_id"`
	Notice              string `gorm:"column:notice;size:512"`
	Level               int32  `gorm:"column:level"`
	ApplyType           int32  `gorm:"column:apply_type"`
	PowerApplyCondition int64  `gorm:"column:power_apply_condition"`
	CityLevelCondition  int32  `gorm:"column:city_level_condition"`
	CreateTime          int64  `gorm:"column:create_time"`
	LastTickTime        int64  `gorm:"column:last_tick_time"`

	AllianceTotalPower int64  `gorm:"-"`
	MemberNum          int32  `gorm:"-"`
	LeaderName         string `gorm:"-"`
}

func (a *AllianceEntity) TableName() string {
	return "alliance"
}

type AllianceMemberEntity struct {
	AllianceId int64 `gorm:"column:alliance_id;index:idx_alliance_id"`
	UserId     int64 `gorm:"column:user_id;primaryKey"`
	Role       int32 `gorm:"column:role"`
	JoinTime   int64 `gorm:"column:join_time"`
}

func (a *AllianceMemberEntity) TableName() string {
	return "alliance_member"
}

type AllianceMemberSignInEntity struct {
	UserId             int64 `gorm:"column:user_id;primaryKey"`
	BasicCount         int32 `gorm:"column:basic_count"`
	AdvertisementCount int32 `gorm:"column:advance_count"`
	LastChangeTime     int64 `gorm:"column:last_change_time"`
}

func (a *AllianceMemberSignInEntity) TableName() string {
	return "alliance_member_sign_in"
}

type SignInModel struct {
	Entity  *AllianceMemberSignInEntity
	UserId  int64
	Changed map[string]interface{}
}

func (m *SignInModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if !tool.IsSameDayByMilli(m.Entity.LastChangeTime, currentTime) {
		m.Refreshable()
	}
}

var _ logicCommon.PlayerModelInterface = (*SignInModel)(nil)

func (m *SignInModel) SaveModelToDB() {
	if len(m.Changed) > 0 {
		easyDB.UpdatePlayerEntity(m.Entity, m.Changed, m.UserId)
	}
	m.Changed = make(map[string]interface{})
}

func (m *SignInModel) Refreshable() {
	m.UpdateLastChangeTime(tool.UnixNowMilli())
	m.UpdateBasicCount(0)
	m.UpdateAdvertisementCount(0)
}

func (m *SignInModel) UpdateBasicCount(count int32) {
	m.Changed["basic_count"] = count
	m.Entity.BasicCount = count
}

func (m *SignInModel) UpdateAdvertisementCount(count int32) {
	m.Changed["advance_count"] = count
	m.Entity.AdvertisementCount = count
}

func (m *SignInModel) UpdateLastChangeTime(time int64) {
	m.Changed["last_change_time"] = time
	m.Entity.LastChangeTime = time
}

func NewSignInModel(userId int64, entity *AllianceMemberSignInEntity) *SignInModel {
	return &SignInModel{
		Entity:  entity,
		UserId:  userId,
		Changed: make(map[string]interface{}),
	}
}

func CreateSignInEntity(entity *AllianceMemberSignInEntity) error {
	return easyDB.CreatePlayerEntity(entity)
}
func LoadSignInModel(userId int64) (*SignInModel, error) {
	entity := &AllianceMemberSignInEntity{}
	row, err := easyDB.GetPlayerEntityByWhere[AllianceMemberSignInEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity.AdvertisementCount = 0
			entity.BasicCount = 0
			entity.LastChangeTime = tool.UnixNowMilli()
			entity.UserId = userId
			return NewSignInModel(userId, entity), CreateSignInEntity(entity)
		} else {
			return nil, err
		}
	}
	entity = row
	return NewSignInModel(userId, entity), nil
}

// 联盟仓库
type AllianceWarehouseEntity struct {
	AllianceId int64 `gorm:"column:alliance_id;primaryKey"`
	ItemId     int32 `gorm:"column:item_id;primaryKey"`
	Count      int64 `gorm:"column:count"`
}

func (a *AllianceWarehouseEntity) TableName() string {
	return "alliance_warehouse"
}
