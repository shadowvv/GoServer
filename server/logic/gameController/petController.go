package gameController

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"google.golang.org/protobuf/proto"
)

// PetController 预留：后续 pet pb 协议落地后在这里注册消息路由。
// 当前仅承载宠物主体（升级/升星/重生）逻辑骨架，保持与现有项目“逻辑写在 controller”风格一致。
type PetController struct{}

func init() {
	RegisterController("pet", &PetController{})
}

var _ LogicControllerInterface = (*PetController)(nil)

func (c *PetController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_DETAIL_REQ, &pb.PetDetailReq{}, PetDetailHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_EQUIP_REQ, &pb.PetEquipReq{}, PetEquipHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_LEVEL_UP_REQ, &pb.PetLevelUpReq{}, PetLevelUpHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_STAR_UP_REQ, &pb.PetStarUpReq{}, PetStarUpHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_REBIRTH_REQ, &pb.PetRebirthReq{}, PetRebirthHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_DECOMPOSE_REQ, &pb.PetDecomposeReq{}, PetDecomposeHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_REQ, &pb.PetMaxLevelMaxStarDetailReq{}, PetMaxLevelMaxStarDetailHandle, enum.FUNCTION_ID_NONE)

	// 宠物缘分：激活/升级
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_AFFINITY_LEVEL_UP_REQ, &pb.PetAffinityLevelUpReq{}, PetAffinityLevelUpHandle, enum.FUNCTION_ID_PET)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_PET_AFFINITY_LIST_REQ, &pb.PetAffinityListReq{}, PetAffinityListHandle, enum.FUNCTION_ID_PET)
}

// 宠物错误码映射（统一入口）
func petErrToCode(err error) pb.ERROR_CODE {
	if err == nil {
		return pb.ERROR_CODE_SUCCESS
	}
	switch err {
	case pet.ErrPetNotFound, pet.ErrPetDeleted:
		return pb.ERROR_CODE_PET_NOT_FOUND // 宠物不存在
	case pet.ErrPetCfgNotFound:
		return pb.ERROR_CODE_CFG_NOT_FOUND // 宠物配置不存在
	case pet.ErrPetLevelMax:
		return pb.ERROR_CODE_PET_LEVEL_MAX // 宠物已达最高等级
	case pet.ErrPetLevelMin:
		return pb.ERROR_CODE_PET_LEVEL_MIN // 宠物等级已最低
	case pet.ErrPetLevelUpConditionNotMet:
		return pb.ERROR_CODE_PET_LEVEL_UP_CONDITION_NOT_MET // 宠物升级条件未满足
	case pet.ErrPetStarMax:
		return pb.ERROR_CODE_PET_STAR_MAX // 宠物星级已达上限
	case pet.ErrPetStarUpConditionNotMet:
		return pb.ERROR_CODE_PET_STAR_UP_CONDITION_NOT_MET // 宠物升星条件未满足
	case pet.ErrPetAffinityCfgNotFound, pet.ErrPetAffinityNotFound:
		return pb.ERROR_CODE_PET_AFFINITY_NOT_FOUND // 缘分配置不存在
	case pet.ErrPetAffinityLevelMax:
		return pb.ERROR_CODE_PET_AFFINITY_LEVEL_MAX // 缘分等级已达上限
	case pet.ErrPetAffinityActivateConditionNotMet:
		return pb.ERROR_CODE_PET_AFFINITY_ACTIVATE_CONDITION_NOT_MET // 缘分激活条件未满足
	case pet.ErrPetAffinityLevelUpConditionNotMet:
		return pb.ERROR_CODE_PET_AFFINITY_LEVEL_UP_CONDITION_NOT_MET // 缘分升级条件未满足
	case pet.ErrPetAffinityNotActive:
		return pb.ERROR_CODE_PET_AFFINITY_NOT_ACTIVATED // 缘分未激活
	case pet.ErrPetSkillCfgNotFound, pet.ErrPetSkillNotFound:
		return pb.ERROR_CODE_PET_SKILL_NOT_FOUND // 宠物技能不存在
	case pet.ErrPetSkillRollFailed:
		return pb.ERROR_CODE_PET_SKILL_ROLL_FAILED // 宠物技能 roll 失败
	default:
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM // 其他错误
	}
}

