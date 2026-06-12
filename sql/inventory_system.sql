-- 简化版背包系统数据库设计（雪花算法ID版本）

-- 物品配置表（与代码模型 ItemConfigEntity 对齐）
CREATE TABLE IF NOT EXISTS item_config (
    item_id INT PRIMARY KEY,
    item_name VARCHAR(64) NOT NULL,
    item_type INT NOT NULL COMMENT '物品类型',
    quality INT NOT NULL COMMENT '物品品质',
    max_stack INT NOT NULL DEFAULT 1 COMMENT '最大堆叠数量',
    is_tradeable BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否可交易',
    is_sellable BOOLEAN NOT NULL DEFAULT TRUE COMMENT '是否可出售',
    sell_price INT NOT NULL DEFAULT 0 COMMENT '出售价格',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_item_type (item_type),
    INDEX idx_item_quality (quality)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 玩家背包表（与代码模型 PlayerInventoryEntity 对齐，启用软删除）
CREATE TABLE IF NOT EXISTS player_inventory (
    id BIGINT PRIMARY KEY COMMENT '雪花算法生成的唯一ID',
    user_id BIGINT NOT NULL,
    item_id INT NOT NULL,
    item_num INT NOT NULL DEFAULT 1 COMMENT '物品数量',
    inventory_type INT NOT NULL DEFAULT 1 COMMENT '背包类型',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    INDEX idx_user_inventory (user_id, inventory_type),
    INDEX idx_item_id (item_id),
    INDEX idx_deleted_at (deleted_at),
    FOREIGN KEY (item_id) REFERENCES item_config(item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 玩家物品数据表（与代码模型 PlayerItemDataEntity 对齐）
CREATE TABLE IF NOT EXISTS player_item_data (
    id BIGINT PRIMARY KEY COMMENT '雪花算法生成的唯一ID',
    inventory_id BIGINT NOT NULL COMMENT '关联的背包项ID',
    data_key VARCHAR(64) NOT NULL COMMENT '数据键',
    data_value VARCHAR(256) NOT NULL COMMENT '数据值',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inventory_id (inventory_id),
    FOREIGN KEY (inventory_id) REFERENCES player_inventory(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 说明：
-- 1) 与 DAO 查询保持一致：使用 snake_case 列名（user_id、item_id、inventory_type、deleted_at）。
-- 2) 软删除列 deleted_at 已添加并建索引，便于 "deleted_at IS NULL" 的查询。
-- 3) 所有主键 ID 采用雪花算法生成，不使用 AUTO_INCREMENT。
-- 4) 未启用 ON DELETE CASCADE，删除逻辑由服务层显式处理。