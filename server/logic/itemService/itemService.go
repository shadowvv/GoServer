package itemService

import (
	"errors"
	"time"

	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"

	"github.com/drop/GoServer/server/logic/platform/eventService"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/adChest"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/inventory"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/vipCard"
)

var service = &ItemService{}
var messageSender logicCommon.MessageSenderInterface
var mainInventoryService inventory.InventoryServiceInterface
var equipInventoryService logicCommon.EquipmentInterface
var eventBus *eventService.EventBus
var passService logicCommon.PassServiceInterface

type ItemService struct {
}

var _ logicCommon.ItemService = (*ItemService)(nil)

func (i *ItemService) AddItem(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	if item == nil {
		return nil
	}
	itemPb := make([]*pb.ItemBasicInfo, 0)

	itemCfg := gameConfig.GetItemCfg(item.ID)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
		return errors.New("item config is nil")
	}
	ReportItemChange(player, item, reason, 0, 0, true)
	itemPb = append(itemPb, &pb.ItemBasicInfo{
		ItemId: item.ID,
		Count:  item.Num,
	})
	if itemCfg.IsAutoUse == 1 {
		return i.autoUseItem(player, item.ID, item.Num)
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
		equipLevel := itemCfg.Level
		if itemCfg.Level == 0 {
			equipLevel = player.GetLevel()
			if equipLevel > gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0] {
				equipLevel = gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0]
			}
		}
		_, err := equipInventoryService.AddEquipment(player.GetUserId(), itemCfg.TargetId, equipLevel)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_VIP_CARD) {
		// 特权卡：不落背包数量（避免把"小时数"当成库存），直接走特权卡子系统
		// item.Num 单位：小时
		err := vipCard.Service.AddVipCardFromItem(player, itemCfg.TargetId, item.Num)
		if err != nil {
			platformLogger.ErrorWithUser("AddVipCard failed", player, err)
			return err
		}
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_PASS) {
		// 通行证进度道具（ShowGroup 21）：不落背包，直接添加到通行证进度
		// item.Num 代表数量，itemCfg.TargetId 代表通行证的 id
		err := passService.AddPassProgressFromItem(player, itemCfg.TargetId, item.Num)
		if err != nil {
			platformLogger.ErrorWithUser("AddPassProgress failed", player, err)
			return err
		}
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_PASS_VIP) {
		// 通行证VIP等级道具（ShowGroup 22）：不落背包，直接设置通行证VIP等级
		//itemCfg.TargetId 代表通行证的 id，itemCfg.Level 代表档位（0=免费, 1=付费档位1, 2=付费档位2）
		err := passService.SetPassVipLevelFromItem(player, itemCfg.TargetId, itemCfg.Level)
		if err != nil {
			platformLogger.ErrorWithUser("SetPassVipLevel failed", player, err)
			return err
		}
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_AD_CHEST) {
		// 广告宝箱（ShowGroup 25）：不落背包，发放到广告宝箱系统并推送
		for n := int64(0); n < item.Num; n++ {
			_, pushMsg, err := adChest.Service.GrantAdChest(player, item.ID)
			if err != nil {
				platformLogger.ErrorWithUser("GrantAdChest failed", player, err)
				break
			}
			if pushMsg != nil {
				messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_AD_CHEST_NEW, pushMsg)
			}
		}
	} else if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_COLLECTION) {
		spidNum := int64(gameConfig.GetConstantCfg(gameConfig.CONSTANT_collectionExchangeSpid).Value[0])
		spidId := itemCfg.TargetId
		spidCfg := gameConfig.GetItemCfg(spidId)
		if spidCfg == nil {
			platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
			return errors.New("item config is nil")
		}
		collectModel := player.(*model.PlayerModel).CollectionModel
		if collectModel.CollectionEntity[spidCfg.TargetId] == nil {
			collectCfg := gameConfig.GetCollectionMainCfgByAtrAndLevel(spidCfg.TargetId, 1)
			if collectCfg == nil {
				platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
				return errors.New("item config is nil")
			}
			err := collectModel.AddCollection(&model.CollectionEntity{
				UserId:                player.GetUserId(),
				CollectLevel:          1,
				CollectionAttribution: collectCfg.Belonging,
				CollectId:             collectCfg.Id,
			})
			if err != nil {
				platformLogger.ErrorWithUser("AddItems failed config = nil", player, err)
				return err
			}
		} else {
			_, err := mainInventoryService.AddItem(player.GetUserId(), spidId, item.Num*spidNum)
			if err != nil {
				platformLogger.ErrorWithUser("AddItems failed", player, err)
				return err
			}
		}
	} else {
		_, err := mainInventoryService.AddItem(player.GetUserId(), item.ID, item.Num)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}

	eventBus.SubmitItemCollectEvent(player.GetUserId(), []*gameConfig.ItemConfig{item})
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:        i.getChangeType(reason),
		AddItemList: itemPb,
		Source:      int32(reason),
	})
	return nil
}

