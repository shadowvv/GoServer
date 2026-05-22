// File: mailController.go
// Description: 邮件系统控制器（处理客户端消息）
// Author: 木村
// Create Time: 2026.01

package gameController

import (
	"fmt"
	"strings"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/mail"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("mail", &MailController{})
}

type MailController struct {
}

var _ LogicControllerInterface = (*MailController)(nil)

var mailService logicCommon.MailServiceInterface

// InitMailService 初始化邮件服务（含 Redis 心跳：检测其他系统插库后的刷新 key，从 DB 重载缓存）
func InitMailService(sessionManager logicCommon.SessionManagerInterface, messageSender logicCommon.MessageSenderInterface, unlock logicCommon.UnlockServiceInterface, serverInfoService *gameServerInfoService.GameServerInfoService) {
	svc := mail.NewMailService(sessionManager, messageSender, unlock, serverInfoService)
	mailService = svc
	svc.StartRefreshHeartbeat(5 * time.Second)
}

// GetMailService 获取邮件服务（供其他controller使用）
func GetMailService() logicCommon.MailServiceInterface {
	return mailService
}

// RegisterLogicMessage 注册邮件消息处理
func (m *MailController) RegisterLogicMessage() {

	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_LIST_REQ, &pb.MailListReq{}, MailListHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_DETAIL_REQ, &pb.MailDetailReq{}, MailDetailHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_READ_REQ, &pb.MailReadReq{}, MailReadHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_CLAIM_REQ, &pb.MailClaimReq{}, MailClaimHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_CLAIM_ALL_REQ, &pb.MailClaimAllReq{}, MailClaimAllHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_DELETE_REQ, &pb.MailDeleteReq{}, MailDeleteHandle, enum.FUNCTION_ID_MAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_MAIL_DELETE_CLAIMED_REQ, &pb.MailDeleteClaimedReq{}, MailDeleteClaimedHandle, enum.FUNCTION_ID_MAIL)

	platformLogger.InfoWithUser("[MailController] All mail messages registered successfully", nil)
}

// getMailErrorCode 将邮件服务错误映射到 ERROR_CODE
func getMailErrorCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "mail not found") {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if strings.Contains(errMsg, "mail expired") {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if strings.Contains(errMsg, "already claimed") {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if strings.Contains(errMsg, "no attachment") || strings.Contains(errMsg, "attachment items is empty") {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if strings.Contains(errMsg, "player not found") {
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
	if strings.Contains(errMsg, "only delete read and claimed mails") {
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM
	}
	if strings.Contains(errMsg, "inventory") || strings.Contains(errMsg, "item") {
		return pb.ERROR_CODE_ADD_ITEM_ERROR
	}
	return pb.ERROR_CODE_SYSTEM_ERROR
}

// MailListHandle 处理获取邮件列表请求
func MailListHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MailListReq)
	if !ok {
		platformLogger.ErrorWithUser("MailListHandle: invalid message type", player, fmt.Errorf("invalid message type"))
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	platformLogger.InfoWithUser(fmt.Sprintf("get mail list: mailType=%d status=%d page=%d pageSize=%d", req.GetMailType(), req.GetStatus(), req.GetPage(), req.GetPageSize()), player)

	// 参数验证和默认值
	mailType := req.GetMailType()
	status := req.GetStatus()
	page := req.GetPage()
	pageSize := req.GetPageSize()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 调用服务获取邮件列表
	mails, total, err := mailService.GetMailList(player.GetUserId(), int32(mailType), int32(status), page, pageSize)
	if err != nil {
		platformLogger.ErrorWithUser(fmt.Sprintf("get mail list failed: %v", err), player, err)
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_LIST_RESP, errorCode)
		return
	}

	platformLogger.InfoWithUser(fmt.Sprintf("get mail list success: total=%d mails=%d", total, len(mails)), player)

	// 转换为protobuf格式
	mailInfos := make([]*pb.MailInfo, 0, len(mails))
	for _, m := range mails {
		var pbAttachment *pb.MailAttachmentInfo
		if len(m.Items) > 0 {
			pbItems := make([]*pb.MailAttachmentItem, 0, len(m.Items))
			for _, it := range m.Items {
				if it == nil {
					continue
				}
				pbItems = append(pbItems, &pb.MailAttachmentItem{
					Type: it.Type,
					Id:   it.ID,
					Num:  it.Num,
				})
			}
			pbAttachment = &pb.MailAttachmentInfo{Items: pbItems}
		}

		mailInfo := &pb.MailInfo{
			MailId:        m.MailID,
			MailType:      pb.MailType(m.MailType),
			Title:         m.Title,
			Content:       m.Content,
			SenderId:      m.SenderID,
			SenderName:    m.SenderName,
			SenderAvatar:  m.SenderAvatar,
			Status:        pb.MailStatus(m.Status),
			HasAttachment: len(m.Items) > 0,
			IsConvenient:  m.IsConvenient,
			TemplateId:    m.TemplateID,
			TitleParams:   m.TitleParams,
			ContentParams: m.ContentParams,
			AttachmentCount: func() int32 {
				if len(m.Items) > 0 {
					return 1
				}
				return 0
			}(),
			Attachment: pbAttachment,
			ExpireTime: m.ExpireTime,
			SendTime:   m.SendTime,
			ReadTime:   m.ReadTime,
			ClaimTime:  m.ClaimTime,
		}
		mailInfos = append(mailInfos, mailInfo)
	}

	// 计算未读数量
	unreadCount := int32(0)
	unClaimedCount := int32(0)
	for _, m := range mails {
		if m.Status == mail.MailStatusUnread {
			unreadCount++
		}
		if len(m.Items) > 0 && m.Status < mail.MailStatusClaimed {
			unClaimedCount++
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_LIST_RESP, &pb.MailListResp{
		Mails:          mailInfos,
		Total:          total,
		UnreadCount:    unreadCount,
		UnClaimedCount: unClaimedCount,
	})
}

// MailDetailHandle 处理获取邮件详情请求
func MailDetailHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MailDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	platformLogger.InfoWithUser("get mail detail", player)

	mailDetail, err := mailService.GetMailDetail(player.GetUserId(), req.GetMailId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_DETAIL_RESP, errorCode)
		return
	}

	// 转换为protobuf格式（附件信息：只有一个，直接用 items）
	var pbAttachment *pb.MailAttachmentInfo
	if len(mailDetail.Items) > 0 {
		pbItems := make([]*pb.MailAttachmentItem, 0, len(mailDetail.Items))
		for _, it := range mailDetail.Items {
			if it == nil {
				continue
			}
			pbItems = append(pbItems, &pb.MailAttachmentItem{
				Type: it.Type,
				Id:   it.ID,
				Num:  it.Num,
			})
		}
		pbAttachment = &pb.MailAttachmentInfo{Items: pbItems}
	}

	mailInfo := &pb.MailInfo{
		MailId:        mailDetail.MailID,
		MailType:      pb.MailType(mailDetail.MailType),
		Title:         mailDetail.Title,
		Content:       mailDetail.Content,
		SenderId:      mailDetail.SenderID,
		SenderName:    mailDetail.SenderName,
		SenderAvatar:  mailDetail.SenderAvatar,
		Status:        pb.MailStatus(mailDetail.Status),
		HasAttachment: len(mailDetail.Items) > 0,
		IsConvenient:  mailDetail.IsConvenient,
		TemplateId:    mailDetail.TemplateID,
		TitleParams:   mailDetail.TitleParams,
		ContentParams: mailDetail.ContentParams,
		AttachmentCount: func() int32 {
			if len(mailDetail.Items) > 0 {
				return 1
			}
			return 0
		}(),
		Attachment: pbAttachment,
		ExpireTime: mailDetail.ExpireTime,
		SendTime:   mailDetail.SendTime,
		ReadTime:   mailDetail.ReadTime,
		ClaimTime:  mailDetail.ClaimTime,
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_DETAIL_RESP, &pb.MailDetailResp{
		Mail: mailInfo,
	})
}

// MailReadHandle 处理阅读邮件请求
func MailReadHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MailReadReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_READ_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	platformLogger.InfoWithUser("read mail", player)

	err := mailService.ReadMail(player.GetUserId(), req.GetMailId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_READ_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_READ_RESP, &pb.MailReadResp{})
}

