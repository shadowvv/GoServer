package gameConfig

import (
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("constant", &ConstantCfgLoader{})
}

type ConstantCfgLoader struct {
	temp1 map[string]*ConstantCfg
}

var _ configLoaderInterface = (*ConstantCfgLoader)(nil)

func (s *ConstantCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/constant.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[string]*ConstantCfg)

	// 加载 constant 表
	if constantData, ok := rawData["constant"]; ok {
		for _, row := range constantData {
			var v ConstantCfg
			v.Id = ParseInt(row["id"])
			v.Name = row["name"]
			v.Value = ParseIntArray(row["value"])
			if v.Id <= 0 {
				continue
			}
			if s.temp1[v.Name] != nil {
				return fmt.Errorf("[gameConfig] load constant error duplicate ID:%d", v.Id)
			}
			s.temp1[v.Name] = &v
		}
	}

	// 加载 privileges_constant 表
	if privilegesConstantData, ok := rawData["privileges_constant"]; ok {
		for _, row := range privilegesConstantData {
			var v ConstantCfg
			v.Id = ParseInt(row["id"])
			v.Name = row["name"]
			v.Value = ParseIntArray(row["value"])
			if v.Id <= 0 {
				continue
			}
			if s.temp1[v.Name] != nil {
				return fmt.Errorf("[gameConfig] load privileges_constant error duplicate name:%s, ID:%d", v.Name, v.Id)
			}
			s.temp1[v.Name] = &v
		}
	}

	return nil
}