func (i *ItemService) AddItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return AddItemsWithPayType(player, items, reason, enum.RECHARGE_TYPE_NONE)
}

func (i *ItemService) AddItemsWithPayType(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason, payType enum.RechargeType) error {
	if items == nil || len(items) == 0 {
		return nil
	}
	itemInventoryMap := make(map[enum.InventoryType]map[int32]int64)
	itemPb := make([]*pb.ItemBasicInfo, 0)

	for _, item := range items {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
			continue
		}
		// 广告宝箱不加入 itemPb，由 PushAdChestNew 单独推送，藏品及碎片需要特判
		if itemCfg.ShowGroup != int32(enum.ITEM_TYPE_AD_CHEST) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_COLLECTION) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_COLLECTION_FRAGMENT) {
			itemPb = append(itemPb, &pb.ItemBasicInfo{
				ItemId: item.ID,
				Count:  item.Num,
			})
		}
		if itemCfg.IsAutoUse == 1 {
			ReportItemChange(player, item, reason, 0, 0, true)
			i.autoUseItem(player, item.ID, item.Num)
			continue
		}
		// 特权卡：不落背包数量，直接走特权卡子系统（item.Num 单位：小时）
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_VIP_CARD) {
			ReportItemChange(player, item, reason, 0, 0, true)
			if err := vipCard.Service.AddVipCardFromItem(player, itemCfg.TargetId, item.Num); err != nil {
				platformLogger.ErrorWithUser("AddVipCard failed", player, err)
				return err
			}
			continue
		}
		// 通行证进度道具（ShowGroup 21）
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_PASS) {
			ReportItemChange(player, item, reason, 0, 0, true)
			if err := passService.AddPassProgressFromItem(player, itemCfg.TargetId, item.Num); err != nil {
				platformLogger.ErrorWithUser("AddPassProgress failed", player, err)
				return err
			}
			continue
		}
		// 通行证VIP等级道具（ShowGroup 22）：添加VIP档位
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_PASS_VIP) {
			ReportItemChange(player, item, reason, 0, 0, true)
			if err := passService.SetPassVipLevelFromItem(player, itemCfg.TargetId, itemCfg.Level); err != nil {
				platformLogger.ErrorWithUser("SetPassVipLevel failed", player, err)
				return err
			}
			continue
		}
		// 广告宝箱（ShowGroup 25）：不落背包，发放到广告宝箱系统并推送
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_AD_CHEST) {
			ReportItemChange(player, item, reason, 0, 0, true)
			for n := int64(0); n < item.Num; n++ {
				_, pushMsg, err := adChest.Service.GrantAdChest(player, item.ID)
				if err != nil {
					platformLogger.ErrorWithUser("GrantAdChest failed", player, err)
					break
				}
				if pushMsg != nil {
					messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_AD_CHEST_NEW, pushMsg)
				}
				// 已达获取上限时 pushMsg 为 nil，停止发放
				if pushMsg == nil {
					break
				}
			}
			continue
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_COLLECTION) {
			spidNum := int64(gameConfig.GetConstantCfg(gameConfig.CONSTANT_collectionExchangeSpid).Value[0])
			spidId := itemCfg.TargetId
			spidCfg := gameConfig.GetItemCfg(spidId)
			if spidCfg == nil {
				platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
				continue
			}
			collectModel := player.(*model.PlayerModel).CollectionModel
			if collectModel.CollectionEntity[spidCfg.TargetId] == nil {
				collectCfg := gameConfig.GetCollectionMainCfgByAtrAndLevel(spidCfg.TargetId, 1)
				if collectCfg == nil {
					platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
					continue
				}
				err := collectModel.AddCollection(&model.CollectionEntity{
					UserId:                player.GetUserId(),
					CollectLevel:          1,
					CollectionAttribution: collectCfg.Belonging,
					CollectId:             collectCfg.Id,
				})
				if err != nil {
					platformLogger.ErrorWithUser("AddItems failed config = nil", player, err)
					continue
				}
				ReportItemChange(player, item, reason, 0, 0, true)
			} else {
				if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
					itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
				}
				if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][spidId]; ok {
					itemInventoryMap[enum.INVENTORY_TYPE_MAIN][spidId] += item.Num * spidNum
				} else {
					itemInventoryMap[enum.INVENTORY_TYPE_MAIN][spidId] = item.Num * spidNum
				}
				ReportItemChange(player, &gameConfig.ItemConfig{ID: spidId, Num: item.Num * spidNum}, reason, 0, 0, true)
				continue
			}
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
			ReportItemChange(player, item, reason, 0, 0, true)
			if itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID] = item.Num
			}
		} else {
			ReportItemChange(player, item, reason, 0, 0, true)
			if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] = item.Num
			}
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] != nil {
		_, err := mainInventoryService.AddItems(player.GetUserId(), itemInventoryMap[enum.INVENTORY_TYPE_MAIN])
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] != nil {
		for itemId, num := range itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] {
			itemCfg := gameConfig.GetItemCfg(itemId)
			if itemCfg == nil {
				continue
			}
			equipLevel := itemCfg.Level
			if itemCfg.Level == 0 {
				equipLevel = player.GetLevel()
				if equipLevel > gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0] {
					equipLevel = gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0]
				}
			}
			for i := 0; i < int(num); i++ {
				_, err := equipInventoryService.AddEquipment(player.GetUserId(), itemCfg.TargetId, equipLevel)
				if err != nil {
					platformLogger.ErrorWithUser("AddItems failed", player, err)
					continue
				}
			}
		}
	}
	eventBus.SubmitItemCollectEvent(player.GetUserId(), items)
	isPay := 0
	if payType == enum.RECHARGE_TYPE_NORMAL {
		isPay = 1
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:        i.getChangeType(reason),
		AddItemList: itemPb,
		Source:      int32(reason),
		IsPay:       int32(isPay),
	})
	return nil
}

