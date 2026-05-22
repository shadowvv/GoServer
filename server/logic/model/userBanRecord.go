package model

type UserBanRecordEntity struct {
	Account   string `gorm:"column:account;primaryKey"`
	ServerId  int32  `gorm:"column:server_id;primaryKey"`
	Reason    int32  `gorm:"column:reason"`
	StartTime int64  `gorm:"column:start_time"`
	EndTime   int64  `gorm:"column:end_time"`
}

func (s *UserBanRecordEntity) TableName() string {
	return "user_ban_record"
}
