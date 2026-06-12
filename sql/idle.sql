-- 挂机奖励表
CREATE TABLE `idle` (
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `idle_level` INT NOT NULL DEFAULT 1 COMMENT '挂机等级',
    `accumulated_time` INT NOT NULL DEFAULT 0 COMMENT '累计挂机时间（秒）',
    `last_settle_time` BIGINT NOT NULL DEFAULT 0 COMMENT '上次结算时间（秒时间戳）',
    `last_claim_time` BIGINT NOT NULL DEFAULT 0 COMMENT '上次领取时间（秒时间戳）',
    `quick_claim_count` INT NOT NULL DEFAULT 0 COMMENT '今日快速领取次数',
    `quick_ad_claim_count` INT NOT NULL DEFAULT 0 COMMENT '今日广告快速领取次数',
    `quick_claim_reset_time` BIGINT NOT NULL DEFAULT 0 COMMENT '快速领取次数重置时间（秒时间戳）',
    `pending_rewards` JSON NOT NULL DEFAULT ('[]') COMMENT '待领取奖励（JSON，预览后不可变更）',
    `quick_claim_preview_rewards` JSON NOT NULL DEFAULT ('[]') COMMENT '快速领取预览奖励（JSON，预览后不可变更）',
    PRIMARY KEY (`user_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='挂机奖励表';

