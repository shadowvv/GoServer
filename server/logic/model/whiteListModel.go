package model

type WhiteListEntity struct {
	Account     string `gorm:"column:account;primaryKey"`
	startTime   int64  `gorm:"column:start_time"`
	endTime     int64  `gorm:"column:end_time"`
	description string `gorm:"column:description"`
}

func (s *WhiteListEntity) TableName() string {
	return "white_list"
}