// MailClaimHandle 处理领取附件请求
func MailClaimHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MailClaimReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_CLAIM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	platformLogger.InfoWithUser("claim mail attachment", player)

	claimedItems, err := mailService.ClaimMailAttachment(player.GetUserId(), req.GetMailId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_CLAIM_RESP, errorCode)
		return
	}

	// 转换为protobuf格式（当前业务：一次性领取，不存在部分成功）
	pbItems := make([]*pb.MailAttachmentItem, 0, len(claimedItems))
	for _, it := range claimedItems {
		if it == nil {
			continue
		}
		pbItems = append(pbItems, &pb.MailAttachmentItem{
			Type: it.Type,
			Id:   it.ID,
			Num:  it.Num,
		})
	}
	claimedAttachment := &pb.MailAttachmentInfo{Items: pbItems}
	failedAttachment := (*pb.MailAttachmentInfo)(nil)

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_CLAIM_RESP, &pb.MailClaimResp{
		ClaimedAttachment: claimedAttachment,
		FailedAttachment:  failedAttachment,
		TotalCount:        int32(len(claimedItems)),
		ClaimedCount:      int32(len(claimedItems)),
		FailedCount:       0,
	})
}

// MailClaimAllHandle 处理一键领取所有附件请求
func MailClaimAllHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("claim all mail attachments", player)

	claimedCount, err := mailService.ClaimAllMailAttachments(player.GetUserId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_CLAIM_ALL_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_CLAIM_ALL_RESP, &pb.MailClaimAllResp{
		ClaimedCount: claimedCount,
	})
}

// MailDeleteHandle 处理删除邮件请求
func MailDeleteHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.MailDeleteReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_DELETE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	platformLogger.InfoWithUser("delete mail", player)

	err := mailService.DeleteMail(player.GetUserId(), req.GetMailId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_DELETE_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_DELETE_RESP, &pb.MailDeleteResp{})
}

// MailDeleteClaimedHandle 处理一键删除已领取邮件请求
func MailDeleteClaimedHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("delete claimed mails", player)

	deletedCount, err := mailService.DeleteClaimedMails(player.GetUserId())
	if err != nil {
		errorCode := getMailErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_MAIL_DELETE_CLAIMED_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_MAIL_DELETE_CLAIMED_RESP, &pb.MailDeleteClaimedResp{
		DeletedCount: deletedCount,
	})
}
