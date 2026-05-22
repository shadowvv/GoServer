package adChest

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
)

var Service = &AdChestService{}

type AdChestService struct{}

// GetDailyOpenLimit 获取今日开启上限（constant 基础值 + 特权卡额外次数）
func (s *AdChestService) GetDailyOpenLimit(player logicCommon.PlayerInterface) int32 {
	base := gameConfig.GetDailyAdChestOpeningAttemptsLimit()
	extra, _ := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_AD_CHEST_OPEN)
	return base + int32(extra)
}

// CanGrantNewChest 检查是否可以发放新宝箱（未达今日开启上限时才能获得新宝箱）
func (s *AdChestService) CanGrantNewChest(player logicCommon.PlayerInterface) bool {
	m := s.getOrLoadAdChestModel(player)
	if m == nil {
		return false
	}
	currentTime := tool.UnixNowMilli()
	openCount := m.GetTodayOpenCount(currentTime)
	limit := s.GetDailyOpenLimit(player)
	return openCount < limit
}

// 同时获取数量
func (s *AdChestService) CanGetNewChest(player logicCommon.PlayerInterface) bool {
	m := s.getOrLoadAdChestModel(player)
	if m == nil {
		return false
	}
	chests := m.GetAllChests()
	currentTime := tool.UnixNowMilli()
	chestCount := 0
	for _, e := range chests {
		cfg := gameConfig.GetLimitedAdChestCfg(e.CfgIndex)
		if cfg == nil {
			continue
		}
		expireTime := e.CreateTime + int64(cfg.Duration)*tool.MINUTE_MILLI
		if currentTime > expireTime {
			m.RemoveChest(e.UniqueId) // 过期则移除，不发给客户端
			continue
		}
		chestCount++
	}
	return chestCount < 5
}

// GrantAdChest 发放广告宝箱，返回唯一ID 和 推送消息（调用方负责发送）；若已达获取上限返回空
func (s *AdChestService) GrantAdChest(player logicCommon.PlayerInterface, itemId int32) (string, *pb.PushAdChestNew, error) {
	if player == nil {
		return "", nil, errors.New("player is nil")
	}
	itemCfg := gameConfig.GetItemCfg(itemId)
	if itemCfg == nil || itemCfg.ShowGroup != int32(enum.ITEM_TYPE_AD_CHEST) {
		return "", nil, errors.New("invalid ad chest item")
	}
	cfg := gameConfig.GetLimitedAdChestCfg(itemCfg.TargetId)
	if cfg == nil {
		return "", nil, errors.New("ad chest config not found")
	}
	if !s.CanGrantNewChest(player) {
		return "", nil, nil // 已达上限，不报错，返回空表示未发放
	}

	if !s.CanGetNewChest(player) {
		return "", nil, nil // 同时只能获取 5个
	}

	m := s.getOrLoadAdChestModel(player)
	if m == nil {
		return "", nil, errors.New("failed to load ad chest model")
	}

	currentTime := tool.UnixNowMilli()
	expireTime := currentTime + int64(cfg.Duration)*60*1000 // Duration 单位：分钟
	uniqueId := m.AddChest(itemId, itemCfg.TargetId, currentTime)

	pushMsg := &pb.PushAdChestNew{
		Chest: &pb.AdChestInfo{
			UniqueId:   uniqueId,
			ItemId:     itemId,
			CfgIndex:   itemCfg.TargetId,
			CreateTime: currentTime,
			ExpireTime: expireTime,
		},
	}
	return uniqueId, pushMsg, nil
}

// OpenAdChest 开启广告宝箱，返回掉落物品列表
func (s *AdChestService) OpenAdChest(player logicCommon.PlayerInterface, uniqueId string, watchAd bool) ([]*gameConfig.ItemConfig, error) {
	if player == nil {
		return nil, errors.New("player is nil")
	}
	adChestModel := s.getOrLoadAdChestModel(player)
	if adChestModel == nil {
		return nil, errors.New("failed to load ad chest model")
	}

	chest := adChestModel.GetChest(uniqueId)
	if chest == nil {
		return nil, errors.New("chest not found")
	}

	cfg := gameConfig.GetLimitedAdChestCfg(chest.CfgIndex)
	if cfg == nil {
		return nil, errors.New("ad chest config not found")
	}

	currentTime := tool.UnixNowMilli()
	expireTime := chest.CreateTime + int64(cfg.Duration)*tool.MINUTE_MILLI
	if currentTime > expireTime+enum.AD_CHEST_OPEN_TOLERANCE_MS {
		return nil, errors.New("chest expired")
	}

	limit := s.GetDailyOpenLimit(player)
	todayCount := adChestModel.GetTodayOpenCount(currentTime)
	if todayCount >= limit {
		return nil, errors.New("daily open limit reached")
	}

	// 选择掉落
	var dropId int32
	if watchAd {
		dropId = cfg.AdDropId
	} else {
		dropId = cfg.DropId
	}
	items := gameConfig.Drop(dropId)

	// 消耗宝箱，增加今日开启计数
	adChestModel.RemoveChest(uniqueId)
	adChestModel.IncrementTodayOpenCount(currentTime)

	// 奖励由 controller 调用 itemService.AddItems 发放
	return items, nil
}

// GetOrLoadAdChestModel 获取或加载广告宝箱模型（供 controller 等调用）
func (s *AdChestService) GetOrLoadAdChestModel(player logicCommon.PlayerInterface) *model.AdChestModel {
	return s.getOrLoadAdChestModel(player)
}

func (s *AdChestService) getOrLoadAdChestModel(player logicCommon.PlayerInterface) *model.AdChestModel {
	if pm, ok := player.(*model.PlayerModel); ok {
		if pm.AdChestModel != nil {
			return pm.AdChestModel
		}
		userId := player.GetUserId()
		m, err := model.LoadAdChestModel(userId)
		if err != nil {
			m = model.CreateAdChestModel(userId)
		}
		pm.AdChestModel = m
		pm.AppendPlayerModel(m)
		return m
	}
	return nil
}
