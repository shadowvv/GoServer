package model

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

type PlayerTokenShopEntity struct {
	UserId      int64 `gorm:"column:user_id;primaryKey"`
	ShopItemId  int32 `gorm:"column:shop_item_id;primaryKey"`
	BuyCount    int32 `gorm:"column:buy_count"`
	LastBuyTime int64 `gorm:"column:last_buy_time"`
	Discount    int32 `gorm:"column:discount"`
}

func (u *PlayerTokenShopEntity) TableName() string {
	return "player_token_shop"
}

type PlayerTokenShopModel struct {
	UserId           int64
	ShopItemEntities map[int32]*PlayerTokenShopEntity

	Changed map[int32]map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*PlayerTokenShopModel)(nil)

func (p *PlayerTokenShopModel) SaveModelToDB() {
	for itemId, changes := range p.Changed {
		if entity := p.ShopItemEntities[itemId]; entity != nil && len(changes) > 0 {
			easyDB.UpdatePlayerEntity(entity, changes, p.UserId)
		}
	}
	p.Changed = make(map[int32]map[string]interface{})
}

func (p *PlayerTokenShopModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		for _, entity := range p.ShopItemEntities {
			cfg := gameConfig.GetTokenShopItemCfg(entity.ShopItemId)
			if cfg == nil {
				continue
			}
			shopCfg := gameConfig.GetTokenShopMainCfg(cfg.ShopId)
			if shopCfg == nil {
				continue
			}
			if entity.Discount > 0 && len(cfg.Discount) == 0 {
				entity.Discount = 0
				if p.Changed[entity.ShopItemId] == nil {
					p.Changed[entity.ShopItemId] = make(map[string]interface{})
				}
				p.Changed[entity.ShopItemId]["discount"] = entity.Discount
				continue
			}
			if len(cfg.Discount) <= 0 {
				continue
			}
			if shopCfg.Refresh == int32(enum.ITEM_REFRESH_TYPE_DAY) {
				entity.Discount = cfg.RandomDiscount()
				if p.Changed[entity.ShopItemId] == nil {
					p.Changed[entity.ShopItemId] = make(map[string]interface{})
				}
				p.Changed[entity.ShopItemId]["discount"] = entity.Discount
			} else if shopCfg.Refresh == int32(enum.ITEM_REFRESH_TYPE_WEEK) && tool.GetNatureWeekDistance(lastTickTime, currentTime) > 0 {
				entity.Discount = cfg.RandomDiscount()
				if p.Changed[entity.ShopItemId] == nil {
					p.Changed[entity.ShopItemId] = make(map[string]interface{})
				}
				p.Changed[entity.ShopItemId]["discount"] = entity.Discount
			}
		}
	}
}

func (p *PlayerTokenShopModel) GetItem(id int32) *PlayerTokenShopEntity {
	return p.ShopItemEntities[id]
}

func (p *PlayerTokenShopModel) UnlockItem(id int32, item *gameConfig.TokenShopItemCfg) *PlayerTokenShopEntity {
	if p.ShopItemEntities[id] == nil {
		cfg := gameConfig.GetTokenShopItemCfg(id)
		if cfg == nil {
			return nil
		}
		entity := &PlayerTokenShopEntity{
			UserId:      p.UserId,
			ShopItemId:  id,
			BuyCount:    0,
			LastBuyTime: 0,
			Discount:    item.RandomDiscount(),
		}
		p.ShopItemEntities[id] = entity
		err := easyDB.CreatePlayerEntity(entity)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (p *PlayerTokenShopModel) RefreshItem(item *PlayerTokenShopEntity) {
	item.BuyCount = 0
	item.LastBuyTime = 0
	if p.Changed[item.ShopItemId] == nil {
		p.Changed[item.ShopItemId] = make(map[string]interface{})
	}

	p.Changed[item.ShopItemId]["buy_count"] = item.BuyCount
	p.Changed[item.ShopItemId]["last_buy_time"] = item.LastBuyTime
}

func (p *PlayerTokenShopModel) BuyItem(item *PlayerTokenShopEntity, num int32) {
	item.BuyCount += num
	item.LastBuyTime = tool.UnixNowMilli()

	if p.Changed[item.ShopItemId] == nil {
		p.Changed[item.ShopItemId] = make(map[string]interface{})
	}

	p.Changed[item.ShopItemId]["buy_count"] = item.BuyCount
	p.Changed[item.ShopItemId]["last_buy_time"] = item.LastBuyTime
}

func LoadPlayerTokenShop(userId int64) (*PlayerTokenShopModel, error) {
	shopItems, err := easyDB.GetPlayerEntitiesByWhere[PlayerTokenShopEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	items := make(map[int32]*PlayerTokenShopEntity)
	for _, entity := range shopItems {
		if entity == nil {
			continue
		}
		items[entity.ShopItemId] = entity
	}
	return &PlayerTokenShopModel{
		UserId:           userId,
		ShopItemEntities: items,
		Changed:          make(map[int32]map[string]interface{}),
	}, nil
}

func CreatePlayerTokenShop(userId int64) (*PlayerTokenShopModel, error) {
	return &PlayerTokenShopModel{
		UserId:           userId,
		ShopItemEntities: make(map[int32]*PlayerTokenShopEntity),
		Changed:          make(map[int32]map[string]interface{}),
	}, nil
}