// 宠物详情
func PetDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet detail req", player)

	req, ok := message.(*pb.PetDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	resp, err := pet.GetPetDetail(player, req.GetPetOwnId())
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DETAIL_RESP, petErrToCode(err))
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_DETAIL_RESP, resp)
}

// 满级满星宠物详情（用于预览展示，不写入背包）
func PetMaxLevelMaxStarDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet max level max star detail req", player)

	req, ok := message.(*pb.PetMaxLevelMaxStarDetailReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	petId := req.GetPetId()
	base := gameConfig.GetPetBaseCfg(petId)
	if base == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	maxLevel := gameConfig.GetPetMaxLevelByPotential(base.PetPotential)
	maxStar := gameConfig.GetPetMaxStarByPetId(petId)

	preview := &model.PetEntity{
		PetOwnID:      0,
		UserID:        0,
		PetID:         petId,
		Level:         maxLevel,
		Star:          maxStar,
		HeroOwnId:     0,
		PassiveSkills: nil, // 按需求：被动技能为空即可
		IsDeleted:     false,
	}
	info := pet.BuildPetDetailInfo(player, preview)
	if info == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}
	// 明确置空，避免 BuildPetDetailInfo 因未来变动填入默认技能
	info.PassiveSkills = nil

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_MAX_LEVEL_MAX_STAR_DETAIL_RESP, &pb.PetMaxLevelMaxStarDetailResp{
		PetInfo: info,
	})
}

// 宠物分解：删除宠物并返还分解产出（按 petBase.salvageYield 配置）
func PetDecomposeHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet decompose req", player)

	req, ok := message.(*pb.PetDecomposeReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	petOwnIds := req.GetPetOwnIds()
	if len(petOwnIds) == 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if player == nil || player.PetModel == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	removedPets := make([]*pb.PetDetailInfo, 0, len(petOwnIds))
	yieldMap := make(map[int32]int64)
	seenPetOwnIDs := make(map[int64]bool, len(petOwnIds))

	// 先校验并汇总产出，避免部分成功/部分失败导致状态不一致
	for _, ownID := range petOwnIds {
		if ownID <= 0 {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
		if seenPetOwnIDs[ownID] {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
		seenPetOwnIDs[ownID] = true
		p := player.PetModel.GetPet(ownID)
		if p == nil || p.IsDeleted {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_PET_NOT_FOUND)
			return
		}
		rewardMap, err := pet.CalcPetDecomposeRewardMap(p)
		if errors.Is(err, pet.ErrPetCfgNotFound) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, petErrToCode(err))
			return
		}
		if detail := pet.BuildPetDetailInfo(player, p); detail != nil {
			removedPets = append(removedPets, detail)
		}
		for id, num := range rewardMap {
			if id <= 0 || num <= 0 {
				continue
			}
			yieldMap[id] += num
		}
	}

	// 统一删除（软删除）并推送移除列表
	for _, ownID := range petOwnIds {
		if petEntity := player.PetModel.GetPet(ownID); petEntity != nil && !petEntity.IsDeleted && petEntity.HeroOwnId != 0 {
			player.PetModel.UnwearPet(ownID)
		}
		player.PetModel.DeletePet(ownID)
	}
	if len(removedPets) > 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_PET_DETAIL, &pb.PushPetDetail{
			RemovePetList: removedPets,
		})
	}

	// 发放分解产出（合并同类项后一并发放）
	yieldItems := make([]*gameConfig.ItemConfig, 0, len(yieldMap))
	for id, num := range yieldMap {
		yieldItems = append(yieldItems, &gameConfig.ItemConfig{ID: id, Num: num})
	}
	if err := itemService.AddItems(player, yieldItems, enum.ITEM_CHANGE_REASON_DECOMPOSE_PET); err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_DECOMPOSE_RESP, &pb.PetDecomposeResp{IsSuccess: true})
}

