// File: equipmentController.go
// Description: 装备系统控制器
// Author: 木村凉太
// Create Time: 2025.11

package gameController

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/equipment"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("equipment", &EquipmentController{})
}

type EquipmentController struct{}

var _ LogicControllerInterface = (*EquipmentController)(nil)

// RegisterLogicMessage 注册装备路由
func (e *EquipmentController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_LIST_REQ, &pb.EquipmentListReq{}, GetEquipmentListHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_DETAIL_REQ, &pb.EquipmentDetailReq{}, GetEquipmentDetailHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_EQUIP_REQ, &pb.EquipmentEquipReq{}, EquipEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_UNEQUIP_REQ, &pb.EquipmentUnequipReq{}, UnequipEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_SWAP_REQ, &pb.EquipmentSwapReq{}, SwapEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_QUICK_EQUIP_REQ, &pb.EquipmentQuickEquipReq{}, QuickEquipHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_QUICK_UNEQUIP_REQ, &pb.EquipmentQuickUnequipReq{}, QuickUnequipHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_LOCK_REQ, &pb.EquipmentLockReq{}, LockEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_REQ, &pb.EquipmentDecomposeReq{}, DecomposeEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)

	//装备2.0功能
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_FORGE_REQ, &pb.EquipmentForgeReq{}, ForgeEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_STRONG_REQ, &pb.EquipmentStrongReq{}, StrongEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_EQUIPMENT_REBIRTH_REQ, &pb.EquipmentRebirthReq{}, RebirthEquipmentHandle, enum.FUNCTION_ID_HERO_EQUIP)
}

var equipmentService logicCommon.EquipmentInterface

// InitEquipmentService 初始化装备服务
func InitEquipmentService() {
	equipmentService = equipment.NewEquipmentServer(sessionManager, messageSender)
}

// getEquipmentErrorCode 将错误信息映射到错误码
func getEquipmentErrorCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	errMsg := err.Error()
	switch {
	case errMsg == "player not found":
		return pb.ERROR_CODE_EQUIPMENT_PLAYER_NOT_FOUND
	case errMsg == "equipment model not loaded":
		return pb.ERROR_CODE_EQUIPMENT_MODEL_NOT_LOADED
	case errMsg == "equipment not found":
		return pb.ERROR_CODE_EQUIPMENT_NOT_FOUND
	case errMsg == "equipment already equipped":
		return pb.ERROR_CODE_EQUIPMENT_ALREADY_EQUIPPED
	case errMsg == "equipment not equipped":
		return pb.ERROR_CODE_EQUIPMENT_NOT_EQUIPPED
	case errMsg == "equipment config not found":
		return pb.ERROR_CODE_EQUIPMENT_CONFIG_NOT_FOUND
	case errMsg == "Equipment and class mismatch":
		return pb.ERROR_CODE_EQUIPMENT_CLASS_MISMATCH
	case errMsg == "equipment is locked":
		return pb.ERROR_CODE_EQUIPMENT_IS_LOCKED
	case errMsg == "equipment is equipped":
		return pb.ERROR_CODE_EQUIPMENT_CANNOT_DECOMPOSE_EQUIPPED
	case errMsg == "equipment quality config not found":
		return pb.ERROR_CODE_EQUIPMENT_QUALITY_CONFIG_NOT_FOUND
	case errMsg == "hero not found" || errMsg == "hero model not loaded":
		return pb.ERROR_CODE_EQUIPMENT_HERO_NOT_FOUND
	case errMsg == "equipment strong config not found":
		return pb.ERROR_CODE_EQUIPMENT_STRONG_CONFIG_NOT_FOUND
	case errMsg == "equipment strong level is max":
		return pb.ERROR_CODE_EQUIPMENT_STRONG_LEVEL_MAX
	case errMsg == "equipment strong cost config not found":
		return pb.ERROR_CODE_EQUIPMENT_STRONG_COST_CONFIG_NOT_FOUND
	case errMsg == "item count is not enough":
		return pb.ERROR_CODE_ITEM_NOT_ENOUGH
	case errMsg == "failed to remove items":
		return pb.ERROR_CODE_REMOVE_ITEM_ERROR
	default:
		return pb.ERROR_CODE_SYSTEM_ERROR
	}
}

