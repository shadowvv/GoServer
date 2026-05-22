package model

type AccountSimpleInfoEntity struct {
	Account           string `gorm:"column:account;primaryKey"`
	LastLoginServerId int32  `gorm:"column:last_login_serverId"`
}

func (u *AccountSimpleInfoEntity) TableName() string {
	return "account_simple_info"
}
