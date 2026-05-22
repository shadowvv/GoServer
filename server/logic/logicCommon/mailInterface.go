// File: mailInterface.go
// Description: 邮件系统服务接口定义
// Author: 木村
// Create Time: 2026.01

package logicCommon

import "github.com/drop/GoServer/server/logic/gameConfig"

// MailServiceInterface 邮件服务接口
type MailServiceInterface interface {
	// ====== 基础操作 ======

	// SendMail 发送邮件（玩家须在线，否则返回错误）
	// userId: 玩家ID
	// mail: 邮件对象，mail.Attachments 支持多个附件（数组）
	// 返回邮件ID和错误信息
	SendMail(userId int64, mail *Mail) (int64, error)

	// SendMailToUserId 向指定玩家发邮件（不要求在线，离线也写入并返回成功；在线则推送红点）
	SendMailToUserId(userId int64, mail *Mail) (int64, error)

	// SendMailByTemplateID 根据邮件模板ID（mailContent 表 id）向指定玩家发送一封邮件
	SendMailByTemplateID(userId int64, templateID int32) (int64, error)

	// SendRewardMailByTemplateID 更改邮件模板的奖励（mailContent 表 id）向指定玩家发送一封邮件
	// titleParams/contentParams: 多语言格式化参数（下标 i 对应 {i}）
	SendRewardMailByTemplateID(userId int64, templateID int32, reward []*gameConfig.ItemConfig, titleParams []string, contentParams []string) (int64, error)

	// SendRewardMailByTemplateIDAndTime 根据邮件模板ID（mailContent 表 id）和时间向指定玩家发送一封邮件
	SendRewardMailByTemplateIDAndTime(userId int64, templateID int32, timestamp int64, reward []*gameConfig.ItemConfig, titleParams []string, contentParams []string) (int64, error)

	// GetMailList 获取邮件列表
	// userId: 玩家ID
	// mailType: 邮件类型（0表示全部）
	// status: 状态筛选（0表示全部）
	// page: 页码（从1开始）
	// pageSize: 每页数量
	// 返回邮件列表、总数和错误信息
	GetMailList(userId int64, mailType int32, status int32, page int32, pageSize int32) ([]*Mail, int32, error)

	// GetMailDetail 获取邮件详情
	// userId: 玩家ID
	// mailId: 邮件ID
	// 返回的Mail包含所有附件列表（Attachments []*MailAttachment）
	GetMailDetail(userId int64, mailId int64) (*Mail, error)

	// ReadMail 阅读邮件（标记为已读）
	// userId: 玩家ID
	// mailId: 邮件ID
	// 返回错误信息
	ReadMail(userId int64, mailId int64) error

	// ClaimMailAttachment 领取邮件附件（业务：一次性领取 items）
	// userId: 玩家ID
	// mailId: 邮件ID
	// 返回成功领取的 items 列表和错误信息
	ClaimMailAttachment(userId int64, mailId int64) ([]*MailAttachmentItem, error)

	// ClaimAllMailAttachments 一键领取所有邮件的所有附件
	// userId: 玩家ID
	// 返回成功领取数量和错误信息
	ClaimAllMailAttachments(userId int64) (int32, error)

	// DeleteMail 删除邮件
	// userId: 玩家ID
	// mailId: 邮件ID
	// 返回错误信息
	DeleteMail(userId int64, mailId int64) error

	// DeleteClaimedMails 一键删除已领取邮件
	// userId: 玩家ID
	// 返回删除数量和错误信息
	DeleteClaimedMails(userId int64) (int32, error)

	// ====== 全服邮件操作 ======

	// SendServerMail 发送全服邮件
	// req: 全服邮件请求（使用interface{}避免循环依赖，实际类型为*pb.SendServerMailRequest）
	// 返回全服邮件ID和错误信息
	SendServerMail(req interface{}) (int64, error)

	// GetServerMailList 获取全服邮件列表（供GM后台查询）
	// page: 页码
	// pageSize: 每页数量
	// 返回全服邮件列表、总数和错误信息
	GetServerMailList(page int32, pageSize int32) ([]*ServerMail, int32, error)

	// ====== 玩家登录时处理 ======

	// OnPlayerLogin 玩家登录时处理
	// userId: 玩家ID
	// 检查并推送全服邮件红点通知
	// 返回错误信息
	OnPlayerLogin(userId int64) error

	// ====== 定时任务 ======

	// CleanExpiredMails 清理过期邮件
	// 返回错误信息
	CleanExpiredMails() error

	// ProcessPendingServerMails 处理待发送的全服邮件
	// 返回错误信息
	ProcessPendingServerMails() error
}
