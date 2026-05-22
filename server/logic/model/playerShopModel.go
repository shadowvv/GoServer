package model

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"
)

type PlayerShopItemEntity struct {
	UserId      int64 `gorm:"column:user_id;primaryKey"`
	ShopItemId  int32 `gorm:"column:shop_item_id;primaryKey"`
	BuyCount    int32 `gorm:"column:buy_count;"`
	UnlockTime  int64 `gorm:"column:unlock_time"`
	LastBuyTime int64 `gorm:"column:last_buy_time"`
}

func (u *PlayerShopItemEntity) TableName() string {
	return "player_shop_item_data"
}

type PlayerPassEntity struct {
	UserId         int64 `gorm:"column:user_id;primaryKey"`
	ShopItemId     int32 `gorm:"column:shop_item_id;primaryKey"`
	AcceptCount    int32 `gorm:"column:accept_count;"`
	LastAcceptTime int64 `gorm:"column:last_accept_time"`
}

func (u *PlayerPassEntity) TableName() string {
	return "player_pass_data"
}

type PlayerShopModel struct {
	UserId       int64
	Player       *PlayerModel
	ItemEntities map[int32]*PlayerShopItemEntity
	Changed      map[int32]map[string]interface{}

	PassEntities map[int32]*PlayerPassEntity
	PassChanged  map[int32]map[string]interface{}

	PushedItemId map[int32]struct{}
}

var _ logicCommon.PlayerModelInterface = (*PlayerShopModel)(nil)

func (p *PlayerShopModel) SaveModelToDB() {
	if len(p.Changed) != 0 {
		for id, changes := range p.Changed {
			easyDB.UpdatePlayerEntity[PlayerShopItemEntity](p.ItemEntities[id], changes, p.UserId)
		}
		p.Changed = make(map[int32]map[string]interface{})
	}

	if len(p.PassChanged) != 0 {
		for id, changes := range p.PassChanged {
			easyDB.UpdatePlayerEntity[PlayerPassEntity](p.PassEntities[id], changes, p.UserId)
		}
		p.PassChanged = make(map[int32]map[string]interface{})
	}
}