func (i *ItemService) CheckItemCount(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig) (bool, error) {
	if item == nil {
		return false, nil
	}
	itemCfg := gameConfig.GetItemCfg(item.ID)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("CheckItemCount failed config = nil", player, nil)
		return false, errors.New("item config is nil")
	}
	if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
		platformLogger.ErrorWithUser("CheckItemCount failed", player, nil)
		return false, errors.New("item config is equip")
	} else {
		return mainInventoryService.HasItem(player.GetUserId(), item.ID, item.Num)
	}
}

func (i *ItemService) CheckItemsCountWithMap(player logicCommon.PlayerInterface, items map[int32]int64) (bool, error) {
	if items == nil || len(items) == 0 {
		return true, nil
	}
	for Id, num := range items {
		itemCfg := gameConfig.GetItemCfg(Id)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("CheckItemsCount failed config = nil", player, nil)
			return false, errors.New("item config is nil")
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
			platformLogger.ErrorWithUser("CheckItemsCount failed", player, nil)
			return false, errors.New("item config is equip")
		} else {
			has, err := mainInventoryService.HasItem(player.GetUserId(), Id, num)
			if !has || err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (i *ItemService) CheckItemsCount(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig) (bool, error) {
	if items == nil || len(items) == 0 {
		return true, nil
	}
	for _, item := range items {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("CheckItemsCount failed config = nil", player, nil)
			return false, errors.New("item config is nil")
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
			platformLogger.ErrorWithUser("CheckItemsCount failed", player, nil)
			return false, errors.New("item config is equip")
		} else {
			has, err := mainInventoryService.HasItem(player.GetUserId(), item.ID, item.Num)
			if !has || err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (i *ItemService) RemoveItem(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	if item == nil {
		return nil
	}
	itemPb := make([]*pb.ItemBasicInfo, 0)

	itemCfg := gameConfig.GetItemCfg(item.ID)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
		return errors.New("item config is nil")
	}
	itemPb = append(itemPb, &pb.ItemBasicInfo{
		ItemId: item.ID,
		Count:  item.Num,
	})
	if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_LOOP_BOX) {
		ReportItemChange(player, item, reason, 0, 0, false)
		platformLogger.ErrorWithUser("RemoveItems failed", player, nil)
	} else {
		ReportItemChange(player, item, reason, 0, 0, false)
		_, err := mainInventoryService.RemoveItem(player.GetUserId(), item.ID, item.Num)
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:           i.getChangeType(reason),
		RemoveItemList: itemPb,
		Source:         int32(reason),
	})
	return nil
}

func (i *ItemService) RemoveItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	if items == nil || len(items) == 0 {
		return nil
	}
	itemInventoryMap := make(map[enum.InventoryType]map[int32]int64)
	itemPb := make([]*pb.ItemBasicInfo, 0)

	for _, item := range items {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
			continue
		}
		itemPb = append(itemPb, &pb.ItemBasicInfo{
			ItemId: item.ID,
			Count:  item.Num,
		})
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_LOOP_BOX) {
			ReportItemChange(player, item, reason, 0, 0, false)
			platformLogger.ErrorWithUser("RemoveItems failed", player, nil)
		} else {
			ReportItemChange(player, item, reason, 0, 0, false)
			if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] = item.Num
			}
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] != nil {
		_, err := mainInventoryService.RemoveItems(player.GetUserId(), itemInventoryMap[enum.INVENTORY_TYPE_MAIN])
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:           i.getChangeType(reason),
		RemoveItemList: itemPb,
		Source:         int32(reason),
	})
	return nil
}

