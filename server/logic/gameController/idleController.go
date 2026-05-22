// File: idleController.go
// Description: 挂机奖励系统控制器
// Author: 木村凉太
// Create Time: 2026.02

package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/idle"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("idle", &IdleController{})
}

type IdleController struct{}

var _ LogicControllerInterface = (*IdleController)(nil)

// RegisterLogicMessage 注册挂机奖励路由
func (i *IdleController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_IDLE_INFO_REQ, &pb.IdleInfoReq{}, GetIdleInfoHandle, enum.FUNCTION_ID_IDLE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_IDLE_CLAIM_REWARD_REQ, &pb.IdleClaimRewardReq{}, ClaimIdleRewardHandle, enum.FUNCTION_ID_IDLE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_IDLE_QUICK_CLAIM_REQ, &pb.IdleQuickClaimReq{}, QuickClaimIdleHandle, enum.FUNCTION_ID_IDLE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_IDLE_UPGRADE_LEVEL_REQ, &pb.IdleUpgradeLevelReq{}, UpgradeIdleLevelHandle, enum.FUNCTION_ID_IDLE)
}

var idleService *idle.IdleServer

// InitIdleService 初始化挂机奖励服务（需在 unlockService 就绪后调用）
func InitIdleService() {
	idleService = idle.NewIdleServer(unlockService)
}

// getIdleErrorCode 将错误信息映射到错误码
func getIdleErrorCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	errMsg := err.Error()
	switch {
	case errMsg == "player not found":
		return pb.ERROR_CODE_IDLE_PLAYER_NOT_FOUND
	case errMsg == "idle model not loaded":
		return pb.ERROR_CODE_IDLE_MODEL_NOT_LOADED
	case errMsg == "idle level config not found":
		return pb.ERROR_CODE_IDLE_LEVEL_CONFIG_NOT_FOUND
	case errMsg == "no reward to claim" || errMsg == "没有可领取的奖励":
		return pb.ERROR_CODE_IDLE_NO_REWARD_TO_CLAIM
	case errMsg == "upgrade condition not met" || errMsg == "升级条件未满足":
		return pb.ERROR_CODE_IDLE_UPGRADE_CONDITION_NOT_MET
	case errMsg == "quick claim config not found":
		return pb.ERROR_CODE_IDLE_QUICK_CLAIM_CONFIG_NOT_FOUND
	case errMsg == "quick claim not unlocked":
		return pb.ERROR_CODE_IDLE_QUICK_CLAIM_NOT_UNLOCKED
	case errMsg == "diamond not enough":
		return pb.ERROR_CODE_IDLE_DIAMOND_NOT_ENOUGH
	default:
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
}

// GetIdleInfoHandle 获取挂机信息
func GetIdleInfoHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.IdleInfoReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("get idle info", player)

	info, err := idleService.GetIdleInfo(player)
	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_INFO_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_IDLE_INFO_RESP, &pb.IdleInfoResp{
		ErrorCode: pb.ERROR_CODE_SUCCESS,
		Info:      info,
	})
}

// ClaimIdleRewardHandle 领取挂机奖励
func ClaimIdleRewardHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.IdleClaimRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_CLAIM_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("claim idle reward", player)

	rewards, err := idleService.ClaimReward(player)
	info, err := idleService.GetIdleInfo(player)
	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_INFO_RESP, errorCode)
		return
	}

	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_CLAIM_REWARD_RESP, errorCode)
		return
	}

	// 操作日志：挂机奖励领取（普通领取）
	operationLogService.OnUserIdleRewardClaim(player.GetUserId(), 0)

	messageSender.SendMessage(player, pb.MESSAGE_ID_IDLE_CLAIM_REWARD_RESP, &pb.IdleClaimRewardResp{
		ErrorCode:       pb.ERROR_CODE_SUCCESS,
		Rewards:         rewards,
		AccumulatedTime: info.AccumulatedTime,
	})
}

// QuickClaimIdleHandle 快速领取
func QuickClaimIdleHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.IdleQuickClaimReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_QUICK_CLAIM_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("quick claim idle", player)

	rewards, nextRewards, err := idleService.QuickClaim(player, req.GetIdleType())
	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_QUICK_CLAIM_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_IDLE_QUICK_CLAIM_RESP, &pb.IdleQuickClaimResp{
		ErrorCode:   pb.ERROR_CODE_SUCCESS,
		Rewards:     rewards,
		NextRewards: nextRewards,
	})

	operationLogService.OnUserIdleRewardClaim(player.GetUserId(), req.GetIdleType())

	eventBusService.SubmitQuickClaimMachineRewardEvent(player.GetUserId(), player.GetUserServerId())
}

// UpgradeIdleLevelHandle 升级挂机等级
func UpgradeIdleLevelHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.IdleUpgradeLevelReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_UPGRADE_LEVEL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("upgrade idle level", player)

	err := idleService.UpgradeIdleLevel(player)
	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_UPGRADE_LEVEL_RESP, errorCode)
		return
	}

	newLevel := int32(1)
	if player.IdleModel != nil && player.IdleModel.Entity != nil {
		newLevel = player.IdleModel.Entity.IdleLevel
	}
	info, err := idleService.GetIdleInfo(player)
	if err != nil {
		errorCode := getIdleErrorCode(err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_IDLE_INFO_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_IDLE_UPGRADE_LEVEL_RESP, &pb.IdleUpgradeLevelResp{
		ErrorCode:  pb.ERROR_CODE_SUCCESS,
		NewLevel:   newLevel,
		CanUpgrade: info.CanUpgrade,
	})
}
