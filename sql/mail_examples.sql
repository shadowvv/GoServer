-- ============================================
-- 邮件系统 SQL 示例集合
-- ============================================

-- ============================================
-- 1. 插入测试邮件（个人邮件）
-- ============================================

-- 1.1 插入一封普通邮件（无附件）
INSERT INTO mail (
    mail_id, user_id, mail_type, title, content, 
    sender_id, sender_name, server_mail_id, template_id, 
    status, has_attachment, attachments, 
    expire_time, send_time, read_time, claim_time
) VALUES (
    1000000000000000001,  -- mail_id（雪花ID，示例值）
    4069511475103744,                 -- user_id（玩家ID）
    1,                     -- mail_type（1=普通邮件）
    '欢迎来到游戏',         -- title
    '亲爱的玩家，欢迎来到我们的游戏世界！',  -- content
    0,                     -- sender_id（0=系统）
    '系统',                -- sender_name
    0,                     -- server_mail_id（0=个人邮件）
    0,                     -- template_id（0=未使用模板）
    0,                     -- status（0=未读）
    FALSE,                 -- has_attachment
    NULL,                  -- attachments（无附件）
    0,                     -- expire_time（0=永不过期）
    UNIX_TIMESTAMP(),      -- send_time（当前时间戳）
    0,                     -- read_time
    0                      -- claim_time
);

-- 1.2 插入一封带附件的邮件（道具+货币）
INSERT INTO mail (
    mail_id, user_id, mail_type, title, content, 
    sender_id, sender_name, server_mail_id, template_id, 
    status, has_attachment, attachments, 
    expire_time, send_time, read_time, claim_time
) VALUES (
    1000000000000000002,  -- mail_id
    4069511475103744,                -- user_id
    3,                    -- mail_type（3=官方邮件）
    '活动奖励发放',        -- title
    '恭喜您完成活动任务，以下是您的奖励！',  -- content
    0,                    -- sender_id（0=系统）
    '活动系统',           -- sender_name
    0,                    -- server_mail_id
    0,                    -- template_id
    0,                    -- status（0=未读）
    TRUE,                 -- has_attachment
    '[{"type":1,"id":1001,"num":10},{"type":2,"id":1001,"num":5000}]',  -- attachments JSON
    1735689600,           -- expire_time（2025-01-01 00:00:00的时间戳，7天后过期）
    UNIX_TIMESTAMP(),     -- send_time
    0,                    -- read_time
    0                     -- claim_time
);

-- 1.3 插入一封玩家邮件（玩家A发给玩家B）
INSERT INTO mail (
    mail_id, user_id, mail_type, title, content, 
    sender_id, sender_name, server_mail_id, template_id, 
    status, has_attachment, attachments, 
    expire_time, send_time, read_time, claim_time
) VALUES (
    1000000000000000003,  -- mail_id
    10002,                -- user_id（接收者ID）
    5,                    -- mail_type（5=玩家邮件）
    '好友问候',           -- title
    '你好，一起组队打副本吧！',  -- content
    4069511475103744,                -- sender_id（发送者ID）
    '玩家A',             -- sender_name
    0,                    -- server_mail_id
    0,                    -- template_id
    0,                    -- status（0=未读）
    FALSE,                -- has_attachment
    NULL,                 -- attachments
    0,                    -- expire_time（0=永不过期）
    UNIX_TIMESTAMP(),     -- send_time
    0,                    -- read_time
    0                     -- claim_time
);

-- ============================================
-- 2. 插入全服邮件
-- ============================================

-- 2.1 插入一封全服邮件（无解锁条件，所有玩家可接收）
INSERT INTO server_mail (
    server_mail_id, mail_type, title, content, 
    template_id, server_id, unlock_list, attachments, 
    send_time, expire_time, status, created_by
) VALUES (
    2000000000000000001,  -- server_mail_id（雪花ID）
    3,                    -- mail_type（3=官方邮件）
    '全服维护通知',        -- title
    '游戏将于今晚22:00进行维护，预计维护2小时，请玩家提前下线。',  -- content
    0,                    -- template_id
    0,                    -- server_id（0=全服）
    '[]',                 -- unlock_list（空数组=无解锁条件）
    NULL,                 -- attachments（无附件）
    UNIX_TIMESTAMP(),     -- send_time
    0,                    -- expire_time（0=永不过期）
    1,                    -- status（1=已发送）
    'admin'               -- created_by（GM账号）
);

-- 2.2 插入一封带附件的全服邮件（有解锁条件）
INSERT INTO server_mail (
    server_mail_id, mail_type, title, content, 
    template_id, server_id, unlock_list, attachments, 
    send_time, expire_time, status, created_by
) VALUES (
    2000000000000000002,  -- server_mail_id
    3,                    -- mail_type
    '开服庆典奖励',       -- title
    '感谢各位玩家的支持，开服庆典奖励已发放！',  -- content
    0,                    -- template_id
    0,                    -- server_id（0=全服）
    '[1001,1002]',        -- unlock_list（需要满足解锁ID 1001和1002）
    '[{"type":1,"id":2001,"num":1},{"type":2,"id":1001,"num":10000}]',  -- attachments
    UNIX_TIMESTAMP(),     -- send_time
    1735689600,           -- expire_time（7天后过期）
    1,                    -- status（1=已发送）
    'gm001'               -- created_by
);