func (i *ItemService) RemoveItemsWithMap(player logicCommon.PlayerInterface, items map[int32]int64, reason enum.ItemChangeReason) error {
	if items == nil || len(items) == 0 {
		return nil
	}
	itemInventoryMap := make(map[enum.InventoryType]map[int32]int64)
	itemPb := make([]*pb.ItemBasicInfo, 0)

	for id, num := range items {
		itemCfg := gameConfig.GetItemCfg(id)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
			continue
		}
		itemPb = append(itemPb, &pb.ItemBasicInfo{
			ItemId: id,
			Count:  num,
		})
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_LOOP_BOX) {
			ReportItemChange(player, &gameConfig.ItemConfig{ID: id, Num: num}, reason, 0, 0, false)
			platformLogger.ErrorWithUser("RemoveItems failed", player, nil)
		} else {
			ReportItemChange(player, &gameConfig.ItemConfig{ID: id, Num: num}, reason, 0, 0, false)
			if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][id]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][id] += num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][id] = num
			}
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] != nil {
		_, err := mainInventoryService.RemoveItems(player.GetUserId(), itemInventoryMap[enum.INVENTORY_TYPE_MAIN])
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:           i.getChangeType(reason),
		RemoveItemList: itemPb,
		Source:         int32(reason),
	})
	return nil
}

