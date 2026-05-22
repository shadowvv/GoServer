package model

type RankBoardInfoEntity struct {
	Rank         int32 `gorm:"column:rank;primaryKey;comment:排名"`
	Id           int64 `gorm:"column:id;primaryKey;comment:id"`
	Score        int64 `gorm:"column:score;comment:积分"`
	ThumbUpCount int32 `gorm:"column:thumb_up_count;comment:点赞数"`
	EnterTime    int64 `gorm:"column:enter_time;comment:进榜时间"`
}

type RankSettleTaskEntity struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	RankId     string `gorm:"column:rank_id;type:varchar(100);not null;uniqueIndex:uk_rank_period,priority:1"`
	SettleType int8   `gorm:"column:settle_type;type:tinyint;not null;uniqueIndex:uk_rank_period,priority:2"`
	Version    string `gorm:"column:version;type:varchar(50);not null;uniqueIndex:uk_rank_period,priority:3"`
	SettleTime int64  `gorm:"column:settle_time;not null;comment:计划结算时间"`
	Status     int8   `gorm:"column:status;type:tinyint;not null;default:0;comment:0=pending,1=running,2=snapshot_done,3=reward_done,4=failed"`
	CreatedAt  int64  `gorm:"column:created_at;not null"`
	UpdatedAt  int64  `gorm:"column:updated_at;not null"`
}

func (RankSettleTaskEntity) TableName() string {
	return "rank_settle_task"
}

type RankSnapshotInfoEntity struct {
	Id           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	TaskId       int64  `gorm:"column:task_id;not null;uniqueIndex:uk_task_user,priority:1;index:idx_task_rank,priority:1"`
	RankId       string `gorm:"column:rank_id;type:varchar(100);not null"`
	SettleType   int8   `gorm:"column:settle_type;type:tinyint;not null"`
	Version      string `gorm:"column:version;type:varchar(50);not null"`
	SourceId     int64  `gorm:"column:source_id;not null;uniqueIndex:uk_task_user,priority:2"`
	Rank         int32  `gorm:"column:rank;not null;index:idx_task_rank,priority:2"`
	Score        int64  `gorm:"column:score;not null"`
	ThumbUpCount int64  `gorm:"column:thumb_up_count;not null;default:0"`
	EnterTime    int64  `gorm:"column:enter_time;not null;default:0"`
	RewardStatus int8   `gorm:"column:reward_status;type:tinyint;not null;default:0;comment:0=未发奖,1=已发奖"`
	CreatedAt    int64  `gorm:"column:created_at;not null"`
}

func (RankSnapshotInfoEntity) TableName() string {
	return "rank_snapshot_info"
}

type RankRewardRecordEntity struct {
	Id        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	TaskId    int64  `gorm:"column:task_id;not null;uniqueIndex:uk_task_source,priority:1"`
	SourceId  int64  `gorm:"column:source_id;not null;uniqueIndex:uk_task_source,priority:2"`
	Reward    string `gorm:"column:reward;type:json;not null"`
	Rank      int32  `gorm:"column:rank;not null"`
	Status    int8   `gorm:"column:status;type:tinyint;not null;default:0;comment:0=pending,1=done"`
	CreatedAt int64  `gorm:"column:created_at;not null"`
	UpdatedAt int64  `gorm:"column:updated_at;not null"`
}

func (RankRewardRecordEntity) TableName() string {
	return "rank_reward_record"
}

type RankBoardServerDataEntity struct {
	NodeId       int32 `gorm:"column:node_id;primaryKey"`
	LastTimeTime int64 `gorm:"column:last_tick_time"`
}

func (m *RankBoardServerDataEntity) TableName() string {
	return "rank_board_server_data"
}
