package gameController

import (
	"context"
	"fmt"
	"math"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/lumber"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	unlockSvc "github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/logic/vipCard"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("Hero", &HeroController{})
}

type HeroController struct {
}

var _ LogicControllerInterface = (*HeroController)(nil)

// RequirePlayerHeroModels：写操作必须调用，若模型缺失则返回错误（不自动创建）
func RequirePlayerHeroModels(player *model.PlayerModel) error {
	if player == nil {
		platformLogger.ErrorWithUser("player is nil", player, nil)
		return fmt.Errorf("player is nil")
	}
	// 根据你的业务选择更严格的判定：可以要求 Entities 非空，或仅要求 model 非 nil
	if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil {
		platformLogger.ErrorWithUser("hero details model not loaded", player, nil)
		return fmt.Errorf("hero details model not loaded")
	}
	if player.AlbumRewardModel == nil || player.AlbumRewardModel.Entity == nil {
		platformLogger.ErrorWithUser("album reward model not loaded", player, nil)
		return fmt.Errorf("album reward model not loaded")
	}
	if player.HeroAlbumModel == nil || player.HeroAlbumModel.Entities == nil {
		platformLogger.ErrorWithUser("hero album model not loaded", player, nil)
		return fmt.Errorf("hero album model not loaded")
	}
	if player.HeroFormationModel == nil || player.HeroFormationModel.Entities == nil {
		platformLogger.ErrorWithUser("hero formation model not loaded", player, nil)
		return fmt.Errorf("hero formation model not loaded")
	}
	return nil
}

func (s *HeroController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_BREAK_REQ, &pb.HeroBreakReq{}, HeroBreakHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_REBITH_REQ, &pb.HeroRebithReq{}, HeroRebithHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_STAR_UP_REQ, &pb.HeroStarUpReq{}, HeroStarUpHandle, enum.FUNCTION_ID_HERO_STAR)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_LEVEL_UP_REQ, &pb.HeroLevelUpReq{}, HeroLevelHandle, enum.FUNCTION_ID_HERO_LEVEL_UP)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_EVOLUTION_REQ, &pb.HeroEvolutionReq{}, HeroEvolutionHandle, enum.FUNCTION_ID_HERO_TRANSFORM)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_ALBUM_REWARD_REQ, &pb.HeroAlbumRewardReq{}, HeroAlbumRewardHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_ALBUM_DETAILS_REQ, &pb.HeroAlbumDetailsReq{}, HeroAlbumDetailsHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_SET_FORMATION_REQ, &pb.HeroSetFormationReq{}, HeroSetFormationHandle, enum.FUNCTION_ID_HERO_FORMATION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_FORMATION_DETAILS_REQ, &pb.HeroFormationDetailsReq{}, HeroFormationHandle, enum.FUNCTION_ID_HERO_FORMATION)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_REQ, &pb.GetPlayerHeroDetailReq{}, GetPlayerHeroDetailHandle, enum.FUNCTION_ID_HERO_DETAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_ALBUM_ITEM_REWARD_REQ, &pb.HeroAlbumItemRewardReq{}, HeroAlbumItemRewardHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_REQ, &pb.HeroExchangeNotLossReq{}, HeroExchangeNotLossHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_HERO_MAX_DETAIL_REQ, &pb.GetHeroMaxDetailReq{}, GetHeroMaxDetailHandle, enum.FUNCTION_ID_HERO_DETAIL)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_HERO_FORMATION_MAX_LEVEL_LIST_REQ, &pb.GetHeroFormationMaxLevelListReq{}, GetHeroFormationMaxLevelListHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_REQ, &pb.BuyDispatchFormationReq{}, BuyDispatchFormationHandle, enum.FUNCTION_ID_NONE)
}

func GetHeroFormationMaxLevelListHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get hero formation max level list", player)
	req, ok := message.(*pb.GetHeroFormationMaxLevelListReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_HERO_FORMATION_MAX_LEVEL_LIST_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	power := int64(0)
	formationLevelResp := player.HeroDetailsModel.GetTop5HeroLevels()
	for _, v := range player.HeroFormationModel.Entities[req.FormationType] {
		if v.IsActive == true {
			for _, hid := range v.HeroOwnIDList {
				power += player.GetHeroAttrForBattle(hid, req.FormationType, v.FormationID)[enum.AttributeBasicCombatPower]
			}
		}
	}
	resp := &pb.GetHeroFormationMaxLevelListResp{
		FormationHeroLevelInfo: formationLevelResp,
		FormationPower:         power,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_HERO_FORMATION_MAX_LEVEL_LIST_RESP, resp)
}

func HeroLevelHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero level req", player)

	req, ok := message.(*pb.HeroLevelUpReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	heroDetail := player.HeroDetailsModel.GetHero(req.HeroOwnId)

	if heroDetail == nil {
		platformLogger.ErrorWithUser("hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}
	var needLevel int32
	if req.LevelUpType == 1 {
		needLevel = 1
	} else if req.LevelUpType == 2 {
		needLevel = 5
	} else if req.LevelUpType == 3 {
		needLevel = 10
	}
	cityLevel := player.ArchitectureModel.GetMainLevel()
	if cityLevel == 0 {
		cityLevel = 1
	}
	if heroDetail.Level+needLevel > gameConfig.GetCityCenterCfg(cityLevel).HeroLevel {
		platformLogger.InfoWithUser("英雄等级不能超过账号等级", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_HERO_LEVEL_MAX)
		return
	}
	itemNeed := hero.CheckLevelUpItem(heroDetail, needLevel)
	flag, err := itemService.CheckItemsCountWithMap(player, itemNeed)
	if !flag || err != nil {
		platformLogger.InfoWithUser("道具数量不足", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItemsWithMap(player, itemNeed, enum.ITEM_CHANGE_REASON_HERO_LEVEL_UP)
	if err != nil {
		platformLogger.ErrorWithUser("remove item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}

	// 记录升级前等级
	beforeLevel := heroDetail.Level
	// 更新等级并记录变更
	newLevel := heroDetail.Level + needLevel
	player.HeroDetailsModel.UpdateLevel(req.HeroOwnId, newLevel)
	if newLevel > player.HeroAlbumModel.Entities[heroDetail.HeroID].HistoryMaxLevel {
		player.HeroAlbumModel.UpdateHistoryMaxLevel(heroDetail.HeroID, newLevel)
	}
	// 返回结果
	resp := &pb.HeroLevelUpResp{
		IsSuccess: true,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, resp)

	if newLevel >= player.StaticData.Entity.HeroHistoryMaxLevel {
		player.StaticData.UpdateHeroHistoryMaxLevel(newLevel)
	}

	// 上报英雄等级日志
	operationLogService.OnUserHeroLevel(player.GetUserId(), int32(heroDetail.HeroID), req.HeroOwnId, beforeLevel, newLevel)

	eventBusService.SubmitHeroLevelUpEvent(player.GetUserId(), int32(heroDetail.HeroID), beforeLevel, newLevel)

	// 记录今日英雄升级次数到 Redis
	ctx := context.Background()
	err = unlockSvc.DailyCache.RecordHeroLevelUp(ctx, player.GetUserId(), needLevel)
	if err != nil {
		platformLogger.ErrorWithUser("record hero level up error", player, err)
	}
}

func HeroBreakHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero break req", player)

	req, ok := message.(*pb.HeroBreakReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_BREAK_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	heroDetail := player.HeroDetailsModel.GetHero(req.HeroOwnId)
	if heroDetail == nil {
		platformLogger.ErrorWithUser("hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_BREAK_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	itemNeed := hero.CheckBreakItem(heroDetail)
	if itemNeed == nil {
		platformLogger.InfoWithUser("hero not exist", player)
		resp := &pb.HeroBreakResp{
			IsSuccess: false,
			HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
		}
		messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_BREAK_RESP, resp)
	}
	flag, err := itemService.CheckItemsCountWithMap(player, itemNeed)
	if !flag || err != nil {
		platformLogger.InfoWithUser("道具数量不足", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItemsWithMap(player, itemNeed, enum.ITEM_CHANGE_REASON_HERO_BREAK)
	if err != nil {
		platformLogger.ErrorWithUser("remove item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_LEVEL_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}

	newBreak := heroDetail.BreakNum + 1
	player.HeroDetailsModel.UpdateBreakNum(req.HeroOwnId, newBreak)
	player.HeroDetailsModel.UpdateIsDirty(req.HeroOwnId, true)

	resp := &pb.HeroBreakResp{
		IsSuccess: true,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_BREAK_RESP, resp)
}

func HeroEvolutionHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero evolution req", player)

	req, ok := message.(*pb.HeroEvolutionReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EVOLUTION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	heroDetail := player.HeroDetailsModel.GetHero(req.HeroOwnId)
	if heroDetail == nil {
		platformLogger.ErrorWithUser("hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EVOLUTION_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	if !hero.CheckEvolutionConditions(heroDetail, req.EvolutionPath, player) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EVOLUTION_RESP, pb.ERROR_CODE_HERO_UNLOCK_CONDITION_NOT_MET)
		return
	}

	// 记录转职前职业
	beforeEvo := heroDetail.EvolutionPath
	player.HeroDetailsModel.UpdateEvolutionPath(req.HeroOwnId, req.EvolutionPath)

	resp := &pb.HeroEvolutionResp{
		IsSuccess: true,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_EVOLUTION_RESP, resp)

	// 上报英雄转职日志
	operationLogService.OnUserHeroEvolutation(player.GetUserId(), int32(heroDetail.HeroID), req.HeroOwnId, beforeEvo, req.EvolutionPath)
}

func HeroAlbumDetailsHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero album details req", player)

	_, ok := message.(*pb.HeroAlbumDetailsReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_DETAILS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	var heroAlbumInfo []*pb.HeroAlbumInfo
	for _, album := range player.HeroAlbumModel.Entities {
		if album == nil {
			continue
		}
		unclaimedScore := hero.CalculateUnclaimedScore(player, album.HeroID)

		info := &pb.HeroAlbumInfo{
			HeroId:         album.HeroID,
			HistoryMaxStar: album.HistoryMaxStar,
			UnclaimedScore: unclaimedScore,
		}
		heroAlbumInfo = append(heroAlbumInfo, info)
	}

	claimed := player.AlbumRewardModel.Entity.ClaimedReward
	allScore := player.AlbumRewardModel.Entity.AllScore

	resp := &pb.HeroAlbumDetailsResp{
		ClaimedReward: claimed,
		AllScore:      allScore,
		HeroAlbumInfo: heroAlbumInfo,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_ALBUM_DETAILS_RESP, resp)

}

func HeroAlbumRewardHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero album reward req", player)

	req, ok := message.(*pb.HeroAlbumRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	if player.AlbumRewardModel == nil || player.HeroAlbumModel == nil {
		platformLogger.ErrorWithUser("hero album reward req", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_REWARD_RESP, pb.ERROR_CODE_HERO_ALBUM_REWARD_NOT_EXIST)
		return
	}

	heroId := req.HeroId
	if heroId == 0 { // 一键领取
		allScore := int32(0)
		for hid, album := range player.HeroAlbumModel.Entities {
			if album == nil {
				continue
			}
			if album.ClaimedStar > 0 {
				allScore += hero.CalculateUnclaimedScore(player, album.HeroID)
				player.HeroAlbumModel.UpdateClaimedStar(hid, player.HeroAlbumModel.Entities[hid].HistoryMaxStar)
			}
		}
		player.AlbumRewardModel.UpdateAllScore(player.AlbumRewardModel.Entity.AllScore + allScore)
	} else {
		album := player.HeroAlbumModel.GetAlbum(heroId)
		hid := req.HeroId
		if album == nil {
			platformLogger.ErrorWithUser("hero album not exist", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_REWARD_RESP, pb.ERROR_CODE_HERO_ALBUM_REWARD_NOT_EXIST)
			return
		}
		score := hero.CalculateUnclaimedScore(player, album.HeroID)
		player.AlbumRewardModel.UpdateAllScore(player.AlbumRewardModel.Entity.AllScore + score)
		player.HeroAlbumModel.UpdateClaimedStar(hid, player.HeroAlbumModel.Entities[hid].HistoryMaxStar)
	}

	resp := &pb.HeroAlbumRewardResp{
		IsSuccess:     true,
		ClaimedReward: player.AlbumRewardModel.Entity.ClaimedReward,
		AllScore:      player.AlbumRewardModel.Entity.AllScore,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_ALBUM_REWARD_RESP, resp)
}

func HeroAlbumItemRewardHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero album item reward req", player)

	_, ok := message.(*pb.HeroAlbumItemRewardReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_ITEM_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	if player.AlbumRewardModel == nil {
		platformLogger.ErrorWithUser("hero album reward req", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_ITEM_REWARD_RESP, pb.ERROR_CODE_HERO_ALBUM_REWARD_NOT_EXIST)
		return
	}
	var reward int32
	claimedReward := player.AlbumRewardModel.Entity.ClaimedReward
	cfgMap := gameConfig.GetAllCodexRewardCfg()
	addItems := make([]*gameConfig.ItemConfig, 0)
	if cfgMap != nil {
		for i := 1; i <= int(len(cfgMap)); i++ {
			if int32(i) <= claimedReward {
				continue
			}
			key := int32(i)
			cfg := cfgMap[key]
			if cfg == nil {
				continue
			}
			if cfg.CodexPoints <= player.AlbumRewardModel.Entity.AllScore {
				reward = key
				for _, item := range cfg.Reward {
					addItems = append(addItems, item)
				}
			}
		}
	}

	err := itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_HERO_ALBUM_REWARD)
	if err != nil {
		platformLogger.ErrorWithUser("hero album item reward add items error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_ALBUM_ITEM_REWARD_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}
	player.AlbumRewardModel.UpdateClaimedReward(reward)
	resp := &pb.HeroAlbumItemRewardResp{
		IsSuccess:     true,
		ClaimedReward: player.AlbumRewardModel.Entity.ClaimedReward,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_ALBUM_ITEM_REWARD_RESP, resp)
}

func HeroStarUpHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero star up req", player)

	req, ok := message.(*pb.HeroStarUpReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	heroDetail := player.HeroDetailsModel.GetHero(req.HeroOwnId)
	if heroDetail == nil {
		platformLogger.ErrorWithUser("hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	flag := hero.HeroStarUpUseHero(heroDetail, player, req.MaterialsCultivated)
	if !flag {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_HERO_STAR_UP_ITEM_ERROR)
		return
	}

	if starEffectCfg := gameConfig.GetStarEffectCfg(int32(heroDetail.HeroID), heroDetail.StarLevel+1); starEffectCfg == nil {
		platformLogger.InfoWithUser("英雄已达最高星级", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_HERO_STAR_MAX)
		return
	}
	// HeroStarUpUseHero已经调用过下列两个配置，所以直接用
	heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
	heroStarCfg := gameConfig.GetHeroStarCfg(heroBaseCfg.HeroPotential, heroBaseCfg.HeroClass, heroDetail.StarLevel+1)
	flag = true
	var err error
	if heroStarCfg.Cost != nil {
		flag, err = itemService.CheckItemCount(player, heroStarCfg.Cost)
	}
	if !flag || err != nil {
		platformLogger.ErrorWithUser("hero star up check item count error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItem(player, heroStarCfg.Cost, enum.ITEM_CHANGE_REASON_HERO_STAR_UP)
	if err != nil {
		platformLogger.ErrorWithUser("hero star up remove items error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}

	// 脱下宠物
	detachedPets := make([]*pb.PetDetailInfo, 0)

	// 删除材料英雄
	for key, materialList := range req.MaterialsCultivated {
		if key == "universalHero" {
			for _, v := range materialList.HeroOwnIds {
				items := &gameConfig.ItemConfig{
					ID:  int32(v),
					Num: 1,
				}
				err = itemService.RemoveItem(player, items, enum.ITEM_CHANGE_REASON_HERO_REBIRTH)
				if err != nil {
					platformLogger.ErrorWithUser("hero star up remove items error", player, err)
					messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
					return
				}
			}
			continue
		}
		for _, id := range materialList.HeroOwnIds {
			// 返还英雄培养材料
			itemInfoList := hero.GetHeroRebithAllItem(player.HeroDetailsModel.GetHero(id))
			err := itemService.AddItems(player, itemInfoList, enum.ITEM_CHANGE_REASON_HERO_REBIRTH)
			if err != nil {
				platformLogger.ErrorWithUser("英雄升星返还材料添加失败", player, err)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
				return
			}

			// 脱装备
			heroDetail := player.HeroDetailsModel.GetHero(id)
			for _, equipID := range heroDetail.EquipmentId {
				if equipID != 0 {
					if err := equipmentService.UnequipEquipment(player.GetUserId(), equipID); err != nil {
						if err.Error() != "equipment not equipped" {
							platformLogger.ErrorWithUser("英雄升星脱装备失败", player, err)
							messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, pb.ERROR_CODE_HERO_START_UP_FAIL)
							return
						}
					}
				}
			}

			// 脱饰品
			for _, v := range player.AccessoryModel.Entities {
				if v.HeroOwnId == id {
					player.AccessoryModel.UnloadAccessory(v.AccessoryId, id, player.GetLevel())
				}
			}

			// 脱宠物
			if player.PetModel != nil {
				if petEntity := player.PetModel.GetEquippedPetByHero(id); petEntity != nil {
					player.PetModel.UnwearPet(petEntity.PetOwnID)
					if updatedPet := player.PetModel.GetPet(petEntity.PetOwnID); updatedPet != nil && !updatedPet.IsDeleted {
						detachedPets = append(detachedPets, pet.BuildPetDetailInfo(player, updatedPet))
					}
				}
			}

			//todo 从内存中删除，该英雄可能在阵容，激活等地方被引用
			for formationType, formationDetail := range player.HeroFormationModel.Entities {
				for formationId, formation := range formationDetail {
					newHeroOwnIDList := make([]int64, 0)
					length := len(formation.HeroOwnIDList)
					for _, hid := range formation.HeroOwnIDList {
						if hid != id {
							newHeroOwnIDList = append(newHeroOwnIDList, hid)
						}
					}
					if length != len(newHeroOwnIDList) {
						player.HeroFormationModel.UpdateHeroOwnIDListByTypeAndId((formationType), (formationId), newHeroOwnIDList)
					}
				}
			}
			// 上报材料英雄消耗（id 填 herobase 配置表 ID，ext 放 heroOwnId）
			itemService.ReportUserItemChange(
				player.GetUserId(),
				int32(enum.ITEM_TYPE_HERO),
				int32(heroDetail.HeroID),
				int32(enum.ITEM_CHANGE_REASON_HERO_STAR_UP_MATERIAL),
				0,
				id,
				"-",
				"-1",
				"-",
			)
			// 删除英雄前，需要删除派驻英雄
			lumber.Service.BeforeAssignedHeroesDelete(player, []int64{id})

			// 使用新的DeleteHero方法，自动维护反向索引和缓存
			player.HeroDetailsModel.DeleteHero(id)
		}
	}

	// 推送脱下宠物
	if len(detachedPets) > 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_PET_DETAIL, &pb.PushPetDetail{
			AddPetList: detachedPets,
		})
	}

	// 记录升星前星级
	beforeStar := heroDetail.StarLevel
	lumber.Service.BeforeAssignedHeroChange(player, req.HeroOwnId)
	player.HeroDetailsModel.UpdateStarLevel(req.HeroOwnId, heroDetail.StarLevel+1)
	if player.HeroAlbumModel.Entities[heroDetail.HeroID] != nil {
		if heroDetail.StarLevel+1 > player.HeroAlbumModel.Entities[heroDetail.HeroID].HistoryMaxStar {
			player.HeroAlbumModel.UpdateHistoryMaxStar(heroDetail.HeroID, heroDetail.StarLevel)
		}
	}
	resp := &pb.HeroStarUpResp{
		IsSuccess: true,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_STAR_UP_RESP, resp)

	// 上报英雄星级日志
	operationLogService.OnUserHeroStar(player.GetUserId(), int32(heroDetail.HeroID), req.HeroOwnId, beforeStar, beforeStar+1)

	eventBusService.SubmitHeroStarUpEvent(player.GetUserId(), int32(heroDetail.HeroID), heroDetail.StarLevel)
}

func HeroFormationHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero formation req", player)

	_, ok := message.(*pb.HeroFormationDetailsReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_FORMATION_DETAILS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	otherPlayerHeroDetails := make(map[int64]*model.HeroDetailsEntity)
	selectedHeroes := player.PlayerGloryArenaModel.SelectedHeroes
	for _, selected := range selectedHeroes {
		if selected == nil || selected.SelectedHero == nil {
			continue
		}
		heroDetail := &model.HeroDetailsEntity{
			HeroID: selected.SelectedHero.Id,
			Power:  selected.SelectedHero.Attr[enum.AttributeBasicCombatPower],
		}
		otherPlayerHeroDetails[selected.SelectedHero.Uid] = heroDetail
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	var formations []*pb.HeroFormationDetails
	for formationType, formationDetail := range player.HeroFormationModel.Entities {
		// 删除荣耀擂台不存在英雄
		if formationType == int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) {
			for _, v := range formationDetail {
				newHeroIdList := make([]int64, 0)
				for _, heroId := range v.HeroOwnIDList {
					if player.HeroDetailsModel.GetHero(heroId) != nil {
						newHeroIdList = append(newHeroIdList, heroId)
					} else if otherHeroDetail, ok := otherPlayerHeroDetails[heroId]; ok && otherHeroDetail.HeroID != 0 {
						newHeroIdList = append(newHeroIdList, heroId)
					}
				}
				if len(newHeroIdList) != len(v.HeroOwnIDList) {
					player.HeroFormationModel.UpdateHeroOwnIDListByTypeAndId(formationType, v.FormationID, newHeroIdList)
				}
			}
		}

		for _, formation := range formationDetail {
			if formation == nil {
				continue
			}
			detail := &pb.HeroFormationDetails{
				FormationId:   int32(formation.FormationID),
				HeroOwnIds:    formation.HeroOwnIDList,
				FormationType: formationType,
			}
			//detail.ComberSkillId, detail.ComberSkillLevel, detail.ClassSynergyId = GetHeroFormationComberSkill(detail, player)
			formations = append(formations, detail)
		}
	}

	resp := &pb.HeroFormationDetailsResp{
		Formations: formations,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_FORMATION_DETAILS_RESP, resp)

}

func HeroSetFormationHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero set formation req", player)

	req, ok := message.(*pb.HeroSetFormationReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// 目前只有一个阵容
	if req.FormationId > 1 {
		if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_EXPEDITION) {
			platformLogger.InfoWithUser("formation id exceeds limit", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
			return
		} else {
			// 钻石购买派遣队列
			BuyDispatchFormationNum := player.StaticData.GetBuyDispatchFormationNum()
			// 特权解锁派遣队列
			vipCards, err := vipCard.Service.GetAllFunctionValues(player)
			if err != nil {
				platformLogger.ErrorWithUser("GetVipCardInfoList failed", player, err)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
				return
			}
			formationNum := int32(0)
			if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_FIRST]; ok {
				formationNum++
			}
			if _, ok := vipCards[enum.VIP_PRIVILEGE_EXPEDITION_QUEUE_SECOND]; ok {
				formationNum++
			}
			if req.FormationId > BuyDispatchFormationNum+formationNum+1 {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
				return
			}
		}
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	otherPlayerHeroDetails := make(map[int64]*model.HeroDetailsEntity)
	//if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) || player.PlayerGloryArenaModel == nil {
	//	messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
	//	return
	//}
	selectedHeroes := player.PlayerGloryArenaModel.SelectedHeroes
	for _, selected := range selectedHeroes {
		if selected == nil || selected.SelectedHero == nil {
			continue
		}
		heroDetail := &model.HeroDetailsEntity{
			HeroID: selected.SelectedHero.Id,
			Power:  selected.SelectedHero.Attr[enum.AttributeBasicCombatPower],
		}
		otherPlayerHeroDetails[selected.SelectedHero.Uid] = heroDetail
	}

	heroOwnIdList := make(map[int64]int32)
	IdList := make(map[int64]int32)
	for _, hid := range req.HeroOwnIds {
		heroDetail := player.HeroDetailsModel.GetHero(hid)
		if heroDetail == nil {
			if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) || player.PlayerGloryArenaModel == nil {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
				return
			}
			if otherHeroDetail, ok := otherPlayerHeroDetails[hid]; !ok || otherHeroDetail.HeroID == 0 {
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
				return
			}
			heroDetail = otherPlayerHeroDetails[hid]
		}
		if _, ok := IdList[hid]; ok {
			platformLogger.InfoWithUser("阵容中存在重复英雄", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_FORMATION_HERO_REPEAT)
			return
		}
		if _, ok := IdList[heroDetail.HeroID]; ok {
			platformLogger.InfoWithUser("阵容中存在重复英雄类型", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_FORMATION_HERO_REPEAT)
			return
		}
		IdList[hid] = 1
		IdList[heroDetail.HeroID] = 1
	}

	if len(req.HeroOwnIds) <= 0 {
		platformLogger.InfoWithUser("formation can not null", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
		return
	}
	formationIdUnlock := gameConfig.GetCompositionTypeCfg(req.FormationType)
	if formationIdUnlock == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
		return
	}
	for i := int32(0); i < int32(len(req.HeroOwnIds)); i++ {
		if formationIdUnlock.Unlock[i] != 0 && !unlockService.CheckUnlock(formationIdUnlock.Unlock[i], player) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
			return
		}
	}
	if _, ok := player.HeroFormationModel.Entities[(req.FormationType)][(req.FormationId)]; !ok {
		// 新增阵容 - 所有请求英雄都是上阵操作
		for _, heroOwnId := range req.HeroOwnIds {
			heroDetail := player.HeroDetailsModel.GetHero(heroOwnId)
			if heroDetail == nil {
				if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) || player.PlayerGloryArenaModel == nil {
					messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
					return
				}
				if otherHeroDetail, ok := otherPlayerHeroDetails[heroOwnId]; !ok || otherHeroDetail.HeroID == 0 {
					messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
					return
				}
				heroDetail = otherPlayerHeroDetails[heroOwnId]
			}
			operationLogService.OnUserHeroFormation(player.GetUserId(), req.FormationType, 1, int32(heroDetail.HeroID), heroOwnId)
		}
		newFormation := &model.HeroFormationEntity{
			UserID:        player.GetUserId(),
			FormationID:   int32(req.FormationId),
			HeroOwnIDList: req.HeroOwnIds,
			FormationType: req.FormationType,
			IsActive:      true,
		}
		err := player.HeroFormationModel.AddHeroFormation(req.FormationType, req.FormationId, newFormation)
		if err != nil {
			platformLogger.InfoWithUser("location not unlock", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
			return
		}
	} else {
		// 获取旧阵容英雄列表
		for _, v := range player.HeroFormationModel.Entities[(req.FormationType)][(req.FormationId)].HeroOwnIDList {
			heroOwnIdList[v] = 1
		}

		// 计算差异并记录日志
		newHeroMap := make(map[int64]bool)
		for _, heroOwnId := range req.HeroOwnIds {
			newHeroMap[heroOwnId] = true
		}

		// 找出下阵的英雄（在旧阵容中但不在新阵容中）
		for heroOwnId := range heroOwnIdList {
			if !newHeroMap[heroOwnId] {
				heroDetail := player.HeroDetailsModel.GetHero(heroOwnId)
				if heroDetail == nil {
					if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) || player.PlayerGloryArenaModel == nil {
						messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
						return
					}
					if otherHeroDetail, ok := otherPlayerHeroDetails[heroOwnId]; !ok || otherHeroDetail.HeroID == 0 {
						// 从阵容中删除不存在英雄
						newHeroList := make([]int64, 0)
						for _, v := range player.HeroFormationModel.Entities[(req.FormationType)][(req.FormationId)].HeroOwnIDList {
							if v != heroOwnId {
								newHeroList = append(newHeroList, v)
							}
						}
						player.HeroFormationModel.UpdateHeroOwnIDListByTypeAndId(req.FormationType, req.FormationId, newHeroList)
					}
					heroDetail = otherPlayerHeroDetails[heroOwnId]
				}

				operationLogService.OnUserHeroFormation(player.GetUserId(), req.FormationType, 0, int32(heroDetail.HeroID), heroOwnId)

			}
		}

		// 找出上阵的英雄（在新阵容中但不在旧阵容中）
		for _, heroOwnId := range req.HeroOwnIds {
			if _, ok := heroOwnIdList[heroOwnId]; !ok {
				heroDetail := player.HeroDetailsModel.GetHero(heroOwnId)
				if heroDetail == nil {
					if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) || player.PlayerGloryArenaModel == nil {
						messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
						return
					}
					if otherHeroDetail, ok := otherPlayerHeroDetails[heroOwnId]; !ok || otherHeroDetail.HeroID == 0 {
						messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, pb.ERROR_CODE_HERO_SET_FORMATION_FAIL)
						return
					}
					heroDetail = otherPlayerHeroDetails[heroOwnId]
				}

				operationLogService.OnUserHeroFormation(player.GetUserId(), req.FormationType, 1, int32(heroDetail.HeroID), heroOwnId)

			}
		}

		player.HeroFormationModel.UpdateHeroOwnIDListByTypeAndId(req.FormationType, req.FormationId, req.HeroOwnIds)
	}
	power := int64(0)
	if req.FormationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN) {
		for _, hid := range req.HeroOwnIds {
			if req.FormationType == int32(pb.HeroFormationType_HERO_FORMATION_TYPE_HONOR_ARENA) && player.HeroDetailsModel.GetHero(hid) == nil {
				power += otherPlayerHeroDetails[hid].Power
			} else {
				power += player.GetHeroAttrForBattle(hid, req.FormationType, req.FormationId)[enum.AttributeBasicCombatPower]
			}
		}
	}
	resp := &pb.HeroSetFormationResp{
		IsSuccess:      true,
		FormationPower: power,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_SET_FORMATION_RESP, resp)
}

func HeroRebithHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero rebith req", player)

	req, ok := message.(*pb.HeroRebithReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_REBITH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	heroOwenId := req.HeroOwnId
	heroDetail := player.HeroDetailsModel.GetHero(heroOwenId)
	if heroDetail == nil {
		platformLogger.ErrorWithUser("hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_REBITH_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	// 使用 invService.AddItem 发放返还物品（不再调用 hero.SendRewardToPlayer）
	items := hero.GetHeroRebithAllItem(heroDetail)
	err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_HERO_REBIRTH)
	if err != nil {
		platformLogger.ErrorWithUser("hero rebith add item error", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_REBITH_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}

	// 重置等级为 1
	player.HeroDetailsModel.UpdateLevel(heroOwenId, 1)
	player.HeroDetailsModel.UpdateBreakNum(heroOwenId, 0)
	// 返回响应并推送物品变更
	resp := &pb.HeroRebithResp{
		IsSuccess: true,
		HeroInfo:  player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.HeroOwnId),
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_REBITH_RESP, resp)
}

func GetPlayerHeroDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get player hero detail req", player)

	req, ok := message.(*pb.GetPlayerHeroDetailReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	userId := req.Uid
	heroOwnId := req.HeroOwnId
	operationType := req.OperationType

	var detail *model.HeroDetailsEntity
	if userId != 0 {
		detail = hero.QueryHeroDetailByUserIdAndHeroOwnId(player, heroOwnId)
		if detail == nil {
			platformLogger.ErrorWithUser("hero not exist", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
			return
		}
	} else {
		detail = player.HeroDetailsModel.GetHero(heroOwnId)
		if detail == nil {
			platformLogger.ErrorWithUser("hero not exist", player, nil)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
			return
		}
	}

	var isCultivated bool = false
	if detail.Level > 1 || detail.StarLevel > gameConfig.GetHeroBaseCfg(int32(detail.HeroID)).HeroStar {
		isCultivated = true
	}
	if !isCultivated {
		for _, v := range detail.EquipmentId {
			if v != 0 {
				isCultivated = true
				break
			}
		}
	}

	res := &pb.HeroBagInfo{}
	if operationType == 0 {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_RESP, &pb.GetPlayerHeroDetailResp{})
	} else {
		res = &pb.HeroBagInfo{
			HeroOwnId:     detail.HeroOwnID,
			HeroId:        detail.HeroID,
			Level:         detail.Level,
			Star:          detail.StarLevel,
			EvolutionPath: detail.EvolutionPath,
			EvolutionTime: detail.EvolutionUpdateTime,
			Attributes:    player.GetHeroAttr(detail.HeroOwnID),
			BreakNum:      detail.BreakNum,
			IsCultivated:  isCultivated,
		}
	}
	player.HeroDetailsModel.GetHeroSkillInfo(res)
	resp := &pb.GetPlayerHeroDetailResp{
		HeroInfo: res,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_PLAYER_HERO_DETAIL_RESP, resp)
}

func HeroExchangeNotLossHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero exchange not loss req", player)

	req, ok := message.(*pb.HeroExchangeNotLossReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	oldHeroDetail := player.HeroDetailsModel.GetHero(req.OldHeroOwnId)
	if oldHeroDetail == nil {
		platformLogger.ErrorWithUser("old hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	newHeroDetail := player.HeroDetailsModel.GetHero(req.NewHeroOwnId)
	if newHeroDetail == nil {
		platformLogger.ErrorWithUser("new hero not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, pb.ERROR_CODE_HERO_NOT_FOUND)
		return
	}

	formation := player.HeroFormationModel.GetHeroFormation(req.FormationType, req.FormationId)
	if formation == nil {
		platformLogger.ErrorWithUser("formation not exist", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, pb.ERROR_CODE_HERO_FORMATION_INVALID)
		return
	}

	newLevel := newHeroDetail.Level
	oldLevel := oldHeroDetail.Level
	newBreakNum := newHeroDetail.BreakNum
	oldBreakNum := oldHeroDetail.BreakNum
	player.HeroDetailsModel.UpdateLevel(req.NewHeroOwnId, oldLevel)
	player.HeroDetailsModel.UpdateLevel(req.OldHeroOwnId, newLevel)
	player.HeroDetailsModel.UpdateBreakNum(req.NewHeroOwnId, oldBreakNum)
	player.HeroDetailsModel.UpdateBreakNum(req.OldHeroOwnId, newBreakNum)

	// 替换阵容中的英雄
	newHeroOwnIDList := make([]int64, 0)
	for _, hid := range formation.HeroOwnIDList {
		if hid == req.OldHeroOwnId {
			newHeroOwnIDList = append(newHeroOwnIDList, req.NewHeroOwnId)
		} else {
			newHeroOwnIDList = append(newHeroOwnIDList, hid)
		}
	}
	player.HeroFormationModel.UpdateHeroOwnIDListByTypeAndId((req.FormationType), (req.FormationId), newHeroOwnIDList)

	heroFormation := &pb.HeroFormationDetails{
		FormationId:   int32(formation.FormationID),
		HeroOwnIds:    newHeroOwnIDList,
		FormationType: req.FormationType,
	}
	//heroFormation.ComberSkillId, heroFormation.ComberSkillLevel, heroFormation.ClassSynergyId = hero.GetHeroFormationComberSkill(heroFormation, player)

	resp := &pb.HeroExchangeNotLossResp{
		IsSuccess:        true,
		OldHeroInfo:      player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.OldHeroOwnId),
		NewHeroInfo:      player.HeroDetailsModel.GetHeroInfoByOwnID(player, req.NewHeroOwnId),
		UpdatedFormation: heroFormation,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, resp)
}

func GetHeroMaxDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("hero max detail req", player)

	req, ok := message.(*pb.GetHeroMaxDetailReq)

	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_HERO_EXCHANGE_NOT_LOSS_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if err := RequirePlayerHeroModels(player); err != nil {
		return
	}

	heroId := req.HeroId
	level := gameConfig.GetHeroBaseCfg(heroId).Cap[0]
	star := gameConfig.GetHeroBaseCfg(heroId).Cap[1]
	potential := gameConfig.GetHeroBaseCfg(heroId).HeroPotential
	class := gameConfig.GetHeroBaseCfg(heroId).HeroClass
	secondAttrMap := make(map[int32]int64)
	attrMap := make(map[int32]int64)
	for _, attrId := range enum.SecondClassAttrIdMap {
		attrMap[attrId] = gameConfig.GetHeroBaseAttr(heroId, attrId)
		secondAttrMap[attrId] = gameConfig.GetHeroBaseAttr(heroId, attrId)
		attrMap[attrId] = gameConfig.GetSecondAttr(potential, class, level, gameConfig.GetMaxBreakNum(potential, class), star, attrId)
		secondAttrMap[attrId] = gameConfig.GetSecondAttr(potential, class, level, gameConfig.GetMaxBreakNum(potential, class), star, attrId)
	}
	for _, attrId := range enum.FirstClassAttrIdMap {
		attrMap[attrId] = gameConfig.GetHeroBaseAttr(heroId, attrId)
		secondAttrMap[attrId] = gameConfig.GetHeroBaseAttr(heroId, attrId)
		attrMap[attrId] = gameConfig.GetSecondAttr(potential, class, level, gameConfig.GetMaxBreakNum(potential, class), star, attrId)
		secondAttrMap[attrId] = gameConfig.GetSecondAttr(potential, class, level, gameConfig.GetMaxBreakNum(potential, class), star, attrId)
	}
	attrMap[enum.AttributeBasicHp] = attrMap[enum.AttributeBasicHp] + int64(math.Ceil(float64(secondAttrMap[enum.AttributeBasicBaseHp])*float64((1+float64(secondAttrMap[enum.AttributeBasicBaseHpPercent])/float64(10000)))))
	attrMap[enum.AttributeBasicPhysicalAttack] = attrMap[enum.AttributeBasicPhysicalAttack] + int64(math.Ceil(float64(secondAttrMap[enum.AttributeBasicBasePhysicalAttack])*float64((1+float64(secondAttrMap[enum.AttributeBasicBasePhysicalAttackPercent])/float64(10000)))))
	attrMap[enum.AttributeBasicMagicalAttack] = attrMap[enum.AttributeBasicMagicalAttack] + int64(math.Ceil(float64(secondAttrMap[enum.AttributeBasicBaseMagicalAttack])*float64((1+float64(secondAttrMap[enum.AttributeBasicBaseMagicalAttackPercent])/float64(10000)))))
	attrMap[enum.AttributeBasicPhysicalDefense] = attrMap[enum.AttributeBasicPhysicalDefense] + int64(math.Ceil(float64(secondAttrMap[enum.AttributeBasicBasePhysicalDefense])*float64((1+float64(secondAttrMap[enum.AttributeBasicBasePhysicalDefensePercent])/float64(10000)))))
	attrMap[enum.AttributeBasicMagicalDefense] = attrMap[enum.AttributeBasicMagicalDefense] + int64(math.Ceil(float64(secondAttrMap[enum.AttributeBasicBaseMagicalDefense])*float64((1+float64(secondAttrMap[enum.AttributeBasicBaseMagicalDefensePercent])/float64(10000)))))

	// 战力
	power := gameConfig.GetAttrMapPower(player.ArchitectureModel.GetMainLevel(), attrMap)
	attrMap[enum.AttributeBasicCombatPower] = int64(math.Ceil(power))

	res := make([]*pb.HeroBagInfo, 0)

	if gameConfig.GetStarEffectCfg(heroId, star).ChangeClass != nil {
		for _, v := range gameConfig.GetStarEffectCfg(heroId, star).ChangeClass {
			detail := &pb.HeroBagInfo{
				HeroId:        int64(heroId),
				Level:         level,
				EvolutionPath: v,
				Attributes:    attrMap,
				Star:          star,
				BreakNum:      gameConfig.GetMaxBreakNum(potential, class),
			}
			player.HeroDetailsModel.GetHeroSkillInfo(detail)
			res = append(res, detail)
		}
	} else {
		detail := &pb.HeroBagInfo{
			HeroId:        int64(heroId),
			Level:         level,
			EvolutionPath: gameConfig.GetHeroBaseCfg(heroId).HeroClass,
			Attributes:    attrMap,
			Star:          star,
			BreakNum:      gameConfig.GetMaxBreakNum(potential, class),
		}
		player.HeroDetailsModel.GetHeroSkillInfo(detail)
		res = append(res, detail)
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_HERO_MAX_DETAIL_RESP, &pb.GetHeroMaxDetailResp{HeroInfo: res, ComboSkill: gameConfig.GetHeroBaseCfg(heroId).ComboSkill})
}

func BuyDispatchFormationHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("buy dispatch formation req", player)

	_, ok := message.(*pb.BuyDispatchFormationReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	cfg := gameConfig.GetConstantCfg(gameConfig.CONSTANT_dispatchQueues)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	playerBuyDispatchFormationNum := player.StaticData.GetBuyDispatchFormationNum()
	if playerBuyDispatchFormationNum >= int32(len(cfg.Value)) {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, pb.ERROR_CODE_DISPATCH_FORMATION_NUM_LIMIT)
		return
	}
	if flag, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: enum.DIAMOND_ITEM_ID, Num: int64(cfg.Value[playerBuyDispatchFormationNum])}); !flag || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err := itemService.RemoveItem(player, &gameConfig.ItemConfig{ID: enum.DIAMOND_ITEM_ID, Num: int64(cfg.Value[playerBuyDispatchFormationNum])}, enum.ITEM_CHANGE_REASON_BUY_DISPATCH_FORMATION)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
		return
	}
	player.StaticData.UpdateBuyDispatchFormationNum(playerBuyDispatchFormationNum + 1)
	messageSender.SendMessage(player, pb.MESSAGE_ID_BUY_DISPATCH_FORMATION_RESP, &pb.BuyDispatchFormationResp{})
}
