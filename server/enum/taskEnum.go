package enum

const (
	TaskStatusUnFinish         = 0 //未完成
	TaskStatusFinishUnReward   = 1 // 完成未领取奖励
	TaskStatusFinishAndReward  = 2 // 已领取奖励
	TaskStatusDeleteInMemory   = 3 // 内存中删除
	TaskStatusTimeOverUnFinish = 4 // 超时未完成
)

const (
	EventTypeHeroLevelUp             = "hero_level_up"              // 英雄升级
	EventTypeHeroStarUp              = "hero_star_up"               // 英雄升星
	EventTypeKillMonster             = "kill_monster"               // 击杀怪物
	EventTypePassInstance            = "pass_instance"              // 通过副本
	EventTypeItemCollect             = "item_collect"               // 收集物品
	EventTypeLuckyLottery            = "lucky_lottery"              // 幸运抽奖
	EventTypePlayerLevelUp           = "player_level_up"            // 玩家升级
	EventTypeJoinInstance            = "join_instance"              // 参加副本
	EventTypeQuickClaimMachineReward = "quick_claim_machine_reward" // 使用挂机奖励的快速领取功能
	EventTypeBuildLevelUp            = "build_level_up"             // 建筑升级
	EventTypeDispatchKillMonster     = "dispatch_kill_monster"      // 派遣杀怪
	EventTypePlayerPowerChange       = "player_power_change"        // 玩家战力变化
	EventTypeAddHeroAlbum            = "add_hero_album"             // 添加英雄到图鉴
	EventTypeDispatchMapUnlock       = "dispatch_map_unlock"        // 派遣地图解锁
	EventTypeAccessoryLevelUp        = "accessory_level_up"         // 首饰系统升级
	EventTypeLoopBoxLevelUp          = "loop_box_level_up"          // 循环宝箱系统升级
	EventTypeEquipmentStrong         = "equipment_strong"           // 装备强化
	EventTypePetLevelUp              = "pet_level_up"               // 宠物升级
	EventTypeAllianceJoin            = "alliance_join"              // 加入联盟
	EventTypePetStarUp               = "pet_star_up"                // 宠物升星
	EventTypeEquipmentForge          = "equipment_forge"            // 装备锻造
	EventTypeEquipmentWear           = "equipment_wear"             // 装备穿戴
	EventTypeArenaScoreChange        = "arena_score_change"         // 竞技场积分变化
	EventTypeAdChestOpen             = "ad_chest_open"              // 限时宝箱打开
	EventTypeMainTaskChange          = "main_task_change"           // 主线任务通过
	EventTypeStoneAttrLevelUp        = "stone_attr_level_up"        // 石像等级提升
	EventTypePlayerLogin             = "player_login"               // 玩家登录
	EventTypeCityAgeChange           = "city_age_change"            // 主城时代变化
)

const (
	TaskAffiliationMain          = 1  // 主线
	TaskAffiliationSide          = 2  // 支线
	TaskAffiliationDaily         = 3  // 每日
	TaskAffiliationWeekly        = 4  // 每周
	TaskAffiliationBounty        = 5  // 悬赏
	TaskAffiliationPassCard      = 6  // 通行证任务
	TaskAffiliationDailyAlliance = 7  // 联盟每日
	TaskAffiliationTrial         = 8  // 七日试炼
	TaskAffiliationCityAge       = 9  // 主城时代
	TaskAffiliationAct           = 10 // 活动玩法任务
)