-- ============================================
-- 3. 查询邮件
-- ============================================

-- 3.1 查询玩家的所有未读邮件
SELECT 
    mail_id, title, content, sender_name, 
    has_attachment, send_time, expire_time
FROM mail
WHERE user_id = 4069511475103744
  AND status = 0  -- 0=未读
  AND deleted_at IS NULL
ORDER BY send_time DESC;

-- 3.2 查询玩家的所有邮件（分页，按发送时间倒序）
SELECT 
    mail_id, mail_type, title, sender_name, 
    status, has_attachment, send_time, expire_time
FROM mail
WHERE user_id = 4069511475103744
  AND deleted_at IS NULL
ORDER BY send_time DESC
LIMIT 10 OFFSET 0;  -- 每页10条，第1页

-- 3.3 查询玩家的未读邮件数量
SELECT COUNT(*) AS unread_count
FROM mail
WHERE user_id = 4069511475103744
  AND status = 0  -- 0=未读
  AND deleted_at IS NULL;

-- 3.4 查询玩家的指定类型邮件（例如：只查询官方邮件）
SELECT 
    mail_id, title, content, sender_name, 
    has_attachment, send_time
FROM mail
WHERE user_id = 4069511475103744
  AND mail_type = 3  -- 3=官方邮件
  AND deleted_at IS NULL
ORDER BY send_time DESC;

-- 3.5 查询玩家的已领取但未删除的邮件
SELECT 
    mail_id, title, sender_name, claim_time
FROM mail
WHERE user_id = 4069511475103744
  AND status = 2  -- 2=已领取
  AND deleted_at IS NULL
ORDER BY claim_time DESC;

-- 3.6 查询指定邮件详情（包含附件JSON）
SELECT 
    mail_id, mail_type, title, content, 
    sender_id, sender_name, status, 
    has_attachment, attachments,  -- 附件JSON字段
    expire_time, send_time, read_time, claim_time
FROM mail
WHERE mail_id = 1000000000000000002
  AND deleted_at IS NULL;

-- 3.7 查询已过期的邮件（需要清理）
SELECT 
    mail_id, user_id, title, expire_time
FROM mail
WHERE expire_time > 0
  AND expire_time < UNIX_TIMESTAMP()  -- 过期时间小于当前时间
  AND deleted_at IS NULL
LIMIT 100;

-- ============================================
-- 4. 更新邮件状态
-- ============================================

-- 4.1 标记邮件为已读
UPDATE mail
SET status = 1,  -- 1=已读
    read_time = UNIX_TIMESTAMP()
WHERE mail_id = 1000000000000000001
  AND user_id = 4069511475103744
  AND deleted_at IS NULL;

-- 4.2 标记邮件为已领取（领取附件后）
UPDATE mail
SET status = 2,  -- 2=已领取
    claim_time = UNIX_TIMESTAMP()
WHERE mail_id = 1000000000000000002
  AND user_id = 4069511475103744
  AND deleted_at IS NULL;

-- 4.3 批量标记为已读（玩家阅读所有未读邮件）
UPDATE mail
SET status = 1,
    read_time = UNIX_TIMESTAMP()
WHERE user_id = 4069511475103744
  AND status = 0  -- 0=未读
  AND deleted_at IS NULL;

-- ============================================
-- 5. 删除邮件（软删除）
-- ============================================

-- 5.1 删除指定邮件（软删除）
UPDATE mail
SET deleted_at = NOW()
WHERE mail_id = 1000000000000000003
  AND user_id = 10002
  AND deleted_at IS NULL;

-- 5.2 删除玩家所有已领取的邮件（软删除）
UPDATE mail
SET deleted_at = NOW()
WHERE user_id = 4069511475103744
  AND status = 2  -- 2=已领取
  AND deleted_at IS NULL;

-- 5.3 物理删除已软删除超过30天的邮件（清理数据）
DELETE FROM mail
WHERE deleted_at IS NOT NULL
  AND deleted_at < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- ============================================
-- 6. 全服邮件查询
-- ============================================

-- 6.1 查询所有已发送的全服邮件
SELECT 
    server_mail_id, mail_type, title, 
    send_time, expire_time, created_by
FROM server_mail
WHERE status = 1  -- 1=已发送
ORDER BY send_time DESC;

-- 6.2 查询待发送的全服邮件
SELECT 
    server_mail_id, title, content, 
    unlock_list, attachments, created_by
FROM server_mail
WHERE status = 0  -- 0=待发送
ORDER BY created_at ASC;

