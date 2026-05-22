package gameController

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/wordFilter"
	"golang.org/x/net/context"
	"gorm.io/gorm"

	"github.com/drop/GoServer/server/tool"

	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

const (
	CHAT_CD         = 3 * 1000
	MAX_CHAT_LENGTH = 150
)

func init() {
	RegisterController("Chat", &ChatController{})
}

type ChatController struct {
}

var _ LogicControllerInterface = (*ChatController)(nil)

func (c *ChatController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_REQ, &pb.SendChatMessageReq{}, SendChatMessageHandle, enum.FUNCTION_ID_CHAT)
}

func SendChatMessageHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.SendChatMessageReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	msgDetail := req.GetChatMessage()
	msg := msgDetail.GetMessageContent()

	if msgDetail.GetSendType() == int32(enum.BROADCAST_TYPE_SERVER_ID) {
		if tool.UnixNowMilli()-player.LastSendChatMessageTime < CHAT_CD {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_RESP, pb.ERROR_CODE_SEND_MESSAGE_TIME_INTERVAL_SHORT)
			return
		}
	}
	if utf8.RuneCountInString(msg) > MAX_CHAT_LENGTH {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_RESP, pb.ERROR_CODE_CONTEXT_IS_LONG_LONG)
		return
	}
	if wordFilter.HasSensitive(msg) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_RESP, pb.ERROR_CODE_CONTEXT_NOT_STANDARDIZATION)
		return
	}

	checkBanRecode, err := easyDB.GetServerEntityByWhere[model.UserBanRecordEntity](map[string]interface{}{"account": player.GetUserAccount(), "server_id": player.GetUserServerId()})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		platformLogger.ErrorWithUser("[chat] get ban record err", player, err)
	}
	// 被ban之后直接返回
	if checkBanRecode != nil && checkBanRecode.StartTime < tool.UnixNowMilli() && checkBanRecode.EndTime > tool.UnixNowMilli() {
		platformLogger.InfoWithUser("[chat] user is banned chat", player)
		return
	}
	player.LastSendChatMessageTime = tool.UnixNowMilli()
	sendMsg := &pb.PushReceivedChatMessage{
		ChatMessage: &pb.ChatMessage{
			PlayerInfo: &pb.PlayerBasicInfo{
				UserId:   player.GetUserId(),
				NickName: player.User.GetNickname(),
			},
			SendType:       int32(enum.BROADCAST_TYPE_SERVER_ID),
			SendTime:       tool.UnixNowMilli(),
			ToObjectId:     msgDetail.GetToObjectId(),
			MessageContent: msgDetail.GetMessageContent(),
		},
	}
	chatKey := enum.GetChatKey(msgDetail.GetSendType(), msgDetail.GetToObjectId())

	ctx := context.Background()
	sendMsgJson, err := json.Marshal(sendMsg)
	if err != nil {
		platformLogger.ErrorWithUser("[chat] send json err", player, err)
		return
	}
	err = dbService.RDB.LPush(ctx, chatKey, sendMsgJson).Err()
	if err != nil {
		platformLogger.ErrorWithUser("[chat] msg in redis err", player, err)
		return
	}
	err = dbService.RDB.LTrim(ctx, chatKey, 0, 99).Err()
	if err != nil {
		platformLogger.ErrorWithUser("[chat] msg in redis Ltrim err", player, err)
		return
	}

	sendMsg.ChatMessage.PlayerInfo.HeadId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHead)
	sendMsg.ChatMessage.PlayerInfo.HeadFrameId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHeadFrame)
	sendMsg.ChatMessage.PlayerInfo.TitleId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeTitle)
	sendMsg.ChatMessage.PlayerInfo.BubbleId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeBubble)
	sendMsg.ChatMessage.PlayerInfo.ImageId = player.AppearanceModel.GetWearAppearance(enum.AvatarTypeImage)

	messageSender.SendMessage(player, pb.MESSAGE_ID_SEND_CHAT_MESSAGE_RESP, &pb.SendChatMessageResp{})

	switch msgDetail.GetSendType() {
	case int32(enum.BROADCAST_TYPE_SERVER_ID):
		messageSender.Broadcast(pb.MESSAGE_ID_PUSH_RECEIVED_CHAT_MESSAGE, sendMsg, enum.BroadcastType(msgDetail.GetSendType()), int32(msgDetail.GetToObjectId()))

	}
}