func (s *ConstantCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return fmt.Errorf("[gameConfig] load constant error invalid ID:%s", id)
		}
		if v.Name == "" {
			return fmt.Errorf("[gameConfig] load constant error invalid Name:%s,configId:%d", v.Name, v.Id)
		}
		if len(v.Value) <= 0 {
			return fmt.Errorf("[gameConfig] load constant error invalid Value:%d,configId:%d", v.Value, v.Id)
		}
	}

	if s.temp1[CONSTANT_MAX_NICKNAME_LENGTH] == nil || s.temp1[CONSTANT_MAX_NICKNAME_LENGTH].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid maxNicknameLength")
	}
	if s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM] == nil || len(s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Value) != 2 || GetItemCfg(s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Value[0]) == nil || s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Value[1] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid nicknameChangeItem")
	} else {
		s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Item = &ItemConfig{
			ID:  s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Value[0],
			Num: int64(s.temp1[CONSTANT_NICKNAME_CHANGE_ITEM].Value[1]),
		}
	}
	if s.temp1[CONSTANT_CHANGE_NICKNAME_FREE_TIMES] == nil || s.temp1[CONSTANT_CHANGE_NICKNAME_FREE_TIMES].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid changeNicknameFreeTimes")
	}
	if s.temp1[CONSTANT_DAILY_PRIVILEGE_DROP_QUANTITY_LIMIT] == nil || s.temp1[CONSTANT_DAILY_PRIVILEGE_DROP_QUANTITY_LIMIT].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid dailyPrivilegeDropQuantityLimit")
	}
	if s.temp1[CONSTANT_arenaSeasonInitialPoints] == nil || s.temp1[CONSTANT_arenaSeasonInitialPoints].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error arenaSeasonInitialPoints not found:%s", CONSTANT_arenaSeasonInitialPoints)
	}
	if s.temp1[CONSTANT_arenaDailyFreeRefreshAttempts] == nil || s.temp1[CONSTANT_arenaDailyFreeRefreshAttempts].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error arenaDailyFreeRefreshAttempts not found:%s", CONSTANT_arenaDailyFreeRefreshAttempts)
	}
	if s.temp1[CONSTANT_refreshDiamondConsumptionQuantity] == nil || s.temp1[CONSTANT_refreshDiamondConsumptionQuantity].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error refreshDiamondConsumptionQuantity not found:%s", CONSTANT_refreshDiamondConsumptionQuantity)
	}
	if s.temp1[CONSTANT_arenaDailyFreeChallengeAttempts] == nil || s.temp1[CONSTANT_arenaDailyFreeChallengeAttempts].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error arenaDailyFreeChallengeAttempts not found:%s", CONSTANT_arenaDailyFreeChallengeAttempts)
	}
	if s.temp1[CONSTANT_arenaVictoryReward] == nil || s.temp1[CONSTANT_arenaVictoryReward].Value[0] < 0 || GetDropCfg(s.temp1[CONSTANT_arenaVictoryReward].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid arenaVictoryReward")
	}
	if s.temp1[CONSTANT_arenaDefeatReward] == nil || s.temp1[CONSTANT_arenaDefeatReward].Value[0] < 0 || GetDropCfg(s.temp1[CONSTANT_arenaDefeatReward].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid arenaDefeatReward")
	}
	if s.temp1[CONSTANT_gloryArenaEntryRequirement] == nil || s.temp1[CONSTANT_gloryArenaEntryRequirement].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaEntryRequirement")
	}
	if s.temp1[CONSTANT_gloryArenaOpponentRank] == nil || s.temp1[CONSTANT_gloryArenaOpponentRank].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaOpponentRank")
	}
	if s.temp1[CONSTANT_gloryArenaChallengeTime] == nil || len(s.temp1[CONSTANT_gloryArenaChallengeTime].Value) != 2 ||
		s.temp1[CONSTANT_gloryArenaChallengeTime].Value[0] < 0 || s.temp1[CONSTANT_gloryArenaChallengeTime].Value[0] > 23 ||
		s.temp1[CONSTANT_gloryArenaChallengeTime].Value[1] <= s.temp1[CONSTANT_gloryArenaChallengeTime].Value[0] || s.temp1[CONSTANT_gloryArenaChallengeTime].Value[1] > 24 {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaChallengeTime")
	}
	if s.temp1[CONSTANT_gloryArenaMaxHP] == nil || s.temp1[CONSTANT_gloryArenaMaxHP].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaMaxHP")
	}
	if s.temp1[CONSTANT_gloryArenaDefeatDrop] == nil || s.temp1[CONSTANT_gloryArenaDefeatDrop].Value[0] <= 0 || GetDropCfg(s.temp1[CONSTANT_gloryArenaDefeatDrop].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaMaxHP")
	}
	if s.temp1[CONSTANT_BindingBonus] == nil || len(s.temp1[CONSTANT_BindingBonus].Value) != 2 || GetItemCfg(s.temp1[CONSTANT_BindingBonus].Value[0]) == nil || s.temp1[CONSTANT_BindingBonus].Value[1] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid nicknameChangeItem")
	} else {
		s.temp1[CONSTANT_BindingBonus].Item = &ItemConfig{
			ID:  s.temp1[CONSTANT_BindingBonus].Value[0],
			Num: int64(s.temp1[CONSTANT_BindingBonus].Value[1]),
		}
	}
	if s.temp1[CONSTANT_weeklyEmail] == nil || s.temp1[CONSTANT_weeklyEmail].Value[0] < 0 || GetMailContentCfg(s.temp1[CONSTANT_weeklyEmail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid weeklyEmail")
	}
	if s.temp1[CONSTANT_maximumStamina] == nil || s.temp1[CONSTANT_maximumStamina].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid maximumStamina")
	}
	if s.temp1[CONSTANT_staminaRecoveryTime] == nil || s.temp1[CONSTANT_staminaRecoveryTime].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid staminaRecoveryTime")
	}
	if s.temp1[CONSTANT_dailyLimitOnFreeStaminaClaims] == nil || s.temp1[CONSTANT_dailyLimitOnFreeStaminaClaims].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid dailyLimitOnFreeStaminaClaims")
	}
	if s.temp1[CONSTANT_freeStaminaClaimCooldown] == nil || s.temp1[CONSTANT_freeStaminaClaimCooldown].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid freeStaminaClaimCooldown")
	}
	if s.temp1[CONSTANT_dispatchAccelerationCost] == nil || s.temp1[CONSTANT_dispatchAccelerationCost].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid dispatchAccelerationCost")
	}
	if s.temp1[CONSTANT_freeStaminaNumb] == nil || s.temp1[CONSTANT_freeStaminaNumb].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid freeStaminaNumb")
	}
	if s.temp1[CONSTANT_dispatchEmailRewards] == nil || s.temp1[CONSTANT_dispatchEmailRewards].Value[0] < 0 || GetMailContentCfg(s.temp1[CONSTANT_dispatchEmailRewards].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid dispatchEmailRewards")
	}
	if s.temp1[CONSTANT_sevenSignMail] == nil || s.temp1[CONSTANT_sevenSignMail].Value[0] < 0 || GetMailContentCfg(s.temp1[CONSTANT_sevenSignMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sevenSignMail")
	}
	if s.temp1[CONSTANT_actSignMail] == nil || s.temp1[CONSTANT_actSignMail].Value[0] < 0 || GetMailContentCfg(s.temp1[CONSTANT_actSignMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid actSignMail")
	}
	if s.temp1[CONSTANT_statLockCost] == nil || s.temp1[CONSTANT_statLockCost].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid statLockCost")
	}
	if s.temp1[CONSTANT_refundRateForEnhanceMaterials] == nil || s.temp1[CONSTANT_refundRateForEnhanceMaterials].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid refundRateForEnhanceMaterials")
	}
	if s.temp1[CONSTANT_statUpgradeProbability] == nil || s.temp1[CONSTANT_statUpgradeProbability].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid statUpgradeProbability")
	}
	if s.temp1[CONSTANT_collectionExchangeSpid] == nil || s.temp1[CONSTANT_collectionExchangeSpid].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid collectionExchangeSpid")
	}
	if s.temp1[CONSTANT_successChanceIncrease] == nil || s.temp1[CONSTANT_successChanceIncrease].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid successChanceIncrease")
	}
	if s.temp1[CONSTANT_trialMail] == nil || s.temp1[CONSTANT_trialMail].Value[0] < 0 || GetMailContentCfg(s.temp1[CONSTANT_trialMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid trialMail")
	}
	if s.temp1[CONSTANT_sectNameChange] == nil || len(s.temp1[CONSTANT_sectNameChange].Value) != 2 {
		return fmt.Errorf("[gameConfig] load constant error invalid sectNameChange")
	}
	if GetItemCfg(s.temp1[CONSTANT_sectNameChange].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sectNameChange item is invalid")
	}
	if s.temp1[CONSTANT_sectCreateConsumption] == nil || len(s.temp1[CONSTANT_sectCreateConsumption].Value) != 2 {
		return fmt.Errorf("[gameConfig] load constant error invalid sectCreateConsumption")
	}
	if GetItemCfg(s.temp1[CONSTANT_sectCreateConsumption].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sectCreateConsumption item is invalid")
	}
	for _, v := range s.temp1[CONSTANT_sectCreateConditions].Value {
		if GetUnlockCfg(v) == nil {
			return fmt.Errorf("[gameConfig] load constant error sectCreateConditions not found:%d", v)
		}
	}
	if s.temp1[CONSTANT_sectNameCharacterCount] == nil || s.temp1[CONSTANT_sectNameCharacterCount].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid sectNameCharacterCount")
	}
	if s.temp1[CONSTANT_sectAnnouncementCharacterCount] == nil || s.temp1[CONSTANT_sectAnnouncementCharacterCount].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid sectAnnouncementCharacterCount")
	}
	if s.temp1[CONSTANT_sectmanifestoCount] == nil || s.temp1[CONSTANT_sectmanifestoCount].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid sectmanifestoCount")
	}
	if s.temp1[CONSTANT_sectRefuseMail] == nil || GetMailContentCfg(s.temp1[CONSTANT_sectRefuseMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sectRefuseMail")
	}
	if s.temp1[CONSTANT_sectdissolveMail] == nil || GetMailContentCfg(s.temp1[CONSTANT_sectdissolveMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sectdissolveMail")
	}
	if s.temp1[CONSTANT_sectLeaveMail] == nil || GetMailContentCfg(s.temp1[CONSTANT_sectLeaveMail].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid sectLeaveMail")
	}
	if s.temp1[CONSTANT_dispatchQueues] == nil || len(s.temp1[CONSTANT_dispatchQueues].Value) == 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid dispatchQueues")
	}
	if s.temp1[CONSTANT_maxEquipmentFromMain] == nil || s.temp1[CONSTANT_maxEquipmentFromMain].Value[0] < 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid maxEquipmentFromMain")
	}
	if s.temp1[CONSTANT_timeLimitedAdventure] == nil || s.temp1[CONSTANT_timeLimitedAdventure].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid timeLimitedAdventure")
	}
	if s.temp1[CONSTANT_dailyAdventureLimit] == nil || s.temp1[CONSTANT_dailyAdventureLimit].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid dailyAdventureLimit")
	}
	if s.temp1[CONSTANT_monsterAdventureProgressValue] == nil || s.temp1[CONSTANT_monsterAdventureProgressValue].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid monsterAdventureProgressValue")
	}
	if s.temp1[CONSTANT_realmStorageCap] == nil || s.temp1[CONSTANT_realmStorageCap].Value[0] <= 0 {
		return fmt.Errorf("[gameConfig] load constant error invalid realmStorageCap")
	}
	if s.temp1[CONSTANT_gloryArenaRefreshItem] == nil || GetItemCfg(s.temp1[CONSTANT_gloryArenaRefreshItem].Value[0]) == nil {
		return fmt.Errorf("[gameConfig] load constant error invalid gloryArenaRefreshItem")
	}
	return nil
}

