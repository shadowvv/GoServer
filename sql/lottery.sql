CREATE TABLE `lottery` (
                           `user_id` BIGINT NOT NULL COMMENT '用户ID',
                           `id` INT NOT NULL COMMENT '卡池ID',
                           `all_count` INT NOT NULL DEFAULT 0 COMMENT '总抽取次数',
                           `last_basic_guarantee_num` INT NOT NULL DEFAULT 0 COMMENT '上次基础保底计数',
                           `last_special_guarantee_num` INT NOT NULL DEFAULT 0 COMMENT '上次循环保底计数',
                           `special_guarantee_count` INT NOT NULL DEFAULT 0 COMMENT '循环保底生效档位',
                           `last_change_time` BIGINT NOT NULL DEFAULT 0 COMMENT '上次更新时间',
                           `last_once_effective_num` INT NOT NULL DEFAULT 0 COMMENT '上次单次保底计数',
                           `once_effective_count` INT NOT NULL DEFAULT 0 COMMENT '单次保底生效档位',
                           `first_drop_free`         INT NOT NULL DEFAULT 0 COMMENT '首次免费次数',
                           PRIMARY KEY (`user_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抽奖记录表';

-- 创建索引（根据需要）
CREATE INDEX idx_user_id ON `lottery` (`user_id`);
CREATE INDEX idx_lottery_id ON `lottery` (`id`);
CREATE INDEX idx_user_lottery ON `lottery` (`user_id`, `id`);


CREATE TABLE `lottery_log` (
                               `user_id` BIGINT NOT NULL COMMENT '用户ID',
                               `id` INT NOT NULL COMMENT '日志ID',
                               `item_id` INT NOT NULL COMMENT '物品ID',
                               `count` INT NOT NULL DEFAULT 0 COMMENT '抽取次数',
                               `add_time` BIGINT NOT NULL COMMENT '添加时间',
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抽奖日志表';

-- 创建常用索引
CREATE INDEX idx_user_id ON `lottery_log` (`user_id`);
CREATE INDEX idx_log_id ON `lottery_log` (`id`);
CREATE INDEX idx_item_id ON `lottery_log` (`item_id`);
CREATE INDEX idx_add_time ON `lottery_log` (`add_time`);
CREATE INDEX idx_user_item ON `lottery_log` (`user_id`, `item_id`);
CREATE INDEX idx_user_log ON `lottery_log` (`user_id`, `id`);