package gameConfig

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("rank", &RankCfgLoader{})
}

type CommonRankCfg struct {
	// 排行榜序号
	Id int32 `json:"id"`
	// 活动内排行榜分组id
	ActId int32 `json:"actId"`
	// 排名依据
	RankType int32 `json:"rankType"`
	// 排行积分类型
	PointType int32 `json:"pointType"`
	// 上榜门槛参数
	RankThreshold int64 `json:"rankThreshold"`
	// 发奖类型
	SendRewardType int32 `json:"sendRewardType"`
	// 排行榜结算类型
	SettlementType []int32 `json:"settlementType"`
	// 榜单显示人数
	PN int32 `json:"pN"`
	// 我未上榜最大显示人数
	PNMax int32 `json:"pNMax"`
	// 邮件id
	MailId []int32 `json:"mailId"`
	// 排名奖励id
	RankRewardsId []int32 `json:"rankRewardsId"`
}

type RankCfgLoader struct {
	temp1 map[int32]*RankCfg
	temp2 map[int32]*RankActCfg
	temp3 map[int32]*RankRewardCfg

	arenaRankId int32
	allRankCfg  map[int32]map[int32]*CommonRankCfg
}

var _ configLoaderInterface = (*RankCfgLoader)(nil)

func (s *RankCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/rank.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*RankCfg)
	for _, row := range rawData["rank"] {
		var v RankCfg
		v.Id = ParseInt(row["id"])
		v.RankType = ParseInt(row["rankType"])
		v.PointType = ParseInt(row["pointType"])
		v.RankThreshold = ParseInt64(row["rankThreshold"])
		v.SendRewardType = ParseInt(row["sendRewardType"])
		v.SettlementType = ParseIntArray(row["settlementType"])
		v.PN = ParseInt(row["pN"])
		v.PNMax = ParseInt(row["pNMax"])
		v.MailId = ParseIntArray(row["mailId"])
		v.RankRewardsId = ParseIntArray(row["rankRewardsId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*RankActCfg)
	for _, row := range rawData["rankAct"] {
		var v RankActCfg
		v.Id = ParseInt(row["id"])
		v.ActId = ParseInt(row["actId"])
		v.RankType = ParseInt(row["rankType"])
		v.PointType = ParseInt(row["pointType"])
		v.RankThreshold = ParseInt64(row["rankThreshold"])
		v.SendRewardType = ParseInt(row["sendRewardType"])
		v.SettlementType = ParseIntArray(row["settlementType"])
		v.PN = ParseInt(row["pN"])
		v.PNMax = ParseInt(row["pNMax"])
		v.MailId = ParseIntArray(row["mailId"])
		v.RankRewardsId = ParseIntArray(row["rankRewardsId"])
		v.AllDropId = ParseInt(row["allDropId"])
		v.LikeDropId = ParseInt(row["likeDropId"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	s.temp3 = make(map[int32]*RankRewardCfg)
	for _, row := range rawData["rankReward"] {
		var v RankRewardCfg
		v.Id = ParseInt(row["id"])
		v.RankRewards = row["rankRewards"]
		if v.Id <= 0 {
			continue
		}
		if s.temp3[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load rankReward error duplicate ID:%d", v.Id))
		}
		s.temp3[v.Id] = &v
	}

	s.allRankCfg = make(map[int32]map[int32]*CommonRankCfg)
	for _, v := range s.temp1 {
		if s.allRankCfg[0] == nil {
			s.allRankCfg[0] = make(map[int32]*CommonRankCfg)
		}
		s.allRankCfg[0][v.Id] = &CommonRankCfg{
			Id:             v.Id,
			ActId:          0,
			RankType:       v.RankType,
			PointType:      v.PointType,
			RankThreshold:  v.RankThreshold,
			SendRewardType: v.SendRewardType,
			SettlementType: append([]int32(nil), v.SettlementType...),
			PN:             v.PN,
			PNMax:          v.PNMax,
			MailId:         append([]int32(nil), v.MailId...),
			RankRewardsId:  append([]int32(nil), v.RankRewardsId...),
		}
	}

	for _, v := range s.temp2 {
		if s.allRankCfg[v.ActId] == nil {
			s.allRankCfg[v.ActId] = make(map[int32]*CommonRankCfg)
		}
		s.allRankCfg[v.ActId][v.Id] = &CommonRankCfg{
			Id:             v.Id,
			ActId:          v.ActId,
			RankType:       v.RankType,
			PointType:      v.PointType,
			RankThreshold:  v.RankThreshold,
			SendRewardType: v.SendRewardType,
			SettlementType: append([]int32(nil), v.SettlementType...),
			PN:             v.PN,
			PNMax:          v.PNMax,
			MailId:         append([]int32(nil), v.MailId...),
			RankRewardsId:  append([]int32(nil), v.RankRewardsId...),
		}
	}

	return nil
}

func isInvalidScoreTypeSettleTypeCombo(pointType int32, settleType int32) bool {
	switch pointType {
	case int32(enum.RANK_BOARD_SCORE_TYPE_LEVEL),
		int32(enum.RANK_BOARD_SCORE_TYPE_MAIN_INSTANCE),
		int32(enum.RANK_BOARD_SCORE_TYPE_BATTLE_POWER),
		int32(enum.RANK_BOARD_SCORE_TYPE_TOWER),
		int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_TOTAL_POWER),
		int32(enum.RANK_BOARD_SCORE_TYPE_ARENA),
		int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_ARENA):
		return settleType == int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER) ||
			settleType == int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER)
	case int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_ROUND_WIN_COUNT),
		int32(enum.RANK_BOARD_SCORE_TYPE_ALLIANCE_GLORY_ARENA_ROUND_WIN_COUNT):
		return settleType != int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_ROUND_OVER)
	case int32(enum.RANK_BOARD_SCORE_TYPE_GLORY_ARENA_SEASON_WIN_COUNT):
		return settleType != int32(enum.RANK_BOARD_SETTLE_TYPE_GLORY_ARENA_SEASON_OVER)
	default:
		return false
	}
}