func (s *ConstantCfgLoader) apply() {
	constant.Store(s.temp1)
}

var constant atomic.Value

type ConstantCfg struct {
	// 常量id
	Id int32 `json:"id"`
	// 常量字段
	Name string `json:"name"`
	// 值
	Value []int32 `json:"value"`

	// 道具
	Item *ItemConfig
}

func GetConstantCfg(id string) *ConstantCfg {
	cfgMap := constant.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[string]*ConstantCfg)[id]
}

const (
	CONSTANT_MAX_NICKNAME_LENGTH                   = "maxNicknameLength"
	CONSTANT_NICKNAME_CHANGE_ITEM                  = "nicknameChangeItem"
	CONSTANT_CHANGE_NICKNAME_FREE_TIMES            = "changeNicknameFreeTimes"
	CONSTANT_DAILY_PRIVILEGE_DROP_QUANTITY_LIMIT   = "dailyPrivilegeDropQuantityLimit"
	CONSTANT_PRIVILEGES_RECRUITMENT                = "privilegesRecruitment"
	CONSTANT_BUILDING_ACCELERATION_MIN_COST        = "buildingAccelerationMinCost"
	CONSTANT_PRODUCTION_ACCELERATION_MIN_COST      = "productionAccelerationMinCost"
	CONSTANT_arenaSeasonInitialPoints              = "arenaSeasonInitialPoints"
	CONSTANT_arenaDailyFreeRefreshAttempts         = "arenaDailyFreeRefreshAttempts"
	CONSTANT_refreshDiamondConsumptionQuantity     = "refreshDiamondConsumptionQuantity"
	CONSTANT_petSummonAutoRefreshInterval          = "petSummonAutoRefreshInterval"
	CONSTANT_petSummonAutoRefreshCount             = "petSummonAutoRefreshCount"
	CONSTANT_petSummonDiscountCount                = "petSummonDiscountCount"
	CONSTANT_petSummonDiscount                     = "petSummonDiscount"
	CONSTANT_petSummonRefreshDiamond               = "petSummonRefreshDiamond"
	CONSTANT_arenaDailyFreeChallengeAttempts       = "arenaDailyFreeChallengeAttempts"
	CONSTANT_BindingBonus                          = "BindingBonus"
	CONSTANT_arenaVictoryReward                    = "arenaVictoryReward"
	CONSTANT_arenaDefeatReward                     = "arenaDefeatReward"
	CONSTANT_gloryArenaChallengeTime               = "gloryArenaChallengeTime"
	CONSTANT_gloryArenaEntryRequirement            = "gloryArenaEntryRequirement"
	CONSTANT_gloryArenaOpponentRank                = "gloryArenaOpponentRank"
	CONSTANT_gloryArenaMaxHP                       = "gloryArenaMaxHP"
	CONSTANT_gloryArenaDefeatDrop                  = "gloryArenaDefeatDrop"
	CONSTANT_weeklyEmail                           = "weeklyEmail"
	CONSTANT_DAILY_AD_CHEST_OPENING_ATTEMPTS_LIMIT = "dailyAdChestOpeningAttemptsLimit"
	CONSTANT_maximumStamina                        = "maximumStamina"
	CONSTANT_staminaRecoveryTime                   = "staminaRecoveryTime"
	CONSTANT_dailyLimitOnFreeStaminaClaims         = "dailyLimitOnFreeStaminaClaims"
	CONSTANT_freeStaminaClaimCooldown              = "freeStaminaClaimCooldown"
	CONSTANT_dispatchAccelerationCost              = "dispatchAccelerationCost"
	CONSTANT_freeStaminaNumb                       = "freeStaminaNumb"
	CONSTANT_dispatchEmailRewards                  = "dispatchEmailRewards"
	CONSTANT_sevenSignMail                         = "sevenSignMail"
	CONSTANT_actSignMail                           = "actSignMail"
	CONSTANT_statLockCost                          = "statLockCost"
	CONSTANT_refundRateForEnhanceMaterials         = "refundRateForEnhanceMaterials"
	CONSTANT_statUpgradeProbability                = "statUpgradeProbability"
	CONSTANT_collectionExchangeSpid                = "collectionExchangeSpid"
	CONSTANT_cityProductionTime                    = "cityProductionTime"
	CONSTANT_lumberBonusHeroClass                  = "lumberBonusHeroClass"
	CONSTANT_cityProductBonusHeroPotential         = "cityProductBonusHeroPotential"
	CONSTANT_cityProductBonusHeroStar              = "cityProductBonusHeroStar"
	CONSTANT_successChanceIncrease                 = "successChanceIncrease"
	CONSTANT_trialMail                             = "trialMail"
	CONSTANT_sectNameChange                        = "sectNameChange"
	CONSTANT_sectCreateConsumption                 = "sectCreateConsumption"
	CONSTANT_sectCreateConditions                  = "sectCreateConditions"
	CONSTANT_sectNameCharacterCount                = "sectNameCharacterCount"
	CONSTANT_sectAnnouncementCharacterCount        = "sectAnnouncementCharacterCount"
	CONSTANT_sectmanifestoCount                    = "sectmanifestoCount"
	CONSTANT_sectRefuseMail                        = "sectRefuseMail"
	CONSTANT_sectdissolveMail                      = "sectdissolveMail"
	CONSTANT_sectLeaveMail                         = "sectLeaveMail"
	CONSTANT_dispatchQueues                        = "dispatchQueuesValue"
	CONSTANT_maxEquipmentFromMain                  = "maxEquipmentFromMain"
	CONSTANT_timeLimitedAdventure                  = "timeLimitedAdventure"
	CONSTANT_dailyAdventureLimit                   = "dailyAdventureLimit"
	CONSTANT_monsterAdventureProgressValue         = "monsterAdventureProgressValue"
	CONSTANT_realmStorageCap                       = "realmStorageCap"
	CONSTANT_gloryArenaRefreshItem                 = "gloryArenaRefreshItem"
)

