-- 特权奖励每日领取记录表
-- 说明：
-- 1) 记录玩家每日领取的特权奖励
-- 2) reward_type: 奖励类型（1=招募权益，后续可扩展其他类型）
-- 3) last_claim_time: 上次领取时间（秒时间戳），用于判断是否跨天

CREATE TABLE IF NOT EXISTS player_privilege_reward (
    user_id BIGINT NOT NULL COMMENT '玩家ID',
    reward_type INT NOT NULL COMMENT '奖励类型（1=招募权益）',
    last_claim_time BIGINT NOT NULL DEFAULT 0 COMMENT '上次领取时间（秒时间戳）',
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, reward_type),
    INDEX idx_last_claim_time (last_claim_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
