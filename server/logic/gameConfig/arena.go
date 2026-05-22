package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/tool"
	"math"
	"strings"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("arena", &ArenaCfgLoader{})
}

type ArenaCfgLoader struct {
	temp1 map[int32]*BotCfg
	temp2 map[int32]*PointsParametersCfg

	robotRankList []*BotCfg
}

var _ configLoaderInterface = (*ArenaCfgLoader)(nil)

func (s *ArenaCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/arena.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*BotCfg)
	for _, row := range rawData["bot"] {
		var v BotCfg
		v.Id = ParseInt(row["id"])
		v.ArenaPoints = ParseInt(row["arenaPoints"])
		v.Power = ParseInt(row["power"])
		v.ArenaLineup = ParseIntArray(row["arenaLineup"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load bot error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*PointsParametersCfg)
	for _, row := range rawData["pointsParameters"] {
		var v PointsParametersCfg
		v.Id = ParseInt(row["id"])
		v.WinBase = ParseInt(row["winBase"])
		v.LoseBase = ParseInt(row["loseBase"])
		v.Coeff1 = ParseInt(row["coeff1"])
		v.Coeff2 = ParseInt(row["coeff2"])
		v.PointsRange = row["pointsRange"]
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}
	s.sortBotByArenaPoints(s.temp1)
	return nil
}

func (s *ArenaCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bot error invalid ID:%d", id))
		}
		if v.ArenaPoints <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bot error invalid arenaPoints:%d,configId:%d", v.ArenaPoints, id))
		}
		if v.Power <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bot error invalid power:%d,configId:%d", v.Power, id))
		}
		if len(v.ArenaLineup) <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load bot error invalid arenaLineUp length = 0,configId:%d", id))
		}
		for _, v := range v.ArenaLineup {
			if GetMonsterCfg(v) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load bot error invalid Lineup:%d,configId:%d", v, id))
			}
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid ID:%d", id))
		}
		if v.WinBase <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid winBase:%d,configId:%d", v.WinBase, id))
		}
		if v.LoseBase <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid loseBase:%d,configId:%d", v.LoseBase, id))
		}
		if v.Coeff1 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid Coeff1:%d,configId:%d", v.Coeff1, id))
		}
		if v.Coeff2 <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid Coeff2:%d,configId:%d", v.Coeff2, id))
		}
		err := v.BuildData()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ArenaCfgLoader) apply() {
	bot.Store(s.temp1)
	pointsParameters.Store(s.temp2)
	rankBot.Store(s.robotRankList)
}

func (s *ArenaCfgLoader) sortBotByArenaPoints(allBot map[int32]*BotCfg) {
	// 将map转换为slice以便排序
	bots := make([]*BotCfg, 0, len(allBot))
	for _, bot := range allBot {
		bots = append(bots, bot)
	}

	// 按照竞技场积分降序排列（积分高的排在前面）
	for i := 0; i < len(bots)-1; i++ {
		for j := i + 1; j < len(bots); j++ {
			if bots[i].ArenaPoints < bots[j].ArenaPoints {
				bots[i], bots[j] = bots[j], bots[i]
			}
		}
	}

	// 存储排序结果
	s.robotRankList = bots
}

var bot atomic.Value
var pointsParameters atomic.Value
var rankBot atomic.Value

type BotCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 备注-战力分段
	Type string `json:"type"`
	// 竞技场积分
	ArenaPoints int32 `json:"arenaPoints"`
	// 战力值
	Power int32 `json:"power"`
	// 阵容配置
	ArenaLineup []int32 `json:"arenaLineup"`
}

type PointsParametersCfg struct {
	// 阶段k
	Id int32 `json:"id"`
	// 积分范围
	PointsRange string `json:"pointsRange"`
	// 胜利基础分-Sbase
	WinBase int32 `json:"winBase"`
	// 失败基础分-Tbase
	LoseBase int32 `json:"loseBase"`
	// 积分差系数α
	Coeff1 int32 `json:"coeff1"`
	// 失败系数β
	Coeff2 int32 `json:"coeff2"`

	RealRange []int32
}

func (c *PointsParametersCfg) BuildData() error {
	c.RealRange = make([]int32, 2)
	if c.PointsRange == "" {
		return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid PointsRange configId:%d", c.Id))
	}
	parts := strings.Split(c.PointsRange, "-")
	for i, v := range parts {
		c.RealRange[i] = ParseInt(v)
	}
	if c.RealRange[1] == 0 {
		c.RealRange[1] = math.MaxInt32
	}
	if c.RealRange[0] <= 0 || c.RealRange[1] <= 0 || c.RealRange[0] > c.RealRange[1] {
		return errors.New(fmt.Sprintf("[gameConfig] load pointsParameters error invalid PointsRange:%s,configId:%d", c.PointsRange, c.Id))
	}
	return nil
}

func GetBotCfg(id int32) *BotCfg {
	cfgMap := bot.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*BotCfg)[id]
}

func GetPointsParametersCfgByScore(score int32) *PointsParametersCfg {
	cfgMap := pointsParameters.Load()
	for _, v := range cfgMap.(map[int32]*PointsParametersCfg) {
		if v.RealRange[0] <= score && score <= v.RealRange[1] {
			return v
		}
	}
	return nil
}

func GetRankBotList() []*BotCfg {
	cfgMap := rankBot.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.([]*BotCfg)
}

func GetBotCfgByScore(scoreLeft, scoreRight, num int32) []*BotCfg {
	bots := make([]*BotCfg, 0)
	for _, botT := range GetRankBotList() {
		if botT.ArenaPoints >= scoreLeft && botT.ArenaPoints <= scoreRight {
			bots = append(bots, botT)
		}
		if int32(len(bots)) >= num {
			break
		}
	}
	return bots
}
