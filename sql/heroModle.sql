-- 图鉴奖励积分表
CREATE TABLE `album_reward_score` (
                                      `user_id` BIGINT NOT NULL COMMENT '用户ID',
                                      `claimed_reward` INT NOT NULL DEFAULT 0 COMMENT '已领取积分档位',
                                      `all_score` INT NOT NULL DEFAULT 0 COMMENT '总积分',
                                      PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='图鉴奖励积分表';

-- 英雄图鉴表
CREATE TABLE `hero_album` (
                              `user_id` BIGINT NOT NULL COMMENT '用户ID',
                              `hero_id` BIGINT NOT NULL COMMENT '英雄ID',
                              `history_max_star` INT NOT NULL DEFAULT 0 COMMENT '历史最高星级',
                              `claimed_star` INT NOT NULL DEFAULT 0 COMMENT '已领取星级档位',
                              PRIMARY KEY (`user_id`, `hero_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='英雄图鉴表';

-- 英雄详情表
CREATE TABLE `hero_details` (
                                `hero_own_id` BIGINT NOT NULL COMMENT '英雄唯一ID',
                                `user_id` BIGINT NOT NULL COMMENT '用户ID',
                                `hero_id` BIGINT NOT NULL COMMENT '英雄ID',
                                `level` INT NOT NULL DEFAULT 1 COMMENT '等级',
                                `star_level` INT NOT NULL DEFAULT 1 COMMENT '星级',
                                `evolution_path` INT NOT NULL DEFAULT 0 COMMENT '转职方向',
                                `evolution_update_time` BIGINT NOT NULL DEFAULT 0 COMMENT '转职更新时间',
                                `break_num` INT NOT NULL DEFAULT 0 COMMENT '进阶次数',
                                `equipment_id` json NOT NULL DEFAULT ('[]') COMMENT '装备ID',
                                `is_deleted` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否删除',
                                PRIMARY KEY (`hero_own_id`),
                                KEY `idx_user_id` (`user_id`),
                                KEY `idx_hero_id` (`hero_id`),
                                KEY `idx_is_deleted` (`is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='英雄详情表';

-- 英雄阵型表
CREATE TABLE `hero_formation` (
                                  `user_id` BIGINT NOT NULL COMMENT '用户ID',
                                  `formation_id` INT NOT NULL COMMENT '阵型ID',
                                  `hero_own_id_list` JSON DEFAULT NULL COMMENT '阵型中的英雄唯一ID列表',
                                  `formation_type` INT NOT NULL COMMENT '阵型类型',
                                  `is_active` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否激活',
                                  PRIMARY KEY (`user_id`, `formation_id`, `formation_type`),
                                  KEY `idx_user_id` (`user_id`),
                                  KEY `idx_formation_type` (`formation_type`),
                                  KEY `idx_user_formation_type` (`user_id`, `formation_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='英雄阵型表';