// File: mailModel.go
// Description: 邮件系统数据模型定义
// Author: 木村
// Create Time: 2026.01

package mail

import (
	"encoding/json"
	"time"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"gorm.io/gorm"
)

// ====== 数据库实体模型 ======

// MailEntity 玩家邮件实体
type MailEntity struct {
	MailID        int64          `gorm:"primaryKey;column:mail_id"`                     // 邮件ID（雪花算法生成）
	UserID        int64          `gorm:"column:user_id;index:idx_user_mail"`            // 玩家ID
	MailType      int32          `gorm:"column:mail_type;default:1"`                    // 邮件类型（1普通 2广告 3官方 4命令 5玩家）
	Title         string         `gorm:"column:title;size:128"`                         // 邮件标题
	TitleParams   string         `gorm:"column:title_params;type:json"`                 // 标题参数JSON（存储 string 数组）
	Content       string         `gorm:"column:content;type:text"`                      // 邮件内容
	ContentParams string         `gorm:"column:content_params;type:json"`               // 内容参数JSON（存储 string 数组）
	SenderID      int64          `gorm:"column:sender_id;default:0"`                    // 发送者ID（0表示系统）
	SenderName    string         `gorm:"column:sender_name;size:64;default:''"`         // 发送者名称
	SenderAvatar  string         `gorm:"column:sender_avatar;size:256;default:''"`      // 发送者头像（URL/资源名）
	ServerMailID  int64          `gorm:"column:server_mail_id;default:0"`               // 关联的全服邮件ID（0表示个人邮件）
	TemplateID    int32          `gorm:"column:template_id;default:0"`                  // 邮件模板ID
	Status        int32          `gorm:"column:status;default:0"`                       // 状态（0未读 1已读 2已领取 3已删除）
	HasAttachment bool           `gorm:"column:has_attachment;default:false"`           // 是否有附件
	IsConvenient  bool           `gorm:"column:is_convenient;type:tinyint(1);not null"` // 是否可一键领取（true=可一键领取；false=只能单独领取）
	Attachments   string         `gorm:"column:attachments;type:json"`                  // 附件物品条目JSON（存储 MailAttachmentItem 数组）
	ExpireTime    int64          `gorm:"column:expire_time;default:0"`                  // 过期时间戳（0表示永不过期）
	SendTime      int64          `gorm:"column:send_time"`                              // 发送时间戳
	ReadTime      int64          `gorm:"column:read_time;default:0"`                    // 阅读时间戳
	ClaimTime     int64          `gorm:"column:claim_time;default:0"`                   // 领取时间戳
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create"`    // 创建时间（仅插入时写入，避免被更新语句改成 0000-00-00）
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime"`              // 更新时间（由 GORM 自动维护）
	DeletedAt     gorm.DeletedAt `gorm:"index"`                                         // 软删除
}

func (MailEntity) TableName() string {
	return "mail"
}