func GetMaxNicknameLength() int32 {
	return GetConstantCfg(CONSTANT_MAX_NICKNAME_LENGTH).Value[0]
}

func GetNicknameChangeItem() *ItemConfig {
	return GetConstantCfg(CONSTANT_NICKNAME_CHANGE_ITEM).Item
}

func GetBindingBonus() *ItemConfig {
	return GetConstantCfg(CONSTANT_BindingBonus).Item
}

func GetChangeNicknameFreeTimes() int32 {
	return GetConstantCfg(CONSTANT_CHANGE_NICKNAME_FREE_TIMES).Value[0]
}

func GetDailyPrivilegeDropQuantityLimit() int32 {
	return GetConstantCfg(CONSTANT_DAILY_PRIVILEGE_DROP_QUANTITY_LIMIT).Value[0]
}

// GetPrivilegesRecruitment 获取招募权益配置（奖励物品）
// 从 privileges_constant 表中 name="privilegesRecruitment" 的配置读取
// 返回: itemId, quantity (格式: value[0]|value[1])
func GetPrivilegesRecruitment() (int32, int32) {
	cfg := GetConstantCfg(CONSTANT_PRIVILEGES_RECRUITMENT)
	if cfg == nil || len(cfg.Value) < 2 {
		return 0, 0
	}
	return cfg.Value[0], cfg.Value[1]
}

