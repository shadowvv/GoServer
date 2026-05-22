package model

import (
	"math"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"github.com/drop/GoServer/server/tool"
)

func NewPlayerModel() *PlayerModel {
	return &PlayerModel{
		PlayerModels:            make([]logicCommon.PlayerModelInterface, 0),
		HeroAttrModels:          make([]logicCommon.HeroAttrInterface, 0),
		LastHeartbeatTime:       tool.UnixNowMilli(),
		LastSendChatMessageTime: 0,
		PlayerCacheInfo: &logicCommon.PlayerRedisInfo{
			BasicInfo: &logicCommon.PlayerBasicInfo{},
			BattleInfo: &logicCommon.PlayerBattleInfo{
				FormationInfo:   make(map[int32]*logicCommon.FormationBasicInfo),
				FormationHeroes: make(map[int64]*logicCommon.HeroBasicInfo),
			},
		},
	}
}

type PlayerModel struct {
	Session        serviceInterface.SessionInterface
	PlayerModels   []logicCommon.PlayerModelInterface
	HeroAttrModels []logicCommon.HeroAttrInterface

	User                  *UserModel
	StaticData            *StaticDataModel
	StoryTriggerModel     *StoryTriggerModel
	InventoryModel        *InventoryModel
	PlayerInstanceModel   *PlayerInstanceModel
	HeroDetailsModel      *HeroDetailsCollectionModel   // 英雄详情集合
	HeroAlbumModel        *HeroAlbumCollectionModel     // 英雄图鉴集合
	AlbumRewardModel      *AlbumRewardModel             // 图鉴奖励（单记录）
	HeroFormationModel    *HeroFormationCollectionModel // 英雄阵型集合
	TaskModel             *TaskModel                    // 任务
	EquipmentModel        *EquipmentCollectionModel     // 装备集合
	LotteryModel          *LotteryModel                 // 抽卡模型
	LoopBoxModel          *LoopBoxModel                 // 循环盒
	AdChestModel          *AdChestModel                 // 广告宝箱
	PlayerShopModel       *PlayerShopModel              // 商店
	ExpeditionModel       *ExpeditionModel
	AccessoryModel        *AccessoryModel
	AccessoryLuckyModel   *AccessoryLuckyModel
	PetModel              *PetModel         // 宠物集合（挂在英雄下面）
	PetRecruitModel       *PetRecruitModel  // 宠物招募（3选1/刷新/倒计时）
	PetAffinityModel      *PetAffinityModel // 宠物缘分（激活/升级）
	BountyModel           *BountyModel
	TaskActiveRewardModel *TaskActiveRewardModel
	PlayerActivityModel   *PlayerActivityModel
	PlayerSignModel       *PlayerSignModel // 签到模型
	IdleModel             *IdleModel       // 挂机奖励模型
	PlayerAdventureModel  *PlayerAdventureModel
	VipCardModel          *VipCardModel         // 特权卡模型
	PassModel             *PassModel            // 通行证模型
	PassTaskModel         *PassCardTaskModel    // 通行证任务
	PrivilegeRewardModel  *PrivilegeRewardModel // 特权奖励模型
	ArchitectureModel     *ArchitectureModel    // 建筑信息
	StoneModel            *StoneModel           // 传承石像信息
	PlayerFunctionModel   *PlayerFunctionModel  // 功能状态
	PlayerArenaModel      *PlayerArenaModel     // 竞技场
	PlayerGloryArenaModel *PlayerGloryArenaModel
	CollectionModel       *CollectionModel
	PlayerTokenShopModel  *PlayerTokenShopModel
	LumberModel           *LumberModel
	FurnitureModel        *FurnitureModel
	TrialModel            *TrialModel
	CityAgeModel          *CityAgeModel
	TurnTableModel        *TurnTableModel
	AppearanceModel       *AppearanceModel

	SceneId           int32
	NodeId            int32
	LastHeartbeatTime int64

	PlayerCacheInfo         *logicCommon.PlayerRedisInfo
	FunctionStatus          map[enum.FunctionIdEnum]int32 // 功能状态
	LastSendChatMessageTime int64
}

var _ logicCommon.PlayerInterface = (*PlayerModel)(nil)

func (p *PlayerModel) GetNodeId() int32 {
	return p.NodeId
}

func (p *PlayerModel) GetSceneId() int32 {
	return p.SceneId
}

func (p *PlayerModel) GetLevel() int32 {
	if p.ArchitectureModel == nil {
		return 1
	}
	if p.ArchitectureModel.GetMainLevel() == 0 {
		return 1
	}
	return p.ArchitectureModel.GetMainLevel()
}

func (p *PlayerModel) GetUserId() int64 {
	return p.User.GetUserId()
}

func (p *PlayerModel) GetUserAccount() string {
	return p.User.GetAccount()
}