func (s *RankCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid ID:%d", id))
		}
		if !enum.IsValidRankBoardRankRule(v.RankType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid RankType:%d,configId:%d", v.RankType, id))
		}
		if !enum.IsValidRankBoardScoreType(v.PointType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid PointType:%d,configId:%d", v.PointType, id))
		}
		if !enum.IsValidRankBoardSendRewardType(v.SendRewardType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid SendRewardType:%d,configId:%d", v.SettlementType, id))
		}
		for _, settleType := range v.SettlementType {
			if !enum.IsValidRankBoardSettleType(settleType) {
				return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid SettlementType:%d,configId:%d", settleType, id))
			}
			if settleType == int32(enum.RANK_BOARD_SETTLE_TYPE_ACTIVITY_OVER) {
				return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid SettlementType for common rank, settlementType:%d,configId:%d", settleType, id))
			}
			if isInvalidScoreTypeSettleTypeCombo(v.PointType, settleType) {
				return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid PointType/SettlementType combo, pointType:%d, settlementType:%d, configId:%d", v.PointType, settleType, id))
			}
		}
		if len(v.SettlementType) != len(v.RankRewardsId) || len(v.RankRewardsId) != len(v.MailId) {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid len(v.SettlementType) != len(v.RankRewardsId) || len(v.RankRewardsId) != len(v.MailId), rankRewardsId:%v, configId:%d", v.RankRewardsId, id))
		}
		if v.PN <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid PN:%d,configId:%d", v.PN, id))
		}
		if v.PNMax <= 0 || v.PNMax < v.PN {
			return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid PNMax:%d,configId:%d", v.PNMax, id))
		}
		for _, mailId := range v.MailId {
			if GetMailContentCfg(mailId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid MailId:%d,configId:%d", mailId, id))
			}
		}
		for _, reward := range v.RankRewardsId {
			if reward != 0 && s.temp3[reward] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load rank error invalid RankRewardsId:%d,configId:%d", reward, id))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid ID:%d", id))
		}
		if !enum.IsValidRankBoardRankRule(v.RankType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid RankType:%d,configId:%d", v.RankType, id))
		}
		if !enum.IsValidRankBoardScoreType(v.PointType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid PointType:%d,configId:%d", v.PointType, id))
		}
		if !enum.IsValidRankBoardSendRewardType(v.SendRewardType) {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid SendRewardType:%d,configId:%d", v.SettlementType, id))
		}
		for _, settleType := range v.SettlementType {
			if !enum.IsValidRankBoardSettleType(settleType) {
				return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid SettlementType:%d,configId:%d", settleType, id))
			}
			if isInvalidScoreTypeSettleTypeCombo(v.PointType, settleType) {
				return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid PointType/SettlementType combo, pointType:%d, settlementType:%d, configId:%d", v.PointType, settleType, id))
			}
		}
		if len(v.SettlementType) != len(v.RankRewardsId) || len(v.RankRewardsId) != len(v.MailId) {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid len(v.SettlementType) != len(v.RankRewardsId) || len(v.RankRewardsId) != len(v.MailId), rankRewardsId:%v, configId:%d", v.RankRewardsId, id))
		}
		if v.PN <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid PN:%d,configId:%d", v.PN, id))
		}
		if v.PNMax <= 0 || v.PNMax < v.PN {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid PNMax:%d,configId:%d", v.PNMax, id))
		}
		for _, mailId := range v.MailId {
			if GetMailContentCfg(mailId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid MailId:%d,configId:%d", mailId, id))
			}
		}
		for _, reward := range v.RankRewardsId {
			if reward != 0 && s.temp3[reward] == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid RankRewardsId:%d,configId:%d", reward, id))
			}
		}
		if v.AllDropId != 0 && GetDropCfg(v.AllDropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid AllDropId:%d,configId:%d", v.AllDropId, id))
		}
		if v.LikeDropId != 0 && GetDropCfg(v.LikeDropId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load rankAct error invalid LikeDropId:%d,configId:%d", v.LikeDropId, id))
		}
	}
	for id, v := range s.temp3 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid ID:%d", id))
		}
		if v.RankRewards == "" {
			return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid RankRewards:%s,configId:%d", v.RankRewards, id))
		}
		v.awards = make([]*rankRewardData, 0)
		rankRewardStrings := strings.Split(v.RankRewards, ";")
		for _, rankRewardStr := range rankRewardStrings {
			rankRewardString := strings.Split(rankRewardStr, "|")
			if len(rankRewardString) != 2 {
				return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid RankRewards:%s,configId:%d", v.RankRewards, id))
			}
			rankStrings := strings.Split(rankRewardString[0], "~")
			if len(rankStrings) != 2 {
				return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid RankRewards:%s,configId:%d", v.RankRewards, id))
			}
			rankLeft := ParseInt(rankStrings[0])
			rankRight := ParseInt(rankStrings[1])
			if rankLeft < 0 || rankRight < 0 || rankLeft > rankRight {
				return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid RankRewards:%s,configId:%d", v.RankRewards, id))
			}
			dropId := ParseInt(rankRewardString[1])
			if GetDropCfg(dropId) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load rankReward error invalid DropId:%d,configId:%d", dropId, id))
			}
			v.awards = append(v.awards, &rankRewardData{
				DropId:    dropId,
				RankLeft:  rankLeft,
				RankRight: rankRight,
			})
		}
	}
	return nil
}

