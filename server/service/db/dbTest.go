package db

type PlayerRepository struct{}

func (p *PlayerRepository) GetPlayerByUID(uid uint64) (*Player, error) {
	var player Player
	if err := DB.Where("userId = ?", uid).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}

func (p *PlayerRepository) SavePlayer(player *Player) error {
	return DB.Create(player).Error
}

type Player struct {
	Account        string `gorm:"uniqueIndex"`
	UserId         int64  `gorm:"size:32"`
	LastLoginTime  int
	LastLogoutTime int64
}

func (p *Player) TableName() string {
	return "account"
}