func (i *ItemService) getChangeType(reason enum.ItemChangeReason) pb.ITEM_CHANGE_TYPE {
	switch reason {
	case enum.ITEM_CHANGE_REASON_LOTTERY:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_LOTTERY
	case enum.ITEM_CHANGE_REASON_MONSTER_DROP:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_MONSTER_DROP
	case enum.ITEM_CHANGE_REASON_CASH_SHOP_BUY:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_CASH_SHOP
	case enum.ITEM_CHANGE_REASON_WEEK_PASS:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_CASH_SHOP
	case enum.ITEM_CHANGE_REASON_MAIL_ATTACHMENT:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_CASH_SHOP
	case enum.ITEM_CHANGE_REASON_IDLE_REWARD:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_IDLE
	case enum.ITEM_CHANGE_REASON_IDLE_QUICK_CLAIM:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_IDLE
	case enum.ITEM_CHANGE_REASON_HERO_ALBUM_REWARD:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_HERO_ALBUM_REWARD
	case enum.ITEM_CHANGE_REASON_VIP_PRIVILEGE_REWARD:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_VIP_CARD
	case enum.ITEM_CHANGE_REASON_TOWER_STAGE_REWARD, enum.ITEM_CHANGE_REASON_ARENA_LOSE, enum.ITEM_CHANGE_REASON_ARENA_WIN, enum.ITEM_CHANGE_REASON_ADVENTURE_REWARD, enum.ITEM_CHANGE_REASON_GLORY_ARENA_WIN, enum.ITEM_CHANGE_REASON_GLORY_ARENA_LOSE:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_DUNGEON
	case enum.ITEM_CHANGE_REASON_DAILY_TASK:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_DAILY_TASK
	case enum.ITEM_CHANGE_REASON_RECOVERY_STAMINA, enum.ITEM_CHANGE_REASON_CLIAM_EXPEDITION_REWARD, enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE_FILED, enum.ITEM_CHANGE_REASON_CREATE_ALLIANCE_NAME_FILED:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_NO_POP
	case enum.ITEM_CHANGE_REASON_PET_RECRUIT:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_NO_POP
	case enum.ITEM_CHANGE_REASON_CITY_LUMBER_COLLECT:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_LUMBERYARD
	case enum.ITEM_CHANGE_REASON_TURN_TABLE_REWARD:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_TURN_TABLE
	default:
		return pb.ITEM_CHANGE_TYPE_ITEM_CHANGE_TYPE_COMMON
	}
}