func (p *PlayerShopModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		for _, shopItem := range p.ItemEntities {
			itemCfg := gameConfig.GetStillShopCfg(shopItem.ShopItemId)
			if itemCfg == nil {
				continue
			}
			if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_DAY) {
				if tool.GetNatureDayDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
					p.UpdateShopItemCount(shopItem.ShopItemId, 0)
				}
				if tool.GetNatureDayDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
					p.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
				}
				continue
			}
			if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_WEEK) {
				if tool.GetNatureWeekDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
					p.UpdateShopItemCount(shopItem.ShopItemId, 0)
				}
				if tool.GetNatureWeekDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
					p.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
				}
				continue
			}
			if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_MONTH) {
				if tool.GetNatureMonthDistance(tool.UnixNowMilli(), shopItem.LastBuyTime) > 0 {
					p.UpdateShopItemCount(shopItem.ShopItemId, 0)
				}
				if tool.GetNatureMonthDistance(tool.UnixNowMilli(), shopItem.UnlockTime) > 0 {
					p.UpdateShopItemUnlockTime(shopItem.ShopItemId, 0)
				}
				continue
			}
			if itemCfg.GiftRefresh == int32(enum.ITEM_REFRESH_TYPE_NOT_IN_PRIVILEGE) {
				if len(p.Player.VipCardModel.GetVipCardByItemId(itemCfg.GiftRefreshParam, tool.UnixNowMilli())) == 0 {
					p.UpdateShopItemCount(shopItem.ShopItemId, 0)
				}
				continue
			}
		}
	}

	pushPB := make([]*pb.ShopItemInfo, 0)
	passInfos := make([]*pb.PassItemInfo, 0)

	allShopItem := gameConfig.GetAllCfgByShopItemType(int32(enum.ShopItemTypeFirstCharge))
	for _, item := range allShopItem {
		if _, ok := p.PushedItemId[item.Id]; ok {
			continue
		}
		if !p.checkItemUnlock(item) {
			continue
		}

		leftTime := int64(0)
		buyCount := int32(0)
		unlockTime := int64(0)
		isFirstBuy := false
		itemEntity := p.GetShopItemInfo(item.Id)
		if itemEntity != nil {
			unlockTime = itemEntity.UnlockTime
			if item.Duration > 0 {
				leftTime = int64(item.Duration)*1000 - (currentTime - itemEntity.UnlockTime)
			}
			passEntity := p.GetShopPassInfo(item.Id)
			if passEntity != nil {
				passCfg := gameConfig.GetWeeklyPassCfg(item.Id)
				if passCfg == nil {
					p.PushedItemId[item.Id] = struct{}{}
					continue
				}
				if passEntity.AcceptCount < passCfg.ValidityPeriod {
					day := tool.GetNatureDayDistance(tool.UnixNowMilli(), passEntity.LastAcceptTime)
					passInfos = append(passInfos, &pb.PassItemInfo{
						ItemId:    item.Id,
						CanAccept: day > 0,
						LeftTimes: passCfg.ValidityPeriod - passEntity.AcceptCount,
					})
				} else {
					p.PushedItemId[item.Id] = struct{}{}
					continue
				}
			} else if leftTime <= 0 {
				p.PushedItemId[item.Id] = struct{}{}
				continue
			}
			buyCount = itemEntity.BuyCount
			if item.LimitBuy != 0 && item.LimitBuy <= buyCount {
				p.PushedItemId[item.Id] = struct{}{}
				continue
			}
			isFirstBuy = itemEntity.LastBuyTime == 0
		}
		pushPB = append(pushPB, &pb.ShopItemInfo{
			ItemId:     item.Id,
			BuyCount:   buyCount,
			LeftTime:   leftTime,
			UnlockTime: unlockTime,
			IsFirstBuy: isFirstBuy,
			Status:     0,
		})
		p.PushedItemId[item.Id] = struct{}{}
	}

	airDropItems := gameConfig.GetAllCfgByShopItemType(int32(enum.ShopItemTypeAirdrop))
	for _, item := range airDropItems {
		if _, ok := p.PushedItemId[item.Id]; ok {
			continue
		}
		if !p.checkItemUnlock(item) {
			continue
		}
		leftTime := int64(0)
		buyCount := int32(0)
		unlockTime := int64(0)
		isFirstBuy := false
		itemEntity := p.GetShopItemInfo(item.Id)
		if itemEntity != nil {
			unlockTime = itemEntity.UnlockTime
			if item.Duration > 0 {
				leftTime = int64(item.Duration)*1000 - (currentTime - itemEntity.UnlockTime)
			}
			if leftTime <= 0 {
				continue
			}
			buyCount = itemEntity.BuyCount
			if item.LimitBuy != 0 && item.LimitBuy <= buyCount {
				p.PushedItemId[item.Id] = struct{}{}
				continue
			}
			isFirstBuy = itemEntity.LastBuyTime == 0
		}
		pushPB = append(pushPB, &pb.ShopItemInfo{
			ItemId:     item.Id,
			BuyCount:   buyCount,
			LeftTime:   leftTime,
			UnlockTime: unlockTime,
			IsFirstBuy: isFirstBuy,
			Status:     0,
		})
		p.PushedItemId[item.Id] = struct{}{}
	}

	if len(pushPB) > 0 || len(passInfos) > 0 {
		if senderMsg {
			messageSender.SendMessage(p.Player, pb.MESSAGE_ID_PUSH_SHOP_ITEM_POP, &pb.PushShopItemPop{
				ItemInfos:       pushPB,
				FirstChargeInfo: passInfos,
			})
		}
	}
}

func (p *PlayerShopModel) checkItemUnlock(item *gameConfig.StillShopCfg) bool {
	if item.ActId != 0 {
		open, _ := p.Player.PlayerActivityModel.CheckActivityOpen(item.ActId)
		if !open {
			return false
		}
	}

	if len(item.UnlockStop) > 0 {
		unlockStop := true
		for _, unlock := range item.UnlockStop {
			if !unlockService.CheckUnlock(unlock, p.Player) {
				unlockStop = false
				break
			}
		}
		if unlockStop {
			return false
		}
	}

	for _, unlock := range item.UnlockShow {
		if !unlockService.CheckUnlock(unlock, p.Player) {
			return false
		}
	}

	// 检测是否可购买
	for _, unlock := range item.UnlockBuy {
		if !unlockService.CheckUnlock(unlock, p.Player) {
			return false
		}
	}

	// 限时道具需要先解锁，已保存解锁时间
	if (p.GetShopItemInfo(item.Id) == nil || p.GetShopItemInfo(item.Id).UnlockTime == 0) && item.Duration > 0 {
		err := p.UnlockShopItem(item.Id, int64(item.Duration*1000))
		if err != nil {
			platformLogger.ErrorWithUser("unlock shop item error", p.Player, err)
			return false
		}
	}
	return true
}

func (p *PlayerShopModel) GetShopItemInfo(shopItemId int32) *PlayerShopItemEntity {
	return p.ItemEntities[shopItemId]
}

func (p *PlayerShopModel) GetShopPassInfo(shopItemId int32) *PlayerPassEntity {
	return p.PassEntities[shopItemId]
}

