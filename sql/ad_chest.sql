-- 广告宝箱系统表结构
-- 说明：
-- 1) player_ad_chest: 玩家拥有的广告宝箱实例，每个宝箱有唯一 unique_id
-- 2) create_time: 毫秒时间戳

CREATE TABLE IF NOT EXISTS player_ad_chest (
    user_id BIGINT NOT NULL COMMENT '玩家ID',
    unique_id VARCHAR(36) NOT NULL COMMENT '宝箱唯一ID',
    item_id INT NOT NULL COMMENT '广告宝箱道具ID',
    cfg_index INT NOT NULL COMMENT 'limitedAdChest 配置索引',
    create_time BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间(毫秒时间戳)',
    PRIMARY KEY (unique_id),
    INDEX idx_user_id (user_id),
    INDEX idx_user_create (user_id, create_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='广告宝箱实例表';