func (i *ItemService) autoUseItem(player logicCommon.PlayerInterface, itemId int32, num int64) error {
	itemCfg := gameConfig.GetItemCfg(itemId)
	if itemCfg == nil || itemCfg.IsAutoUse == 0 {
		return nil
	}
	switch itemCfg.ShowGroup {
	case int32(enum.ITEM_TYPE_HERO):
		for i := int64(0); i < num; i++ {
			_, err := hero.AddHeroDetail(player.(*model.PlayerModel), int64(itemCfg.TargetId))
			if heroCfg := gameConfig.GetHeroBaseCfg(itemCfg.TargetId); heroCfg != nil {
				eventBus.SubmitHeroStarUpEvent(player.GetUserId(), itemCfg.TargetId, heroCfg.HeroStar)
			}
			if err != nil {
				platformLogger.ErrorWithUser("AddHeroDetail failed", player, err)
				return err
			}
		}
	case int32(enum.ITEM_TYPE_PET):
		// 自动使用宠物卡：根据 targetId 生成对应宠物
		p, ok := player.(*model.PlayerModel)
		if !ok || p == nil {
			platformLogger.ErrorWithUser("autoUseItem pet: player cast failed", player, nil)
			return errors.New("player cast failed")
		}
		for i := int64(0); i < num; i++ {
			ent, err := pet.ObtainPetByCard(p, itemCfg.Id)
			if err != nil {
				platformLogger.ErrorWithUser("ObtainPetByCard failed", player, err)
				continue
			}
			messageSender.SendMessage(p, pb.MESSAGE_ID_PUSH_PET_DETAIL, &pb.PushPetDetail{
				AddPetList: []*pb.PetDetailInfo{pet.BuildPetDetailInfo(p, ent)},
			})
		}
	case int32(enum.ITEM_TYPE_LOOP_BOX):
		player.(*model.PlayerModel).LoopBoxModel.AddLoopBox(itemCfg.TargetId, int32(num))
	case int32(enum.ITEM_TYPE_ACCESSORY):
		err := player.(*model.PlayerModel).AccessoryModel.AddAccessory(itemCfg.TargetId, int32(num))
		if err != nil {
			platformLogger.ErrorWithUser("AddAccessory failed", player, err)
			return err
		}
	case int32(enum.ITEM_TYPE_PLAYER_EXP):
		// TODO:玩家不在有经验和等级，走主城等级
		//playerModel, ok := player.(*model.PlayerModel)
		//if !ok {
		//	platformLogger.ErrorWithUser( "AddExp failed", player, nil)
		//}
		//err := playerModel.User.AddExp(num)
		//if err != nil {
		//	platformLogger.ErrorWithUser( "AddExp failed", player, err)
		//}
	case int32(enum.ITEM_TYPE_VIP_CARD):
		// item.Num 单位：小时
		if err := vipCard.Service.AddVipCardFromItem(player, itemCfg.TargetId, num); err != nil {
			platformLogger.ErrorWithUser("AddVipCard failed", player, err)
			return err
		}
	case int32(enum.ITEM_TYPE_PASS):
		// 通行证进度道具
		itemCfg := gameConfig.GetItemCfg(itemId)
		if itemCfg != nil {
			if err := passService.AddPassProgressFromItem(player, itemCfg.TargetId, num); err != nil {
				platformLogger.ErrorWithUser("AddPassProgress failed", player, err)
				return err
			}
		}
	case int32(enum.ITEM_TYPE_PASS_VIP):
		// 通行证VIP等级道具
		itemCfg := gameConfig.GetItemCfg(itemId)
		if itemCfg != nil {
			if err := passService.SetPassVipLevelFromItem(player, itemCfg.TargetId, itemCfg.Level); err != nil {
				platformLogger.ErrorWithUser("SetPassVipLevel failed", player, err)
				return err
			}
		}
	case int32(enum.ITEM_TYPE_APPEARANCE):
		endTime := int64(0)
		if itemCfg.TargetId2 != 0 {
			endTime = tool.UnixNowMilli() + int64(itemCfg.TargetId2)*int64(time.Hour)
		}
		if err := player.(*model.PlayerModel).AppearanceModel.AddAppearance(itemCfg.TargetId, endTime); err != nil {
			platformLogger.ErrorWithUser("AddAppearance failed", player, err)
			return err
		}
		res := make([]*pb.AvatarDetail, 0)
		res = append(res, &pb.AvatarDetail{
			Id:      itemCfg.TargetId,
			EndTime: endTime,
			IsWear:  false,
		})
		messageSender.SendMessage(player.(*model.PlayerModel), pb.MESSAGE_ID_PUSH_AVATAR_DETAIL, &pb.PushAvatarDetail{
			AvatarDetail: res,
		})
	}
	return nil
}