// GetEquipmentListHandle 获取装备列表
func GetEquipmentListHandle(message proto.Message, player *model.PlayerModel) {
	_, ok := message.(*pb.EquipmentListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("get equipment list", player)

	equipments, err := equipmentService.GetEquipmentList(player.GetUserId())
	errorCode := getEquipmentErrorCode(err)

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_LIST_RESP, &pb.EquipmentListResp{
		ErrorCode:  errorCode,
		Equipments: equipments,
	})
}

// GetEquipmentDetailHandle 获取装备详情
func GetEquipmentDetailHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("get equipment detail", player)

	detail, err := equipmentService.GetEquipmentDetail(player.GetUserId(), req.GetEquipmentOwnId())
	errorCode := getEquipmentErrorCode(err)

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_DETAIL_RESP, &pb.EquipmentDetailResp{
		ErrorCode: errorCode,
		Detail:    detail,
	})
}

// EquipEquipmentHandle 穿戴装备
func EquipEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentEquipReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_EQUIP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("equip equipment", player)
	heroInfos := make([]*pb.HeroBagInfo, 0)
	oldHeroOwnId := int64(0)
	equipmentId := req.GetEquipmentOwnId()
	equipmentEntity := player.EquipmentModel.GetEquipment(equipmentId)
	if equipmentEntity.HeroOwnID != 0 {
		oldHeroOwnId = equipmentEntity.HeroOwnID
	}
	equipmentService.UnequipEquipment(player.GetUserId(), req.GetEquipmentOwnId())
	if oldHeroOwnId != 0 {
		heroInfos = append(heroInfos, player.HeroDetailsModel.GetHeroInfoByOwnID(player, oldHeroOwnId))
	}
	err := equipmentService.EquipEquipment(player.GetUserId(), req.GetEquipmentOwnId(), req.GetHeroOwnId())
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	heroInfos = append(heroInfos, player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.GetHeroOwnId()))
	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_EQUIP_RESP, &pb.EquipmentEquipResp{
		ErrorCode: errorCode,
		IsSuccess: isSuccess,
	})
	if isSuccess {
		heroId := int32(0)
		if hd := player.HeroDetailsModel.Entities[req.GetHeroOwnId()]; hd != nil {
			heroId = int32(hd.HeroID)
		}
		operationLogService.OnUserEquipmentOperation(player.GetUserId(), heroId, equipmentEntity.SlotIndex, 0, equipmentEntity.EquipmentID)
		eventBusService.SubmitEquipmentWearEvent(player.GetUserId())
	}
}

// UnequipEquipmentHandle 卸下装备
func UnequipEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentUnequipReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_UNEQUIP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("unequip equipment", player)

	equipmentEntity := player.EquipmentModel.GetEquipment(req.GetEquipmentOwnId())
	heroOwnId := equipmentEntity.HeroOwnID

	err := equipmentService.UnequipEquipment(player.GetUserId(), req.GetEquipmentOwnId())
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_UNEQUIP_RESP, &pb.EquipmentUnequipResp{
		ErrorCode: errorCode,
		IsSuccess: isSuccess,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroOwnId),
	})
}

// SwapEquipmentHandle 替换装备
func SwapEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentSwapReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_SWAP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("swap equipment", player)

	// 记录装备原持有英雄（若在背包则为 0），UnequipEquipment 后会被清空

	equipmentService.UnequipEquipment(player.GetUserId(), req.GetEquipmentOwnId())

	replacedEquipmentOwnID, err := equipmentService.SwapEquipment(player.GetUserId(), req.GetEquipmentOwnId(), req.GetHeroOwnId())
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_SWAP_RESP, &pb.EquipmentSwapResp{
		ErrorCode:              errorCode,
		IsSuccess:              isSuccess,
		ReplacedEquipmentOwnId: replacedEquipmentOwnID,
	})
	if isSuccess {
		eventBusService.SubmitEquipmentWearEvent(player.GetUserId())
	}
}

