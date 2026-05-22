package logicCommon

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/go-redis/redis/v8"
)

const PLAYER_REDIS_TIMEOUT = 30 * 24 * time.Hour

// 玩家redis存储的信息
type PlayerRedisInfo struct {
	BasicInfo  *PlayerBasicInfo  // 玩家基本信息
	BattleInfo *PlayerBattleInfo // 玩家战斗信息
}

type PlayerBasicInfo struct {
	Id                     int64  // 玩家id
	ServerId               int32  // 服务器id
	Name                   string // 昵称
	MainCityLevel          int32  // 主城等级
	ArenaVersion           string // 当前竞技场版本
	ArenaScore             int32  // 竞技场分数
	GloryArenaVersion      string // 荣耀擂台当前轮version
	GloryArenaBestWinCount int32  // 荣耀擂台当前轮最好的成绩
	ShowHeroId             int32  // 显示英雄id TODO:当前无用
	ShowClassId            int32  // 显示英雄class TODO:当前无用
	HeadId                 int32  // 头像id
	FrameId                int32  // 头像框id
	LastLoginTime          int64  // 上次登录时间
	LastOfflineTime        int64  // 上次离线时间
	Title                  int32  // 称号id
	BubbleId               int32  // 对话框id
	ImageId                int32  // 小人id
}

type PlayerAllianceInfo struct {
	ArenaJoined  bool
	RoundBestWin int32
	UserId       int64  // 玩家id
	AllianceId   int64  // 联盟id
	AllianceName string // 联盟名字
	JoinTime     int64  // 入盟时间（毫秒时间戳）
}

type PlayerBattleInfo struct {
	UserId          int64                         // 玩家id
	FormationInfo   map[int32]*FormationBasicInfo // 布阵信息
	FormationHeroes map[int64]*HeroBasicInfo      // 布阵的英雄信息
}

func (b *PlayerBattleInfo) GetMainFormationPower() int64 {
	if b.FormationInfo == nil {
		return 0
	}
	mainFormation := b.FormationInfo[int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN)]
	if mainFormation == nil {
		return 0
	}
	return mainFormation.BattlePower
}

type HeroBasicInfo struct {
	Uid         int64             // 唯一id
	Id          int64             // 英雄id
	ClassId     int32             // 英雄class
	Star        int32             // 英雄星级
	Level       int32             // 英雄等级
	Units       int32             // 英雄模型id
	AtkSpeed    int32             // 攻击速度
	MoveSpeed   int32             // 移动速度
	PatrolRange int32             // 巡逻范围
	AggroRange  int32             // 索敌范围
	AttackRange int32             // 移动范围
	NormalAtk   int32             // 普攻技能
	Attr        map[int32]int64   // 英雄属性
	Skill       []int32           // 技能
	PetInfo     *pb.PetBattleInfo // 宠物信息
}

type FormationBasicInfo struct {
	Heroes      []int64 // 英雄id
	BattlePower int64   // 战力
}

func GetPlayerRedisInfo(userId int64) *PlayerRedisInfo {
	ctx := context.Background()

	basicInfoKey := enum.GetPlayerBasicInfoKey(userId)
	battleInfoKey := enum.GetPlayerBattleInfoKey(userId)

	// 使用 Pipelined，自动执行并返回每个命令对象
	var (
		basicCmd  *redis.StringCmd
		battleCmd *redis.StringCmd
	)

	_, err := dbService.RDB.Pipelined(ctx, func(p redis.Pipeliner) error {
		// GETEX: 读取并续期 24h
		basicCmd = p.GetEx(ctx, basicInfoKey, PLAYER_REDIS_TIMEOUT)
		battleCmd = p.GetEx(ctx, battleInfoKey, PLAYER_REDIS_TIMEOUT)
		return nil
	})
	if err != nil && err != redis.Nil {
		logger.ErrorBySprintf("[redis] pipeline exec failed userId=%d err=%v", userId, err)
		return nil
	}

	// 1. basicInfo：必须存在
	basicInfoStr, err := basicCmd.Result()
	if err == redis.Nil {
		// 核心基础信息不存在，直接认为该玩家缓存无效
		logger.ErrorBySprintf("[redis] basic info missing key=%s userId=%d", basicInfoKey, userId)
		return nil
	}
	if err != nil {
		logger.ErrorBySprintf("[redis] get basic info failed key=%s userId=%d err=%v", basicInfoKey, userId, err)
		return nil
	}

	var basicInfo PlayerBasicInfo
	if err := json.Unmarshal([]byte(basicInfoStr), &basicInfo); err != nil {
		logger.ErrorBySprintf("[redis] unmarshal basic info failed key=%s userId=%d err=%v", basicInfoKey, userId, err)
		return nil
	}

	// 2. battleInfo：允许不存在
	var battleInfoPtr *PlayerBattleInfo
	battleInfoStr, err := battleCmd.Result()
	switch err {
	case nil:
		var battleInfo PlayerBattleInfo
		if uerr := json.Unmarshal([]byte(battleInfoStr), &battleInfo); uerr != nil {
			logger.ErrorBySprintf("[redis] unmarshal battle info failed key=%s userId=%d err=%v", battleInfoKey, userId, uerr)
			return nil
		}
		battleInfoPtr = &battleInfo
	case redis.Nil:
		// 没有战斗信息，允许为空
		battleInfoPtr = nil
	default:
		logger.ErrorBySprintf("[redis] get battle info failed key=%s userId=%d err=%v", battleInfoKey, userId, err)
		return nil
	}

	return &PlayerRedisInfo{
		BasicInfo:  &basicInfo,
		BattleInfo: battleInfoPtr,
	}
}