func (i *ItemService) ExchangeItem(player logicCommon.PlayerInterface, costItems []*gameConfig.ItemConfig, addItems []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	itemInventoryMap := make(map[enum.InventoryType]map[int32]int64)
	removeItemPb := make([]*pb.ItemBasicInfo, 0)

	for _, item := range costItems {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("RemoveItems failed config = nil", player, nil)
			continue
		}
		removeItemPb = append(removeItemPb, &pb.ItemBasicInfo{
			ItemId: item.ID,
			Count:  item.Num,
		})
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) || itemCfg.ShowGroup == int32(enum.ITEM_TYPE_LOOP_BOX) {
			ReportItemChange(player, item, reason, 0, 0, false)
			platformLogger.ErrorWithUser("RemoveItems failed", player, nil)
		} else {
			ReportItemChange(player, item, reason, 0, 0, false)
			if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] = item.Num
			}
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] != nil {
		_, err := mainInventoryService.RemoveItems(player.GetUserId(), itemInventoryMap[enum.INVENTORY_TYPE_MAIN])
		if err != nil {
			platformLogger.ErrorWithUser("RemoveItems failed", player, err)
			return err
		}
	}

	clear(itemInventoryMap)

	AddItemPb := make([]*pb.ItemBasicInfo, 0)
	for _, item := range addItems {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
			continue
		}
		AddItemPb = append(AddItemPb, &pb.ItemBasicInfo{
			ItemId: item.ID,
			Count:  item.Num,
		})
		if itemCfg.IsAutoUse == 1 {
			ReportItemChange(player, item, reason, 0, 0, true)
			err := i.autoUseItem(player, item.ID, item.Num)
			if err != nil {
				platformLogger.ErrorWithUser("AddItems failed", player, err)
				return err
			}
			continue
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
			ReportItemChange(player, item, reason, 0, 0, true)
			if itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_EQUIP][item.ID] = item.Num
			}
		} else {
			ReportItemChange(player, item, reason, 0, 0, true)
			if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] == nil {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN] = make(map[int32]int64)
			}
			if _, ok := itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID]; ok {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] += item.Num
			} else {
				itemInventoryMap[enum.INVENTORY_TYPE_MAIN][item.ID] = item.Num
			}
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_MAIN] != nil {
		_, err := mainInventoryService.AddItems(player.GetUserId(), itemInventoryMap[enum.INVENTORY_TYPE_MAIN])
		if err != nil {
			platformLogger.ErrorWithUser("AddItems failed", player, err)
			return err
		}
	}
	if itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] != nil {
		for itemId, _ := range itemInventoryMap[enum.INVENTORY_TYPE_EQUIP] {
			itemCfg := gameConfig.GetItemCfg(itemId)
			if itemCfg == nil {
				platformLogger.ErrorWithUser("AddItems failed config = nil", player, nil)
				continue
			}
			equipLevel := player.GetLevel()
			if equipLevel > gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0] {
				equipLevel = gameConfig.GetConstantCfg(gameConfig.CONSTANT_maxEquipmentFromMain).Value[0]
			}
			if reason == enum.ITEM_CHANGE_REASON_FORGE_EQUIPMENT {
				equipLevel = itemCfg.Level
			}
			_, err := equipInventoryService.AddEquipment(player.GetUserId(), itemCfg.TargetId, equipLevel)
			if err != nil {
				platformLogger.ErrorWithUser("AddItems failed", player, err)
				continue
			}
		}
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_ITEM_CHANGE, &pb.PushItemChange{
		Type:           i.getChangeType(reason),
		RemoveItemList: removeItemPb,
		AddItemList:    AddItemPb,
		Source:         int32(reason),
	})
	return nil
}

func (i *ItemService) ResetItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	if items == nil || len(items) == 0 {
		return nil
	}
	addItems := make([]*gameConfig.ItemConfig, 0)
	for _, item := range items {
		itemCfg := gameConfig.GetItemCfg(item.ID)
		if itemCfg == nil {
			platformLogger.ErrorWithUser("ResetItems failed config = nil", player, nil)
			return errors.New("item config is nil")
		}
		if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
			platformLogger.ErrorWithUser("ResetItems failed", player, nil)
			return errors.New("item config is equip")
		} else {
			count, err := mainInventoryService.GetItemCount(player.GetUserId(), item.ID)
			if err != nil {
				platformLogger.ErrorWithUser("ResetItem failed", player, err)
				return err
			}
			if count >= item.Num {
				continue
			} else {
				addItems = append(addItems, &gameConfig.ItemConfig{
					ID:  item.ID,
					Num: item.Num - count,
				})
			}
		}
	}
	return i.AddItems(player, addItems, reason)
}

