package model

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type UserEntity struct {
	Account         string `gorm:"column:account;primaryKey;size:64"`
	UserId          int64  `gorm:"column:user_id"`
	ServerId        int32  `gorm:"column:server_id;primaryKey"`
	Nickname        string `gorm:"column:nickname;size:64"`
	HeadId          int32  `gorm:"column:head_id"`
	HeadFrameId     int32  `gorm:"column:head_frame_id"`
	ChannelId       int32  `gorm:"column:channel_id"`
	TitleId         int32  `gorm:"column:title_id"`
	Level           int32  `gorm:"column:level"`
	Vip             int32  `gorm:"column:vip"`
	ChargeCount     int32  `gorm:"column:charge_count"`
	LastChargeTime  int64  `gorm:"column:last_charge_time"`
	LastLoginTime   int64  `gorm:"column:last_login_time"`
	LastOfflineTime int64  `gorm:"column:last_offline_time"`
	RegisterTime    int64  `gorm:"column:register_time"`

	AllianceName string `gorm:"-"`
}

func (u *UserEntity) TableName() string {
	return "account"
}

type UserModel struct {
	Player  *PlayerModel
	Entity  *UserEntity
	Changed map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*UserModel)(nil)

func NewUserModel(entity *UserEntity, player *PlayerModel) *UserModel {
	return &UserModel{
		Player:  player,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func (u *UserModel) UpdateLevel(level int32) {
	u.Entity.Level = level
	u.Changed["level"] = level
	if u.Player != nil {
		u.Player.UpdatePlayerBasicInfoToRedis()
	}
}

func (u *UserModel) UpdateNickname(nickname string) {
	u.Entity.Nickname = nickname
	u.Changed["nickname"] = nickname
	if u.Player != nil {
		u.Player.UpdatePlayerBasicInfoToRedis()
	}
}

func (u *UserModel) UpdateHeadId(headId int32) {
	u.Entity.HeadId = headId
	u.Changed["head_id"] = headId
}

func (u *UserModel) UpdateHeadFrameId(headFrameId int32) {
	u.Entity.HeadFrameId = headFrameId
	u.Changed["head_frame_id"] = headFrameId
}

func (u *UserModel) UpdateTitleId(titleId int32) {
	u.Entity.TitleId = titleId
	u.Changed["title_id"] = titleId
}

func (u *UserModel) UpdateVip(vip int32) {
	u.Entity.Vip = vip
	u.Changed["vip"] = vip
}

func (u *UserModel) UpdateChargeCount(ChargeCount int32) {
	u.Entity.ChargeCount = ChargeCount
	u.Changed["charge_count"] = ChargeCount
}

func (u *UserModel) UpdateLastChargeTime(time int64) {
	u.Entity.LastChargeTime = time
	u.Changed["last_charge_time"] = time
}

func (u *UserModel) UpdateLastLoginTime(time int64) {
	u.Entity.LastLoginTime = time
	u.Changed["last_login_time"] = time
}

func (u *UserModel) UpdateLastOfflineTime(time int64) {
	u.Entity.LastOfflineTime = time
	u.Changed["last_offline_time"] = time
	if u.Player != nil {
		u.Player.UpdatePlayerBasicInfoToRedis()
	}
}

func (u *UserModel) GetAccount() string {
	return u.Entity.Account
}

func (u *UserModel) GetUserId() int64 {
	return u.Entity.UserId
}

func (u *UserModel) GetServerId() int32 {
	return u.Entity.ServerId
}

func (u *UserModel) GetNickname() string {
	return u.Entity.Nickname
}

func (u *UserModel) GetHeadId() int32 {
	return u.Entity.HeadId
}

func (u *UserModel) GetHeadFrameId() int32 {
	return u.Entity.HeadFrameId
}

func (u *UserModel) GetTitleId() int32 {
	return u.Entity.TitleId
}

func (u *UserModel) GetChannelId() int32 {
	return u.Entity.ChannelId
}

func (u *UserModel) GetVip() int32 {
	return u.Entity.Vip
}

func (u *UserModel) GetChargeCount() int32 {
	return u.Entity.ChargeCount
}

func (u *UserModel) GetLastChargeTime() int64 {
	return u.Entity.LastChargeTime
}

func (u *UserModel) GetLastLoginTime() int64 {
	return u.Entity.LastLoginTime
}

func (u *UserModel) GetLastOfflineTime() int64 {
	return u.Entity.LastOfflineTime
}

func (u *UserModel) GetRegisterTime() int64 {
	return u.Entity.RegisterTime
}

func (u *UserModel) SaveModelToDB() {
	if u.Changed == nil || len(u.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(u.Entity, u.Changed, u.Entity.UserId)
	u.Changed = make(map[string]interface{})
}

func (u *UserModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	//nothing to do
}

func CreateUserModel(account string, userId int64, serverId int32, player *PlayerModel) (*UserModel, error) {
	entity := &UserEntity{
		Account:         account,
		UserId:          userId,
		Nickname:        gameConfig.RandomNickname(),
		LastLoginTime:   0, // 初始登录时间为 0
		LastOfflineTime: 0,
		RegisterTime:    tool.UnixNowMilli(),
		ServerId:        serverId,
	}
	err := easyDB.CreatePlayerEntity[UserEntity](entity)
	if err != nil {
		return nil, err
	}
	userModel := NewUserModel(entity, player)
	return userModel, nil
}
