package hero

import (
	"errors"

	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"

	"github.com/drop/GoServer/server/logic/platform/easyDB"

	"github.com/drop/GoServer/server/logic/platform/platformLogger"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
)

var HeroBagMaxNum = 1200

var HeroIdGenerator *tool.IdGenerator

func InitHero() {
	HeroIdGenerator = tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_HERO))
}

func CheckLevelUpItem(heroDetail *model.HeroDetailsEntity, needLevel int32) map[int32]int64 {
	res := make(map[int32]int64)
	// 占位逻辑：根据 needLevel 构建空需求
	heroPotential := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID)).HeroPotential
	heroClass := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID)).HeroClass
	for i := heroDetail.Level; i < heroDetail.Level+needLevel; i++ {
		for id, j := range gameConfig.GetHeroLevelCfg(heroPotential, heroClass, ((i-1)%100)+1).LevelMaterials {

			// 获取材料倍率
			cost := gameConfig.GetLevelRatioCfg((i-1)/100 + 1).CostRatio[id]

			var add = int64(gameConfig.GetHeroLevelCfg(heroPotential, heroClass, ((i-1)%100)+1).Cost[id]*cost) / 10000
			if _, ok := res[j]; !ok {
				res[j] = add
			} else {
				res[j] += add
			}
		}
	}
	return res
}

func CheckBreakItem(detail *model.HeroDetailsEntity) map[int32]int64 {
	res := make(map[int32]int64)

	heroPotential := gameConfig.GetHeroBaseCfg(int32(detail.HeroID)).HeroPotential
	rank := detail.BreakNum + 1
	heroClass := gameConfig.GetHeroBaseCfg(int32(detail.HeroID)).HeroClass
	if gameConfig.GetHeroBreakCfg(heroPotential, heroClass, rank) == nil {
		return nil
	}
	for id, j := range gameConfig.GetHeroBreakCfg(heroPotential, heroClass, rank).BreakMaterials {
		if _, ok := res[j]; !ok {
			res[j] = int64(gameConfig.GetHeroBreakCfg(heroPotential, heroClass, rank).BreakCost[id])
		} else {
			res[j] += int64(gameConfig.GetHeroBreakCfg(heroPotential, heroClass, rank).BreakCost[id])
		}

	}
	return res
}

// 时间，星级，副本，专转职方向是否允许
func CheckEvolutionConditions(detail *model.HeroDetailsEntity, evolution int32, player *model.PlayerModel) bool {
	if detail == nil || player == nil {
		return false
	}
	heroId := detail.HeroID
	star := detail.StarLevel
	heroClassCfg := gameConfig.GetHeroClassCfg(int32(detail.EvolutionPath))
	if heroClassCfg == nil {
		return false
	}
	vis := 0
	if heroClassCfg.ChangeClass != nil {
		for _, v := range heroClassCfg.ChangeClass {
			if v == evolution {
				vis = 1
				break
			}
		}
		if vis == 0 {
			if heroClassCfg.SwitchClass == evolution {
				vis = 2
			}
		}
	} else {
		if heroClassCfg.SwitchClass == evolution {
			vis = 2
		}
	}
	if vis == 0 {
		platformLogger.InfoWithUser("转职方向不合法", player)
		return false
	}
	for i := gameConfig.GetHeroBaseCfg(int32(detail.HeroID)).HeroStar; i <= star; i++ {
		starCfg := gameConfig.GetStarEffectCfg(int32(heroId), i)
		flag := false
		if starCfg == nil {
			continue
		}
		for _, v := range starCfg.ChangeClass {
			if v == evolution {
				flag = true
				break
			}
		}
		if flag {
			if starCfg.UnlockClass != 0 && !unlockService.CheckUnlock(starCfg.UnlockClass, player) {
				platformLogger.InfoWithUser("转职副本未解锁", player)
				return false
			}
			break
		}
	}

	if tool.UnixNowMilli()-detail.EvolutionUpdateTime < gameConfig.HeroEvolutionTimeCfg && vis == 2 {
		platformLogger.InfoWithUser("转职cd时间未到", player)
		return false
	}
	if vis == 2 {
		player.HeroDetailsModel.UpdateEvolutionUpdateTime(detail.HeroOwnID, tool.UnixNowMilli())
	}

	return true
}