func (p *PlayerShopModel) UpdateShopItemCount(shopItemId int32, count int32) {
	entity := p.ItemEntities[shopItemId]
	if entity == nil {
		return
	}
	entity.BuyCount = count
	if p.Changed[shopItemId] == nil {
		p.Changed[shopItemId] = make(map[string]interface{})
	}
	p.Changed[shopItemId]["buy_count"] = count
	if count != 0 {
		entity.LastBuyTime = tool.UnixNowMilli()
		p.Changed[shopItemId]["last_buy_time"] = entity.LastBuyTime
	}
	return
}

func (p *PlayerShopModel) UnlockShopItem(shopItemId int32, duration int64) error {
	entity := p.ItemEntities[shopItemId]
	if entity == nil {
		entity = &PlayerShopItemEntity{
			UserId:      p.UserId,
			ShopItemId:  shopItemId,
			BuyCount:    0,
			UnlockTime:  tool.UnixNowMilli(),
			LastBuyTime: 0,
		}
		p.ItemEntities[shopItemId] = entity
		err := easyDB.CreatePlayerEntity[PlayerShopItemEntity](entity)
		if err != nil {
			return err
		}
	} else {
		currentTime := tool.UnixNowMilli()
		if currentTime-entity.UnlockTime > duration {
			if p.Changed[shopItemId] == nil {
				p.Changed[shopItemId] = make(map[string]interface{})
			}
			entity.UnlockTime = tool.UnixNowMilli()
			p.Changed[shopItemId]["unlock_time"] = entity.UnlockTime
		}
	}
	return nil
}

func (p *PlayerShopModel) UpdateShopPassData(passEntity *PlayerPassEntity, currentTime int64, acceptCount int32) {
	passEntity.LastAcceptTime = currentTime
	passEntity.AcceptCount = acceptCount

	if p.PassChanged[passEntity.ShopItemId] == nil {
		p.PassChanged[passEntity.ShopItemId] = make(map[string]interface{})
	}
	p.PassChanged[passEntity.ShopItemId]["last_accept_time"] = currentTime
	p.PassChanged[passEntity.ShopItemId]["accept_count"] = acceptCount
}

func (p *PlayerShopModel) ResetPassInfo(shopItemId int32) {
	passEntity := p.PassEntities[shopItemId]
	if passEntity == nil {
		passEntity = &PlayerPassEntity{
			UserId:         p.UserId,
			ShopItemId:     shopItemId,
			AcceptCount:    0,
			LastAcceptTime: 0,
		}
		p.PassEntities[shopItemId] = passEntity
		err := easyDB.CreatePlayerEntity[PlayerPassEntity](passEntity)
		if err != nil {
			return
		}
	} else {
		passEntity.AcceptCount = 0
		passEntity.LastAcceptTime = 0

		if p.PassChanged[passEntity.ShopItemId] == nil {
			p.PassChanged[passEntity.ShopItemId] = make(map[string]interface{})
		}
		p.PassChanged[passEntity.ShopItemId]["accept_count"] = 0
		p.PassChanged[passEntity.ShopItemId]["last_accept_time"] = 0
	}
}

func (p *PlayerShopModel) UpdateShopItemUnlockTime(shopItemId int32, unlockTime int64) {
	entity := p.ItemEntities[shopItemId]
	if entity == nil {
		return
	}
	entity.UnlockTime = unlockTime
	if p.Changed[shopItemId] == nil {
		p.Changed[shopItemId] = make(map[string]interface{})
	}
	entity.UnlockTime = tool.UnixNowMilli()
	p.Changed[shopItemId]["unlock_time"] = entity.UnlockTime
}

func NewPlayerShopModel(userId int64, player *PlayerModel, entity map[int32]*PlayerShopItemEntity, passEntity map[int32]*PlayerPassEntity) *PlayerShopModel {
	return &PlayerShopModel{
		UserId:       userId,
		Player:       player,
		ItemEntities: entity,
		PassEntities: passEntity,
		Changed:      make(map[int32]map[string]interface{}),
		PassChanged:  make(map[int32]map[string]interface{}),

		PushedItemId: make(map[int32]struct{}),
	}
}

func LoadPlayerShopModel(userId int64, player *PlayerModel) (*PlayerShopModel, error) {
	entity, err := easyDB.GetPlayerEntitiesByWhere[PlayerShopItemEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	itemMap := make(map[int32]*PlayerShopItemEntity)
	for _, e := range entity {
		itemMap[e.ShopItemId] = e
	}

	passEntity, err := easyDB.GetPlayerEntitiesByWhere[PlayerPassEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return nil, err
	}
	passMap := make(map[int32]*PlayerPassEntity)
	for _, e := range passEntity {
		passMap[e.ShopItemId] = e
	}
	return NewPlayerShopModel(userId, player, itemMap, passMap), nil
}