func UpdatePlayerRedisInfo(playerRedisInfo *PlayerRedisInfo) error {
	if playerRedisInfo == nil {
		return fmt.Errorf("playerRedisInfo is nil")
	}
	if playerRedisInfo.BasicInfo == nil {
		return fmt.Errorf("playerRedisInfo.basicInfo is nil")
	}

	userId := playerRedisInfo.BasicInfo.Id
	if userId <= 0 {
		return fmt.Errorf("invalid userId: %d", userId)
	}

	ctx := context.Background()
	expire := PLAYER_REDIS_TIMEOUT

	basicInfoKey := enum.GetPlayerBasicInfoKey(userId)
	battleInfoKey := enum.GetPlayerBattleInfoKey(userId)

	basicInfoBytes, err := json.Marshal(playerRedisInfo.BasicInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal basic info failed userId=%d err=%v", userId, err)
		return err
	}

	var battleInfoBytes []byte
	if playerRedisInfo.BattleInfo != nil {
		battleInfoBytes, err = json.Marshal(playerRedisInfo.BattleInfo)
		if err != nil {
			logger.ErrorBySprintf("[redis] marshal battle info failed userId=%d err=%v", userId, err)
			return err
		}
	}

	_, err = dbService.RDB.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, basicInfoKey, basicInfoBytes, expire)

		if playerRedisInfo.BattleInfo != nil {
			pipe.Set(ctx, battleInfoKey, battleInfoBytes, expire)
		} else {
			pipe.Del(ctx, battleInfoKey)
		}

		return nil
	})
	if err != nil {
		logger.ErrorBySprintf("[redis] update player redis info failed userId=%d err=%v", userId, err)
		return err
	}

	return nil
}

func GetPlayerRedisInfos(userIds []int64) map[int64]*PlayerRedisInfo {
	result := make(map[int64]*PlayerRedisInfo)
	if len(userIds) == 0 {
		return result
	}

	ctx := context.Background()

	type playerCmdSet struct {
		userId      int64
		basicKey    string
		allianceKey string
		battleKey   string
		basicCmd    *redis.StringCmd
		battleCmd   *redis.StringCmd
	}

	cmdSets := make([]*playerCmdSet, 0, len(userIds))

	_, err := dbService.RDB.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, userId := range userIds {
			cmdSet := &playerCmdSet{
				userId:    userId,
				basicKey:  enum.GetPlayerBasicInfoKey(userId),
				battleKey: enum.GetPlayerBattleInfoKey(userId),
			}

			cmdSet.basicCmd = p.GetEx(ctx, cmdSet.basicKey, PLAYER_REDIS_TIMEOUT)
			cmdSet.battleCmd = p.GetEx(ctx, cmdSet.battleKey, PLAYER_REDIS_TIMEOUT)

			cmdSets = append(cmdSets, cmdSet)
		}
		return nil
	})
	if err != nil && err != redis.Nil {
		logger.ErrorBySprintf("[redis] batch pipeline exec failed err=%v", err)
		return result
	}

	for _, cmdSet := range cmdSets {
		// basicInfo 必须存在
		basicInfoStr, err := cmdSet.basicCmd.Result()
		if err == redis.Nil {
			// 基础信息不存在，直接跳过这个玩家
			continue
		}
		if err != nil {
			logger.ErrorBySprintf("[redis] get basic info failed key=%s userId=%d err=%v", cmdSet.basicKey, cmdSet.userId, err)
			continue
		}

		var basicInfo PlayerBasicInfo
		if err := json.Unmarshal([]byte(basicInfoStr), &basicInfo); err != nil {
			logger.ErrorBySprintf("[redis] unmarshal basic info failed key=%s userId=%d err=%v", cmdSet.basicKey, cmdSet.userId, err)
			continue
		}

		// battleInfo 允许不存在
		var battleInfoPtr *PlayerBattleInfo
		battleInfoStr, err := cmdSet.battleCmd.Result()
		if err == nil {
			var battleInfo PlayerBattleInfo
			if uerr := json.Unmarshal([]byte(battleInfoStr), &battleInfo); uerr != nil {
				logger.ErrorBySprintf("[redis] unmarshal battle info failed key=%s userId=%d err=%v", cmdSet.battleKey, cmdSet.userId, uerr)
				continue
			}
			battleInfoPtr = &battleInfo
		} else if err != redis.Nil {
			logger.ErrorBySprintf("[redis] get battle info failed key=%s userId=%d err=%v", cmdSet.battleKey, cmdSet.userId, err)
			continue
		}

		result[cmdSet.userId] = &PlayerRedisInfo{
			BasicInfo:  &basicInfo,
			BattleInfo: battleInfoPtr,
		}
	}

	return result
}

