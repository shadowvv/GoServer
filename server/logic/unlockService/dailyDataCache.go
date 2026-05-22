package unlockService

import (
	"context"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/service/dbService"
)

// DailyDataCache 每日数据缓存管理器
// 用于管理 Redis 中的每日统计数据
type DailyDataCache struct {
	// 使用项目已有的 Redis 客户端
}

const (
	// 每日数据过期时间：27小时（比24小时多3小时，防止跨时区问题）
	DailyDataExpire = 27 * time.Hour
)

// DailyCache 全局每日数据缓存实例
var DailyCache = &DailyDataCache{}

// RecordHeroLevelUp 记录英雄升级
// 在英雄升级时调用此方法，增加今日升级次数
func (d *DailyDataCache) RecordHeroLevelUp(ctx context.Context, userId int64, levelUpCount int32) error {
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyHeroLevelUpKey(today)
	userField := strconv.FormatInt(userId, 10)

	// 使用 HINCRBY 增加升级次数
	_, err := dbService.RDB.HIncrBy(ctx, redisKey, userField, int64(levelUpCount)).Result()
	if err != nil {
		return err
	}

	// 设置过期时间（只在第一次设置时需要）
	dbService.RDB.Expire(ctx, redisKey, DailyDataExpire)

	return nil
}

// RecordLottery 记录抽卡
// 在抽卡时调用此方法，记录抽卡次数和抽到的物品
func (d *DailyDataCache) RecordLottery(ctx context.Context, userId int64, lotteryId int32, itemId int32, count int32) error {
	today := time.Now().Format("20060102")
	userField := strconv.FormatInt(userId, 10)

	// 1. 记录抽卡次数
	countKey := enum.GetDailyLotteryCountKey(today, lotteryId)
	_, err := dbService.RDB.HIncrBy(ctx, countKey, userField, int64(count)).Result()
	if err != nil {
		return err
	}
	dbService.RDB.Expire(ctx, countKey, DailyDataExpire)

	// 2. 检查物品类型和品质
	itemCfg := gameConfig.GetItemCfg(itemId)
	if itemCfg == nil {
		return nil
	}

	// 3. 如果是英雄类型，记录抽到的英雄
	if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) {
		heroKey := enum.GetDailyLotteryHeroKey(today)
		heroSetKey := heroKey + ":" + userField

		_, err = dbService.RDB.SAdd(ctx, heroSetKey, itemCfg.TargetId).Result()
		if err != nil {
			return err
		}
		dbService.RDB.Expire(ctx, heroSetKey, DailyDataExpire)
	}

	// 4. 记录抽到的品质
	qualityKey := enum.GetDailyLotteryQualityKey(today, itemCfg.Quality)
	qualitySetKey := qualityKey + ":" + userField

	_, err = dbService.RDB.SAdd(ctx, qualitySetKey, itemId).Result()
	if err != nil {
		return err
	}
	dbService.RDB.Expire(ctx, qualitySetKey, DailyDataExpire)

	return nil
}

// GetDailyHeroLevelUpCount 获取今日英雄升级次数
func (d *DailyDataCache) GetDailyHeroLevelUpCount(ctx context.Context, userId int64) (int32, error) {
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyHeroLevelUpKey(today)
	userField := strconv.FormatInt(userId, 10)

	countStr, err := dbService.RDB.HGet(ctx, redisKey, userField).Result()
	if err != nil {
		return 0, nil // 没有记录返回0
	}
	count, _ := strconv.ParseInt(countStr, 10, 32)
	return int32(count), nil
}

// GetDailyLotteryCount 获取今日指定卡池的抽卡次数
func (d *DailyDataCache) GetDailyLotteryCount(ctx context.Context, userId int64, lotteryId int32) (int32, error) {
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyLotteryCountKey(today, lotteryId)
	userField := strconv.FormatInt(userId, 10)

	countStr, err := dbService.RDB.HGet(ctx, redisKey, userField).Result()
	if err != nil {
		return 0, nil
	}
	count, _ := strconv.ParseInt(countStr, 10, 32)
	return int32(count), nil
}

// HasLotteryHeroToday 检查今日是否抽到过指定英雄
func (d *DailyDataCache) HasLotteryHeroToday(ctx context.Context, userId int64, heroId int32) (bool, error) {
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyLotteryHeroKey(today)
	userField := strconv.FormatInt(userId, 10)
	heroSetKey := redisKey + ":" + userField

	exists, err := dbService.RDB.SIsMember(ctx, heroSetKey, heroId).Result()
	if err != nil {
		return false, err
	}
	return exists, nil
}

// HasLotteryQualityToday 检查今日是否抽到过指定品质的物品
func (d *DailyDataCache) HasLotteryQualityToday(ctx context.Context, userId int64, quality int32) (bool, error) {
	today := time.Now().Format("20060102")
	redisKey := enum.GetDailyLotteryQualityKey(today, quality)
	userField := strconv.FormatInt(userId, 10)
	qualitySetKey := redisKey + ":" + userField

	count, err := dbService.RDB.SCard(ctx, qualitySetKey).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ClearExpiredData 清理过期数据（可选，Redis会自动过期）
// 如果需要手动清理，可以在每日0点执行
func (d *DailyDataCache) ClearExpiredData(ctx context.Context, date string) error {
	// 删除指定日期的所有 key
	patterns := []string{
		enum.REDIS_DAILY_HERO_LEVELUP + date + "*",
		enum.REDIS_DAILY_LOTTERY_COUNT + date + "*",
		enum.REDIS_DAILY_LOTTERY_QUALITY + date + "*",
		enum.REDIS_DAILY_LOTTERY_HERO + date + "*",
	}

	for _, pattern := range patterns {
		keys, err := dbService.RDB.Keys(ctx, pattern).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			dbService.RDB.Del(ctx, keys...)
		}
	}

	return nil
}