func (p *PlayerModel) GetUserServerId() int32 {
	return p.User.GetServerId()
}

func (p *PlayerModel) GetSession() serviceInterface.SessionInterface {
	return p.Session
}

func (p *PlayerModel) SavePlayerToDB() {
	for _, playerModel := range p.PlayerModels {
		playerModel.SaveModelToDB()
	}
}

// CheckAndPushHeroChange 在Save之前读取各HeroAttrModel的Changed，感知英雄变化并推送
// 规则：
// - 有 allDirty=true 的 model：推主线全部阵容 + PushType=1
// - 否则：有特定英雄变化的推这些英雄 + PushType=0
func (p *PlayerModel) CheckAndPushHeroChange() {
	if p.HeroDetailsModel == nil || p.HeroFormationModel == nil {
		return
	}

	// 汇总所有变化
	allDirty := false
	changedOwnIDs := make(map[int64]bool)

	for _, model := range p.HeroAttrModels {
		if model == nil {
			continue
		}
		ownIDs, dirty := model.GetChangedHeroOwnIDs()
		if dirty {
			allDirty = true
		}
		for _, ownID := range ownIDs {
			changedOwnIDs[ownID] = true
		}
	}
	ownIDs, _ := p.HeroFormationModel.GetChangedHeroOwnIDs()
	for _, ownID := range ownIDs {
		changedOwnIDs[ownID] = true
	}

	// 有全局脏标记：type=1，客户端自行拉取，服务端只发通知
	if allDirty || p.HeroDetailsModel.heroAttrTree.Root {
		p.HeroDetailsModel.refreshHeroAttrTree()
		infos := make([]*pb.HeroBagInfo, 0)
		allPower := int64(0)
		for _, v := range p.HeroFormationModel.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
			if v.IsActive == true {
				for _, heroOwnId := range v.HeroOwnIDList {
					infos = append(infos, p.HeroDetailsModel.GetHeroInfoByOwnID(p, heroOwnId))
					allPower += p.HeroDetailsModel.GetHeroInfoByOwnID(p, heroOwnId).Attributes[enum.AttributeBasicCombatPower]
				}
			}
		}
		eventServer.SubmitPlayerPowerChangeEvent(p.GetUserId(), p.GetUserServerId(), allPower)
		messageSender.SendMessage(p, pb.MESSAGE_ID_PUSH_HERO_POWER_CHANGE, &pb.PushHeroPowerChange{
			HeroInfos: infos,
			PushType:  1,
		})
		p.updatePowerRank()
		return
	}

	// 无全局脏：特定英雄变化推这些英雄 + type=0
	if len(changedOwnIDs) > 0 {
		infos := make([]*pb.HeroBagInfo, 0)
		for ownID := range changedOwnIDs {
			p.HeroDetailsModel.GetHero(ownID).isDirty = true
			if info := p.HeroDetailsModel.GetHeroInfoByOwnID(p, ownID); info != nil {
				infos = append(infos, info)
			}
		}
		if len(infos) > 0 {
			for _, info := range infos {
				if p.checkIsMainHero(info.HeroId) {
					p.updatePowerRank()
					break
				}
			}
			messageSender.SendMessage(p, pb.MESSAGE_ID_PUSH_HERO_POWER_CHANGE, &pb.PushHeroPowerChange{
				HeroInfos: infos,
				PushType:  0,
			})
		}
		// 计算总战力
		allPower := int64(0)
		for _, v := range p.HeroFormationModel.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
			if v.IsActive == true {
				for _, heroOwnId := range v.HeroOwnIDList {
					allPower += p.HeroDetailsModel.GetHeroInfoByOwnID(p, heroOwnId).Attributes[enum.AttributeBasicCombatPower]
				}
			}
		}
		eventServer.SubmitPlayerPowerChangeEvent(p.GetUserId(), p.GetUserServerId(), allPower)
		p.updatePowerRank()
		return
	}
}

func (p *PlayerModel) checkIsMainHero(heroId int64) bool {
	formation := p.HeroFormationModel.GetHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN), 1)
	if formation == nil {
		return false
	}
	for _, playerHeroId := range formation.HeroOwnIDList {
		if playerHeroId == heroId {
			return true
		}
	}
	return false
}

func (p *PlayerModel) Heartbeat(currentTime int64) error {
	if currentTime-p.LastHeartbeatTime < 1*500 {
		return nil
	}
	for _, model := range p.PlayerModels {
		day := tool.GetNatureDayDistance(currentTime, p.LastHeartbeatTime)
		model.Heartbeat(p.LastHeartbeatTime, currentTime, day, true)
		if day > 0 {
			enum.PublishLogin(dbService.RDB, p.GetUserId(), p.GetUserAccount(), 1)
		}
	}
	if playerHeartbeatService != nil {
		playerHeartbeatService.Heartbeat(p, currentTime)
	}
	p.LastHeartbeatTime = currentTime
	return nil
}

