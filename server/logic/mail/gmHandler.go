// File: gmHandler.go
// Description: 邮件系统GM命令处理器
// Author: 木村
// Create Time: 2026.01

package mail

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

var _ logicCommon.GMCommandHandler = (*MailGMHandler)(nil)

// MailGMHandler 邮件系统GM命令处理器
type MailGMHandler struct {
	mailService logicCommon.MailServiceInterface
}

// NewMailGMHandler 创建邮件系统GM命令处理器
func NewMailGMHandler(mailService logicCommon.MailServiceInterface) *MailGMHandler {
	return &MailGMHandler{
		mailService: mailService,
	}
}

// GetSupportedCommands 返回支持的命令类型
func (h *MailGMHandler) GetSupportedCommands() []pb.GMCommandType {
	return []pb.GMCommandType{
		pb.GMCommandType_GM_CMD_SEND_MAIL,
		pb.GMCommandType_GM_CMD_SEND_SERVER_MAIL,
	}
}

// HandleCommand 处理GM命令
func (h *MailGMHandler) HandleCommand(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	cmdType := req.GetCmdType()

	switch cmdType {
	case pb.GMCommandType_GM_CMD_SEND_MAIL:
		return h.handleSendMail(req, userId)
	case pb.GMCommandType_GM_CMD_SEND_SERVER_MAIL:
		return h.handleSendServerMail(req, userId)
	default:
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_CMD,
			Message: fmt.Sprintf("邮件系统不支持的命令类型: %d", cmdType),
		}
	}
}

// handleSendMail 处理发送个人邮件命令
func (h *MailGMHandler) handleSendMail(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	// 从extraParams解析邮件参数（JSON格式）
	extraParams := req.GetExtraParams()
	if extraParams == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "邮件参数不能为空，请通过extraParams传递JSON格式的邮件信息",
		}
	}

	var mailParams struct {
		Title        string                `json:"title"`
		Content      string                `json:"content"`
		MailType     int32                 `json:"mail_type"`
		SenderAvatar string                `json:"sender_avatar"` // 发送者头像（可选）
		Items        []*MailAttachmentItem `json:"items"`         // 附件物品条目（业务约定：只有一个附件，直接传 items）
		ExpireDays   int32                 `json:"expire_days"`
		IsConvenient *bool                 `json:"is_convenient"` // 是否可一键领取（可选，默认true）
	}

	if err := json.Unmarshal([]byte(extraParams), &mailParams); err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("解析邮件参数失败: %v", err),
		}
	}

	// 参数验证
	if mailParams.Title == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "邮件标题不能为空",
		}
	}
	if mailParams.Content == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "邮件内容不能为空",
		}
	}
	if mailParams.MailType <= 0 {
		mailParams.MailType = MailTypeNormal
	}
	if mailParams.ExpireDays <= 0 {
		mailParams.ExpireDays = 7 // 默认7天
	}

	// 计算过期时间
	expireTime := int64(0)
	if mailParams.ExpireDays > 0 {
		expireTime = time.Now().Unix() + int64(mailParams.ExpireDays)*24*3600
	}

	// 创建邮件对象
	isConvenient := true
	if mailParams.IsConvenient != nil {
		isConvenient = *mailParams.IsConvenient
	}
	mail := &Mail{
		UserID:       userId,
		MailType:     mailParams.MailType,
		Title:        mailParams.Title,
		Content:      mailParams.Content,
		SenderID:     0, // 系统发送
		SenderName:   "系统",
		SenderAvatar: mailParams.SenderAvatar,
		Status:       MailStatusUnread,
		IsConvenient: isConvenient,
		Items:        mailParams.Items,
		ExpireTime:   expireTime,
		SendTime:     time.Now().Unix(),
	}

	// 发送邮件
	mailID, err := h.mailService.SendMail(userId, mail)
	if err != nil {
		logger.ErrorWithZapFields("[MailGMHandler] Failed to send mail", zap.Error(err), zap.Int64("user_id", userId))
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_FAILED,
			Message: fmt.Sprintf("发送邮件失败: %v", err),
		}
	}

	return &pb.MessageGmResp{
		Result:  pb.GMResult_GM_RESULT_SUCCESS,
		Message: "发送邮件成功",
		ExtraData: map[string]string{
			"mail_id": fmt.Sprintf("%d", mailID),
			"user_id": fmt.Sprintf("%d", userId),
			"title":   mailParams.Title,
		},
	}
}