func (i *ItemService) ResetItem(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	if item == nil {
		return nil
	}
	itemCfg := gameConfig.GetItemCfg(item.ID)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("ResetItem failed config = nil", player, nil)
		return errors.New("item config is nil")
	}
	if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
		platformLogger.ErrorWithUser("CheckItemCount failed", player, nil)
		return errors.New("item config is equip")
	} else {
		count, err := mainInventoryService.GetItemCount(player.GetUserId(), item.ID)
		if err != nil {
			platformLogger.ErrorWithUser("ResetItem failed", player, err)
			return err
		}
		if count >= item.Num {
			return nil
		} else {
			addItem := &gameConfig.ItemConfig{
				ID:  item.ID,
				Num: item.Num - count,
			}
			return i.AddItem(player, addItem, reason)
		}
	}
}

func (i *ItemService) GetItemCount(player logicCommon.PlayerInterface, itemId int32) int64 {
	itemCfg := gameConfig.GetItemCfg(itemId)
	if itemCfg == nil {
		platformLogger.ErrorWithUser("ResetItem failed config = nil", player, nil)
		return 0
	}
	if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_EQUIP) {
		platformLogger.ErrorWithUser("CheckItemCount failed", player, nil)
		return 0
	} else {
		count, err := mainInventoryService.GetItemCount(player.GetUserId(), itemId)
		if err != nil {
			platformLogger.ErrorWithUser("ResetItem failed", player, err)
			return 0
		}
		return count
	}
}

func RegisterItemService(sender logicCommon.MessageSenderInterface, inventory inventory.InventoryServiceInterface, equipInventory logicCommon.EquipmentInterface, Bus *eventService.EventBus, passS logicCommon.PassServiceInterface) {
	messageSender = sender
	mainInventoryService = inventory
	equipInventoryService = equipInventory
	eventBus = Bus
	passService = passS
}

func AddItem(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	//cfg := gameConfig.GetItemCfg(item.ID)
	//if cfg == nil {
	//	platformLogger.ErrorWithUser( "AddItem failed config = nil", player, nil)
	//	return errors.New("item config is nil")
	//}
	//ReportUserItemChange(player.GetUserId(), cfg.Type, item.ID, int32(reason), item.Num, 0, 0, 0, 0)
	//ReportUserItemChange(player.GetUserId(), int32(enum.ITEM_TYPE_ADD), item.ID, reason, item.Num, 0, 0, 0, 0)
	return service.AddItem(player, item, reason)
}

func AddItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.AddItems(player, items, reason)
}

func AddItemsWithPayType(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason, payType enum.RechargeType) error {
	return service.AddItemsWithPayType(player, items, reason, payType)
}

func CheckItemCount(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig) (bool, error) {
	return service.CheckItemCount(player, item)
}

func CheckItemsCountWithMap(player logicCommon.PlayerInterface, items map[int32]int64) (bool, error) {
	return service.CheckItemsCountWithMap(player, items)
}

func CheckItemsCount(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig) (bool, error) {
	return service.CheckItemsCount(player, items)
}

func RemoveItem(player logicCommon.PlayerInterface, item *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.RemoveItem(player, item, reason)
}

func RemoveItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.RemoveItems(player, items, reason)
}

func RemoveItemsWithMap(player logicCommon.PlayerInterface, items map[int32]int64, reason enum.ItemChangeReason) error {
	return service.RemoveItemsWithMap(player, items, reason)
}

func ExchangeItem(player logicCommon.PlayerInterface, costItems []*gameConfig.ItemConfig, addItems []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.ExchangeItem(player, costItems, addItems, reason)
}

func ResetItems(player logicCommon.PlayerInterface, items []*gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.ResetItems(player, items, reason)
}

func ResetItem(player logicCommon.PlayerInterface, items *gameConfig.ItemConfig, reason enum.ItemChangeReason) error {
	return service.ResetItem(player, items, reason)
}

func GetItemCount(player logicCommon.PlayerInterface, itemId int32) int64 {
	if service == nil {
		return 0
	}
	return service.GetItemCount(player, itemId)
}

func GetItemService() *ItemService {
	return service
}