func (p *PlayerModel) BuildPlayerCacheInfo() {
	p.BuildPlayerBasicInfo()
	p.RefreshPlayerBattleInfo()
}

func (p *PlayerModel) BuildPlayerBasicInfo() {
	basicInfo := &logicCommon.PlayerBasicInfo{
		Id:              p.GetUserId(),
		ServerId:        p.GetUserServerId(),
		Name:            p.User.GetNickname(),
		MainCityLevel:   p.GetLevel(),
		ShowHeroId:      0,
		ShowClassId:     0,
		LastLoginTime:   p.User.GetLastLoginTime(),
		LastOfflineTime: p.User.GetLastOfflineTime(),
	}

	if p.PlayerArenaModel != nil {
		basicInfo.ArenaVersion = p.PlayerArenaModel.GetVersion()
		basicInfo.ArenaScore = p.PlayerArenaModel.GetScore()
	}

	if p.PlayerGloryArenaModel != nil {
		basicInfo.GloryArenaVersion = p.PlayerGloryArenaModel.GetPoolVersion()
		basicInfo.GloryArenaBestWinCount = p.PlayerGloryArenaModel.GetRoundBestWinCount()
	}

	if p.AppearanceModel != nil {
		basicInfo.HeadId = p.AppearanceModel.GetWearAppearance(enum.AvatarTypeHead)
		basicInfo.FrameId = p.AppearanceModel.GetWearAppearance(enum.AvatarTypeHeadFrame)
		basicInfo.BubbleId = p.AppearanceModel.GetWearAppearance(enum.AvatarTypeBubble)
		basicInfo.ImageId = p.AppearanceModel.GetWearAppearance(enum.AvatarTypeImage)
		basicInfo.Title = p.AppearanceModel.GetWearAppearance(enum.AvatarTypeTitle)
	}

	p.PlayerCacheInfo.BasicInfo = basicInfo
}

func (p *PlayerModel) UpdatePlayerBasicInfoToRedis() {
	if p == nil || p.PlayerCacheInfo == nil {
		return
	}
	p.BuildPlayerBasicInfo()
	if p.PlayerCacheInfo.BasicInfo == nil {
		return
	}
	logicCommon.UpdatePlayerBasicInfo(p.PlayerCacheInfo.BasicInfo)
}

func (p *PlayerModel) RefreshPlayerBattleInfo() {
	p.PlayerCacheInfo.BattleInfo.UserId = p.User.Entity.UserId
	formation := p.HeroFormationModel.GetHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN), 1)
	fromFormationType := pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN
	buildFormationBasicInfo(p.PlayerCacheInfo.BattleInfo, p, formation, pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN, fromFormationType)

	otherFormation := p.HeroFormationModel.GetHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_ATK), 1)
	fromFormationType = pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_ATK
	if otherFormation == nil {
		otherFormation = formation
		fromFormationType = pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN
	}
	buildFormationBasicInfo(p.PlayerCacheInfo.BattleInfo, p, otherFormation, pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_ATK, fromFormationType)

	otherFormation = p.HeroFormationModel.GetHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF), 1)
	fromFormationType = pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF
	if otherFormation == nil {
		otherFormation = formation
		fromFormationType = pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN
	}
	buildFormationBasicInfo(p.PlayerCacheInfo.BattleInfo, p, otherFormation, pb.HeroFormationType_HERO_FORMATION_TYPE_ARENA_DEF, fromFormationType)
}

func buildFormationBasicInfo(battleInfo *logicCommon.PlayerBattleInfo, player *PlayerModel, formation *HeroFormationEntity, formationId pb.HeroFormationType, fromFormationId pb.HeroFormationType) {
	battleInfo.FormationHeroes = make(map[int64]*logicCommon.HeroBasicInfo)
	if formation == nil {
		return
	}
	power := int64(0)
	formationBasicInfo := &logicCommon.FormationBasicInfo{
		Heroes:      make([]int64, 0),
		BattlePower: 0,
	}
	for _, heroId := range formation.HeroOwnIDList {
		info := player.HeroDetailsModel.GetHeroInfoByOwnID(player, heroId)
		if info == nil {
			continue
		}
		cfg := gameConfig.GetHeroBaseCfg(int32(info.HeroId))
		if cfg == nil {
			continue
		}
		formationBasicInfo.Heroes = append(formationBasicInfo.Heroes, heroId)
		info.Attributes = player.GetHeroAttr(heroId)
		power += info.Attributes[enum.AttributeBasicCombatPower]
		if _, ok := battleInfo.FormationHeroes[heroId]; ok {
			continue
		}
		skill := make([]int32, 0)
		if info.ActiveSkill != 0 {
			skill = append(skill, info.ActiveSkill)
		}
		if info.PassiveSkill1 != 0 {
			skill = append(skill, info.PassiveSkill1)
		}
		if info.PassiveSkill2 != 0 {
			skill = append(skill, info.PassiveSkill2)
		}
		if info.ClassSkill != 0 {
			skill = append(skill, info.ClassSkill)
		}
		battleInfo.FormationHeroes[heroId] = &logicCommon.HeroBasicInfo{
			Uid:         heroId,
			Id:          info.HeroId,
			Units:       gameConfig.GetHeroUnitsId(int32(info.HeroId), info.Star, info.EvolutionPath),
			Star:        info.Star,
			Level:       info.Level,
			ClassId:     info.EvolutionPath,
			AtkSpeed:    int32(info.Attributes[enum.AttributeBasicAttackSpeed]),
			MoveSpeed:   int32(info.Attributes[enum.AttributeBasicMoveSpeed]),
			PatrolRange: cfg.PatrolRange,
			AggroRange:  cfg.AggroRange,
			AttackRange: cfg.AttackRange,
			NormalAtk:   info.BasicSkill,
			Attr:        info.Attributes,
			Skill:       skill,
			PetInfo:     GetHeroPetBattleInfo(heroId, player.PetModel),
		}
	}
	formationBasicInfo.BattlePower = power
	battleInfo.FormationInfo[int32(formationId)] = formationBasicInfo
}