// 校验英雄升星材料是否满足要求
func HeroStarUpUseHero(heroDetail *model.HeroDetailsEntity, player *model.PlayerModel, allItems map[string]*pb.MaterialList) bool {
	if heroDetail == nil || player == nil {
		return false
	}
	// 获取目标英雄配置
	baseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
	if baseCfg == nil {
		platformLogger.ErrorWithUser("目标英雄 baseCfg nil", player, nil)
		return false
	}
	heroPotential := baseCfg.HeroPotential
	heroClass := baseCfg.HeroClass
	heroStar := heroDetail.StarLevel + 1

	starCfg := gameConfig.GetHeroStarCfg(heroPotential, heroClass, heroStar)
	if starCfg == nil {
		platformLogger.ErrorWithUser("heroStar cfg nil", player, nil)
		return false
	}

	used := make(map[int64]bool) // 防止同一个 heroOwnId 被重复使用

	for key, value := range allItems {
		var required []int32
		switch key {
		case "baseCard":
			required = starCfg.BaseCard
		case "sameClass":
			required = starCfg.SameClass
		case "anyClass":
			required = starCfg.AnyClass
		case "universalHero":
			continue
		default:
			return false
		}

		if value == nil {
			platformLogger.ErrorWithUser("材料列表为空", player, nil)
			return false
		}

		if len(required) < 2 {
			if len(value.HeroOwnIds) > 0 {
				platformLogger.InfoWithUser("材料未配置但传入了材料，拒绝请求", player)
				return false
			}
			continue
		}
		requiredStar := required[0]
		requiredCount := required[1]

		if int32(len(value.HeroOwnIds)) != requiredCount {
			if key == "sameClass" {
				// 检查是否有 universalHero 补充
				if allItems["universalHero"] != nil && allItems["universalHero"].HeroOwnIds != nil {
					universalCount := int32(len(allItems["universalHero"].HeroOwnIds))
					if int32(len(value.HeroOwnIds))+universalCount != requiredCount {
						platformLogger.InfoWithUser("材料英雄数量不足或超出", player)
						return false
					}
					// 校验通用英雄道具是否在配置中
					for _, itemID := range allItems["universalHero"].HeroOwnIds {
						found := false
						for _, cfgItemID := range starCfg.UniversalHero {
							if int32(itemID) == cfgItemID {
								found = true
								break
							}
						}
						if !found {
							platformLogger.InfoWithUser("通用英雄道具不在配置中", player)
							return false
						}
					}
				} else {
					platformLogger.InfoWithUser("材料英雄数量不足或超出", player)
					return false
				}
			} else {
				platformLogger.InfoWithUser("材料英雄数量不足或超出", player)
				return false
			}
		}

		for _, ownID := range value.HeroOwnIds {
			// 不能用目标英雄自身做材料
			if heroDetail.HeroOwnID != 0 && ownID == heroDetail.HeroOwnID {
				platformLogger.InfoWithUser("不能使用目标英雄自身作为材料", player)
				return false
			}
			// 防重复使用
			if used[ownID] {
				platformLogger.InfoWithUser("材料重复使用", player)
				return false
			}
			used[ownID] = true

			// 存在性与删除检查
			if player.HeroDetailsModel == nil || player.HeroDetailsModel.Entities == nil || player.HeroDetailsModel.Entities[ownID] == nil {
				platformLogger.ErrorWithUser("材料英雄不存在", player, nil)
				return false
			}
			mat := player.HeroDetailsModel.Entities[ownID]
			if mat.IsDeleted {
				platformLogger.ErrorWithUser("材料英雄已删除", player, nil)
				return false
			}
			// 星级校验
			if mat.StarLevel != requiredStar {
				platformLogger.InfoWithUser("材料英雄星级不足", player)
				return false
			}

			// 类型特有校验
			switch key {
			case "baseCard":
				// 同名英雄（HeroID 必须一致）
				if mat.HeroID != heroDetail.HeroID {
					platformLogger.InfoWithUser("baseCard 必须为同名英雄", player)
					return false
				}
			case "sameClass":
				// 同职业英雄
				matBase := gameConfig.GetHeroBaseCfg(int32(mat.HeroID))
				if matBase == nil || matBase.HeroClass != heroClass {
					platformLogger.InfoWithUser("sameClass 必须为同职业英雄", player)
					return false
				}
			default:
				// anyClass 无额外校验
			}
		}
	}

	return true
}