// GetVipCardRecruitment 获取招募权益对应的特权卡配置
// 从 privileges_constant 表中 name="privilegesRecruitment" 的配置读取
// 返回: vipCardItemId, rewardItemId (如果配置包含特权卡itemId，否则返回0,0)
// 注意：根据配置格式，如果 value 有3个值，则 value[0] 是特权卡itemId，value[1] 是奖励itemId，value[2] 是数量
// 如果 value 只有2个值，则 value[0] 是奖励itemId，value[1] 是数量（需要任意特权卡）
func GetVipCardRecruitment() (int32, int32) {
	cfg := GetConstantCfg(CONSTANT_PRIVILEGES_RECRUITMENT)
	if cfg == nil {
		return 0, 0
	}
	// 如果配置有3个值，返回特权卡itemId和奖励itemId
	if len(cfg.Value) >= 3 {
		return cfg.Value[0], cfg.Value[1] // vipCardItemId, rewardItemId
	}
	// 如果只有2个值，说明不需要特定特权卡，返回0表示需要任意特权卡
	return 0, cfg.Value[0] // 0表示任意特权卡, rewardItemId
}

// GetBuildingMinCost 获取建筑加速的最小消耗
func GetBuildingMinCost() int32 {
	cfg := GetConstantCfg(CONSTANT_BUILDING_ACCELERATION_MIN_COST)
	if cfg == nil || len(cfg.Value) == 0 {
		return 100 // 默认100
	}
	return cfg.Value[0]
}