func (p *PlayerModel) updatePowerRank() {
	p.RefreshPlayerBattleInfo()
	logicCommon.UpdatePlayerBattleInfo(p.PlayerCacheInfo.BattleInfo)
	if _, ok := p.PlayerCacheInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]; !ok {
		return
	}
	info := &rpcPb.NotifyUpdateRankInfo{
		Id:    p.GetUserId(),
		Score: p.PlayerCacheInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)].BattlePower,
	}
	commonRankConfigs := gameConfig.GetAllRankCfg()
	for _, rankCfgMap := range commonRankConfigs {
		for _, rankCfg := range rankCfgMap {
			fullRankCfg := gameConfig.GetRankCfgByIds(rankCfg.ActId, rankCfg.Id)
			if fullRankCfg == nil {
				continue
			}
			rankId := ""
			var err error
			if rankCfg.ActId != 0 {
				settled, version := p.PlayerActivityModel.CheckActivitySettled(rankCfg.ActId)
				if settled {
					continue
				}
				if int64(fullRankCfg.RankThreshold) > info.Score {
					continue
				}
				rankId, err = logicCommon.GetRankUniqueId(0, rankCfg.ActId, rankCfg.Id, p.GetUserServerId(), version)
				if err != nil {
					logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
					return
				}
			} else {
				if int64(fullRankCfg.RankThreshold) > info.Score {
					continue
				}
				rankId, err = logicCommon.GetRankUniqueId(rankCfg.Id, 0, 0, p.GetUserServerId(), "")
				if err != nil {
					logger.ErrorBySprintf("arena GetRankUniqueId error:%+v", err)
					return
				}
			}
			if rankCfg.PointType == int32(enum.RANK_BOARD_SCORE_TYPE_BATTLE_POWER) {
				err = rpcMessageSender.SendMessageToRankBoard(p.GetUserId(), rankId, 0, rpcPb.RPC_MESSAGE_ID_RPC_MESSAGE_UPDATE_PLAYER_RANK_INFO, info)
				if err != nil {
					logger.ErrorBySprintf("notify rank info to rankBoard node error: %v", err)
					return
				}
			}
		}
	}
}

func (p *PlayerModel) GetHeroAttr(heroId int64) map[int32]int64 {
	return p.HeroDetailsModel.FindHeroAttrByHeroId(heroId)
}