// handleSendServerMail 处理发送全服邮件命令
func (h *MailGMHandler) handleSendServerMail(req *pb.MessageGmReq, userId int64) *pb.MessageGmResp {
	// 从extraParams解析全服邮件参数（JSON格式）
	extraParams := req.GetExtraParams()
	if extraParams == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "全服邮件参数不能为空，请通过extraParams传递JSON格式的邮件信息",
		}
	}

	var serverMailParams struct {
		Title        string                `json:"title"`
		Content      string                `json:"content"`
		MailType     int32                 `json:"mail_type"`
		ServerID     int32                 `json:"server_id"`
		UnlockList   []int32               `json:"unlock_list"` // 解锁条件列表（unlockID数组）
		Items        []*MailAttachmentItem `json:"items"`       // 附件物品条目（业务约定：只有一个附件，直接传 items）
		ExpireDays   int32                 `json:"expire_days"`
		CreatedBy    string                `json:"created_by"`
		IsConvenient *bool                 `json:"is_convenient"` // 是否可一键领取（可选，默认true）
	}

	if err := json.Unmarshal([]byte(extraParams), &serverMailParams); err != nil {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: fmt.Sprintf("解析全服邮件参数失败: %v", err),
		}
	}

	// 参数验证
	if serverMailParams.Title == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "邮件标题不能为空",
		}
	}
	if serverMailParams.Content == "" {
		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_INVALID_PARAM,
			Message: "邮件内容不能为空",
		}
	}
	if serverMailParams.MailType <= 0 {
		serverMailParams.MailType = MailTypeNormal
	}
	if serverMailParams.ExpireDays <= 0 {
		serverMailParams.ExpireDays = 7 // 默认7天
	}
	if serverMailParams.CreatedBy == "" {
		serverMailParams.CreatedBy = "GM"
	}

	return nil

	// TODO: 实现全服邮件发送，需要proto代码生成后使用正确的类型
	// 这里暂时返回错误，等proto代码生成后再实现
	//return &pb.MessageGmResp{
	//	Result:  pb.GMResult_GM_RESULT_FAILED,
	//	Message: "发送全服邮件功能待proto代码生成后实现",
	//}, nil
	//
	// 以下代码等proto生成后启用
	/*
	       	// 转换附件为protobuf格式（业务约定：一封邮件只有一个附件，附件内 items 列表）
	       	var pbAttachment *pb.MailAttachmentInfo
	       	if len(serverMailParams.Items) > 0 {
	       		pbItems := make([]*pb.MailAttachmentItem, 0, len(serverMailParams.Items))
	       		for _, it := range serverMailParams.Items {
	       			pbItems = append(pbItems, &pb.MailAttachmentItem{
	       				Type: it.Type,
	       				Id:   it.ID,
	       				Num:  it.Num,
	       			})
	       		}
	       		pbAttachment = &pb.MailAttachmentInfo{
	       			Items:     pbItems,
	       		}
	       	}

	   		// 创建全服邮件请求
	   		serverMailReq := &pb.SendServerMailRequest{
	   			MailType:     serverMailParams.MailType,
	   			Title:        serverMailParams.Title,
	   			Content:      serverMailParams.Content,
	   			TemplateId:   0,
	   			ServerId:     serverMailParams.ServerID,
	   			UnlockList:   serverMailParams.UnlockList,
	   			Attachment:   pbAttachment,
	   			ExpireDays:   serverMailParams.ExpireDays,
	   			CreatedBy:    serverMailParams.CreatedBy,
	   		}

	   		// 发送全服邮件
	   		serverMailID, err := h.mailService.SendServerMail(serverMailReq)
	*/
	// 等proto生成后启用
	/*
		if err != nil {
			logger.ErrorWithZapFields("[MailGMHandler] Failed to send server mail", zap.ErrorWithZapFields(err))
			return &pb.MessageGmResp{
				Result:  pb.GMResult_GM_RESULT_FAILED,
				Message: fmt.Sprintf("发送全服邮件失败: %v", err),
			}
		}

		return &pb.MessageGmResp{
			Result:  pb.GMResult_GM_RESULT_SUCCESS,
			Message: "发送全服邮件成功",
			ExtraData: map[string]string{
				"server_mail_id": fmt.Sprintf("%d", serverMailID),
				"title":          serverMailParams.Title,
				"server_id":      fmt.Sprintf("%d", serverMailParams.ServerID),
			},
		}
	*/
}