// GetProductionMinCost 获取生产加速的最小消耗
func GetProductionMinCost() int32 {
	cfg := GetConstantCfg(CONSTANT_PRODUCTION_ACCELERATION_MIN_COST)
	if cfg == nil || len(cfg.Value) == 0 {
		return 100 // 默认100
	}
	return cfg.Value[0]
}

func GetArenaInitScore() int32 {
	return GetConstantCfg(CONSTANT_arenaSeasonInitialPoints).Value[0]
}

func GetArenaDailyFreeRefreshTimes() int32 {
	return GetConstantCfg(CONSTANT_arenaDailyFreeRefreshAttempts).Value[0]
}

func GetRefreshDiamondConsumptionQuantity() int32 {
	return GetConstantCfg(CONSTANT_refreshDiamondConsumptionQuantity).Value[0]
}

// GetPetSummonAutoRefreshIntervalSeconds 宠物招募系统自动刷新间隔（秒）
func GetPetSummonAutoRefreshIntervalSeconds() int32 {
	cfg := GetConstantCfg(CONSTANT_petSummonAutoRefreshInterval)
	if cfg == nil || len(cfg.Value) == 0 {
		return 1800
	}
	return cfg.Value[0]
}

// GetPetSummonAutoRefreshCount 宠物招募系统每日系统刷新次数上限
func GetPetSummonAutoRefreshCount() int32 {
	cfg := GetConstantCfg(CONSTANT_petSummonAutoRefreshCount)
	if cfg == nil || len(cfg.Value) == 0 {
		return 10
	}
	return cfg.Value[0]
}

