-- 特权卡系统表结构
-- 说明：
-- 1) 特权卡本质无“实体”，按 (user_id, item_id) 唯一一条状态
-- 2) expire_time = -1 表示永久；否则为秒级 Unix 时间戳
-- 3) item_id 对应 item.json 中 showGroup=20 的物品 id

CREATE TABLE IF NOT EXISTS player_vip_card (
    user_id BIGINT NOT NULL COMMENT '玩家ID',
    item_id INT NOT NULL COMMENT '特权卡物品ID',
    expire_time BIGINT NOT NULL DEFAULT -1 COMMENT '过期时间(秒时间戳);-1=永久',
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, item_id),
    INDEX idx_expire_time (expire_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