-- 6.3 查询已过期的全服邮件
SELECT 
    server_mail_id, title, expire_time
FROM server_mail
WHERE expire_time > 0
  AND expire_time < UNIX_TIMESTAMP()
  AND status = 1;  -- 只查询已发送的

-- 6.4 更新全服邮件状态为已过期
UPDATE server_mail
SET status = 2  -- 2=已过期
WHERE expire_time > 0
  AND expire_time < UNIX_TIMESTAMP()
  AND status = 1;

-- ============================================
-- 7. 统计查询
-- ============================================

-- 7.1 统计玩家的邮件总数（按状态分组）
SELECT 
    status,
    COUNT(*) AS count
FROM mail
WHERE user_id = 4069511475103744
  AND deleted_at IS NULL
GROUP BY status;

-- 7.2 统计玩家的邮件总数（按类型分组）
SELECT 
    mail_type,
    COUNT(*) AS count
FROM mail
WHERE user_id = 4069511475103744
  AND deleted_at IS NULL
GROUP BY mail_type;

-- 7.3 统计所有玩家的邮件数量（Top 10）
SELECT 
    user_id,
    COUNT(*) AS mail_count
FROM mail
WHERE deleted_at IS NULL
GROUP BY user_id
ORDER BY mail_count DESC
LIMIT 10;

-- 7.4 统计有附件的邮件数量
SELECT COUNT(*) AS attachment_mail_count
FROM mail
WHERE has_attachment = TRUE
  AND deleted_at IS NULL;

-- ============================================
-- 8. 维护操作
-- ============================================

-- 8.1 清理过期邮件（软删除）
UPDATE mail
SET deleted_at = NOW()
WHERE expire_time > 0
  AND expire_time < UNIX_TIMESTAMP()
  AND deleted_at IS NULL;

-- 8.2 查询玩家的邮件数量（检查是否超过上限）
SELECT COUNT(*) AS mail_count
FROM mail
WHERE user_id = 4069511475103744
  AND deleted_at IS NULL;
-- 如果 mail_count >= 100，需要提示玩家清理邮件

-- 8.3 查询玩家最早的一封邮件（用于自动删除最旧的邮件）
SELECT mail_id, send_time
FROM mail
WHERE user_id = 4069511475103744
  AND deleted_at IS NULL
ORDER BY send_time ASC
LIMIT 1;

-- ============================================
-- 9. JSON 字段操作示例（MySQL 5.7+）
-- ============================================

-- 9.1 查询附件中包含特定道具ID的邮件
SELECT 
    mail_id, title, attachments
FROM mail
WHERE JSON_CONTAINS(attachments, '{"id": 1001}', '$')
  AND deleted_at IS NULL;

-- 9.2 查询附件数量大于0的邮件
SELECT 
    mail_id, title, 
    JSON_LENGTH(attachments) AS attachment_count
FROM mail
WHERE has_attachment = TRUE
  AND JSON_LENGTH(attachments) > 0
  AND deleted_at IS NULL;

-- 9.3 提取附件JSON中的第一个道具信息
SELECT 
    mail_id, title,
    JSON_EXTRACT(attachments, '$[0].type') AS first_item_type,
    JSON_EXTRACT(attachments, '$[0].id') AS first_item_id,
    JSON_EXTRACT(attachments, '$[0].num') AS first_item_num
FROM mail
WHERE has_attachment = TRUE
  AND deleted_at IS NULL
LIMIT 10;

-- ============================================
-- 10. 复杂查询示例
-- ============================================

-- 10.1 查询玩家最近7天收到的邮件数量
SELECT COUNT(*) AS mail_count
FROM mail
WHERE user_id = 4069511475103744
  AND send_time >= UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 7 DAY))
  AND deleted_at IS NULL;

-- 10.2 查询玩家未领取附件的邮件列表
SELECT 
    mail_id, title, sender_name, 
    attachments, send_time
FROM mail
WHERE user_id = 4069511475103744
  AND has_attachment = TRUE
  AND status < 2  -- 0=未读 1=已读，但未领取（2=已领取）
  AND deleted_at IS NULL
ORDER BY send_time DESC;

-- 10.3 查询全服邮件中需要特定解锁条件的邮件
SELECT 
    server_mail_id, title, unlock_list
FROM server_mail
WHERE JSON_CONTAINS(unlock_list, '1001', '$')
  AND status = 1;  -- 已发送

-- ============================================
-- 注意事项：
-- 1. mail_id 和 server_mail_id 使用雪花算法生成，示例中的ID仅作参考
-- 2. 时间戳字段使用 UNIX_TIMESTAMP() 获取当前时间戳（秒级）
-- 3. attachments 和 unlock_list 字段存储 JSON 格式数据
-- 4. 使用软删除（deleted_at），物理删除需谨慎
-- 5. 查询时注意添加 deleted_at IS NULL 条件
-- ============================================

