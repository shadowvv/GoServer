package model

type AllianceEntity struct {
	AllianceId          int64  `gorm:"column:alliance_id;primaryKey"`
	ServerId            int32  `gorm:"column:server_id;index:idx_server_name,priority:1"`
	Name                string `gorm:"column:name;size:64;index:idx_server_name,priority:2"`
	Announce            string `gorm:"column:announce;size:512"`
	BadgeId             int32  `gorm:"column:badge_id"`
	Notice              string `gorm:"column:notice;size:512"`
	Level               int32  `gorm:"column:level"`
	Exp                 int32  `gorm:"column:exp"`
	ApplyType           int32  `gorm:"column:apply_type"`
	PowerApplyCondition int64  `gorm:"column:power_apply_condition"`
	CityLevelCondition  int32  `gorm:"column:city_level_condition"`
	CreateTime          int64  `gorm:"column:create_time"`

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