// GetPetSummonDiscountCount 宠物招募钻石折扣次数（预留给招募扣费逻辑使用）
func GetPetSummonDiscountCount() int32 {
	cfg := GetConstantCfg(CONSTANT_petSummonDiscountCount)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

// GetPetSummonDiscount 宠物招募钻石折扣（预留给招募扣费逻辑使用）
func GetPetSummonDiscount() []int32 {
	cfg := GetConstantCfg(CONSTANT_petSummonDiscount)
	if cfg == nil {
		return nil
	}
	return cfg.Value
}

// GetPetSummonRefreshDiamond 宠物招募手动刷新钻石消耗
func GetPetSummonRefreshDiamond() int32 {
	cfg := GetConstantCfg(CONSTANT_petSummonRefreshDiamond)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetArenaDailyFreeChallengeTimes() int32 {
	return GetConstantCfg(CONSTANT_arenaDailyFreeChallengeAttempts).Value[0]
}

func GetArenaVictoryReward() int32 {
	return GetConstantCfg(CONSTANT_arenaVictoryReward).Value[0]
}

func GetArenaDefeatReward() int32 {
	return GetConstantCfg(CONSTANT_arenaDefeatReward).Value[0]
}

func GetGloryArenaEntryRequirement() int32 {
	return GetConstantCfg(CONSTANT_gloryArenaEntryRequirement).Value[0]
}

func GetGloryArenaOpponentRank() int32 {
	return GetConstantCfg(CONSTANT_gloryArenaOpponentRank).Value[0]
}

func GetGloryArenaMaxHP() int32 {
	cfg := GetConstantCfg(CONSTANT_gloryArenaMaxHP)
	if cfg == nil || len(cfg.Value) == 0 || cfg.Value[0] <= 0 {
		return 3
	}
	return cfg.Value[0]
}

func GetGloryArenaDefeatDrop() int32 {
	cfg := GetConstantCfg(CONSTANT_gloryArenaDefeatDrop)
	if cfg == nil || len(cfg.Value) == 0 || cfg.Value[0] <= 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetGloryArenaChallengeTime() (int32, int32) {
	cfg := GetConstantCfg(CONSTANT_gloryArenaChallengeTime)
	if cfg == nil || len(cfg.Value) < 2 {
		return 12, 22
	}
	return cfg.Value[0], cfg.Value[1]
}

func GetMaximumStamina() int32 {
	cfg := GetConstantCfg(CONSTANT_maximumStamina)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetStaminaRecoveryTime() int32 {
	cfg := GetConstantCfg(CONSTANT_staminaRecoveryTime)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetFreeStaminaNumb() int32 {
	cfg := GetConstantCfg(CONSTANT_freeStaminaNumb)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetDailyLimitOnFreeStaminaClaims() int32 {
	cfg := GetConstantCfg(CONSTANT_dailyLimitOnFreeStaminaClaims)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetFreeStaminaClaimCooldown() int32 {
	cfg := GetConstantCfg(CONSTANT_freeStaminaClaimCooldown)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetDispatchAccelerationCost() int32 {
	cfg := GetConstantCfg(CONSTANT_dispatchAccelerationCost)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetDailyAdChestOpeningAttemptsLimit() int32 {
	cfg := GetConstantCfg(CONSTANT_DAILY_AD_CHEST_OPENING_ATTEMPTS_LIMIT)
	if cfg == nil || len(cfg.Value) == 0 {
		return 20
	}
	return cfg.Value[0]
}

func GetStatLockCost() int32 {
	cfg := GetConstantCfg(CONSTANT_statLockCost)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetRefundRateForEnhanceMaterials() int32 {
	cfg := GetConstantCfg(CONSTANT_refundRateForEnhanceMaterials)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetStatUpgradeProbability() int32 {
	cfg := GetConstantCfg(CONSTANT_statUpgradeProbability)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetCollectionExchangeSpid() int32 {
	cfg := GetConstantCfg(CONSTANT_collectionExchangeSpid)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetWeeklyEmailId() int32 {
	return GetConstantCfg(CONSTANT_weeklyEmail).Value[0]
}

func GetSevenSignMailId() int32 {
	return GetConstantCfg(CONSTANT_sevenSignMail).Value[0]
}

func GetActSignMailId() int32 {
	return GetConstantCfg(CONSTANT_actSignMail).Value[0]
}

func GetTrialExpireMailTemplateID() int32 {
	return GetConstantCfg(CONSTANT_trialMail).Value[0]
}

func GetCityProductionTime() int32 {
	cfg := GetConstantCfg(CONSTANT_cityProductionTime)
	if cfg == nil || len(cfg.Value) == 0 {
		return 5
	}
	return cfg.Value[0]
}

// GetLumberBonusHeroClass 伐木场职业加成，返回 (heroClass, bonusPercent)
func GetLumberBonusHeroClass() (int32, int32) {
	cfg := GetConstantCfg(CONSTANT_lumberBonusHeroClass)
	if cfg == nil || len(cfg.Value) < 2 {
		return 0, 0
	}
	return cfg.Value[0], cfg.Value[1]
}

func GetCityProductBonusHeroPotential() int32 {
	cfg := GetConstantCfg(CONSTANT_cityProductBonusHeroPotential)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}

func GetCityProductBonusHeroStar() int32 {
	cfg := GetConstantCfg(CONSTANT_cityProductBonusHeroStar)
	if cfg == nil || len(cfg.Value) == 0 {
		return 0
	}
	return cfg.Value[0]
}
func GetChangeAllianceNameItem() *ItemConfig {
	cfg := GetConstantCfg(CONSTANT_sectNameChange)
	if cfg == nil || len(cfg.Value) == 0 {
		return nil
	}
	return &ItemConfig{
		ID:  cfg.Value[0],
		Num: int64(cfg.Value[1]),
	}
}
func GetCreateAllianceItem() *ItemConfig {
	cfg := GetConstantCfg(CONSTANT_sectCreateConsumption)
	if cfg == nil || len(cfg.Value) == 0 {
		return nil
	}
	return &ItemConfig{
		ID:  cfg.Value[0],
		Num: int64(cfg.Value[1]),
	}
}

func GetCreateAllianceUnlock() []int32 {
	cfg := GetConstantCfg(CONSTANT_sectCreateConditions)
	if cfg == nil || len(cfg.Value) == 0 {
		return make([]int32, 0)
	}
	return cfg.Value
}

func GetAllianceNameMaxLength() int32 {
	return GetConstantCfg(CONSTANT_sectNameCharacterCount).Value[0]
}

func GetAllianceNoticeMaxLength() int32 {
	return GetConstantCfg(CONSTANT_sectAnnouncementCharacterCount).Value[0]
}

func GetAllianceAnnounceMaxLength() int32 {
	return GetConstantCfg(CONSTANT_sectmanifestoCount).Value[0]
}

func GetAllianceRefuseMailId() int32 {
	return GetConstantCfg(CONSTANT_sectRefuseMail).Value[0]
}

func GetAllianceDissolveMailId() int32 {
	return GetConstantCfg(CONSTANT_sectdissolveMail).Value[0]
}

func GetAllianceKickMailId() int32 {
	return GetConstantCfg(CONSTANT_sectLeaveMail).Value[0]
}

func GetTimeLimitedAdventure() int32 {
	return GetConstantCfg(CONSTANT_timeLimitedAdventure).Value[0]
}

func GetDailyAdventureLimit() int32 {
	return GetConstantCfg(CONSTANT_dailyAdventureLimit).Value[0]
}

func GetMonsterAdventureProgressValue() int32 {
	return GetConstantCfg(CONSTANT_monsterAdventureProgressValue).Value[0]
}

func GetRealmStorageCap() int32 {
	return GetConstantCfg(CONSTANT_realmStorageCap).Value[0]
}

func GetGloryArenaRefreshItem() int32 {
	return GetConstantCfg(CONSTANT_gloryArenaRefreshItem).Value[0]
}