// SetItems 序列化附件物品条目列表为JSON字符串
func (e *MailEntity) SetItems(items []*MailAttachmentItem) error {
	if len(items) == 0 {
		// MySQL JSON 列不允许写入空字符串：会报 Invalid JSON text: "The document is empty."
		// 这里统一写入 "null"（JSON null），GetItems 已兼容处理。
		e.Attachments = "null"
		e.HasAttachment = false
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	e.Attachments = string(data)
	e.HasAttachment = true
	return nil
}

// GetItems 从JSON字符串反序列化为附件物品条目列表
func (e *MailEntity) GetItems() ([]*MailAttachmentItem, error) {
	if e.Attachments == "" || e.Attachments == "null" {
		return []*MailAttachmentItem{}, nil
	}
	var items []*MailAttachmentItem
	err := json.Unmarshal([]byte(e.Attachments), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// encodeStringSliceForJSONColumn 序列化字符串切片为JSON字符串
func encodeStringSliceForJSONColumn(v []string) (string, error) {
	if len(v) == 0 {
		return "null", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeStringSliceFromJSONColumn 从JSON字符串反序列化为字符串切片
func decodeStringSliceFromJSONColumn(raw string) ([]string, error) {
	if raw == "" || raw == "null" {
		return []string{}, nil
	}
	var v []string
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	return v, nil
}

func (e *MailEntity) SetTitleParams(params []string) error {
	s, err := encodeStringSliceForJSONColumn(params)
	if err != nil {
		return err
	}
	e.TitleParams = s
	return nil
}

func (e *MailEntity) GetTitleParams() ([]string, error) {
	return decodeStringSliceFromJSONColumn(e.TitleParams)
}

func (e *MailEntity) SetContentParams(params []string) error {
	s, err := encodeStringSliceForJSONColumn(params)
	if err != nil {
		return err
	}
	e.ContentParams = s
	return nil
}

func (e *MailEntity) GetContentParams() ([]string, error) {
	return decodeStringSliceFromJSONColumn(e.ContentParams)
}

// ServerMailEntity 全服邮件实体
type ServerMailEntity struct {
	ServerMailID  int64     `gorm:"primaryKey;column:server_mail_id"`              // 全服邮件ID（雪花算法生成）
	MailType      int32     `gorm:"column:mail_type;default:1"`                    // 邮件类型
	Title         string    `gorm:"column:title;size:128"`                         // 邮件标题
	TitleParams   string    `gorm:"column:title_params;type:json"`                 // 标题参数JSON（存储 string 数组）
	Content       string    `gorm:"column:content;type:text"`                      // 邮件内容
	ContentParams string    `gorm:"column:content_params;type:json"`               // 内容参数JSON（存储 string 数组）
	TemplateID    int32     `gorm:"column:template_id;default:0"`                  // 邮件模板ID
	ServerID      int32     `gorm:"column:server_id;default:0;index"`              // 服务器ID（0表示全服）
	AllianceID    int64     `gorm:"column:alliance_id;default:0;index"`            // 联盟ID（0表示非联盟邮件）
	SenderAvatar  string    `gorm:"column:sender_avatar;size:256;default:''"`      // 发送者头像（URL/资源名）
	UnlockList    string    `gorm:"column:unlock_list;type:json"`                  // 解锁条件列表JSON格式（存储unlockID数组）
	IsConvenient  bool      `gorm:"column:is_convenient;type:tinyint(1);not null"` // 是否可一键领取（true=可一键领取；false=只能单独领取）
	Attachments   string    `gorm:"column:attachments;type:json"`                  // 附件物品条目JSON（存储 MailAttachmentItem 数组）
	SendTime      int64     `gorm:"column:send_time"`                              // 发送时间戳
	ExpireTime    int64     `gorm:"column:expire_time;default:0"`                  // 过期时间戳（0表示永不过期）
	Status        int32     `gorm:"column:status;default:0"`                       // 状态（0待发送 1已发送 2已过期）
	CreatedBy     string    `gorm:"column:created_by;size:64"`                     // 创建者（GM账号）
	CreatedAt     time.Time `gorm:"column:created_at"`                             // 创建时间
	UpdatedAt     time.Time `gorm:"column:updated_at"`                             // 更新时间
}

func (ServerMailEntity) TableName() string {
	return "server_mail"
}

// SetItems 序列化附件物品条目列表为JSON字符串
func (e *ServerMailEntity) SetItems(items []*MailAttachmentItem) error {
	if len(items) == 0 {
		// MySQL JSON 列不允许写入空字符串
		e.Attachments = "null"
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	e.Attachments = string(data)
	return nil
}

// GetItems 从JSON字符串反序列化为附件物品条目列表
func (e *ServerMailEntity) GetItems() ([]*MailAttachmentItem, error) {
	if e.Attachments == "" || e.Attachments == "null" {
		return []*MailAttachmentItem{}, nil
	}
	var items []*MailAttachmentItem
	err := json.Unmarshal([]byte(e.Attachments), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (e *ServerMailEntity) SetTitleParams(params []string) error {
	s, err := encodeStringSliceForJSONColumn(params)
	if err != nil {
		return err
	}
	e.TitleParams = s
	return nil
}

func (e *ServerMailEntity) GetTitleParams() ([]string, error) {
	return decodeStringSliceFromJSONColumn(e.TitleParams)
}

func (e *ServerMailEntity) SetContentParams(params []string) error {
	s, err := encodeStringSliceForJSONColumn(params)
	if err != nil {
		return err
	}
	e.ContentParams = s
	return nil
}

func (e *ServerMailEntity) GetContentParams() ([]string, error) {
	return decodeStringSliceFromJSONColumn(e.ContentParams)
}

// SetUnlockList 序列化解锁ID列表为JSON字符串
func (e *ServerMailEntity) SetUnlockList(unlockIDs []int32) error {
	if len(unlockIDs) == 0 {
		e.UnlockList = "[]"
		return nil
	}
	data, err := json.Marshal(unlockIDs)
	if err != nil {
		return err
	}
	e.UnlockList = string(data)
	return nil
}

// GetUnlockList 从JSON字符串反序列化为解锁ID列表
func (e *ServerMailEntity) GetUnlockList() ([]int32, error) {
	if e.UnlockList == "" || e.UnlockList == "[]" {
		return []int32{}, nil
	}
	var unlockIDs []int32
	err := json.Unmarshal([]byte(e.UnlockList), &unlockIDs)
	if err != nil {
		return nil, err
	}
	return unlockIDs, nil
}

// ====== 业务逻辑模型 ======

// 使用 logicCommon 中的类型定义，避免循环依赖
type MailAttachmentItem = logicCommon.MailAttachmentItem
type Mail = logicCommon.Mail
type ServerMail = logicCommon.ServerMail

// MailToEntity 将 Mail 转换为数据库实体
func MailToEntity(m *Mail) *MailEntity {
	return buildMailEntityFromMail(m)
}

// MailFromEntity 从数据库实体转换为 Mail
func MailFromEntity(entity *MailEntity) (*Mail, error) {
	m := &Mail{
		MailID:       entity.MailID,
		UserID:       entity.UserID,
		MailType:     entity.MailType,
		Title:        entity.Title,
		Content:      entity.Content,
		SenderID:     entity.SenderID,
		SenderName:   entity.SenderName,
		SenderAvatar: entity.SenderAvatar,
		ServerMailID: entity.ServerMailID,
		TemplateID:   entity.TemplateID,
		Status:       entity.Status,
		IsConvenient: entity.IsConvenient,
		ExpireTime:   entity.ExpireTime,
		SendTime:     entity.SendTime,
		ReadTime:     entity.ReadTime,
		ClaimTime:    entity.ClaimTime,
	}

	items, err := entity.GetItems()
	if err != nil {
		return nil, err
	}
	m.Items = items

	titleParams, err := entity.GetTitleParams()
	if err != nil {
		return nil, err
	}
	m.TitleParams = titleParams
	contentParams, err := entity.GetContentParams()
	if err != nil {
		return nil, err
	}
	m.ContentParams = contentParams
	return m, nil
}

// ServerMailToEntity 将 ServerMail 转换为数据库实体
func ServerMailToEntity(s *ServerMail) *ServerMailEntity {
	return buildServerMailEntityFromServerMail(s)
}

// ServerMailFromEntity 从数据库实体转换为 ServerMail
func ServerMailFromEntity(entity *ServerMailEntity) (*ServerMail, error) {
	s := &ServerMail{
		ServerMailID: entity.ServerMailID,
		MailType:     entity.MailType,
		Title:        entity.Title,
		Content:      entity.Content,
		TemplateID:   entity.TemplateID,
		ServerID:     entity.ServerID,
		AllianceID:   entity.AllianceID,
		SenderAvatar: entity.SenderAvatar,
		IsConvenient: entity.IsConvenient,
		SendTime:     entity.SendTime,
		ExpireTime:   entity.ExpireTime,
		Status:       entity.Status,
		CreatedBy:    entity.CreatedBy,
	}

	items, err := entity.GetItems()
	if err != nil {
		return nil, err
	}
	s.Items = items

	unlockList, err := entity.GetUnlockList()
	if err != nil {
		return nil, err
	}
	s.UnlockList = unlockList
	return s, nil
}

// MailManager 邮件管理器（玩家邮件集合）
type MailManager struct {
	UserID       int64
	Mails        map[int64]*Mail // 邮件映射，key为mailID
	ChangedMails map[int64]*Mail // 变更的邮件
	NewMailIDs   map[int64]bool  // 新建邮件ID（需 INSERT，非 UPDATE）
	DeletedMails map[int64]bool  // 删除的邮件
	Changed      bool            // 是否有变更
}

// NewMailManager 创建邮件管理器
func NewMailManager(userID int64) *MailManager {
	return &MailManager{
		UserID:       userID,
		Mails:        make(map[int64]*Mail),
		ChangedMails: make(map[int64]*Mail),
		NewMailIDs:   make(map[int64]bool),
		DeletedMails: make(map[int64]bool),
		Changed:      false,
	}
}

// AddMail 添加邮件（新建邮件，需 INSERT）
func (mm *MailManager) AddMail(mail *Mail) {
	mm.Mails[mail.MailID] = mail
	mm.ChangedMails[mail.MailID] = mail
	mm.NewMailIDs[mail.MailID] = true
	mm.Changed = true
}

// RemoveMail 移除邮件（同时从 ChangedMails/NewMailIDs 移除，避免 SaveMailManager 对已删邮件执行 UPDATE 导致状态被覆盖）
func (mm *MailManager) RemoveMail(mailID int64) {
	delete(mm.Mails, mailID)
	delete(mm.ChangedMails, mailID)
	delete(mm.NewMailIDs, mailID)
	mm.DeletedMails[mailID] = true
	mm.Changed = true
}

// GetMail 获取邮件
func (mm *MailManager) GetMail(mailID int64) *Mail {
	return mm.Mails[mailID]
}

// ClearChanged 清除变更标记
func (mm *MailManager) ClearChanged() {
	mm.ChangedMails = make(map[int64]*Mail)
	mm.NewMailIDs = make(map[int64]bool)
	mm.DeletedMails = make(map[int64]bool)
	mm.Changed = false
}

// ====== 常量定义 ======

// 邮件状态常量
const (
	MailStatusUnread  = 0 // 未读
	MailStatusRead    = 1 // 已读
	MailStatusClaimed = 2 // 已领取
	MailStatusDeleted = 3 // 已删除
)

// 邮件类型常量
const (
	MailTypeNormal        = 1 // 普通邮件
	MailTypeAdvertisement = 2 // 广告邮件
	MailTypeOfficial      = 3 // 官方邮件
	MailTypeCommand       = 4 // 命令邮件
	MailTypePlayer        = 5 // 玩家邮件
)

// 全服邮件状态常量
const (
	ServerMailStatusPending = 0 // 待发送
	ServerMailStatusSent    = 1 // 已发送
	ServerMailStatusExpired = 2 // 已过期
)

// 附件类型常量
const (
	AttachmentItemTypeItem     = 1 // 道具
	AttachmentItemTypeCurrency = 2 // 货币
	AttachmentItemTypeExp      = 3 // 经验
)

// 邮件容量限制
const (
	MaxMailCount = 100 // 单个玩家最多保存100封邮件
)
