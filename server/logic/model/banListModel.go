package model

type BanListEntity struct {
	Account   string `gorm:"column:account;primaryKey"`
	ServerId  int32  `gorm:"column:server_id"`
	Reason    int32  `gorm:"column:reason"`
	StartTime int64  `gorm:"column:start_time"`
	EndTime   int64  `gorm:"column:end_time"`
}

func (s *BanListEntity) TableName() string {
	return "ban_list"
}
