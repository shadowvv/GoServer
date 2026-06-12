-- 玩家渠道绑定表：记录玩家绑定的渠道及领取状态
CREATE TABLE IF NOT EXISTS `account_channel_bind` (
    `user_id` BIGINT NOT NULL COMMENT '玩家ID',
    `channel` VARCHAR(64) NOT NULL COMMENT '渠道标识，由客户端传入',
    `claim_status` TINYINT NOT NULL DEFAULT 0 COMMENT '领取状态：0未领取 1已绑定 2已领取，必须 0→1→2',
    PRIMARY KEY (`user_id`, `channel`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='玩家渠道绑定表';
