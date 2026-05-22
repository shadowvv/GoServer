package gameConfig

import (
	"errors"
	"fmt"
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/tool"
	"sync/atomic"
)

func init() {
	RegisterConfigLoader("stillShop", &StillShopCfgLoader{})
}

type StillShopCfgLoader struct {
	temp1 map[int32]*StillShopCfg

	shopItemTypeToCfgMap map[int32]map[int32]*StillShopCfg
	shopTypeToCfgMap     map[int32]map[int32]*StillShopCfg
}

var _ configLoaderInterface = (*StillShopCfgLoader)(nil)

func (s *StillShopCfgLoader) loadData() error {
	var rawData map[string]map[string]map[string]string
	if err := tool.LoadJSON(`gameConfig/stillshop.json`, &rawData); err != nil {
		return err
	}

	s.temp1 = make(map[int32]*StillShopCfg)
	s.shopItemTypeToCfgMap = make(map[int32]map[int32]*StillShopCfg)
	s.shopTypeToCfgMap = make(map[int32]map[int32]*StillShopCfg)
	for _, row := range rawData["stillshop"] {
		var v StillShopCfg
		v.Id = ParseInt(row["id"])
		v.ActId = ParseInt(row["actId"])
		v.TypeId = ParseInt(row["typeId"])
		v.ShopType = ParseInt(row["shopType"])
		v.ProductId = ParseInt(row["productId"])
		v.DropId = ParseIntArray(row["dropId"])
		v.LimitBuy = ParseInt(row["limitBuy"])
		v.UnlockShow = ParseIntArray(row["unlockShow"])
		v.UnlockBuy = ParseIntArray(row["unlockBuy"])
		v.NextPackage = ParseInt(row["nextPackage"])
		v.GiftRefreshParam = ParseInt(row["giftRefreshParam"])
		v.GiftRefresh = ParseInt(row["giftRefresh"])
		v.FirstBuy = ParseInt(row["firstBuy"])
		v.Diamond = ParseInt(row["diamond"])
		v.Adv = ParseInt(row["adv"])
		v.UnlockStop = ParseIntArray(row["unlockStop"])
		v.Duration = ParseInt(row["duration"])
		if v.Id <= 0 {
			continue
		}
		if s.temp1[v.Id] != nil {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error duplicate ID:%d", v.Id))
		}
		s.temp1[v.Id] = &v

		if s.shopItemTypeToCfgMap[v.TypeId] == nil {
			s.shopItemTypeToCfgMap[v.TypeId] = make(map[int32]*StillShopCfg)
		}
		s.shopItemTypeToCfgMap[v.TypeId][v.Id] = &v

		if s.shopTypeToCfgMap[v.ShopType] == nil {
			s.shopTypeToCfgMap[v.ShopType] = make(map[int32]*StillShopCfg)
		}
		s.shopTypeToCfgMap[v.ShopType][v.Id] = &v
	}

	return nil
}

func (s *StillShopCfgLoader) checkData() error {
	for id, v := range s.temp1 {
		if v.Id <= 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d", id))
		}
		if !enum.IsValidShopItemType(v.TypeId) {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,typeId not valid:%d", id, v.TypeId))
		}
		if GetProductCfg(v.ProductId) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,not in product productId:%d", id, v.ProductId))
		}
		for i := 0; i < len(v.DropId); i++ {
			if GetDropCfg(v.DropId[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,dropId[%d] not in drop", id, v.DropId[i]))
			}
		}
		if v.LimitBuy < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,limitBuy <= 0", id))
		}
		for i := 0; i < len(v.UnlockShow); i++ {
			if v.UnlockShow[i] != 0 && GetUnlockCfg(v.UnlockShow[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,unlockShow[%d] not in unlock", id, v.UnlockShow[i]))
			}
		}
		for i := 0; i < len(v.UnlockBuy); i++ {
			if v.UnlockBuy[i] != 0 && GetUnlockCfg(v.UnlockBuy[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,unlockBuy[%d] not in unlock", id, v.UnlockBuy[i]))
			}
		}
		for i := 0; i < len(v.UnlockStop); i++ {
			if v.UnlockStop[i] != 0 && GetUnlockCfg(v.UnlockStop[i]) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,unlockStop[%d] not in unlock", id, v.UnlockStop[i]))
			}
		}
		if v.NextPackage != 0 && s.temp1[v.NextPackage] == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,nextPackage not in stillShop", id))
		}
		if !enum.IsValidItemRefreshType(v.GiftRefresh) {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,giftRefresh not valid", id))
		}
		if v.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_NOT_IN_PRIVILEGE) {
			if GetVipCardCfg(v.GiftRefreshParam) == nil {
				return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,giftRefreshParam not valid", id))
			}
		}
		if v.FirstBuy != 0 && GetDropCfg(v.FirstBuy) == nil {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,FirstBuy[%d] not in drop", id, v.FirstBuy))
		}
		if v.Duration < 0 {
			return errors.New(fmt.Sprintf("[gameConfig] load stillShop error invalid ID:%d,duration <= 0", id))
		}
		if v.NextPackage != 0 {
			s.temp1[v.NextPackage].PrePackage = v.Id
		}
	}
	return nil
}

func (s *StillShopCfgLoader) apply() {
	stillShop.Store(s.temp1)
	stillShopItemType.Store(s.shopItemTypeToCfgMap)
	stillShopType.Store(s.shopTypeToCfgMap)
}

var stillShop atomic.Value
var stillShopItemType atomic.Value
var stillShopType atomic.Value

type StillShopCfg struct {
	// 序号
	Id int32 `json:"id"`
	// 活动id
	ActId int32 `json:"actId"`
	// 商店类型
	TypeId int32 `json:"typeId"`
	// 商店
	ShopType int32 `json:"shopType"`
	// 礼包id
	ProductId int32 `json:"productId"`
	// 自选掉落
	DropId []int32 `json:"dropId"`
	// 限购次数
	LimitBuy int32 `json:"limitBuy"`
	// 显示解锁
	UnlockShow []int32 `json:"unlockShow"`
	// 购买解锁
	UnlockBuy []int32 `json:"unlockBuy"`
	// 下一礼包
	NextPackage int32 `json:"nextPackage"`
	// 商品刷新规则
	GiftRefresh int32 `json:"giftRefresh"`
	// 商品刷新参数
	GiftRefreshParam int32 `json:"giftRefreshParam"`
	// 首次购买赠送
	FirstBuy int32 `json:"firstBuy"`
	// 是否给予钻石
	Diamond int32 `json:"diamond"`
	// 广告
	Adv int32 `json:"adv"`
	// 解锁后不显示
	UnlockStop []int32 `json:"unlockStop"`
	// 限购时间
	Duration int32 `json:"duration"`
	// 上一个礼包
	PrePackage int32
}

func GetStillShopCfg(id int32) *StillShopCfg {
	cfgMap := stillShop.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]*StillShopCfg)[id]
}

func GetAllCfgWithShopType(shopType int32) map[int32]*StillShopCfg {
	cfgMap := stillShopType.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]map[int32]*StillShopCfg)[shopType]
}

func GetAllCfgByShopItemType(shopItemType int32) map[int32]*StillShopCfg {
	cfgMap := stillShopItemType.Load()
	if cfgMap == nil {
		return nil
	}
	return cfgMap.(map[int32]map[int32]*StillShopCfg)[shopItemType]
}
