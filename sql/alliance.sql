CREATE TABLE IF NOT EXISTS `alliance` (
  `alliance_id` BIGINT NOT NULL,
  `server_id` INT NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `announce` VARCHAR(512) NOT NULL DEFAULT '',
  `badge_id` INT NOT NULL DEFAULT 0,
  `notice` VARCHAR(512) NOT NULL DEFAULT '',
  `level` INT NOT NULL DEFAULT 1,
  `exp` INT NOT NULL DEFAULT 0,
  `apply_type` INT NOT NULL DEFAULT 0,
  `power_apply_condition` BIGINT NOT NULL DEFAULT 0,
  `city_level_condition` INT NOT NULL DEFAULT 0,
  `alliance_total_power` BIGINT NOT NULL DEFAULT 0,
  `create_time` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`alliance_id`),
  KEY `idx_server_name` (`server_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `alliance_member` (
  `alliance_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `role` INT NOT NULL DEFAULT 0,
  `join_time` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`user_id`),
  KEY `idx_alliance_id` (`alliance_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

