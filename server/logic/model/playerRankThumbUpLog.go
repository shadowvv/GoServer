package model

type PlayerRankThumbUpLog struct {
	Id           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	RankId       string `gorm:"column:rank_id"`
	FromPlayerId int64  `gorm:"column:from_player_id"`
	ThumbUpTime  int64  `gorm:"column:thumb_up_time"`
}

func (*PlayerRankThumbUpLog) TableName() string {
	return "player_rank_thumb_up_log"
}

type PlayerRankClaimChestsLog struct {
	Id           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	RankId       string `gorm:"column:rank_id"`
	FromPlayerId int64  `gorm:"column:from_player_id"`
	ClaimTime    int64  `gorm:"column:claim_time"`
}

func (*PlayerRankClaimChestsLog) TableName() string {
	return "player_rank_claim_chest_log"
}
