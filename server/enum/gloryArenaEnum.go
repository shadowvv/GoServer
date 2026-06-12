package enum

type GloryArenaSeasonType int32

const (
	GLORY_ARENA_SEASON_TYPE_PRE    GloryArenaSeasonType = 0 // 季前赛
	GLORY_ARENA_SEASON_TYPE_FIRST  GloryArenaSeasonType = 1 // 第一赛季
	GLORY_ARENA_SEASON_TYPE_SECOND GloryArenaSeasonType = 2 // 第二赛季
	GLORY_ARENA_SEASON_TYPE_POST   GloryArenaSeasonType = 3 // 季后赛

	GLORY_ARENA_BATTLE_POWER_RANK_ID int32 = 3
)

func IsValidGloryArenaSeasonType(v int32) bool {
	return v >= int32(GLORY_ARENA_SEASON_TYPE_PRE) || v <= int32(GLORY_ARENA_SEASON_TYPE_POST)
}