const (
	ObjectiveTypeKillAnyMonsterHowMany          = 1  // 击杀任意怪物数量
	ObjectiveTypeKillWhatMonsterHowMany         = 2  // 击杀指定怪物数量
	ObjectiveTypeWhereKillWhatMonsterHowMany    = 3  // 在指定地点击杀指定怪物数量
	ObjectiveTypeGetTypeOrQualityItemsHowMany   = 4  // 获得指定类型或品质物品数量
	ObjectiveTypeGetWhatItemsHowMany            = 5  // 获得指定物品数量
	ObjectiveTypeAnyHeroReachWhatLevel          = 6  // 任意英雄达到指定等级
	ObjectiveTypeAnyHeroLevelUpHowMany          = 7  // 任意英雄升级指定次数
	ObjectiveTypePassWhatMainLevel              = 8  // 通关指定主线关卡
	ObjectiveTypePassHowManyMainLevel           = 9  // 通关指定数量主线关卡
	ObjectiveTypeAccessoryLuckyHowMany          = 10 // 首饰抽取多少次
	ObjectiveTypeHeroLotteryHowMany             = 11 // 英雄抽取多少次
	ObjectiveTypeLoopBoxLotteryHowMany          = 12 // 循环宝箱抽取多少次
	ObjectiveTypeTowerChallengeHowMany          = 13 // 挑战5v5爬塔多少次
	ObjectiveTypeArenaParticipateHowMany        = 14 // 参与竞技场多少次
	ObjectiveTypeQuickClaimMachineRewardHowMany = 15 // 使用多少次挂机奖励的快速领取功能
	ObjectiveTypeBuildLevelUpHowMany            = 16 // 升级建筑多少次

	ObjectiveTypeStrongEquipmentHowMany      = 20 // 装备强化多少次
	ObjectiveTypeHowManyHeroReachWhatLevel   = 21 // 多少英雄达到指定等级
	ObjectiveTypeHowManyHeroReachWhatStar    = 22 // 多少英雄达到指定星级
	ObjectiveTypeTowerChallengePassWhatLevel = 23 // 5v5爬塔通关到多少关

	ObjectiveTypeHeroStarUpHowMany = 25 // 英雄升星多少次

	ObjectiveTypeWhatBuildLevelUpWhat                 = 27 // 指定建筑升级到某个等级
	ObjectiveTypeDispatchKillMonsterHowMany           = 28 // 派遣杀怪多少次
	ObjectiveTypeCumulativeDispatchKillMonsterHowMany = 29 // 累计派遣杀怪多少次
	ObjectiveTypePlayerPowerReachWhat                 = 30 // 玩家战力达到多少
	ObjectiveTypeAllBuildLevelReachWhat               = 31 // 所有建筑总等级达到多少级
	ObjectiveTypeHeroQuantityReachWhat                = 32 // 拥有英雄数量达到几个
	ObjectiveTypeWhatDispatchMapUnlockWhatStage       = 33 // 派遣地图某张地图解锁到几阶段
	ObjectiveTypeAccessorySystemLevelReachWhat        = 35 // 首饰系统等级达到多少级
	ObjectiveTypeHowManyHeroReachWhatPotential        = 36 // 多少英雄达到指定潜力
	ObjectiveTypeHowManyEquipStrongReachWhatLevel     = 37 // 多少装备强化达到指定等级
	ObjectiveTypeHowManyPetReachWhatLevel             = 38 // 多少宠物达到指定等级
	ObjectiveTypeJoinAlliance                         = 39 // 加入联盟
	ObjectiveTypeAdventureParticipateHowMany          = 40 // 参与奇遇副本多少次
	ObjectiveTypeHowManyPetReachWhatStar              = 41 // 多少宠物达到指定星级
	ObjectiveTypeEquipmentForgeHowMany                = 42 // 多少装备锻造达到指定等级
	ObjectiveTypeWearHowManyEquipmentQuality          = 43 // 多少装备穿戴达到指定品质
	ObjectiveTypeArenaScoreReachWhat                  = 44 // 竞技场积分达到多少
	ObjectiveTypeAdChestOpenHowMany                   = 45 // 多少次限时宝箱打开
	ObjectiveTypeMainTaskPassWhatNum                  = 46 // 主线任务通过多少次
	ObjectiveTypeStoneClassTotalLevelReachWhat        = 47 // 石像总等级达到多少级
	ObjectiveTypeLoopBoxSystemLevelReachWhat          = 48 // 循环宝箱系统等级达到多少级
	ObjectiveTypeGloryArenaChallengeHowMany           = 49 // 挑战荣耀擂台多少次
	ObjectiveTypePetRecruitHowMany                    = 50 // 宠物招募次数
	ObjectiveTypePetRecruitHowManyCumulative          = 51 // 宠物招募次数（累计）
	ObjectiveTypeCollectionLotteryHowMany             = 52 // 藏品抽取次数
	ObjectiveTypeCollectionLotteryHowManyCumulative   = 53 // 藏品抽取次数(累计)
	ObjectiveTypeDungeonParticipateHowMany            = 54 // 常驻副本参与次数
	ObjectiveTypeDungeonParticipateHowManyCumulative  = 55 // 常驻副本参与次数（累计）
	ObjectiveTypeDungeonPassWhatStage                 = 56 // 常驻副本通关层数
	ObjectiveTypeGloryArenaWinHowMany                 = 59 // 荣耀擂台胜场次数
	ObjectiveTypeWearHowManyEquipmentLevel            = 60 // 穿戴多少装备达到指定阶数

	ObjectiveTypeLoginGame                            = 61 // 登陆游戏
	ObjectiveTypeStoneWhatClassWhatAttrLevelUpHowMany = 62 // 石像某个职业某个属性达到多少级
	ObjectiveTypeCityAgeReachWhatStage                = 63 // 时代达到什么阶段
)