func (s *RankCfgLoader) apply() {
	rank.Store(s.temp1)
	rankAct.Store(s.temp2)
	rankReward.Store(s.temp3)
	arenaRankId.Store(s.arenaRankId)
	allRankCfg.Store(s.allRankCfg)
}

var rank atomic.Value
var rankAct atomic.Value
var rankReward atomic.Value
var arenaRankId atomic.Int32
var allRankCfg atomic.Value

type RankCfg struct {
	// 排行榜序号
	Id int32 `json:"id"`
	// 排名依据
	RankType int32 `json:"rankType"`
	// 排行积分类型
	PointType int32 `json:"pointType"`
	// 胜场上榜门槛
	RankThreshold int64 `json:"rankThreshold"`
	// 发奖类型
	SendRewardType int32 `json:"sendRewardType"`
	// 排行榜结算类型
	SettlementType []int32 `json:"settlementType"`
	// 榜单显示人数
	PN int32 `json:"pN"`
	// 我未上榜最大显示人数
	PNMax int32 `json:"pNMax"`
	// 发奖邮件
	MailId []int32 `json:"mailId"`
	// 排名奖励id
	RankRewardsId []int32 `json:"rankRewardsId"`
}

type RankActCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 活动表id
	ActId int32 `json:"actId"`
	// 排名依据
	RankType int32 `json:"rankType"`
	// 排行积分类型
	PointType int32 `json:"pointType"`
	// 胜场上榜门槛
	RankThreshold int64 `json:"rankThreshold"`
	// 发奖类型
	SendRewardType int32 `json:"sendRewardType"`
	// 排行榜结算类型
	SettlementType []int32 `json:"settlementType"`
	// 榜单人数
	PN int32 `json:"pN"`
	// 我未上榜最大显示人数
	PNMax int32 `json:"pNMax"`
	// 发奖邮件
	MailId []int32 `json:"mailId"`
	// 排名奖励id
	RankRewardsId []int32 `json:"rankRewardsId"`
	// 全民奖励
	AllDropId int32 `json:"allDropId"`
	// 点赞每日奖励
	LikeDropId int32 `json:"likeDropId"`
}

type RankRewardCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 排名奖励
	RankRewards string `json:"rankRewards"`

	awards []*rankRewardData
}

type rankRewardData struct {
	// 排行
	RankLeft int32
	// 排名
	RankRight int32
	// 掉落id
	DropId int32
}

func GetAllRankCfg() map[int32]map[int32]*CommonRankCfg {
	cfgMap := allRankCfg.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]map[int32]*CommonRankCfg)
}

func GetRankCfgByIds(actId, rankId int32) *CommonRankCfg {
	cfgMap := allRankCfg.Load()
	if cfgMap == nil {
		return nil
	}
	temp := cfgMap.(map[int32]map[int32]*CommonRankCfg)
	if temp[actId] == nil {
		return nil
	}
	return temp[actId][rankId]
}

func GetRankCfg(id int32) *RankCfg {
	cfgMap := rank.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*RankCfg)[id]
}

func GetRankActCfg(id int32) *RankActCfg {
	cfgMap := rankAct.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*RankActCfg)[id]
}

// TODO:优化成enum
func GetArenaRankId() int32 {
	return 6
}

func GetRankRewardCfgWithRank(awardId, rank int32) int32 {
	cfgMap := rankReward.Load()
	if cfgMap == nil {
		return 0
	}
	cfg := cfgMap.(map[int32]*RankRewardCfg)[awardId]
	if cfg == nil {
		return 0
	}
	for _, v := range cfg.awards {
		if rank >= v.RankLeft && rank <= v.RankRight {
			return v.DropId
		}
	}
	return 0
}