func (p *PlayerModel) getHeroAttr(heroId int64) map[int32]int64 {
	//isMain := false
	//heroDetailList := make(map[int64]*HeroDetailsEntity)
	//formationId := int32(0)
	//for _, v := range p.HeroFormationModel.Entities[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)] {
	//	if v.IsActive == true {
	//		formationId = v.FormationID
	//		for _, heroOwnId := range v.HeroOwnIDList {
	//			if heroId == heroOwnId {
	//				for _, heroOwnId := range v.HeroOwnIDList {
	//					heroDetailList[heroOwnId] = p.HeroDetailsModel.GetHero(heroOwnId)
	//				}
	//				isMain = true
	//				break
	//			}
	//		}
	//		break
	//	}
	//}

	//classSynergyIdList := p.GetClassSynergy(heroDetailList, int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN), formationId)

	// 获取英雄职业
	heroDetail := p.HeroDetailsModel.GetHero(heroId)
	heroClass := int32(0)
	if heroDetail != nil {
		heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
		if heroBaseCfg != nil {
			heroClass = heroBaseCfg.HeroClass
		}
	}

	secondAttrMap := make(map[int32]int64)
	for _, attrId := range enum.SecondClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			attr += model.GetBuffAttr(heroId, attrId)
			attr += model.GetHeroAttr(heroId, attrId)
		}
		//if isMain {
		//	attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		//}
		secondAttrMap[attrId] = attr
	}

	thirdAttrMap := make(map[int32]int64)
	for _, attrId := range enum.ThirdClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			attr += model.GetBuffAttr(heroId, attrId)
			attr += model.GetHeroAttr(heroId, attrId)
		}
		//if isMain {
		//	attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		//}
		thirdAttrMap[attrId] = attr
	}

	// 根据职业获取对应的职业基础属性值
	classBaseHp := p.getClassBaseAttr(heroClass, secondAttrMap, "Hp")
	classBaseHpPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "HpPercent")
	classBasePhysicalAttack := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalAttack")
	classBasePhysicalAttackPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalAttackPercent")
	classBaseMagicalAttack := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalAttack")
	classBaseMagicalAttackPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalAttackPercent")
	classBasePhysicalDefense := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalDefense")
	classBasePhysicalDefensePercent := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalDefensePercent")
	classBaseMagicalDefense := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalDefense")
	classBaseMagicalDefensePercent := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalDefensePercent")

	attrMap := make(map[int32]int64)
	for _, attrId := range enum.FirstClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			if attrId == enum.AttributeBasicSkillAttackPower {
				attr *= model.GetHeroAttr(heroId, attrId)
				attr *= model.GetBuffAttr(heroId, attrId)
			} else {
				attr += model.GetHeroAttr(heroId, attrId)
				attr += model.GetBuffAttr(heroId, attrId)
			}
		}
		//if isMain {
		//	if attrId == enum.AttributeBasicSkillAttackPower {
		//		attr *= p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		//	} else {
		//		attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		//	}
		//}
		attrMap[attrId] = attr
	}

	// 计算总基础属性值 = 普通基础属性 + 职业基础属性
	totalBaseHp := secondAttrMap[enum.AttributeBasicBaseHp] + classBaseHp
	totalBaseHpPercent := secondAttrMap[enum.AttributeBasicBaseHpPercent] + classBaseHpPercent
	totalBasePhysicalAttack := secondAttrMap[enum.AttributeBasicBasePhysicalAttack] + classBasePhysicalAttack
	totalBasePhysicalAttackPercent := secondAttrMap[enum.AttributeBasicBasePhysicalAttackPercent] + classBasePhysicalAttackPercent
	totalBaseMagicalAttack := secondAttrMap[enum.AttributeBasicBaseMagicalAttack] + classBaseMagicalAttack
	totalBaseMagicalAttackPercent := secondAttrMap[enum.AttributeBasicBaseMagicalAttackPercent] + classBaseMagicalAttackPercent
	totalBasePhysicalDefense := secondAttrMap[enum.AttributeBasicBasePhysicalDefense] + classBasePhysicalDefense
	totalBasePhysicalDefensePercent := secondAttrMap[enum.AttributeBasicBasePhysicalDefensePercent] + classBasePhysicalDefensePercent
	totalBaseMagicalDefense := secondAttrMap[enum.AttributeBasicBaseMagicalDefense] + classBaseMagicalDefense
	totalBaseMagicalDefensePercent := secondAttrMap[enum.AttributeBasicBaseMagicalDefensePercent] + classBaseMagicalDefensePercent

	attrMap[enum.AttributeBasicHp] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicHp]+
		int64(math.Ceil(float64(totalBaseHp)*(1+float64(totalBaseHpPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicHpPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalAttack] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalAttack]+
		int64(math.Ceil(float64(totalBasePhysicalAttack)*(1+float64(totalBasePhysicalAttackPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalAttackPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalAttack] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalAttack]+
		int64(math.Ceil(float64(totalBaseMagicalAttack)*(1+float64(totalBaseMagicalAttackPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicMagicalAttackPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalDefense] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalDefense]+
		int64(math.Ceil(float64(totalBasePhysicalDefense)*(1+float64(totalBasePhysicalDefensePercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalDefensePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalDefense] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalDefense]+
		int64(math.Ceil(float64(totalBaseMagicalDefense)*(1+float64(totalBaseMagicalDefensePercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicMagicalDefensePercent])/float64(10000))))

	attrMap[enum.AttributeBasicPhysicalCritRate] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalCritRate]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalCritRatePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalCritRate] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalCritRate]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalCritRatePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalCritResistance] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalCritResistance]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalCritResistancePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalCritResistance] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalCritResistance]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalCritResistancePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalHit] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalHit]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalHitPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalHit] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalHit]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalHitPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalDodge] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalDodge]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalDodgePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalDodge] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalDodge]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalDodgePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalPenetration] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalPenetration]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalPenetrationPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalPenetration] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalPenetration]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalPenetrationPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalBlock] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalBlock]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalBlockPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalBlock] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalBlock]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalBlockPercent])/float64(10000))))

	// 战力
	power := gameConfig.GetAttrMapPower(p.ArchitectureModel.GetMainLevel(), attrMap)
	attrMap[enum.AttributeBasicCombatPower] = int64(math.Ceil(power))
	res := make(map[int32]int64)
	for key, v := range attrMap {
		if v != 0 {
			res[key] = v
		}
	}
	return res
}

