CREATE TABLE `loop_box` (
                            `user_Id` BIGINT NOT NULL COMMENT '用户ID',
                            `system_ex` INT NOT NULL DEFAULT 0 COMMENT '系统经验',
                            `system_level` INT NOT NULL DEFAULT 0 COMMENT '系统等级',
                            `system_point` INT NOT NULL DEFAULT 0 COMMENT '系统积分',
                            `loop_id` INT NOT NULL DEFAULT 0 COMMENT '当前循环id',
                            `box_list` JSON NOT NULL DEFAULT (JSON_ARRAY(0,0,0,0,0)) COMMENT '下标0-4 表示用户拥有1-5某个箱子多少个',
                            PRIMARY KEY (`user_Id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='循环宝箱系统表';

-- 创建索引
CREATE INDEX idx_user_id ON `loop_box` (`user_Id`);
CREATE INDEX idx_system_level ON `loop_box` (`system_level`);
CREATE INDEX idx_loop_id ON `loop_box` (`loop_id`);