// 宠物装备
func PetEquipHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet equip req", player)

	req, ok := message.(*pb.PetEquipReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_EQUIP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil || player.HeroDetailsModel == nil {
		platformLogger.InfoWithUser("player petModel or heroModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_EQUIP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	petEntity, _, err := pet.EquipPet(player, req.GetPetOwnId(), req.GetHeroOwnId(), req.GetOperationType())
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_EQUIP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_EQUIP_RESP, &pb.PetEquipResp{
		Pet: pet.BuildPetDetailInfo(player, petEntity),
	})
}

// 宠物升级
func PetLevelUpHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet level up req", player)

	req, ok := message.(*pb.PetLevelUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil {
		platformLogger.InfoWithUser("player or petModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	delta := int32(0)
	switch req.GetLevelUpType() {
	case 1:
		delta = 1
	case 2:
		delta = 5
	case 3:
		delta = 10
	default:
		platformLogger.InfoWithUser("invalid level up type", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	targetLevel, costMap, err := pet.LevelUp(player, req.GetPetOwnId(), delta)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, petErrToCode(err))
		return
	}

	// 校验并扣除升级材料
	ok, err = itemService.CheckItemsCountWithMap(player, costMap)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	if err := itemService.RemoveItemsWithMap(player, costMap, enum.ITEM_CHANGE_REASON_HERO_LEVEL_UP); err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}

	petEntity := player.PetModel.GetPet(req.GetPetOwnId())
	oldLevel := petEntity.Level
	petId := petEntity.PetID
	player.PetModel.UpdateLevel(req.GetPetOwnId(), targetLevel)

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_LEVEL_UP_RESP, &pb.PetLevelUpResp{
		Pet: pet.BuildPetDetailInfo(player, petEntity),
	})
	if petEntity != nil && petEntity.Level > oldLevel {
		operationLogService.OnUserPetLevelUp(player.GetUserId(), petId, petEntity.PetOwnID, oldLevel, petEntity.Level)
		eventBusService.SubmitPetLevelUpEvent(player.GetUserId(), petEntity.PetOwnID, oldLevel, petEntity.Level)
	}

}

// 宠物升星
func PetStarUpHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet star up req", player)

	req, ok := message.(*pb.PetStarUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil {
		platformLogger.InfoWithUser("player or petModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	petEntityBefore := player.PetModel.GetPet(req.GetPetOwnId())
	oldStar := int32(0)
	petId := int32(0)
	if petEntityBefore != nil {
		oldStar = petEntityBefore.Star
		petId = petEntityBefore.PetID
	}
	newStar, sacrificeIds, err := pet.StarUp(player, req.GetPetOwnId(), req.GetPetOwnIds())
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, petErrToCode(err))
		return
	}
	// 先预计算升星后的被动技能；失败则不落地任何状态（避免"升星成功但 roll 技能失败"的半成功）
	newSkills, err := pet.PreviewPetPassiveSkillsAfterStarUp(player, req.GetPetOwnId(), newStar)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, petErrToCode(err))
		return
	}

	// costNum2：升星额外道具消耗（先检测再扣除；扣除成功后才会删除材料宠物并落库）。
	petEntityForCost := player.PetModel.GetPet(req.GetPetOwnId())
	if petEntityForCost == nil || petEntityForCost.IsDeleted {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_PET_NOT_FOUND)
		return
	}
	starCfg := gameConfig.GetPetStarCfgByPetIdStar(petEntityForCost.PetID, petEntityForCost.Star)
	if starCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	if starCfg.CostNum2 != nil && starCfg.CostNum2.ID > 0 && starCfg.CostNum2.Num > 0 {
		costItems := []*gameConfig.ItemConfig{{ID: starCfg.CostNum2.ID, Num: starCfg.CostNum2.Num}}
		ok, err := itemService.CheckItemsCount(player, costItems)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
		if !ok {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		if err := itemService.RemoveItems(player, costItems, enum.ITEM_CHANGE_REASON_USE_ITEM); err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
	}

	removedPets := make([]*pb.PetDetailInfo, 0, len(sacrificeIds))
	for _, sid := range sacrificeIds {
		if mat := player.PetModel.GetPet(sid); mat != nil && !mat.IsDeleted {
			removedPets = append(removedPets, pet.BuildPetDetailInfo(player, mat))
		}
		if mat := player.PetModel.GetPet(sid); mat != nil && !mat.IsDeleted && mat.HeroOwnId != 0 {
			player.PetModel.UnwearPet(sid)
		}
		player.PetModel.DeletePet(sid)
	}
	if len(removedPets) > 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_PET_DETAIL, &pb.PushPetDetail{
			RemovePetList: removedPets,
		})
	}

	// 提交落地：更新星级并写入预计算好的被动技能
	player.PetModel.UpdateStar(req.GetPetOwnId(), newStar)
	player.PetModel.UpdatePassiveSkills(req.GetPetOwnId(), newSkills)
	petEntity := player.PetModel.GetPet(req.GetPetOwnId())

	// 图鉴簿变更推送：petId -> 历史最高星
	if player.PetAffinityModel != nil && petEntity != nil {
		if ent := player.PetAffinityModel.BookEntities[petEntity.PetID]; ent != nil {
			messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_PET_STAR_BOOK_CHANGE, &pb.PushPetStarBookChange{
				Changed: []*pb.PetStarBookEntry{
					{PetId: petEntity.PetID, MaxStar: ent.MaxStar},
				},
			})
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_STAR_UP_RESP, &pb.PetStarUpResp{
		Pet: pet.BuildPetDetailInfo(player, petEntity),
	})
	if petEntity != nil {
		if petId == 0 {
			petId = petEntity.PetID
		}
		operationLogService.OnUserPetStarUp(player.GetUserId(), petId, petEntity.PetOwnID, oldStar, newStar)
	}
	eventBusService.SubmitPetStarUpEvent(player.GetUserId(), req.GetPetOwnId(), newStar-1, newStar)
}

