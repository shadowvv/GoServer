package enum

// 排行榜分数类型枚举
type RankBoardScoreType int32

const (
	RANK_BOARD_SCORE_TYPE_LEVEL                                RankBoardScoreType = iota + 1 // 等级
	RANK_BOARD_SCORE_TYPE_MAIN_INSTANCE                                                      // 主线副本
	RANK_BOARD_SCORE_TYPE_BATTLE_POWER                                                       // 战力
	RANK_BOARD_SCORE_TYPE_TOWER                                                              // 爬塔
	RANK_BOARD_SCORE_TYPE_ALLIANCE_TOTAL_POWER                                               // 联盟战力
	RANK_BOARD_SCORE_TYPE_ARENA                                                              // 竞技场积分
	RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT                                        // 荣耀竞技场轮次胜场
	RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT                                       // 荣耀竞技场赛季胜场
	RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT                               // 荣耀竞技场联盟轮次胜场
	RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA                                                     // 联盟竞技场积分
)

func IsValidRankBoardScoreType(v int32) bool {
	return v >= int32(RANK_BOARD_SCORE_TYPE_LEVEL) && v <= int32(RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA)
}

// 排行榜排名规则枚举
type RankBoardRankRule int32

const (
	RANK_BOARD_RANK_RULE_SCORE      RankBoardRankRule = iota + 1 // 分数
	RANK_BOARD_RANK_RULE_ENTER_TIME                              // 满足条件的时间
)

func IsValidRankBoardRankRule(v int32) bool {
	return v >= int32(RANK_BOARD_RANK_RULE_SCORE) && v <= int32(RANK_BOARD_RANK_RULE_ENTER_TIME)
}

// 排行榜结算类型枚举
type RankBoardSettleType int32

const (
	RANK_BOARD_SETTLE_TYPE_DAY                     RankBoardSettleType = iota + 1 // 天结算
	RANK_BOARD_SETTLE_TYPE_WEEK                                                   // 周结算
	RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER                                 // 荣耀擂台轮结束
	RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER                                // 荣耀擂台赛季结束
	RANK_BOARD_SETTLE_TYPE_ACTIVITY_OVER                                          // 活动结束
)

func IsValidRankBoardSettleType(v int32) bool {
	return v >= int32(RANK_BOARD_SETTLE_TYPE_DAY) && v <= int32(RANK_BOARD_SETTLE_TYPE_ACTIVITY_OVER)
}

// 排行榜发奖类型枚举
type RankBoardSendRewardType int32

const (
	RANK_BOARD_SEND_REWARD_TYPE_RESOLVE RankBoardSendRewardType = iota + 1 // 榜单结算发奖
	RANK_BOARD_SEND_REWARD_TYPE_ENTER                                      // 进入榜单发奖
)

func IsValidRankBoardSendRewardType(v int32) bool {
	return v >= int32(RANK_BOARD_SEND_REWARD_TYPE_RESOLVE) && v <= int32(RANK_BOARD_SEND_REWARD_TYPE_ENTER)
}

const (
	RankSettleTaskStatusPending      int8 = 0
	RankSettleTaskStatusRunning      int8 = 1
	RankSettleTaskStatusSnapshotDone int8 = 2
	RankSettleTaskStatusRewardDone   int8 = 3
	RankSettleTaskStatusFailed       int8 = 4
)

const (
	RankRewardStatusPending int8 = 0
	RankRewardStatusDone    int8 = 1
)
