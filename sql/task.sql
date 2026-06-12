CREATE TABLE IF NOT EXISTS task (
                                    user_id BIGINT NOT NULL COMMENT '用户ID',
                                    slot_id INT NOT NULL COMMENT '槽位ID',
                                    task_id INT NOT NULL DEFAULT 0 COMMENT '任务ID',
                                    task_attribution INT NOT NULL COMMENT '任务归属',
                                    progress_data INT NOT NULL DEFAULT 0 COMMENT '进度数据',
                                    status INT NOT NULL DEFAULT 0 COMMENT '状态:0进行中,1完成,2已领奖',
                                    add_time BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
                                    update_time BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
                                    PRIMARY KEY (user_id, slot_id, task_attribution),
    INDEX idx_user_id (user_id),
    INDEX idx_task_id (task_id),
    INDEX idx_status (status),
    INDEX idx_add_time (add_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务表';

CREATE TABLE bounty (
                        user_id BIGINT NOT NULL COMMENT '用户ID',
                        bounty_id INT NOT NULL COMMENT '悬赏ID',
                        end_time BIGINT NOT NULL DEFAULT 0 COMMENT '结束时间',
                        status INT NOT NULL DEFAULT 0 COMMENT '状态',
                        slot_list JSON COMMENT '槽位列表',
                        PRIMARY KEY (user_id, bounty_id),
                        INDEX idx_status (status),
                        INDEX idx_end_time (end_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='悬赏表';

CREATE TABLE task_active_reward (
                                    user_id BIGINT NOT NULL COMMENT '用户ID',
                                    daily_point INT NOT NULL DEFAULT 0 COMMENT '每日积分',
                                    daily_update_time BIGINT NOT NULL DEFAULT 0 COMMENT '每日更新时间',
                                    daily_box INT NOT NULL DEFAULT 0 COMMENT '每日宝箱',
                                    week_point INT NOT NULL DEFAULT 0 COMMENT '每周积分',
                                    week_update_time BIGINT NOT NULL DEFAULT 0 COMMENT '每周更新时间',
                                    week_box INT NOT NULL DEFAULT 0 COMMENT '每周宝箱',
                                    PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务活跃奖励表';

CREATE TABLE `pass_card_task` (
                                  `user_id` BIGINT NOT NULL,
                                  `pass_card_id` INT NOT NULL,
                                  `status` INT NOT NULL DEFAULT 0,
                                  `task_slot_list` JSON,
                                  `task_finish_count` JSON,
                                  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                  PRIMARY KEY (`user_id`, `pass_card_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 添加索引（根据查询需求）
CREATE INDEX idx_user_id ON `pass_card_task` (`user_id`);
CREATE INDEX idx_pass_card_id ON `pass_card_task` (`pass_card_id`);
CREATE INDEX idx_status ON `pass_card_task` (`status`);