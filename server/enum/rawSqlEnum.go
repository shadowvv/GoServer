package enum

const (
	SELECT_ARENA_DEFEND_LOG_SQL = `
SELECT attack_user_id, defend_user_id, challenge_time, defend_score_change
FROM player_arena_log
WHERE defend_user_id = ? AND version = ?
ORDER BY challenge_time DESC
LIMIT 30` // 查询竞技场防守日志

	SELECT_ARENA_DEFEND_NOT_RESOLVE_SQL = `
SELECT battle_id, defend_score_change
FROM player_arena_log
WHERE defend_user_id = ? AND version = ? AND defend_resolved = 0` // 查询竞技场防守未结算日志

	SELECT_RECENT_PLAYER_SQL = `
SELECT account, user_id, server_id, nickname, head_id, head_frame_id, title_id, level, vip, charge_count, last_charge_time, last_login_time, last_offline_time, register_time, channel_id
FROM account` // 查询玩家信息

	SELECT_PLAYER_SQL_BY_ID_SQL = `
SELECT account, user_id, server_id, nickname, head_id, head_frame_id, title_id, level, vip, charge_count, last_charge_time, last_login_time, last_offline_time, register_time, channel_id
FROM account
WHERE user_id = ?` // 通过id获得玩家信息
)