//var EventToObjectiveTypes = map[string][]int32{
//	EventTypeKillMonster: {ObjectiveTypeKillAnyMonsterHowMany, ObjectiveTypeKillWhatMonsterHowMany, ObjectiveTypeWhereKillWhatMonsterHowMany},
//	EventTypeHeroLevelUp: {ObjectiveTypeAnyHeroReachWhatLevel, ObjectiveTypeAnyHeroLevelUpHowMany, ObjectiveTypeHowManyHeroReachWhatLevel},
//	EventTypeHeroStarUp:  {ObjectiveTypeHeroStarUpHowMany, ObjectiveTypeHowManyHeroReachWhatStar},
//	EventTypeItemCollect: {ObjectiveTypeGetWhatItemsHowMany, ObjectiveTypeGetTypeOrQualityItemsHowMany},
//	EventTypePassInstance: {ObjectiveTypePassWhatMainLevel, ObjectiveTypePassHowManyMainLevel, ObjectiveTypeTowerChallengePassWhatLevel,
//		ObjectiveTypeDungeonPassWhatStage, ObjectiveTypeGloryArenaWinHowMany},
//	EventTypeLuckyLottery: {ObjectiveTypeAccessoryLuckyHowMany, ObjectiveTypeHeroLotteryHowMany, ObjectiveTypeLoopBoxLotteryHowMany,
//		ObjectiveTypeAccessorySystemLevelReachWhat, ObjectiveTypeCollectionLotteryHowMany, ObjectiveTypeCollectionLotteryHowManyCumulative,
//		ObjectiveTypePetRecruitHowMany, ObjectiveTypePetRecruitHowManyCumulative},
//	EventTypeJoinInstance: {ObjectiveTypeArenaParticipateHowMany, ObjectiveTypeTowerChallengeHowMany, ObjectiveTypeAdventureParticipateHowMany,
//		ObjectiveTypeGloryArenaChallengeHowMany, ObjectiveTypeDungeonParticipateHowMany, ObjectiveTypeDungeonParticipateHowManyCumulative},
//	EventTypeQuickClaimMachineReward: {ObjectiveTypeQuickClaimMachineRewardHowMany},
//	EventTypeBuildLevelUp:            {ObjectiveTypeBuildLevelUpHowMany, ObjectiveTypeWhatBuildLevelUpWhat, ObjectiveTypeAllBuildLevelReachWhat},
//	EventTypeDispatchKillMonster:     {ObjectiveTypeDispatchKillMonsterHowMany, ObjectiveTypeCumulativeDispatchKillMonsterHowMany},
//	EventTypePlayerPowerChange:       {ObjectiveTypePlayerPowerReachWhat},
//	EventTypeAddHeroAlbum:            {ObjectiveTypeHeroQuantityReachWhat, ObjectiveTypeHowManyHeroReachWhatPotential},
//	EventTypeDispatchMapUnlock:       {ObjectiveTypeWhatDispatchMapUnlockWhatStage},
//	EventTypeLoopBoxLevelUp:          {ObjectiveTypeLoopBoxSystemLevelReachWhat},
//	EventTypeEquipmentStrong:         {ObjectiveTypeHowManyEquipStrongReachWhatLevel, ObjectiveTypeStrongEquipmentHowMany},
//	EventTypePetLevelUp:              {ObjectiveTypeHowManyPetReachWhatLevel},
//	EventTypeAllianceJoin:            {ObjectiveTypeJoinAlliance},
//	EventTypePetStarUp:               {ObjectiveTypeHowManyPetReachWhatStar},
//	EventTypeEquipmentForge:          {ObjectiveTypeEquipmentForgeHowMany},
//	EventTypeEquipmentWear:           {ObjectiveTypeWearHowManyEquipmentQuality, ObjectiveTypeWearHowManyEquipmentLevel},
//	EventTypeArenaScoreChange:        {ObjectiveTypeArenaScoreReachWhat},
//	EventTypeAdChestOpen:             {ObjectiveTypeAdChestOpenHowMany},
//	EventTypeMainTaskChange:          {ObjectiveTypeMainTaskPassWhatNum},
//	EventTypeStoneAttrLevelUp:        {ObjectiveTypeStoneClassTotalLevelReachWhat},
//}
//
//var NeedCheckTasks = map[int32]bool{
//	ObjectiveTypeAnyHeroReachWhatLevel:                true,
//	ObjectiveTypePassWhatMainLevel:                    true,
//	ObjectiveTypeHowManyHeroReachWhatLevel:            true,
//	ObjectiveTypeHowManyHeroReachWhatStar:             true,
//	ObjectiveTypeTowerChallengePassWhatLevel:          true,
//	ObjectiveTypeWhatBuildLevelUpWhat:                 true,
//	ObjectiveTypeCumulativeDispatchKillMonsterHowMany: true,
//	ObjectiveTypePlayerPowerReachWhat:                 true,
//	ObjectiveTypeAllBuildLevelReachWhat:               true,
//	ObjectiveTypeHeroQuantityReachWhat:                true,
//	ObjectiveTypeWhatDispatchMapUnlockWhatStage:       true,
//	ObjectiveTypeLoopBoxSystemLevelReachWhat:          true,
//	ObjectiveTypeAccessorySystemLevelReachWhat:        true,
//	ObjectiveTypeHowManyHeroReachWhatPotential:        true,
//	ObjectiveTypeHowManyEquipStrongReachWhatLevel:     true,
//	ObjectiveTypeHowManyPetReachWhatLevel:             true,
//	ObjectiveTypeJoinAlliance:                         true,
//	ObjectiveTypeHowManyPetReachWhatStar:              true,
//	ObjectiveTypeWearHowManyEquipmentQuality:          true,
//	ObjectiveTypeArenaScoreReachWhat:                  true,
//	ObjectiveTypeMainTaskPassWhatNum:                  true,
//	ObjectiveTypeStoneClassTotalLevelReachWhat:        true,
//	ObjectiveTypePetRecruitHowManyCumulative:          true,
//	ObjectiveTypeCollectionLotteryHowManyCumulative:   true,
//	ObjectiveTypeDungeonParticipateHowManyCumulative:  true,
//	ObjectiveTypeWearHowManyEquipmentLevel:            true,
//}