func UpdatePlayerBasicInfo(basicInfo *PlayerBasicInfo) {
	key := enum.GetPlayerBasicInfoKey(basicInfo.Id)
	info, err := json.Marshal(basicInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal failed key=%s  err=%v", key, err)
	}
	err = dbService.RDB.SetEX(context.Background(), key, string(info), PLAYER_REDIS_TIMEOUT).Err()
	if err != nil {
		logger.ErrorBySprintf("[redis] set failed key=%s  err=%v", key, err)
	}
}

func UpdatePlayerBattleInfo(battleInfo *PlayerBattleInfo) {
	key := enum.GetPlayerBattleInfoKey(battleInfo.UserId)
	info, err := json.Marshal(battleInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal failed key=%s  err=%v", key, err)
	}
	err = dbService.RDB.SetEX(context.Background(), key, string(info), PLAYER_REDIS_TIMEOUT).Err()
	if err != nil {
		logger.ErrorBySprintf("[redis] set failed key=%s  err=%v", key, err)
	}
}

func UpdatePlayerAllianceInfo(allianceInfo *PlayerAllianceInfo) {
	key := enum.GetPlayerAllianceInfoKey(allianceInfo.UserId)
	info, err := json.Marshal(allianceInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal failed key=%s  err=%v", key, err)
	}
	err = dbService.RDB.Set(context.Background(), key, string(info), 0).Err()
	if err != nil {
		logger.ErrorBySprintf("[redis] set failed key=%s  err=%v", key, err)
	}
}

func GetPlayerAllianceInfoFromRedis(userId int64) *PlayerAllianceInfo {
	var allianceInfo PlayerAllianceInfo
	allianceInfo.UserId = userId

	allianceInfoKey := enum.GetPlayerAllianceInfoKey(userId)
	alliance, err := dbService.RDB.Get(context.Background(), allianceInfoKey).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorBySprintf("[redis] get alliance info failed userId=%d err=%v", userId, err)
		}
		return &allianceInfo
	}
	err = json.Unmarshal([]byte(alliance), &allianceInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal failed key=%s  err=%v", allianceInfoKey, err)
		return &allianceInfo
	}
	return &allianceInfo
}

func GetPlayerBasicInfoFromRedis(userId int64) *PlayerBasicInfo {
	var basicInfo PlayerBasicInfo
	basicInfoKey := enum.GetPlayerBasicInfoKey(userId)
	basicInfoString, err := dbService.RDB.GetEx(context.Background(), basicInfoKey, PLAYER_REDIS_TIMEOUT).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorBySprintf("[redis] get redis info failed userId=%d err=%v", userId, err)
		}
		return nil
	}
	err = json.Unmarshal([]byte(basicInfoString), &basicInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] marshal failed key=%s  err=%v", basicInfoKey, err)
		return nil
	}
	return &basicInfo
}

