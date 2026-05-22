package enum

import "fmt"

const (
	// Account/login
	REDIS_ACCOUNT_LOGIN_TOKEN = "login:token:"
	REDIS_ONLINE_PLAYER       = "online:"
	REDIS_MEMORY_PLAYER       = "memory:"
	REDIS_REGISTER_COUNT      = "register:"

	// Mail
	REDIS_MAIL_REFRESH_USERS  = "mail:refresh:users"
	REDIS_MAIL_REFRESH_SERVER = "mail:refresh:server"

	// Player
	REDIS_PLAYER_BASIC_INFO    = "player:BasicInfo:"
	REDIS_PLAYER_ALLIANCE_INFO = "player:AllianceInfo:"
	REDIS_PLAYER_BATTLE_INFO   = "player:BattleInfo:"

	// Arena
	REDIS_SERVER_ARENA_SCORE_RANK = "ArenaScoreRank:"

	// Chat
	REDIS_CHAT = "chat:"

	// Alliance
	REDIS_ALLIANCE_BASIC_INFO = "alliance:basic:"
	REDIS_ALLIANCE_NAME_INDEX = "alliance:name:index:"
	REDIS_ALLIANCE_APPLY_LIST = "alliance:apply:list:"
	REDIS_ALLIANCE_SERVER_SET = "alliance:server:"

	// Activity
	REDIS_ACTIVITY_ALL_CONFIG = "activity:config:"
	REDIS_ACTIVITY_OPEN       = "activity:open:"

	// Glory arena
	REDIS_GLORY_ARENA_POOL_OPPONENT = "glory_arena:pool:opponent:"
	REDIS_GLORY_ARENA_POOL_QUALIFY  = "glory_arena:pool:qualify:"
	REDIS_GLORY_ARENA_OPS_STATE     = "glory_arena:ops:state"

	// Daily logs
	REDIS_DAILY_HERO_LEVELUP    = "daily:hero:levelup:"
	REDIS_DAILY_LOTTERY_COUNT   = "daily:lottery:count:"
	REDIS_DAILY_LOTTERY_QUALITY = "daily:lottery:quality:"
	REDIS_DAILY_LOTTERY_HERO    = "daily:lottery:hero:"

	// Throughput monitor
	REDIS_GATEWAY_PROCESS_KEY   = "throughput:gateway:process:"
	REDIS_LOGIN_PROCESS_KEY     = "throughput:login:process:"
	REDIS_SCENE_PROCESS_KEY     = "throughput:scene:process:"
	REDIS_RANKBOARD_PROCESS_KEY = "throughput:rankBoard:process:"
	REDIS_ALLIANCE_PROCESS_KEY  = "throughput:alliance:process:"
)

func GetActivityOpenKey(serverId int32) string {
	return fmt.Sprintf(REDIS_ACTIVITY_OPEN+"%d", serverId)
}

func GetLoginTokenKey(account, token string, serverId int32) string {
	return fmt.Sprintf(REDIS_ACCOUNT_LOGIN_TOKEN+"%s:%s:%d", account, token, serverId)
}

func GetOnlinePlayerKey(serverId int32) string {
	return fmt.Sprintf(REDIS_ONLINE_PLAYER+"%d", serverId)
}

func GetRegisterConst(serverId int32) string {
	return fmt.Sprintf(REDIS_REGISTER_COUNT+"%d", serverId)
}

func GetChatKey(msgType int32, msgId int64) string {
	return fmt.Sprintf(REDIS_CHAT+":%d:%d", msgType, msgId)
}

func GetPlayerBasicInfoKey(userId int64) string {
	return fmt.Sprintf(REDIS_PLAYER_BASIC_INFO+"%d", userId)
}

func GetArenaScoreInfoKey(serverId int32, version string) string {
	return fmt.Sprintf(REDIS_SERVER_ARENA_SCORE_RANK+"%d:%s", serverId, version)
}

func GetPlayerAllianceInfoKey(userId int64) string {
	return fmt.Sprintf(REDIS_PLAYER_ALLIANCE_INFO+"%d", userId)
}

func GetPlayerBattleInfoKey(userId int64) string {
	return fmt.Sprintf(REDIS_PLAYER_BATTLE_INFO+"%d", userId)
}

func GetAllianceBasicInfoKey(allianceId int64) string {
	return fmt.Sprintf(REDIS_ALLIANCE_BASIC_INFO+"%d", allianceId)
}

func GetAllianceNameIndexKey(serverId int32) string {
	return fmt.Sprintf(REDIS_ALLIANCE_NAME_INDEX+"%d", serverId)
}

func GetAllianceApplyListKey(allianceId int64) string {
	return fmt.Sprintf(REDIS_ALLIANCE_APPLY_LIST+"%d", allianceId)
}

func GetServerAllianceSetKey(serverId int32) string {
	return fmt.Sprintf(REDIS_ALLIANCE_SERVER_SET+"%d", serverId)
}

func GetGloryArenaPoolOpponentRoundKey(version string) string {
	return fmt.Sprintf(REDIS_GLORY_ARENA_POOL_OPPONENT+"%s", version)
}

func GetGloryArenaPoolQualifyRoundKey(version string) string {
	return fmt.Sprintf(REDIS_GLORY_ARENA_POOL_QUALIFY+"%s", version)
}

func GetGloryArenaOpsStateKey() string {
	return REDIS_GLORY_ARENA_OPS_STATE
}

func GetDailyHeroLevelUpKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_HERO_LEVELUP+"%s", date)
}

func GetDailyLotteryCountKey(date string, lotteryId int32) string {
	return fmt.Sprintf(REDIS_DAILY_LOTTERY_COUNT+"%s:%d", date, lotteryId)
}

func GetDailyLotteryQualityKey(date string, quality int32) string {
	return fmt.Sprintf(REDIS_DAILY_LOTTERY_QUALITY+"%s:%d", date, quality)
}

func GetDailyLotteryHeroKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_LOTTERY_HERO+"%s", date)
}

func GetSceneProcessKey(sceneTemplateId int32, id int32) string {
	return fmt.Sprintf(REDIS_SCENE_PROCESS_KEY+"%d:%d", sceneTemplateId, id)
}

func GetLoginProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_LOGIN_PROCESS_KEY+"%d", processId)
}

func GetRankBoardProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_RANKBOARD_PROCESS_KEY+"%d", processId)
}

func GetGatewayProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_GATEWAY_PROCESS_KEY+"%d", processId)
}

func GetAllianceProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_ALLIANCE_PROCESS_KEY+"%d", processId)
}
