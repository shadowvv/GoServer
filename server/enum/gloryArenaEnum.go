package enum

// GloryArenaSeasonType defines glory arena season phase.
type GloryArenaSeasonType int32

const (
	GLORY_ARENA_SEASON_TYPE_PRE    GloryArenaSeasonType = 0
	GLORY_ARENA_SEASON_TYPE_FIRST  GloryArenaSeasonType = 1
	GLORY_ARENA_SEASON_TYPE_SECOND GloryArenaSeasonType = 2
	GLORY_ARENA_SEASON_TYPE_POST   GloryArenaSeasonType = 3

	GLORY_ARENA_ARENA_RANK_TOP_N_DEFAULT int32 = 100
	GLORY_ARENA_BATTLE_POWER_RANK_ID     int32 = 3
)

func IsValidGloryArenaSeasonType(v int32) bool {
	return v >= int32(GLORY_ARENA_SEASON_TYPE_PRE) || v <= int32(GLORY_ARENA_SEASON_TYPE_POST)
}

// GetGloryArenaSeasonTypeBySeasonId maps legacy season id to phase:
// odd id => preseason, even id => postseason.
func GetGloryArenaSeasonTypeBySeasonId(seasonId int32) GloryArenaSeasonType {
	if seasonId > 0 {
		return GLORY_ARENA_SEASON_TYPE_POST
	}
	return GLORY_ARENA_SEASON_TYPE_PRE
}