func (p *PlayerModel) AppendPlayerModel(model logicCommon.PlayerModelInterface) {
	p.PlayerModels = append(p.PlayerModels, model)
}

func (p *PlayerModel) AppendHeroAttrModel(model logicCommon.HeroAttrInterface) {
	p.HeroAttrModels = append(p.HeroAttrModels, model)
}

func (p *PlayerModel) IsOnline() bool {
	return p.Session.IsActive()
}

//func (p *PlayerModel) GetClassSynergy(heroIdList map[int64]*HeroDetailsEntity, formationType int32, formationId int32) []int32 {
//	heroClassMap := make(map[int32]int32)
//	f := p.HeroFormationModel
//	res := make([]int32, 0)
//	if heroIdList == nil || len(heroIdList) == 0 {
//		return nil
//	}
//	if f.Entities[int32(formationType)] == nil {
//		return nil
//	}
//	if f.Entities[int32(formationType)][formationId] == nil {
//		return nil
//	}
//	for _, v := range f.Entities[int32(formationType)][formationId].HeroOwnIDList {
//		//heroClassMap[heroIdList[v].EvolutionPath]++
//		heroCfg := gameConfig.GetHeroBaseCfg(int32(heroIdList[v].HeroID))
//		if heroCfg == nil {
//			continue
//		}
//		heroClassMap[heroCfg.HeroClass]++
//	}
//	for classId, num := range heroClassMap {
//		for index, num1 := range gameConfig.GetHeroClassCfg(classId).SynergyLevel {
//			if num1 == num {
//				res = append(res, gameConfig.GetHeroClassCfg(classId).ClassSynergy[index])
//				break
//			}
//		}
//	}
//	return res
//}

