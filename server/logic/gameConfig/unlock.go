package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
	"github.com/robfig/cron/v3"
)

func init() {
	RegisterConfigLoader("unlock", &UnlockCfgLoader{})
}

type UnlockCfgLoader struct {
	temp1 map[int32]*UnlockCfg
}

var _ configLoaderInterface = (*UnlockCfgLoader)(nil)

func (s *UnlockCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/unlock.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*UnlockCfg)
	for _, row := range rawData["unlock"] {
		var v UnlockCfg
		v.Id = ParseInt(row["id"])
		v.UnlockType = ParseInt(row["unlockType"])
		v.UnlockParam = ParseInt(row["unlockParam"])
		v.UnlockValue = row["unlockValue"]
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load unlock error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	return nil
}

func (s *UnlockCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid ID:%d", id))
		}
		if !enum.IsValidUnlockType(v.UnlockType) {
			return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid UnlockType:%d,id:%d", v.UnlockType, id))
		}
		switch enum.UnlockType(v.UnlockType) {
		case enum.UNLOCK_TYPE_PLAYER_IN_MAIN_INSTANCE, enum.UNLOCK_TYPE_PLAYER_FINISH_MAIN_INSTANCE:
			stageId := ParseInt(v.UnlockValue)
			if GetMainStageCfg(stageId) == nil || GetMainStageCfg(stageId).InstanceId != int32(enum.MAIN_INSTANCE_ID) {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid stageId:%d,id:%d", stageId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IN_INSTANCE, enum.UNLOCK_TYPE_PLAYER_FINISH_INSTANCE:
			instanceId := v.UnlockParam
			if GetInstanceCfg(instanceId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid instanceId:%d,id:%d", instanceId, id))
			}
			stageId := ParseInt(v.UnlockValue)
			if GetMainStageCfg(stageId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid stageId:%d,id:%d", stageId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_LEVEL:
			level := ParseInt(v.UnlockValue)
			if level <= 0 || GetRoleLevelCfg(level) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid level:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_VIP_LEVEL:
			level := ParseInt(v.UnlockValue)
			if level <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid vipLevel:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IN_MAIN_TASK, enum.UNLOCK_TYPE_PLAYER_FINISH_MAIN_TASK:
			taskId := ParseInt(v.UnlockValue)
			if taskId <= 0 || GetMainCfg(taskId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid mainTaskId:%d,id:%d", taskId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IN_SUB_TASK, enum.UNLOCK_TYPE_PLAYER_FINISH_SUB_TASK:
			taskId := ParseInt(v.UnlockValue)
			if GetSecondaryCfg(taskId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid subTaskId:%d,id:%d", taskId, id))
			}
		case enum.UNLOCK_TYPE_SERVER_OPEN_TIME:
			timeInterval := ParseInt(v.UnlockValue)
			if timeInterval < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid timeInterval:%d,id:%d", timeInterval, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_REGISTER_TIME:
			timeInterval := ParseInt(v.UnlockValue)
			if timeInterval < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid timeInterval:%d,id:%d", timeInterval, id))
			}
		case enum.UNLOCK_TYPE_SERVER_TIME:
			timeInterval := ParseInt(v.UnlockValue)
			if timeInterval < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid timeInterval:%d,id:%d", timeInterval, id))
			}
		case enum.UNLOCK_TYPE_SERVER_CURRENT_TIME:
			if v.UnlockParam == 0 {
				timeInterval, err := tool.ParseTime2TimeStamp(v.UnlockValue)
				if err != nil || timeInterval < 0 {
					return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid timeInterval:%d,id:%d", timeInterval, id))
				}
			} else {
				if !tool.ValidateCron("0 0 " + v.UnlockValue) {
					return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid cron:%s,id:%d", v.UnlockValue, id))
				}
			}
		case enum.UNLOCK_TYPE_SERVER_REGISTER_COUNT:
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_SERVER_ACTIVE_PLAYER_COUNT:
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_ALLIANCE_MEMBER_COUNT:
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_CHARGE_COUNT:
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_CHARGE_TIMES:
			times := ParseInt(v.UnlockValue)
			if times < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid times:%d,id:%d", times, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_CHARGE_DAY:
			day := ParseInt(v.UnlockValue)
			if day <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid day:%d,id:%d", day, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_BUY_TARGET_SHOP_ITEM:
			shopItemId := ParseInt(v.UnlockValue)
			if GetStillShopCfg(shopItemId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid shopItemId:%d,id:%d", shopItemId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_BUY_PRIVILEGE:
			shopItemId := ParseInt(v.UnlockValue)
			if GetVipCardCfg(shopItemId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid shopItemId:%d,id:%d", shopItemId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_LOGIN_DAYS:
			day := ParseInt(v.UnlockValue)
			if day <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid day:%d,id:%d", day, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_REGISTER_DAYS:
			day := ParseInt(v.UnlockValue)
			if day <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid day:%d,id:%d", day, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_MIN_LEVEL:
			level := ParseInt(v.UnlockValue)
			if level <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid level:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_LEVEL:
			level := ParseInt(v.UnlockValue)
			if level <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid level:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_MIN_STAR:
			star := ParseInt(v.UnlockValue)
			if star <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid star:%d,id:%d", star, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_HISTORY_MAX_STAR:
			star := ParseInt(v.UnlockValue)
			if star <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid star:%d,id:%d", star, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HAVE_HERO:
			heroId := ParseInt(v.UnlockValue)
			if heroId <= 0 || GetHeroBaseCfg(heroId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid heroId:%d,id:%d", heroId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_DRAW_CARD_TIMES:
			times := ParseInt(v.UnlockValue)
			if times < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid times:%d,id:%d", times, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_FAILED_IN_MAIN_INSTANCE:
			stageId := ParseInt(v.UnlockValue)
			if GetMainStageCfg(stageId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid stageId:%d,id:%d", stageId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IN_PRIVILEGE:
			privilegeId := ParseInt(v.UnlockValue)
			if !enum.IsValidVipPrivilegeType(privilegeId) {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid privilegeId:%d,id:%d", privilegeId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_LEVEL_UP_SUM_TODAY:
			num := ParseInt(v.UnlockValue)
			if num < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid times:%d,id:%d", num, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO:
			heroId := ParseInt(v.UnlockValue)
			if heroId <= 0 || GetHeroBaseCfg(heroId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid heroId:%d,id:%d", heroId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IS_LOTTERY_HERO_TODAY:
			heroId := ParseInt(v.UnlockValue)
			if heroId <= 0 || GetHeroBaseCfg(heroId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid heroId:%d,id:%d", heroId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY:
			quality := ParseInt(v.UnlockValue)
			if quality <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid quality:%d,id:%d", quality, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_IS_FIRST_LOTTERY_HERO_QUALITY_TODAY:
			quality := ParseInt(v.UnlockValue)
			if quality <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid quality:%d,id:%d", quality, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_STAR:
			star := ParseInt(v.UnlockValue)
			if star <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid star:%d,id:%d", star, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_HERO_NOW_MAX_LEVEL:
			level := ParseInt(v.UnlockValue)
			if level <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid level:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_ARCHITECTURE_LEVEL:
			level := ParseInt(v.UnlockValue)
			if tType := v.UnlockParam; tType == 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid unlockParam:%d,id:%d", v.UnlockParam, id))
			}
			if level <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid level:%d,id:%d", level, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_COLLECTION_NUM:
			num := ParseInt(v.UnlockValue)
			if num <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid num:%d,id:%d", num, id))
			}
		case enum.UNLOCK_TYPE_ACTIVITY_OPEN_DAY:
			activityId := v.UnlockParam
			day := ParseInt(v.UnlockValue)
			if activityId <= 0 || GetAllOriginalActivityCfg()[activityId] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid activityId:%d,id:%d", activityId, id))
			}
			if day <= 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid day:%d,id:%d", day, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_CITY_AGE:
			cityAgeId := ParseInt(v.UnlockValue)
			if cityAgeId <= 0 || GetCityAgeUpCfg(cityAgeId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid cityAgeId:%d,id:%d", cityAgeId, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_LOST:
		case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_ENROLL_WIN_COUNT:
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid winCount:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_GLORY_ARENA_FIRST_ENTER:
		case enum.UNLOCK_TYPE_PLAYER_PET_LOTTERY_DRAW_COUNT:
			if v.UnlockParam != 0 && v.UnlockParam != 1 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid unlockParam:%d,id:%d", v.UnlockParam, id))
			}
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_COLLECTION_LOTTERY_DRAW_COUNT:
			if v.UnlockParam != 0 && v.UnlockParam != 1 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid unlockParam:%d,id:%d", v.UnlockParam, id))
			}
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		case enum.UNLOCK_TYPE_PLAYER_EXPEDITION_COUNT:
			if v.UnlockParam != 0 && v.UnlockParam != 1 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid param:%d,id:%d", v.UnlockParam, id))
			}
			count := ParseInt(v.UnlockValue)
			if count < 0 {
				return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid count:%d,id:%d", count, id))
			}
		default:
			return errors.New(fmt.Sprintf("[gameConfig] load unlock error invalid unlockType:%d,id:%d", v.UnlockType, id))
		}
	}
	return nil
}

func (s *UnlockCfgLoader) apply() {
	unlockMap := make(map[int32]UnlockInterface)
	for _, v := range s.temp1 {
		switch enum.UnlockType(v.UnlockType) {
		case enum.UNLOCK_TYPE_SERVER_CURRENT_TIME:
			unlockBase := &UnlockTimeValueBase{
				Id:          v.Id,
				UnlockType:  enum.UnlockType(v.UnlockType),
				UnlockParam: v.UnlockParam,
			}
			if v.UnlockParam == 0 {
				unlockMap[v.Id].(*UnlockTimeValueBase).UnlockValue, _ = tool.ParseTime2TimeStamp(v.UnlockValue)
			} else {
				unlockBase.Cron, _ = cron.ParseStandard("0 0 " + v.UnlockValue)
			}
			unlockMap[v.Id] = unlockBase
		default:
			unlockMap[v.Id] = &UnlockIntValueBase{
				Id:          v.Id,
				UnlockType:  enum.UnlockType(v.UnlockType),
				UnlockParam: v.UnlockParam,
				UnlockValue: ParseInt(v.UnlockValue),
			}
		}
	}
	unlock.Store(unlockMap)
}

var unlock atomic.Value

type UnlockCfg struct {
	// id
	Id int32 `json:"id"`
	// 解锁类型
	UnlockType int32 `json:"unlockType"`
	// 解锁参数项
	UnlockParam int32 `json:"unlockParam"`
	// 解锁参数
	UnlockValue string `json:"unlockValue"`
}

func GetUnlockCfg(id int32) UnlockInterface {
	cfgMap := unlock.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]UnlockInterface)[id]
}

// 解锁服务接口
type UnlockInterface interface {
	// 获取解锁类型
	GetUnlockType() enum.UnlockType
}

// 解锁基础结构
type UnlockIntValueBase struct {
	Id          int32           // 解锁ID
	UnlockType  enum.UnlockType // 解锁类型
	UnlockParam int32           // 解锁参数
	UnlockValue int32           // 解锁值
}

// 获取解锁类型
func (u *UnlockIntValueBase) GetUnlockType() enum.UnlockType {
	return u.UnlockType
}

// 解锁时间结构
type UnlockTimeValueBase struct {
	Id          int32           // 解锁ID
	UnlockType  enum.UnlockType // 解锁类型
	UnlockParam int32           // 解锁参数
	UnlockValue int64           // 解锁时间戳
	Cron        cron.Schedule   // 解锁时间
}

// 获取解锁类型
func (u *UnlockTimeValueBase) GetUnlockType() enum.UnlockType {
	return u.UnlockType
}
