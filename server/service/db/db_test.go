package db

import "gorm.io/gorm"

type PlayerRepository struct{}

func (p *PlayerRepository) GetPlayerByUID(uid uint64) (*Player, error) {
	var player Player
	if err := DB.Where("uid = ?", uid).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}

func (p *PlayerRepository) SavePlayer(player *Player) error {
	return DB.Save(player).Error
}

type Player struct {
	gorm.Model
	UID     uint64 `gorm:"uniqueIndex"`
	Name    string `gorm:"size:32"`
	Level   int
	Exp     int64
	SceneID int
	PosX    float32
	PosY    float32
	PosZ    float32
}