func (p *PlayerModel) GetHeroAttrForBattle(heroId int64, formationType int32, formationId int32) map[int32]int64 {
	heroDetailList := make(map[int64]*HeroDetailsEntity)
	if p.HeroFormationModel.Entities[formationType] == nil {
		return nil
	}
	if p.HeroFormationModel.Entities[formationType][formationId] == nil {
		return nil
	}
	for _, v := range p.HeroFormationModel.Entities[formationType][formationId].HeroOwnIDList {
		heroDetailList[v] = p.HeroDetailsModel.GetHero(v)
	}

	//classSynergyIdList := p.GetClassSynergy(heroDetailList, int32(formationType), formationId)

	// 获取英雄职业，用于计算职业基础属性
	heroDetail := p.HeroDetailsModel.GetHero(heroId)
	heroClass := int32(0)
	if heroDetail != nil {
		heroBaseCfg := gameConfig.GetHeroBaseCfg(int32(heroDetail.HeroID))
		if heroBaseCfg != nil {
			heroClass = heroBaseCfg.HeroClass
		}
	}

	// 非主线阵容时，设置等级替换
	if formationType != int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN) {
		heroOwnIDList := p.HeroFormationModel.Entities[formationType][formationId].HeroOwnIDList
		for idx, ownID := range heroOwnIDList {
			if ownID == heroId {
				top5 := p.HeroDetailsModel.GetTop5HeroLevels()
				if idx < len(top5) {
					p.HeroDetailsModel.SetOverrideLevel(top5[idx])
				}
				break
			}
		}
	}
	defer p.HeroDetailsModel.ClearOverrideLevel()

	secondAttrMap := make(map[int32]int64)
	for _, attrId := range enum.SecondClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			attr += model.GetHeroAttr(heroId, attrId)
			attr += model.GetBuffAttr(heroId, attrId)
			//attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		}
		secondAttrMap[attrId] = attr
	}

	thirdAttrMap := make(map[int32]int64)
	for _, attrId := range enum.ThirdClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			attr += model.GetBuffAttr(heroId, attrId)
			attr += model.GetHeroAttr(heroId, attrId)
			//attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
		}
		thirdAttrMap[attrId] = attr
	}

	// 根据职业获取对应的职业基础属性值
	classBaseHp := p.getClassBaseAttr(heroClass, secondAttrMap, "Hp")
	classBaseHpPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "HpPercent")
	classBasePhysicalAttack := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalAttack")
	classBasePhysicalAttackPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalAttackPercent")
	classBaseMagicalAttack := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalAttack")
	classBaseMagicalAttackPercent := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalAttackPercent")
	classBasePhysicalDefense := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalDefense")
	classBasePhysicalDefensePercent := p.getClassBaseAttr(heroClass, secondAttrMap, "PhysicalDefensePercent")
	classBaseMagicalDefense := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalDefense")
	classBaseMagicalDefensePercent := p.getClassBaseAttr(heroClass, secondAttrMap, "MagicalDefensePercent")

	// 计算总基础属性值 = 普通基础属性 + 职业基础属性
	totalBaseHp := secondAttrMap[enum.AttributeBasicBaseHp] + classBaseHp
	totalBaseHpPercent := secondAttrMap[enum.AttributeBasicBaseHpPercent] + classBaseHpPercent
	totalBasePhysicalAttack := secondAttrMap[enum.AttributeBasicBasePhysicalAttack] + classBasePhysicalAttack
	totalBasePhysicalAttackPercent := secondAttrMap[enum.AttributeBasicBasePhysicalAttackPercent] + classBasePhysicalAttackPercent
	totalBaseMagicalAttack := secondAttrMap[enum.AttributeBasicBaseMagicalAttack] + classBaseMagicalAttack
	totalBaseMagicalAttackPercent := secondAttrMap[enum.AttributeBasicBaseMagicalAttackPercent] + classBaseMagicalAttackPercent
	totalBasePhysicalDefense := secondAttrMap[enum.AttributeBasicBasePhysicalDefense] + classBasePhysicalDefense
	totalBasePhysicalDefensePercent := secondAttrMap[enum.AttributeBasicBasePhysicalDefensePercent] + classBasePhysicalDefensePercent
	totalBaseMagicalDefense := secondAttrMap[enum.AttributeBasicBaseMagicalDefense] + classBaseMagicalDefense
	totalBaseMagicalDefensePercent := secondAttrMap[enum.AttributeBasicBaseMagicalDefensePercent] + classBaseMagicalDefensePercent

	attrMap := make(map[int32]int64)
	for _, attrId := range enum.FirstClassAttrIdMap {
		attr := int64(0)
		for _, model := range p.HeroAttrModels {
			if attrId == enum.AttributeBasicSkillAttackPower {
				attr *= model.GetHeroAttr(heroId, attrId)
				attr *= model.GetBuffAttr(heroId, attrId)
				//attr *= p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
			} else {
				attr += model.GetHeroAttr(heroId, attrId)
				attr += model.GetBuffAttr(heroId, attrId)
				//attr += p.HeroFormationModel.GetHeroAttr(p.HeroDetailsModel.GetHero(heroId), attrId, classSynergyIdList)
			}
		}
		attrMap[attrId] = attr
	}

	attrMap[enum.AttributeBasicHp] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicHp]+
		int64(math.Ceil(float64(totalBaseHp)*(1+float64(totalBaseHpPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicHpPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalAttack] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalAttack]+
		int64(math.Ceil(float64(totalBasePhysicalAttack)*(1+float64(totalBasePhysicalAttackPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalAttackPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalAttack] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalAttack]+
		int64(math.Ceil(float64(totalBaseMagicalAttack)*(1+float64(totalBaseMagicalAttackPercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicMagicalAttackPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalDefense] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalDefense]+
		int64(math.Ceil(float64(totalBasePhysicalDefense)*(1+float64(totalBasePhysicalDefensePercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalDefensePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalDefense] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalDefense]+
		int64(math.Ceil(float64(totalBaseMagicalDefense)*(1+float64(totalBaseMagicalDefensePercent)/float64(10000))))) *
		(1 + float64(thirdAttrMap[enum.AttributeBasicMagicalDefensePercent])/float64(10000))))

	attrMap[enum.AttributeBasicPhysicalCritRate] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalCritRate]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalCritRatePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalCritRate] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalCritRate]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalCritRatePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalCritResistance] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalCritResistance]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalCritResistancePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalCritResistance] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalCritResistance]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalCritResistancePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalHit] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalHit]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalHitPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalHit] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalHit]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalHitPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalDodge] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalDodge]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalDodgePercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalDodge] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalDodge]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalDodgePercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalPenetration] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalPenetration]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalPenetrationPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalPenetration] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalPenetration]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalPenetrationPercent])/float64(10000))))
	attrMap[enum.AttributeBasicPhysicalBlock] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicPhysicalBlock]) * (1 + float64(thirdAttrMap[enum.AttributeBasicPhysicalBlockPercent])/float64(10000))))
	attrMap[enum.AttributeBasicMagicalBlock] = int64(math.Ceil(float64(attrMap[enum.AttributeBasicMagicalBlock]) * (1 + float64(thirdAttrMap[enum.AttributeBasicMagicalBlockPercent])/float64(10000))))

	// 战力
	power := gameConfig.GetAttrMapPower(p.ArchitectureModel.GetMainLevel(), attrMap)
	attrMap[enum.AttributeBasicCombatPower] = int64(math.Ceil(power))
	return attrMap
}