// 计算未领取图鉴积分
func CalculateUnclaimedScore(player *model.PlayerModel, heroId int64) int32 {
	if player == nil || player.HeroAlbumModel == nil || player.HeroAlbumModel.Entities == nil {
		return 0
	}
	album, ok := player.HeroAlbumModel.Entities[heroId]
	if !ok || album == nil {
		return 0
	}
	addScore := gameConfig.GetHeroCodexCfg(int32(heroId)).StarPoints
	return (album.HistoryMaxStar - album.ClaimedStar) * addScore
}

// 获取玩家英雄背包信息（返回 heroOwnID -> *HeroBagInfo）
func GetHeroBagInfo(player *model.PlayerModel) map[int64]*pb.HeroBagInfo {
	if player == nil {
		platformLogger.ErrorWithUser("GetHeroBagInfo player is nil", player, errors.New("player is nil"))
		return nil
	}

	var heroInfoMap = make(map[int64]*pb.HeroBagInfo)
	for key, value := range player.HeroDetailsModel.Entities {
		if value == nil || value.IsDeleted {
			continue
		}
		// 获取英雄信息
		res := player.HeroDetailsModel.GetHeroInfoByOwnID(player, key)
		heroInfoMap[value.HeroOwnID] = res
	}
	return heroInfoMap
}

func GetHeroRebithAllItem(detail *model.HeroDetailsEntity) []*gameConfig.ItemConfig {
	if detail.Level == 1 {
		return nil
	}
	res := make([]*gameConfig.ItemConfig, 0)

	baseCfg := gameConfig.GetHeroBaseCfg(int32(detail.HeroID))
	if baseCfg == nil {
		return res
	}
	heroPotential := baseCfg.HeroPotential
	heroClass := baseCfg.HeroClass
	rank := detail.BreakNum
	heroLevel := detail.Level

	agg := make(map[int32]int64)

	// 突破材料（带越界保护）
	for i := int32(1); i <= rank; i++ {
		breakCfg := gameConfig.GetHeroBreakCfg(heroPotential, heroClass, i)
		if breakCfg == nil {
			continue
		}
		for idx, matID := range breakCfg.BreakMaterials {
			if idx < 0 || idx >= len(breakCfg.BreakCost) {
				continue
			}
			agg[matID] += int64(breakCfg.BreakCost[idx])
		}
	}

	// 升级材料：复用 CheckLevelUpItem（返回 map[int32]*int32）
	if heroLevel > 1 {
		detail.Level = 1
		levelMap := CheckLevelUpItem(detail, heroLevel-1)
		if levelMap != nil {
			for id, pnum := range levelMap {
				agg[id] += pnum
			}
		}
	}

	// 转为切片返回
	for id, num := range agg {
		res = append(res, &gameConfig.ItemConfig{ID: id, Num: int64(num)})
	}
	return res
}

