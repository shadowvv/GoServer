// File: mailTypes.go
// Description: 邮件系统类型定义（避免循环依赖）
// Author: 木村
// Create Time: 2026.01

package logicCommon

// MailAttachmentItem 附件内的道具条目
// 注意：这里的 Type 不是"附件类型"，而是"道具条目类型"，用于决定如何发放
// 约定：1=道具 2=货币 3=经验（与历史 AttachmentType 常量保持一致，便于迁移）
type MailAttachmentItem struct {
	Type int32 `json:"type"` // 条目类型（1道具 2货币 3经验）
	ID   int32 `json:"id"`   // 道具/货币/资源ID
	Num  int32 `json:"num"`  // 数量
}

// Mail 邮件业务模型
type Mail struct {
	MailID        int64                 // 邮件ID（雪花ID）
	UserID        int64                 // 玩家ID
	MailType      int32                 // 邮件类型（普通/广告/官方/命令等，见 MailType 常量）
	Title         string                // 邮件标题
	TitleParams   []string              // 邮件标题参数（多语言格式化参数；下标 i 对应 {i}）
	Content       string                // 邮件正文
	ContentParams []string              // 邮件正文参数（多语言格式化参数；下标 i 对应 {i}）
	SenderID      int64                 // 发送者ID（0=系统）
	SenderName    string                // 发送者名称（如"系统"）
	SenderAvatar  string                // 发送者头像（当非玩家邮件时使用；可存URL/资源名）
	ServerMailID  int64                 // 关联的全服邮件ID（0=个人邮件；>0 表示由全服邮件派生）
	TemplateID    int32                 // 邮件模板ID（0=未使用模板）
	Status        int32                 // 邮件状态（0未读/1已读/2已领取/3已删除）
	IsConvenient  bool                  // 是否可一键领取（true=可一键领取；false=只能单独领取）
	Items         []*MailAttachmentItem `json:"items"` // 附件物品条目列表（业务约定：只有一个附件，直接存 items）
	ExpireTime    int64                 // 过期时间戳（秒，0=永不过期）
	SendTime      int64                 // 发送时间戳（秒）
	ReadTime      int64                 // 阅读时间戳（秒，0=未读/未记录）
	ClaimTime     int64                 // 领取时间戳（秒，0=未领取/未记录）
}

// ServerMail 全服邮件业务模型
type ServerMail struct {
	ServerMailID  int64
	MailType      int32
	Title         string
	Content       string
	ContentParams []string
	TemplateID    int32
	ServerID      int32
	AllianceID    int64                 // 联盟ID（0=非联盟邮件；>0 表示联盟邮件）
	SenderAvatar  string                // 发送者头像（全服邮件展示用；可存URL/资源名）
	UnlockList    []int32               // 解锁条件列表（unlockID数组）
	IsConvenient  bool                  // 是否可一键领取（true=可一键领取；false=只能单独领取）
	Items         []*MailAttachmentItem // 全服邮件附件物品条目列表（业务约定：只有一个附件，直接存 items）
	SendTime      int64
	ExpireTime    int64
	Status        int32
	CreatedBy     string
}