// getClassBaseAttr 根据英雄职业从secondAttrMap中获取对应的职业基础属性值
func (p *PlayerModel) getClassBaseAttr(heroClass int32, secondAttrMap map[int32]int64, attrType string) int64 {
	switch heroClass {
	case 110: // 剑士 Swordsman
		switch attrType {
		case "Hp":
			return secondAttrMap[enum.AttributeBasicSwordsmanBaseHp]
		case "HpPercent":
			return secondAttrMap[enum.AttributeBasicSwordsmanBaseHpPercent]
		case "PhysicalAttack":
			return secondAttrMap[enum.AttributeBasicSwordsmanBasePhysicalAttack]
		case "PhysicalAttackPercent":
			return secondAttrMap[enum.AttributeBasicSwordsmanBasePhysicalAttackPercent]
		case "PhysicalDefense":
			return secondAttrMap[enum.AttributeBasicSwordsmanBasePhysicalDefense]
		case "PhysicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicSwordsmanBasePhysicalDefensePercent]
		case "MagicalDefense":
			return secondAttrMap[enum.AttributeBasicSwordsmanBaseMagicalDefense]
		case "MagicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicSwordsmanBaseMagicalDefensePercent]
		}
	case 210: // 枪手 Gunner
		switch attrType {
		case "Hp":
			return secondAttrMap[enum.AttributeBasicGunnerBaseHp]
		case "HpPercent":
			return secondAttrMap[enum.AttributeBasicGunnerBaseHpPercent]
		case "PhysicalAttack":
			return secondAttrMap[enum.AttributeBasicGunnerBasePhysicalAttack]
		case "PhysicalAttackPercent":
			return secondAttrMap[enum.AttributeBasicGunnerBasePhysicalAttackPercent]
		case "PhysicalDefense":
			return secondAttrMap[enum.AttributeBasicGunnerBasePhysicalDefense]
		case "PhysicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicGunnerBasePhysicalDefensePercent]
		case "MagicalDefense":
			return secondAttrMap[enum.AttributeBasicGunnerBaseMagicalDefense]
		case "MagicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicGunnerBaseMagicalDefensePercent]
		}
	case 310: // 法师 Mage
		switch attrType {
		case "Hp":
			return secondAttrMap[enum.AttributeBasicMageBaseHp]
		case "HpPercent":
			return secondAttrMap[enum.AttributeBasicMageBaseHpPercent]
		case "MagicalAttack":
			return secondAttrMap[enum.AttributeBasicMageBaseMagicalAttack]
		case "MagicalAttackPercent":
			return secondAttrMap[enum.AttributeBasicMageBaseMagicalAttackPercent]
		case "PhysicalDefense":
			return secondAttrMap[enum.AttributeBasicMageBasePhysicalDefense]
		case "PhysicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicMageBasePhysicalDefensePercent]
		case "MagicalDefense":
			return secondAttrMap[enum.AttributeBasicMageBaseMagicalDefense]
		case "MagicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicMageBaseMagicalDefensePercent]
		}
	case 410: // 格斗家 Brawler
		switch attrType {
		case "Hp":
			return secondAttrMap[enum.AttributeBasicBrawlerBaseHp]
		case "HpPercent":
			return secondAttrMap[enum.AttributeBasicBrawlerBaseHpPercent]
		case "PhysicalAttack":
			return secondAttrMap[enum.AttributeBasicBrawlerBasePhysicalAttack]
		case "PhysicalAttackPercent":
			return secondAttrMap[enum.AttributeBasicBrawlerBasePhysicalAttackPercent]
		case "PhysicalDefense":
			return secondAttrMap[enum.AttributeBasicBrawlerBasePhysicalDefense]
		case "PhysicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicBrawlerBasePhysicalDefensePercent]
		case "MagicalDefense":
			return secondAttrMap[enum.AttributeBasicBrawlerBaseMagicalDefense]
		case "MagicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicBrawlerBaseMagicalDefensePercent]
		}
	case 510: // 使者 Envoy
		switch attrType {
		case "Hp":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseHp]
		case "HpPercent":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseHpPercent]
		case "MagicalAttack":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseMagicalAttack]
		case "MagicalAttackPercent":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseMagicalAttackPercent]
		case "PhysicalDefense":
			return secondAttrMap[enum.AttributeBasicEnvoyBasePhysicalDefense]
		case "PhysicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicEnvoyBasePhysicalDefensePercent]
		case "MagicalDefense":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseMagicalDefense]
		case "MagicalDefensePercent":
			return secondAttrMap[enum.AttributeBasicEnvoyBaseMagicalDefensePercent]
		}
	}
	return 0
}

func (p *PlayerModel) GetMainFormationPower() int64 {
	info := p.PlayerCacheInfo.BattleInfo.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]
	if info != nil {
		return info.BattlePower
	}
	return 0
}