// QuickEquipHandle 一键穿戴
func QuickEquipHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentQuickEquipReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_QUICK_EQUIP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("quick equip equipment", player)

	equipments, err := equipmentService.QuickEquip(player.GetUserId(), req.GetHeroOwnIds())
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	heroInfo := make([]*pb.HeroBagInfo, 0)
	for _, heroOwnId := range req.GetHeroOwnIds() {
		heroInfo = append(heroInfo, player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroOwnId))
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_QUICK_EQUIP_RESP, &pb.EquipmentQuickEquipResp{
		ErrorCode:  errorCode,
		IsSuccess:  isSuccess,
		Equipments: equipments,
		HeroInfos:  heroInfo,
	})
	if isSuccess {
		eventBusService.SubmitEquipmentWearEvent(player.GetUserId())
	}
}

// QuickUnequipHandle 一键卸下
func QuickUnequipHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentQuickUnequipReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_QUICK_UNEQUIP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("quick unequip equipment", player)

	err := equipmentService.QuickUnequip(player.GetUserId(), req.GetHeroOwnIds())
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	heroInfo := make([]*pb.HeroBagInfo, 0)
	for _, heroOwnId := range req.GetHeroOwnIds() {
		heroInfo = append(heroInfo, player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroOwnId))
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_QUICK_UNEQUIP_RESP, &pb.EquipmentQuickUnequipResp{
		ErrorCode: errorCode,
		IsSuccess: isSuccess,
		HeroInfos: heroInfo,
	})
}

// LockEquipmentHandle 锁定/解锁装备
func LockEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentLockReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_LOCK_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("lock/unlock equipment", player)

	var err error
	if req.GetLock() {
		err = equipmentService.LockEquipment(player.GetUserId(), req.GetEquipmentOwnId())
	} else {
		err = equipmentService.UnlockEquipment(player.GetUserId(), req.GetEquipmentOwnId())
	}
	errorCode := getEquipmentErrorCode(err)
	isSuccess := errorCode == pb.ERROR_CODE_SUCCESS

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_LOCK_RESP, &pb.EquipmentLockResp{
		ErrorCode: errorCode,
		IsSuccess: isSuccess,
	})
}

// DecomposeEquipmentHandle 分解装备
func DecomposeEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentDecomposeReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("decompose equipment", player)

	// 分解前重生装备
	items := make([]*gameConfig.ItemConfig, 0)
	itemsMap := make(map[int32]int64)
	for _, v := range req.EquipmentOwnIds {
		equipmentDetail := player.EquipmentModel.GetEquipment(v)
		if equipmentDetail == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, pb.ERROR_CODE_EQUIPMENT_NOT_FOUND)
			return
		}
		addItems := gameConfig.GetRebirthEquipItems(equipmentDetail.StrongLevel)
		if addItems == nil && equipmentDetail.StrongLevel > 0 {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		for key, v := range addItems {
			itemsMap[key] += v
		}
	}

	resp, err := equipmentService.DecomposeEquipments(player.GetUserId(), req.GetEquipmentOwnIds())
	errorCode := getEquipmentErrorCode(err)

	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, &pb.EquipmentDecomposeResp{
			ErrorCode: errorCode,
			DropId:    []int32{},
		})
		return
	}

	// 上报每件装备分解消耗（id 填装备配置表模板 ID，ext 放 ownId）
	for _, ownId := range req.GetEquipmentOwnIds() {
		if player.EquipmentModel != nil {
			if equip := player.EquipmentModel.GetEquipment(ownId); equip != nil {
				itemService.ReportUserItemChange(
					player.GetUserId(),
					int32(enum.ITEM_TYPE_EQUIP),
					equip.EquipmentID,
					int32(enum.ITEM_CHANGE_REASON_DECOMPOSE_EQUIP_CONSUME),
					0,
					ownId,
					"-",
					"-1",
					"-",
				)
			}
		}
	}
	for _, item := range gameConfig.GetDropMap(resp.DropId) {
		itemsMap[item.ID] += item.Num
	}

	for key, v := range itemsMap {
		items = append(items, &gameConfig.ItemConfig{ID: key, Num: v})
	}

	err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_DECOMPOSE_EQUIP)
	if err != nil {
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, &pb.EquipmentDecomposeResp{
		ErrorCode: pb.ERROR_CODE_SUCCESS,
		DropId:    resp.DropId,
	})
}

func ForgeEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentForgeReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	useItems := make([]*gameConfig.ItemConfig, 0)
	platformLogger.InfoWithUser("forge equipment", player)
	paperId := req.GetPaperId()
	num := req.GetCount()
	paperCfg := gameConfig.GetEquipBlueprintCfg(paperId)
	if paperCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	for _, itemCfg := range paperCfg.Cost {
		useItems = append(useItems, &gameConfig.ItemConfig{ID: itemCfg.ID, Num: itemCfg.Num * int64(num)})
	}
	if paperCfg.Star > 1 {
		useItems = append(useItems, &gameConfig.ItemConfig{ID: paperCfg.Id, Num: int64(num)})
	}
	equipmentItemCfg := gameConfig.GetItemCfg(paperCfg.Equipment)
	if equipmentItemCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	addItems := make([]*gameConfig.ItemConfig, 0)
	for i := int32(0); i < num; i++ {
		addItems = append(addItems, &gameConfig.ItemConfig{ID: equipmentItemCfg.Id, Num: 1})
	}
	flag, err := itemService.CheckItemsCount(player, useItems)
	if err != nil || !flag {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItems(player, useItems, enum.ITEM_CHANGE_REASON_FORGE_EQUIPMENT)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_FORGE_EQUIPMENT)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, &pb.EquipmentForgeResp{})
	operationLogService.OnUserEquipmentOperation2_0Forge(player.GetUserId(), paperCfg.Equipment)
	eventBusService.SubmitEquipmentForgeEvent(player.GetUserId(), num)
}

func StrongEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentStrongReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_STRONG_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("strong equipment", player)
	// 强化逻辑
	equipmentOwnId := req.GetEquipmentOwnId()
	// 获取旧等级
	oldStrongLevel := int32(0)
	equipmentId := int32(0)
	if player.EquipmentModel != nil {
		if detail := player.EquipmentModel.GetEquipment(equipmentOwnId); detail != nil {
			oldStrongLevel = detail.StrongLevel
			equipmentId = detail.EquipmentID
		}
	}
	resp, err := equipmentService.StrongEquipment(player.GetUserId(), equipmentOwnId, req.GetIsUseStone())
	errorCode := getEquipmentErrorCode(err)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_STRONG_RESP, errorCode)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_STRONG_RESP, resp)

	newStrongLevel := int32(0)
	if resp.GetEquipmentDetail() != nil {
		newStrongLevel = resp.GetEquipmentDetail().GetStrongLevel()
	}
	isSuccess := int32(0)
	if newStrongLevel > oldStrongLevel {
		isSuccess = 1
		eventBusService.SubmitEquipmentStrongEvent(player.GetUserId(), equipmentOwnId, oldStrongLevel, newStrongLevel)
	}
	operationLogService.OnUserEquipmentOperation2_0Refine(player.GetUserId(), isSuccess, equipmentId, oldStrongLevel, newStrongLevel)
}

func RebirthEquipmentHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.EquipmentRebirthReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_REBIRTH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	platformLogger.InfoWithUser("rebirth equipment", player)
	resp, err := equipmentService.RebirthEquipment(player.GetUserId(), req.GetEquipmentOwnId())
	errorCode := getEquipmentErrorCode(err)
	if errorCode != pb.ERROR_CODE_SUCCESS {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_EQUIPMENT_REBIRTH_RESP, errorCode)
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_EQUIPMENT_REBIRTH_RESP, resp)
}
