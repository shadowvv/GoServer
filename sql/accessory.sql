CREATE TABLE `accessory` (
                             `user_id` BIGINT NOT NULL COMMENT '用户ID',
                             `accessory_id` INT NOT NULL COMMENT '配件ID',
                             `accessory_level` INT NOT NULL DEFAULT 0 COMMENT '配件等级',
                             `num` INT NOT NULL DEFAULT 0 COMMENT '配件数量',
                             `hero_own_id` BIGINT NULL DEFAULT NULL COMMENT '装备的英雄ID（如未装备则为NULL）',
                             PRIMARY KEY (`user_id`, `accessory_id`),
                             INDEX `idx_user_id` (`user_id`),
                             INDEX `idx_hero_own_id` (`hero_own_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='配件表';

CREATE TABLE IF NOT EXISTS accessory_lucky (
                                               user_id BIGINT NOT NULL COMMENT '用户ID',
                                               lucky_level INT NOT NULL DEFAULT 0 COMMENT '幸运等级',
                                               lucky_id INT NOT NULL COMMENT '幸运ID',
                                               lucky_num INT NOT NULL DEFAULT 0 COMMENT '幸运数量',
                                               free_num INT NOT NULL DEFAULT 0 COMMENT '免费次数',
                                               free_update_time BIGINT NOT NULL DEFAULT 0 COMMENT '免费次数更新时间',
    free_used_num int not null default 0 comment '免费抽取使用次数'
                                               PRIMARY KEY (user_id, lucky_id),
    INDEX idx_user_id (user_id),
    INDEX idx_lucky_id (lucky_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='饰品幸运表';