-- 邮件系统数据库设计（雪花算法ID版本）

-- 玩家邮件表（与代码模型 MailEntity 对齐，启用软删除）
CREATE TABLE IF NOT EXISTS mail (
    mail_id BIGINT PRIMARY KEY COMMENT '邮件ID（雪花算法生成）',
    user_id BIGINT NOT NULL COMMENT '玩家ID',
    mail_type INT NOT NULL DEFAULT 1 COMMENT '邮件类型（1普通 2广告 3官方 4命令 5玩家）',
    title VARCHAR(128) NOT NULL COMMENT '邮件标题',
    content TEXT NOT NULL COMMENT '邮件内容',
    sender_id BIGINT NOT NULL DEFAULT 0 COMMENT '发送者ID（0表示系统）',
    sender_name VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发送者名称',
    sender_avatar VARCHAR(256) NOT NULL DEFAULT '' COMMENT '发送者头像（当非玩家邮件时使用；可存URL/资源名）',
    server_mail_id BIGINT NOT NULL DEFAULT 0 COMMENT '关联的全服邮件ID（0表示个人邮件）',
    template_id INT NOT NULL DEFAULT 0 COMMENT '邮件模板ID',
    status INT NOT NULL DEFAULT 0 COMMENT '状态（0未读 1已读 2已领取 3已删除）',
    has_attachment BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否有附件',
    is_convenient TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否可一键领取（1可一键领取；0只能单独领取）',
    attachments JSON COMMENT '附件物品条目JSON（存储 MailAttachmentItem 数组）',
    expire_time BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间戳（0表示永不过期）',
    send_time BIGINT NOT NULL COMMENT '发送时间戳',
    read_time BIGINT NOT NULL DEFAULT 0 COMMENT '阅读时间戳',
    claim_time BIGINT NOT NULL DEFAULT 0 COMMENT '领取时间戳',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    INDEX idx_user_mail (user_id, status, deleted_at),
    INDEX idx_server_mail_id (server_mail_id),
    INDEX idx_expire_time (expire_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='玩家邮件表';

-- 全服邮件表（与代码模型 ServerMailEntity 对齐）
CREATE TABLE IF NOT EXISTS server_mail (
    server_mail_id BIGINT PRIMARY KEY COMMENT '全服邮件ID（雪花算法生成）',
    mail_type INT NOT NULL DEFAULT 1 COMMENT '邮件类型',
    title VARCHAR(128) NOT NULL COMMENT '邮件标题',
    content TEXT NOT NULL COMMENT '邮件内容',
    template_id INT NOT NULL DEFAULT 0 COMMENT '邮件模板ID',
    server_id INT NOT NULL DEFAULT 0 COMMENT '服务器ID（0表示全服）',
    sender_avatar VARCHAR(256) NOT NULL DEFAULT '' COMMENT '发送者头像（全服邮件展示用；可存URL/资源名）',
    unlock_list JSON COMMENT '解锁条件列表JSON格式（存储unlockID数组）',
    is_convenient TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否可一键领取（1可一键领取；0只能单独领取）',
    attachments JSON COMMENT '附件物品条目JSON（存储 MailAttachmentItem 数组）',
    send_time BIGINT NOT NULL COMMENT '发送时间戳',
    expire_time BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间戳（0表示永不过期）',
    status INT NOT NULL DEFAULT 0 COMMENT '状态（0待发送 1已发送 2已过期）',
    created_by VARCHAR(64) NOT NULL COMMENT '创建者（GM账号）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX idx_server_id (server_id),
    INDEX idx_status (status),
    INDEX idx_send_time (send_time),
    INDEX idx_expire_time (expire_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='全服邮件表';

-- 附件JSON格式示例：
-- [
--   {
--     "type": 1,
--     "id": 1001,
--     "num": 10
--   },
--   {
--     "type": 2,
--     "id": 1001,
--     "num": 1000
--   }
-- ]
-- 
-- 说明：
-- 1) type: 1=道具 2=货币 3=经验
-- 2) id: 道具ID/货币ID/资源ID
-- 3) num: 数量
-- 
-- unlock_list JSON格式示例：
-- [1001, 1002, 1003]
-- 
-- 说明：
-- 1) 存储解锁ID数组，玩家需要满足所有解锁条件才能接收该全服邮件