func GetPlayerBasicInfosFromRedis(userIds []int64) map[int64]*PlayerBasicInfo {
	result := make(map[int64]*PlayerBasicInfo)
	if len(userIds) == 0 || dbService.RDB == nil {
		return result
	}

	ctx := context.Background()

	type playerCmd struct {
		userId int64
		key    string
		cmd    *redis.StringCmd
	}

	cmdList := make([]*playerCmd, 0, len(userIds))
	dupMap := make(map[int64]struct{}, len(userIds))

	_, err := dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, userId := range userIds {
			if userId <= 0 {
				continue
			}
			if _, ok := dupMap[userId]; ok {
				continue
			}
			dupMap[userId] = struct{}{}

			key := enum.GetPlayerBasicInfoKey(userId)
			cmd := pipe.GetEx(ctx, key, PLAYER_REDIS_TIMEOUT)
			cmdList = append(cmdList, &playerCmd{
				userId: userId,
				key:    key,
				cmd:    cmd,
			})
		}
		return nil
	})
	if err != nil && err != redis.Nil {
		logger.ErrorBySprintf("[redis] batch get player basic info exec failed err=%v", err)
		return result
	}

	for _, item := range cmdList {
		basicInfoString, err := item.cmd.Result()
		if err != nil {
			if err != redis.Nil {
				logger.ErrorBySprintf("[redis] get redis info failed userId=%d key=%s err=%v", item.userId, item.key, err)
			}
			continue
		}

		var basicInfo PlayerBasicInfo
		if err := json.Unmarshal([]byte(basicInfoString), &basicInfo); err != nil {
			logger.ErrorBySprintf("[redis] unmarshal failed key=%s userId=%d err=%v", item.key, item.userId, err)
			continue
		}

		result[item.userId] = &basicInfo
	}

	return result
}

func GetPlayerBattleInfoFromRedis(userId int64) *PlayerBattleInfo {
	var battleInfo PlayerBattleInfo
	battleInfo.UserId = userId

	battleInfoKey := enum.GetPlayerBattleInfoKey(userId)
	battleInfoString, err := dbService.RDB.GetEx(context.Background(), battleInfoKey, PLAYER_REDIS_TIMEOUT).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorBySprintf("[redis] get battle info failed userId=%d err=%v", userId, err)
		}
		return nil
	}
	err = json.Unmarshal([]byte(battleInfoString), &battleInfo)
	if err != nil {
		logger.ErrorBySprintf("[redis] unmarshal failed key=%s  err=%v", battleInfoKey, err)
		return nil
	}
	return &battleInfo
}

func GetPlayerBattleInfosFromRedis(userIds []int64) map[int64]*PlayerBattleInfo {
	result := make(map[int64]*PlayerBattleInfo)
	if len(userIds) == 0 || dbService.RDB == nil {
		return result
	}

	ctx := context.Background()

	type playerCmd struct {
		userId int64
		key    string
		cmd    *redis.StringCmd
	}

	cmdList := make([]*playerCmd, 0, len(userIds))
	dupMap := make(map[int64]struct{}, len(userIds))

	_, err := dbService.RDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, userId := range userIds {
			if userId <= 0 {
				continue
			}
			if _, ok := dupMap[userId]; ok {
				continue
			}
			dupMap[userId] = struct{}{}

			key := enum.GetPlayerBattleInfoKey(userId)
			cmd := pipe.GetEx(ctx, key, PLAYER_REDIS_TIMEOUT)
			cmdList = append(cmdList, &playerCmd{
				userId: userId,
				key:    key,
				cmd:    cmd,
			})
		}
		return nil
	})
	if err != nil && err != redis.Nil {
		logger.ErrorBySprintf("[redis] batch get player battle info exec failed err=%v", err)
		return result
	}

	for _, item := range cmdList {
		battleInfoString, err := item.cmd.Result()
		if err != nil {
			if err != redis.Nil {
				logger.ErrorBySprintf("[redis] get battle info failed userId=%d key=%s err=%v", item.userId, item.key, err)
			}
			continue
		}

		var battleInfo PlayerBattleInfo
		battleInfo.UserId = item.userId

		if err := json.Unmarshal([]byte(battleInfoString), &battleInfo); err != nil {
			logger.ErrorBySprintf("[redis] unmarshal failed key=%s userId=%d err=%v", item.key, item.userId, err)
			continue
		}

		result[item.userId] = &battleInfo
	}

	return result
}

func GetOtherPlayerArenaScoreFromRedis(userId int64) int64 {
	basicInfo := GetPlayerBasicInfoFromRedis(userId)
	if basicInfo == nil {
		return 0
	}
	return int64(basicInfo.ArenaScore)
}

func GetOtherPlayerGloryArenaRoundBestWin(userId int64) int32 {
	basicInfo := GetPlayerBasicInfoFromRedis(userId)
	if basicInfo == nil {
		return 0
	}
	return basicInfo.GloryArenaBestWinCount
}

func parseYYYYMMDDToDateInt(version string) (int64, bool) {
	if len(version) != 8 {
		return 0, false
	}
	if _, err := strconv.Atoi(version); err != nil {
		return 0, false
	}
	t, err := time.ParseInLocation("20060102", version, time.Local)
	if err != nil {
		return 0, false
	}
	return int64(t.Year()*10000 + int(t.Month())*100 + t.Day()), true
}

func dateIntToTime(date int64) (time.Time, bool) {
	s := fmt.Sprintf("%08d", date)
	t, err := time.ParseInLocation("20060102", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
