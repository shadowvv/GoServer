package enum

import "fmt"

const (
	// Account/login
	REDIS_ACCOUNT_LOGIN_TOKEN = "login:token:" // login:token:account:token:serverId eg:login:token:drop:testToken:1
	REDIS_ONLINE_PLAYER       = "online:"      // online:serverId eg:online:1
	REDIS_REGISTER_COUNT      = "register:"    // register:serverId eg:register:1

	// Mail
	REDIS_MAIL_REFRESH_USERS  = "mail:refresh:users"  // mail:refresh:users eg:mail:refresh:users
	REDIS_MAIL_REFRESH_SERVER = "mail:refresh:server" // mail:refresh:server eg:mail:refresh:server

	// Player
	REDIS_PLAYER_BASIC_INFO    = "player:BasicInfo:"    // player:BasicInfo:userId eg:player:BasicInfo:10001
	REDIS_PLAYER_ALLIANCE_INFO = "player:AllianceInfo:" // player:AllianceInfo:userId eg:player:AllianceInfo:10001
	REDIS_PLAYER_BATTLE_INFO   = "player:BattleInfo:"   // player:BattleInfo:userId eg:player:BattleInfo:10001

	// Arena
	REDIS_SERVER_ARENA_SCORE_RANK = "ArenaScoreRank:" // ArenaScoreRank:serverId:version eg:ArenaScoreRank:1:v1

	// Chat
	REDIS_CHAT = "chat:" // chat::msgType:msgId eg:chat::1:1001

	// Alliance
	REDIS_ALLIANCE_BASIC_INFO  = "alliance:basic:"
	REDIS_ALLIANCE_MEMBER_INFO = "alliance:member:"
	REDIS_ALLIANCE_NAME_INDEX  = "alliance:name:index:"
	REDIS_ALLIANCE_APPLY_LIST  = "alliance:apply:list:"
	REDIS_ALLIANCE_SERVER_SET  = "alliance:server:"

	// Activity
	REDIS_ACTIVITY_ALL_CONFIG = "activity:config:" // activity:config: eg:activity:config:
	REDIS_ACTIVITY_OPEN       = "activity:open:"   // activity:open:serverId eg:activity:open:1

	// Glory arena
	REDIS_GLORY_ARENA_POOL_OPPONENT = "glory_arena:pool:opponent:" // glory_arena:pool:opponent:version eg:glory_arena:pool:opponent:v1
	REDIS_GLORY_ARENA_POOL_QUALIFY  = "glory_arena:pool:qualify:"  // glory_arena:pool:qualify:version eg:glory_arena:pool:qualify:v1
	REDIS_GLORY_ARENA_OPS_STATE     = "glory_arena:ops:state"      // glory_arena:ops:state eg:glory_arena:ops:state

	// Daily logs
	REDIS_DAILY_HERO_LEVELUP             = "daily:hero:levelup:"
	REDIS_DAILY_LOTTERY_COUNT            = "daily:lottery:count:"
	REDIS_DAILY_LOTTERY_QUALITY          = "daily:lottery:quality:"
	REDIS_DAILY_LOTTERY_HERO             = "daily:lottery:hero:"
	REDIS_DAILY_PET_RECRUIT_COUNT        = "daily:pet:recruit:count:"
	REDIS_DAILY_COLLECTION_LOTTERY_COUNT = "daily:collection:lottery:count:"
	REDIS_DAILY_EXPEDITION_COUNT         = "daily:expedition:count:"

	// Throughput monitor
	REDIS_GATEWAY_PROCESS_KEY   = "throughput:gateway:process:" // throughput:gateway:process:processId eg:throughput:gateway:process:1
	REDIS_SIDEWAY_PROCESS_KEY   = "throughput:sideway:process:"
	REDIS_LOGIN_PROCESS_KEY     = "throughput:login:process:"     // throughput:login:process:processId eg:throughput:login:process:1
	REDIS_SCENE_PROCESS_KEY     = "throughput:scene:process:"     // throughput:scene:process:sceneTemplateId:id eg:throughput:scene:process:1001:1
	REDIS_RANKBOARD_PROCESS_KEY = "throughput:rankBoard:process:" // throughput:rankBoard:process:processId eg:throughput:rankBoard:process:1
	REDIS_ALLIANCE_PROCESS_KEY  = "throughput:alliance:process:"  // throughput:alliance:process:processId eg:throughput:alliance:process:1
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

func GetAllianceMemberInfoKey(allianceId int64) string {
	return fmt.Sprintf(REDIS_ALLIANCE_MEMBER_INFO+"%d", allianceId)
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

func GetDailyExpeditionCountKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_EXPEDITION_COUNT+"%s", date)
}

func GetDailyLotteryQualityKey(date string, quality int32) string {
	return fmt.Sprintf(REDIS_DAILY_LOTTERY_QUALITY+"%s:%d", date, quality)
}

func GetDailyLotteryHeroKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_LOTTERY_HERO+"%s", date)
}

func GetDailyPetRecruitCountKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_PET_RECRUIT_COUNT+"%s", date)
}

func GetDailyCollectionLotteryCountKey(date string) string {
	return fmt.Sprintf(REDIS_DAILY_COLLECTION_LOTTERY_COUNT+"%s", date)
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

func GetSidewayProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_SIDEWAY_PROCESS_KEY+"%d", processId)
}

func GetAllianceProcessKey(processId int32) string {
	return fmt.Sprintf(REDIS_ALLIANCE_PROCESS_KEY+"%d", processId)
}
