CREATE TABLE IF NOT EXISTS `player_expedition_battlefield` (
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `battlefield_id` INT NOT NULL COMMENT '战场ID',
    `battlefield_level` INT NOT NULL DEFAULT 1 COMMENT '战场等级',
    `last_refresh_time` BIGINT NOT NULL DEFAULT 0 COMMENT '上次刷新时间(毫秒时间戳)',
    `update_time` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(毫秒时间戳)',
    PRIMARY KEY (`user_id`, `battlefield_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家派遣战场等级';

CREATE TABLE IF NOT EXISTS `player_expedition_slot` (
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `slot_id` INT NOT NULL COMMENT '队伍槽位ID',
    `is_unlocked` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已解锁',
    `slot_status` INT NOT NULL DEFAULT 0 COMMENT '槽位状态 0空闲 1派遣中',
    `unlock_time` BIGINT NOT NULL DEFAULT 0 COMMENT '解锁时间(毫秒时间戳)',
    `update_time` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(毫秒时间戳)',
    PRIMARY KEY (`user_id`, `slot_id`),
    KEY `idx_user_unlocked` (`user_id`, `is_unlocked`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家派遣队伍槽位';

CREATE TABLE IF NOT EXISTS `player_expedition_point_state` (
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `battlefield_id` INT NOT NULL COMMENT '战场ID',
    `point_id` INT NOT NULL COMMENT '怪物刷新点ID',
    `monster_id` INT NOT NULL DEFAULT 0 COMMENT '当前怪物ID',
    `point_status` INT NOT NULL DEFAULT 0 COMMENT '点位状态 0空闲 1占用',
    `last_refresh_time` BIGINT NOT NULL DEFAULT 0 COMMENT '上次刷新时间(毫秒时间戳)',
    `next_refresh_time` BIGINT NOT NULL DEFAULT 0 COMMENT '下次刷新时间(毫秒时间戳)',
    `update_time` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(毫秒时间戳)',
    PRIMARY KEY (`user_id`, `battlefield_id`, `point_id`),
    KEY `idx_user_bf_status` (`user_id`, `battlefield_id`, `point_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家派遣怪物点位状态';

CREATE TABLE IF NOT EXISTS `player_expedition_active` (
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `slot_id` INT NOT NULL COMMENT '队伍槽位ID',
    `battlefield_id` INT NOT NULL COMMENT '战场ID',
    `point_id` INT NOT NULL COMMENT '怪物刷新点ID',
    `formation_type` INT NOT NULL COMMENT '阵容类型',
    `formation_id` INT NOT NULL COMMENT '阵容ID',
    `status` INT NOT NULL DEFAULT 0 COMMENT '任务状态 0进行中 1已完成待手动结束',
    `start_time` BIGINT NOT NULL DEFAULT 0 COMMENT '开始时间(毫秒时间戳)',
    `end_time` BIGINT NOT NULL DEFAULT 0 COMMENT '结束时间(毫秒时间戳)',
    `update_time` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间(毫秒时间戳)',
    PRIMARY KEY (`user_id`, `slot_id`),
    UNIQUE KEY `uk_user_formation` (`user_id`, `formation_type`, `formation_id`),
    UNIQUE KEY `uk_user_point` (`user_id`, `battlefield_id`, `point_id`),
    KEY `idx_user_end_time` (`user_id`, `end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家进行中的派遣任务';