func QueryHeroDetailByUserIdAndHeroOwnId(player *model.PlayerModel, heroOwnId int64) *model.HeroDetailsEntity {
	res := &model.HeroDetailsEntity{}
	if heroOwnId == 0 {
		return nil
	}
	if player != nil {
		if player.HeroDetailsModel != nil && player.HeroDetailsModel.Entities != nil {
			if detail, ok := player.HeroDetailsModel.Entities[heroOwnId]; ok && detail != nil {
				return detail
			}
		}
	}
	ent, err := easyDB.GetPlayerEntityByWhere[model.HeroDetailsEntity](map[string]interface{}{"user_id": player.GetUserId(), "hero_own_id": heroOwnId})
	if err != nil {
		platformLogger.ErrorWithUser("QueryHeroDetailByUserIdAndHeroOwnId is fail ", nil, err)
		return res
	}
	if ent == nil {
		return nil
	}

	return ent
}

func AddHeroDetail(player *model.PlayerModel, heroID int64) (*model.HeroDetailsEntity, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}
	if len(player.HeroDetailsModel.Entities)+1 > HeroBagMaxNum {
		return nil, errors.New("hero bag full")
	}
	heroOwnId := HeroIdGenerator.NextId()
	flag, err := player.HeroDetailsModel.AddHero(player, heroID, heroOwnId)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, errors.New("add hero detail fail")
	}
	player.HeroDetailsModel.AddHeroForMemory(heroOwnId)
	//messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ADD_HERO_DETAIL, &pb.PushAddHeroDetail{
	//	HeroInfo: player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroOwnId),
	//})
	return player.HeroDetailsModel.Entities[heroOwnId], nil
}

//func GetHeroFormationComberSkill(detail *pb.HeroFormationDetails, player *model.PlayerModel) ([]int32, []int32, []int32) {
//	heroIdMap := make(map[int64]bool)
//	heroOwnIdMap := make(map[int64]*model.HeroDetailsEntity)
//	comberSkillMap := make(map[int32]bool)
//	comberskillId := make([]int32, 0)
//	comberskillLevel := make([]int32, 0)
//	classSynergyId := make([]int32, 0)
//	for _, heroOwnId := range detail.HeroOwnIds {
//		hero := player.HeroDetailsModel.GetHero(heroOwnId)
//		if hero == nil {
//			continue
//		}
//		heroId := hero.HeroID
//		heroOwnIdMap[heroOwnId] = hero
//		heroIdMap[heroId] = true
//		heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroId))
//		if heroBaseCfg == nil {
//			continue
//		}
//		for _, comberId := range heroBaseCfg.ComboSkill {
//			comberSkillMap[comberId] = true
//		}
//	}
//	//classSynergyId = player.GetClassSynergy(heroOwnIdMap, detail.FormationType, detail.FormationId)
//	for comberId, _ := range comberSkillMap {
//		comboSkillCfg := gameConfig.GetComboSkillCfg(comberId)
//		if comboSkillCfg == nil {
//			continue
//		}
//		for i := len(comboSkillCfg.Level) - 1; i >= 0; i-- {
//			flag := true
//			//for _, heroid := range comboSkillCfg.LimitedHero[i] {
//			//	if _, ok := heroIdMap[int64(heroid)]; !ok {
//			//		flag = false
//			//		break
//			//	}
//			//}
//			if flag {
//				comberskillId = append(comberskillId, comberId)
//				comberskillLevel = append(comberskillLevel, int32(i+1))
//				break
//			}
//		}
//	}
//	return comberskillId, comberskillLevel, classSynergyId
//}

func GetHeroFormationInfo(player *model.PlayerModel) []*pb.HeroFormationDetails {
	var formations []*pb.HeroFormationDetails
	for formationType, formationDetail := range player.HeroFormationModel.Entities {
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

	return formations
}

var unlockService logicCommon.UnlockServiceInterface
var messageSender logicCommon.MessageSenderInterface

func InitHeroService(unlock logicCommon.UnlockServiceInterface, sender logicCommon.MessageSenderInterface) {
	unlockService = unlock
	messageSender = sender
}
