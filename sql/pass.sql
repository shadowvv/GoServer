-- 通行证进度表
CREATE TABLE IF NOT EXISTS `player_pass_progress` (
  `user_id` BIGINT NOT NULL COMMENT '玩家ID',
  `pass_id` INT NOT NULL COMMENT '通行证ID',
  `progress` INT NOT NULL DEFAULT 0 COMMENT '当前进度',
  `level` INT NOT NULL DEFAULT 0 COMMENT '当前等级',
  `loop_progress` INT NOT NULL DEFAULT 0 COMMENT '循环积分（满级后多出的积分）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `pass_id`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通行证进度表';

-- 通行证VIP等级表
CREATE TABLE IF NOT EXISTS `player_pass_vip` (
  `user_id` BIGINT NOT NULL COMMENT '玩家ID',
  `pass_id` INT NOT NULL COMMENT '通行证ID',
  `vip_level` INT NOT NULL DEFAULT 0 COMMENT 'VIP等级：0=免费, 1=付费档位1, 2=付费档位2',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `pass_id`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通行证VIP等级表';

-- 通行证奖励领取记录表
CREATE TABLE IF NOT EXISTS `player_pass_reward` (
  `user_id` BIGINT NOT NULL COMMENT '玩家ID',
  `pass_id` INT NOT NULL COMMENT '通行证ID',
  `level` INT NOT NULL COMMENT '等级',
  `reward_level` INT NOT NULL COMMENT '奖励档位：0=免费, 1=付费档位1, 2=付费档位2',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `pass_id`, `level`, `reward_level`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通行证奖励领取记录表';

-- 通行证掉落选择记录表（当drop有多个道具时，记录玩家选择）
CREATE TABLE IF NOT EXISTS `player_pass_drop_choice` (
  `user_id` BIGINT NOT NULL COMMENT '玩家ID',
  `pass_id` INT NOT NULL COMMENT '通行证ID',
  `level` INT NOT NULL COMMENT '等级',
  `reward_level` INT NOT NULL COMMENT '奖励档位：0=免费, 1=付费档位1, 2=付费档位2',
  `drop_id` INT NOT NULL COMMENT '掉落ID',
  `chosen_item_id` INT NOT NULL COMMENT '选择的道具ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`, `pass_id`, `level`, `reward_level`, `drop_id`),
  INDEX `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通行证掉落选择记录表';
