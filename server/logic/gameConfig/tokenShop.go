package gameConfig

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
)

func init() {
	RegisterConfigLoader("tokenShop", &TokenShopCfgLoader{})
}

type TokenShopCfgLoader struct {
	temp1 map[int32]*TokenShopMainCfg
	temp2 map[int32]*TokenShopItemCfg
}

var _ configLoaderInterface = (*TokenShopCfgLoader)(nil)

func (s *TokenShopCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/tokenShop.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*TokenShopMainCfg)
	for _, row := range rawData["main"] {
		var v TokenShopMainCfg
		v.Id = ParseInt(row["id"])
		v.SystemId = ParseInt(row["systemId"])
		v.ActId = ParseInt(row["actId"])
		v.Refresh = ParseInt(row["refresh"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load main error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v
	}

	s.temp2 = make(map[int32]*TokenShopItemCfg)
	for _, row := range rawData["reward"] {
		var v TokenShopItemCfg
		v.Id = ParseInt(row["id"])
		v.ExchangeId = ParseInt(row["exchangeId"])
		v.ShopId = ParseInt(row["shopId"])
		v.LimitBuy = ParseInt(row["limitBuy"])
		v.UnlockShow = ParseInt(row["unlockShow"])
		v.UnlockBuy = ParseInt(row["unlockBuy"])
		v.UnlockStop = ParseInt(row["unlockStop"])
		v.Discount = ParseIntArray(row["discount"])
		v.DiscountWeight = ParseIntArray(row["discountWeight"])
		if v.Id <= 0 {
			continue
		}
		if s.temp2[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error duplicate ID:%d", v.Id))
		}
		s.temp2[v.Id] = &v
	}

	return nil
}

func (s *TokenShopCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load main error invalid ID:%d", id))
		}
		if v.SystemId != 0 && GetSystemUnlockCfg(v.SystemId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load main error invalid systemId:%d,configId:%d", v.SystemId, id))
		}
		if v.Refresh != 0 && !enum.IsValidItemRefreshType(v.Refresh) {
			return errors.New(fmt.Sprintf("[gameConfig] load main error invalid refresh:%d,configId:%d", v.Refresh, id))
		}
	}
	for id, v := range s.temp2 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid ID:%d", id))
		}
		if GetExchangeCfg(v.ExchangeId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid exchangeId:%d,configId:%d", v.ExchangeId, id))
		}
		if v.LimitBuy < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid limitBuy:%d,configId:%d", v.LimitBuy, id))
		}
		if v.UnlockShow != 0 && GetUnlockCfg(v.UnlockShow) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid unlockShow:%d,configId:%d", v.UnlockShow, id))
		}
		if v.UnlockBuy != 0 && GetUnlockCfg(v.UnlockBuy) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid unlockBuy:%d,configId:%d", v.UnlockBuy, id))
		}
		if v.UnlockStop != 0 && GetUnlockCfg(v.UnlockStop) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid UnlockStop:%d,configId:%d", v.UnlockStop, id))
		}
		if len(v.Discount) != len(v.DiscountWeight) {
			return errors.New(fmt.Sprintf("[gameConfig] load reward error invalid discount,configId:%d", id))
		}
	}
	return nil
}

func (s *TokenShopCfgLoader) apply() {
	tokenShopMain.Store(s.temp1)
	reward.Store(s.temp2)
}

var tokenShopMain atomic.Value
var reward atomic.Value

type TokenShopMainCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 商店类型
	ShopType int32 `json:"shopType"`
	// 功能id
	SystemId int32 `json:"systemId"`
	// 活动id
	ActId int32 `json:"actId"`
	// 刷新条件
	Refresh int32 `json:"refresh"`
}

type TokenShopItemCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 兑换id
	ExchangeId int32 `json:"exchangeId"`
	// 页签
	ShopId int32 `json:"shopId"`
	// 限购次数
	LimitBuy int32 `json:"limitBuy"`
	// 显示解锁
	UnlockShow int32 `json:"unlockShow"`
	// 购买解锁
	UnlockBuy int32 `json:"unlockBuy"`
	//达到条件后不显示
	UnlockStop int32 `json:"unlockStop"`
	// 折扣
	Discount []int32 `json:"discount"`
	// 折扣权重
	DiscountWeight []int32 `json:"discountWeight"`
}

func (c *TokenShopItemCfg) RandomDiscount() int32 {
	if len(c.Discount) == 0 {
		return 0
	}
	return tool.RandomWeight(c.Discount, c.DiscountWeight)
}

func GetTokenShopMainCfg(id int32) *TokenShopMainCfg {
	cfgMap := tokenShopMain.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TokenShopMainCfg)[id]
}

func GetAllTokenShopItem() map[int32]*TokenShopItemCfg {
	cfgMap := reward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TokenShopItemCfg)
}

func GetTokenShopItemCfg(id int32) *TokenShopItemCfg {
	cfgMap := reward.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*TokenShopItemCfg)[id]
}