// 宠物重生
func PetRebirthHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet rebirth req", player)

	req, ok := message.(*pb.PetRebirthReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_REBIRTH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil {
		platformLogger.InfoWithUser("player or petModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_REBIRTH_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	refundItems, err := pet.Rebirth(player, req.GetPetOwnId())
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_REBIRTH_RESP, petErrToCode(err))
		return
	}
	if len(refundItems) > 0 {
		if err := itemService.AddItems(player, refundItems, enum.ITEM_CHANGE_REASON_HERO_REBIRTH); err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_REBIRTH_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
			return
		}
	}

	// 重置等级为 1
	player.PetModel.UpdateLevel(req.GetPetOwnId(), 1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_REBIRTH_RESP, &pb.PetRebirthResp{
		Pet: pet.BuildPetDetailInfo(player, player.PetModel.GetPet(req.GetPetOwnId())),
	})
}

// 宠物缘分升级
func PetAffinityLevelUpHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet affinity level up req", player)

	req, ok := message.(*pb.PetAffinityLevelUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil || player.PetAffinityModel == nil {
		platformLogger.InfoWithUser("player petModel or petAffinityModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LEVEL_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	resp, err := pet.UpgradeAffinity(player, req.GetAffinityId())
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LEVEL_UP_RESP, petErrToCode(err))
		return
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LEVEL_UP_RESP, resp)

}

// 宠物缘分列表
func PetAffinityListHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("pet affinity list req", player)

	_, ok := message.(*pb.PetAffinityListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if player == nil || player.PetModel == nil || player.PetAffinityModel == nil {
		platformLogger.InfoWithUser("player petModel or petAffinityModel nil", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LIST_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PET_AFFINITY_LIST_RESP, pet.GetPetAffinityList(player))
}
