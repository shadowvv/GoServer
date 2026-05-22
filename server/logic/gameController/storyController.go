package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"google.golang.org/protobuf/proto"
)

// 注册 Controller
func init() {
	RegisterController("story", &StoryController{})
}

var _ LogicControllerInterface = (*StoryController)(nil)

type StoryController struct{}

// 注册协议
func (p *StoryController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_STORY_FINISH_REQ, &pb.StoryFinishReq{}, StoryFinishHandler, enum.FUNCTION_ID_NONE)
}

func StoryFinishHandler(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.StoryFinishReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STORY_FINISH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	newId := req.GetStoryId()
	if newId <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STORY_FINISH_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	// 记录这次触发（次数 +1）
	if player.StoryTriggerModel != nil {
		player.StoryTriggerModel.AddStoryTrigger(newId)
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_STORY_FINISH_RESP, &pb.StoryFinishResp{})
}