type DispatchTarget int32

const (
	TargetPlayer   DispatchTarget = 1
	TargetAlliance DispatchTarget = 2
)

type ObjectiveMeta struct {
	EventType string
	Target    DispatchTarget
	NeedCheck bool
}

var ObjectiveMetaList = map[int32]*ObjectiveMeta{
	ObjectiveTypeKillAnyMonsterHowMany:        {EventType: EventTypeKillMonster, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeKillWhatMonsterHowMany:       {EventType: EventTypeKillMonster, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeWhereKillWhatMonsterHowMany:  {EventType: EventTypeKillMonster, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeGetTypeOrQualityItemsHowMany: {EventType: EventTypeItemCollect, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeGetWhatItemsHowMany:          {EventType: EventTypeItemCollect, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeAnyHeroReachWhatLevel:        {EventType: EventTypeHeroLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeAnyHeroLevelUpHowMany:        {EventType: EventTypeHeroLevelUp, Target: TargetPlayer, NeedCheck: false},

	ObjectiveTypePassWhatMainLevel:              {EventType: EventTypePassInstance, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypePassHowManyMainLevel:           {EventType: EventTypePassInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeAccessoryLuckyHowMany:          {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeHeroLotteryHowMany:             {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeLoopBoxLotteryHowMany:          {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeTowerChallengeHowMany:          {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeArenaParticipateHowMany:        {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeQuickClaimMachineRewardHowMany: {EventType: EventTypeQuickClaimMachineReward, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeBuildLevelUpHowMany:            {EventType: EventTypeBuildLevelUp, Target: TargetPlayer, NeedCheck: false},

	ObjectiveTypeStrongEquipmentHowMany:      {EventType: EventTypeEquipmentStrong, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeHowManyHeroReachWhatLevel:   {EventType: EventTypeHeroLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeHowManyHeroReachWhatStar:    {EventType: EventTypeHeroStarUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeTowerChallengePassWhatLevel: {EventType: EventTypePassInstance, Target: TargetPlayer, NeedCheck: true},

	ObjectiveTypeHeroStarUpHowMany: {EventType: EventTypeHeroStarUp, Target: TargetPlayer, NeedCheck: true},

	ObjectiveTypeWhatBuildLevelUpWhat:                 {EventType: EventTypeBuildLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeDispatchKillMonsterHowMany:           {EventType: EventTypeDispatchKillMonster, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeCumulativeDispatchKillMonsterHowMany: {EventType: EventTypeDispatchKillMonster, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypePlayerPowerReachWhat:                 {EventType: EventTypePlayerPowerChange, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeAllBuildLevelReachWhat:               {EventType: EventTypeBuildLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeHeroQuantityReachWhat:                {EventType: EventTypeAddHeroAlbum, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeWhatDispatchMapUnlockWhatStage:       {EventType: EventTypeDispatchMapUnlock, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeAccessorySystemLevelReachWhat:        {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeHowManyHeroReachWhatPotential:        {EventType: EventTypeAddHeroAlbum, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeHowManyEquipStrongReachWhatLevel:     {EventType: EventTypeEquipmentStrong, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeHowManyPetReachWhatLevel:             {EventType: EventTypePetLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeJoinAlliance:                         {EventType: EventTypeAllianceJoin, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeAdventureParticipateHowMany:          {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeHowManyPetReachWhatStar:              {EventType: EventTypePetStarUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeEquipmentForgeHowMany:                {EventType: EventTypeEquipmentForge, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeWearHowManyEquipmentQuality:          {EventType: EventTypeEquipmentWear, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeArenaScoreReachWhat:                  {EventType: EventTypeArenaScoreChange, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeAdChestOpenHowMany:                   {EventType: EventTypeAdChestOpen, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeMainTaskPassWhatNum:                  {EventType: EventTypeMainTaskChange, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeStoneClassTotalLevelReachWhat:        {EventType: EventTypeStoneAttrLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeLoopBoxSystemLevelReachWhat:          {EventType: EventTypeLoopBoxLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeGloryArenaChallengeHowMany:           {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},

	ObjectiveTypePetRecruitHowMany:                   {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypePetRecruitHowManyCumulative:         {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeCollectionLotteryHowMany:            {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeCollectionLotteryHowManyCumulative:  {EventType: EventTypeLuckyLottery, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeDungeonParticipateHowMany:           {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeDungeonParticipateHowManyCumulative: {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeDungeonPassWhatStage:                {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeGloryArenaWinHowMany:                {EventType: EventTypeJoinInstance, Target: TargetPlayer, NeedCheck: false},
	ObjectiveTypeWearHowManyEquipmentLevel:           {EventType: EventTypeEquipmentWear, Target: TargetPlayer, NeedCheck: true},

	ObjectiveTypeLoginGame:                            {EventType: EventTypePlayerLogin, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeStoneWhatClassWhatAttrLevelUpHowMany: {EventType: EventTypeStoneAttrLevelUp, Target: TargetPlayer, NeedCheck: true},
	ObjectiveTypeCityAgeReachWhatStage:                {EventType: EventTypeCityAgeChange, Target: TargetPlayer, NeedCheck: true},
}

var PlayerEventTypes map[string][]int32
var AllianceEventTypes map[string][]int32
var NeedCheckTask map[int32]bool

func init() {
	PlayerEventTypes = make(map[string][]int32)
	AllianceEventTypes = make(map[string][]int32)
	NeedCheckTask = make(map[int32]bool)
	for key, v := range ObjectiveMetaList {
		switch v.Target {
		case TargetPlayer:
			PlayerEventTypes[v.EventType] = append(PlayerEventTypes[v.EventType], key)
		case TargetAlliance:
			AllianceEventTypes[v.EventType] = append(AllianceEventTypes[v.EventType], key)
		}
		if v.NeedCheck {
			NeedCheckTask[key] = true
		}
	}
}
