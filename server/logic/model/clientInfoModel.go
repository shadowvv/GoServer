package model

type GameClientVersionEntity struct {
	Version      string `gorm:"column:version;primaryKey"`
	HotfixConfig string `gorm:"column:hotfix_config"`
	Examine      int32  `gorm:"column:examine"`
}

func (s *GameClientVersionEntity) TableName() string {
	return "client_version"
}
