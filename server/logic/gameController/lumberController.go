package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/lumber"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("CityLumber", &LumberController{})
}

// LumberController 伐木场控制器
// 处理伐木场相关的所有客户端请求：查询详情、领取产物、派驻英雄、家具升级
type LumberController struct{}

var _ LogicControllerInterface = (*LumberController)(nil)

// RegisterLogicMessage 注册伐木场相关协议处理器
// 所有伐木场协议都需要 FUNCTION_ID_LUMBERYARD 功能解锁
func (c *LumberController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_LUMBER_DETAIL_REQ, &pb.CityLumberDetailReq{}, CityLumberDetailHandle, enum.FUNCTION_ID_LUMBERYARD)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_LUMBER_COLLECT_REQ, &pb.CityLumberCollectReq{}, CityLumberCollectHandle, enum.FUNCTION_ID_LUMBERYARD)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_CITY_LUMBER_ASSIGN_HERO_REQ, &pb.CityLumberAssignHeroReq{}, CityLumberAssignHeroHandle, enum.FUNCTION_ID_LUMBERYARD)
}

// CityLumberDetailHandle 查询伐木场详情
// 请求：CityLumberDetailReq（无参数）
// 响应：CityLumberDetailResp（包含完整的伐木场信息）
func CityLumberDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("city lumber detail Req", player)

	if _, ok := message.(*pb.CityLumberDetailReq); !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	info, err := lumber.Service.GetDetailInfo(player)
	if err != nil {
		platformLogger.ErrorWithUser("get lumber detail error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_DETAIL_RESP, getCityLumberErrorCode(err))
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_LUMBER_DETAIL_RESP, &pb.CityLumberDetailResp{
		Info: info,
	})
}

// CityLumberCollectHandle 领取暂存产物
// 请求：CityLumberCollectReq（指定要领取的道具ID）
// 响应：CityLumberCollectResp（领取奖励和剩余暂存）
func CityLumberCollectHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("city lumber collect Req", player)

	req, ok := message.(*pb.CityLumberCollectReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_COLLECT_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	reward, stored, err := lumber.Service.Collect(player, req.ItemId)
	if err != nil {
		errCode := getCityLumberErrorCode(err)
		platformLogger.ErrorWithUser("lumber collect error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_COLLECT_RESP, errCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_LUMBER_COLLECT_RESP, &pb.CityLumberCollectResp{
		Reward: reward,
		Stored: stored,
	})
}

// CityLumberAssignHeroHandle 派驻英雄
// 请求：CityLumberAssignHeroReq（新的英雄OwnID列表）
// 响应：CityLumberAssignHeroResp（最终派驻列表和当前暂存）
func CityLumberAssignHeroHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("city lumber assign hero Req", player)

	req, ok := message.(*pb.CityLumberAssignHeroReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_ASSIGN_HERO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	heroOwnIDs := req.GetHeroOwnId()
	if req.GetClearAll() {
		heroOwnIDs = nil
	}
	_, _, err := lumber.Service.AssignHero(player, heroOwnIDs)
	if err != nil {
		errCode := getCityLumberErrorCode(err)
		platformLogger.ErrorWithUser("lumber assign hero error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_CITY_LUMBER_ASSIGN_HERO_RESP, errCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_CITY_LUMBER_ASSIGN_HERO_RESP, &pb.CityLumberAssignHeroResp{
		IsSuccess: true,
	})
}

func getCityLumberErrorCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	switch err.Error() {
	case "building not built", "architecture type not supported":
		return pb.ERROR_CODE_CITY_LUMBER_NOT_BUILT // 伐木场未建造
	case "store empty":
		return pb.ERROR_CODE_CITY_LUMBER_STORE_EMPTY // 暂存为空，无可领取
	case "hero invalid":
		return pb.ERROR_CODE_CITY_LUMBER_HERO_INVALID // 英雄不存在或已删除
	case "hero duplicate":
		return pb.ERROR_CODE_CITY_LUMBER_HERO_DUPLICATE // 英雄重复派驻
	case "hero slot not enough":
		return pb.ERROR_CODE_CITY_LUMBER_HERO_SLOT_NOT_ENOUGH // 英雄槽位不足
	case "furniture not unlock":
		return pb.ERROR_CODE_CITY_FURNITURE_NOT_UNLOCK // 家具未解锁
	case "furniture level max":
		return pb.ERROR_CODE_CITY_FURNITURE_LEVEL_MAX // 家具已达最大等级
	case "furniture type error":
		return pb.ERROR_CODE_CITY_FURNITURE_TYPE_ERROR // 家具类型不合法
	case "item not enough":
		return pb.ERROR_CODE_ITEM_NOT_ENOUGH // 材料不足
	default:
		return pb.ERROR_CODE_SYSTEM_ERROR // 系统错误
	}
}
