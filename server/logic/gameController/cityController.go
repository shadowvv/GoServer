package gameController

import (
	"math"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/lumber"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("City", &CityController{})
}

type CityController struct {
}

var _ LogicControllerInterface = (*CityController)(nil)

func (a *CityController) RegisterLogicMessage() {
	//主城建筑升级部分
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ALL_ARCHITECTURE_DETAIL_REQ, &pb.AllArchitectureDetailReq{}, AllArchitectureDetailReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_REQ, &pb.ArchitectureLevelUpReq{}, ArchitectureLevelUpReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_REQ, &pb.ArchitectureAccelerateReq{}, ArchitectureAccelerateReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ARCHITECTURE_FINISH_REQ, &pb.ArchitectureFinishReq{}, ArchitectureFinishReqHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_ARCHITECTURE_STOP_REQ, &pb.ArchitectureStopReq{}, ArchitectureStopReqHandle, enum.FUNCTION_ID_NONE)

	//传承石像
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_STONE_DETAIL_REQ, &pb.StoneDetailReq{}, StoneDetailHandle, enum.FUNCTION_ID_STONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_STONE_ATTR_UP_REQ, &pb.StoneAttrUpReq{}, StoneAttrUpHandle, enum.FUNCTION_ID_STONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_STONE_ATTR_RESET_REQ, &pb.StoneAttrResetReq{}, StoneAttrResetHandle, enum.FUNCTION_ID_STONE)

	// 收藏系统
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_COLLECTION_REQ, &pb.CollectionReq{}, CollectionHandle, enum.FUNCTION_ID_STONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_REQ, &pb.CollectionItemLevelUpReq{}, CollectionItemLevelUpHandle, enum.FUNCTION_ID_STONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_REQ, &pb.CollectionEntryLevelUpReq{}, CollectionEntryLevelUpHandle, enum.FUNCTION_ID_STONE)
}

func AllArchitectureDetailReqHandle(message proto.Message, player *model.PlayerModel) {
	architectureInfo := make([]*pb.ArchitectureInfo, 0)
	for _, v := range player.ArchitectureModel.Entities {
		architectureDetail := &pb.ArchitectureInfo{
			Type:        v.Type,
			Level:       v.Level,
			Status:      v.Status,
			UpStartTime: v.UpStartTime,
		}
		architectureInfo = append(architectureInfo, architectureDetail)
	}
	resp := &pb.AllArchitectureDetailResp{
		ArInfoList: architectureInfo,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_ALL_ARCHITECTURE_DETAIL_RESP, resp)
}

func ArchitectureLevelUpReqHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ArchitectureLevelUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if req.Type != int32(enum.ARCHITECTURE_TYPE_MAIN) {
		systemUnlockId := enum.GetArchitectureTypeName(req.Type)

		flag := unlockService.CheckSystemUnlock(systemUnlockId, player)
		if !flag {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
			return
		}
	}

	architectureEntity, ok := player.ArchitectureModel.Entities[req.Type]
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ARCHITECTURE_TYPE_ERROR)
		return
	}

	if architectureEntity.Status == 2 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ARCHITECTURE_IS_UPGRADING)
		return
	}
	level := architectureEntity.Level
	//nextCfg := gameConfig.GetCityLevelCfg(req.Type, architectureEntity.Level+1)
	//if nextCfg == nil {
	//	platformLogger.InfoWithUser("architecture level max", player)
	//	messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ARCHITECTURE_LEVEL_MAX)
	//	return
	//}

	//centerCfg := gameConfig.GetCityCenterCfg(architectureEntity.Level)

	cfg := gameConfig.GetCityLevelCfg(req.Type, level+1)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ARCHITECTURE_LEVEL_MAX)
		return
	}
	for _, v := range cfg.GetUnlock() {
		if v != 0 && !unlockService.CheckUnlock(v, player) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
			return
		}
	}
	if !lumber.Service.CheckUpgradeProgress(player, req.Type) {
		platformLogger.InfoWithUser("production building progress not enough", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_CITY_LUMBER_PROGRESS_NOT_ENOUGH)
		return
	}

	ok, err := itemService.CheckItemsCount(player, cfg.GetItem())
	if !ok || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	err = itemService.RemoveItems(player, cfg.GetItem(), enum.ITEM_CHANGE_REASON_ARCHITECTURE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	// TODO: status改为枚举或者const
	if architectureEntity.Status == 0 {
		player.ArchitectureModel.UpdateStatus(req.Type, 3)
	} else if architectureEntity.Status == 1 {
		player.ArchitectureModel.UpdateStatus(req.Type, 2)
	}
	player.ArchitectureModel.UpdateUpStartTime(req.Type, tool.UnixNowMilli())

	messageSender.SendMessage(player, pb.MESSAGE_ID_ARCHITECTURE_LEVEL_UP_RESP, &pb.ArchitectureLevelUpResp{
		ArInfo: &pb.ArchitectureInfo{
			Type:        architectureEntity.Type,
			Level:       architectureEntity.Level,
			Status:      architectureEntity.Status,
			UpStartTime: architectureEntity.UpStartTime,
		},
	})
}

func ArchitectureAccelerateReqHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ArchitectureAccelerateReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	architectureEntity, ok := player.ArchitectureModel.Entities[req.Type]
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_ARCHITECTURE_TYPE_ERROR)
		return
	}

	if architectureEntity.Status != 2 && architectureEntity.Status != 3 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_ARCHITECTURE_IS_NOT_UPGRADING)
		return
	}

	cfg := gameConfig.GetCityLevelCfg(req.Type, architectureEntity.Level+1)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	lossTime := int64(0)
	for _, v := range req.ItemInfo {
		itemCfg := gameConfig.GetItemCfg(v.ItemId)
		if itemCfg == nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		itemInfo := &gameConfig.ItemConfig{
			ID:  v.ItemId,
			Num: v.Count,
		}
		ok, err := itemService.CheckItemCount(player, itemInfo)
		if !ok || err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
		lossTime += int64(itemCfg.TargetId*1000) * v.Count
		err = itemService.RemoveItem(player, itemInfo, enum.ITEM_CHANGE_REASON_ARCHITECTURE)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
			return
		}
	}

	player.ArchitectureModel.UpdateUpStartTime(req.Type, architectureEntity.UpStartTime-lossTime)
	if tool.UnixNowMilli()-player.ArchitectureModel.Entities[req.Type].UpStartTime >= int64(cfg.GetTime())*1000 {
		player.ArchitectureModel.OnUpgradeComplete(req.Type, architectureEntity.Level, true)
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_ARCHITECTURE_ACCELERATE_RESP, &pb.ArchitectureAccelerateResp{
		ArInfo: &pb.ArchitectureInfo{
			Type:        architectureEntity.Type,
			Level:       architectureEntity.Level,
			Status:      architectureEntity.Status,
			UpStartTime: architectureEntity.UpStartTime,
		},
	})
}

func ArchitectureFinishReqHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ArchitectureFinishReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	architectureEntity, ok := player.ArchitectureModel.Entities[req.Type]
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_ARCHITECTURE_TYPE_ERROR)
		return
	}

	if architectureEntity.Status != 2 && architectureEntity.Status != 3 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_ARCHITECTURE_IS_NOT_UPGRADING)
		return
	}
	level := architectureEntity.Level
	if architectureEntity.Status == 3 {
		level = 0
	}
	cfg := gameConfig.GetCityLevelCfg(req.Type, level+1)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	for _, v := range cfg.GetUnlock() {
		if v != 0 && !unlockService.CheckUnlock(v, player) {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
			return
		}
	}

	finishMinConst := gameConfig.GetBuildingMinCost()
	nowTime := tool.UnixNowMilli()
	cfgTime := cfg.GetTime()
	runSec := float64(nowTime-architectureEntity.UpStartTime) / 1000
	remainMin := math.Ceil(float64(float64(cfgTime)-runSec) / 60.0)
	needTime := int32(remainMin)
	//needTime := int32(math.Ceil(float64(float64(cfg.GetTime())-float64(tool.UnixNowMilli()-architectureEntity.UpStartTime)/1000) / float64(60)))
	if needTime <= 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_ARCHITECTURE_IS_ALMOST_FINISHED)
		return
	}

	ok, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{
		ID:  enum.DIAMOND_ITEM_ID,
		Num: int64(needTime * finishMinConst),
	})
	if !ok || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItem(player, &gameConfig.ItemConfig{
		ID:  enum.DIAMOND_ITEM_ID,
		Num: int64(needTime * finishMinConst),
	}, enum.ITEM_CHANGE_REASON_ARCHITECTURE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	player.ArchitectureModel.OnUpgradeComplete(req.Type, level, true)

	messageSender.SendMessage(player, pb.MESSAGE_ID_ARCHITECTURE_FINISH_RESP, &pb.ArchitectureFinishResp{
		ArInfo: &pb.ArchitectureInfo{
			Type:        architectureEntity.Type,
			Level:       architectureEntity.Level,
			Status:      architectureEntity.Status,
			UpStartTime: architectureEntity.UpStartTime,
		},
	})
}

func ArchitectureStopReqHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.ArchitectureStopReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	architectureEntity, ok := player.ArchitectureModel.Entities[req.Type]
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, pb.ERROR_CODE_ARCHITECTURE_TYPE_ERROR)
		return
	}

	if architectureEntity.Status != 2 && architectureEntity.Status != 3 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, pb.ERROR_CODE_ARCHITECTURE_IS_NOT_UPGRADING)
		return
	}
	cfg := gameConfig.GetCityLevelCfg(req.Type, architectureEntity.Level)
	if cfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	err := itemService.AddItems(player, cfg.GetItem(), enum.ITEM_CHANGE_REASON_ARCHITECTURE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, pb.ERROR_CODE_ADD_ITEM_ERROR)
		return
	}
	itemInfos := make([]*pb.ItemBasicInfo, 0)
	for _, v := range cfg.GetItem() {
		itemInfos = append(itemInfos, &pb.ItemBasicInfo{
			ItemId: v.ID,
			Count:  v.Num,
		})
	}
	player.ArchitectureModel.UpdateStatus(req.Type, 1)

	messageSender.SendMessage(player, pb.MESSAGE_ID_ARCHITECTURE_STOP_RESP, &pb.ArchitectureStopResp{
		ArInfo: &pb.ArchitectureInfo{
			Type:        architectureEntity.Type,
			Level:       architectureEntity.Level,
			Status:      architectureEntity.Status,
			UpStartTime: architectureEntity.UpStartTime,
		},
		ItemInfo: itemInfos,
	})
}

func StoneDetailHandle(message proto.Message, player *model.PlayerModel) {
	stoneInfo := make([]*pb.StoneInfo, 0)
	for _, v := range player.StoneModel.Entities {
		cfgAttr := gameConfig.GetStatueAttrInfoByLevel()[v.Class]
		atteLevel := make(map[int32]int32)
		for key, _ := range cfgAttr {
			if _, ok := gameConfig.GetStatueAttrLevelMap()[v.Class][key]; !ok {
				platformLogger.InfoWithUser("stone index cfg info is null", player)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_DETAIL_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
				return
			}
			if len(gameConfig.GetStatueAttrLevelMap()[v.Class][key].Attr) > len(v.AttrLevel) {
				for i := int32(len(v.AttrLevel)); i < gameConfig.GetStatueAttrIndexMap()[v.Class][key]+1; i++ {
					v.AttrLevel = append(v.AttrLevel, 0)
				}
				player.StoneModel.UpdateAttrLevel(v.Class, v.AttrLevel)
			}
			atteLevel[key] = v.AttrLevel[gameConfig.GetStatueAttrIndexMap()[v.Class][key]]
		}
		stoneInfo = append(stoneInfo, &pb.StoneInfo{
			StoneClass: v.Class,
			AttrLevel:  atteLevel,
		})
	}

	messageSender.SendMessage(player, pb.MESSAGE_ID_STONE_DETAIL_RESP, &pb.StoneDetailResp{
		StoneInfoList: stoneInfo,
	})
}

func StoneAttrUpHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.StoneAttrUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	// 检查该职业是否有配置（惰性加载：配置存在但 entity 为空时自动创建）
	if _, hasConfig := gameConfig.GetStatueAttrIndexMap()[req.StoneClass]; !hasConfig {
		platformLogger.InfoWithUser("stone class config not found", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_STONE_CLASS_ERROR)
		return
	}

	stoneEntity, ok := player.StoneModel.Entities[req.StoneClass]
	if !ok {
		// 配置存在但 entity 不存在，自动创建
		attrCount := len(gameConfig.GetStatueAttrIndexMap()[req.StoneClass])
		initialAttrLevel := make([]int32, attrCount)
		err := player.StoneModel.AddStoneEntity(req.StoneClass, initialAttrLevel)
		if err != nil {
			platformLogger.ErrorWithUser("add stone entity error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		stoneEntity = player.StoneModel.Entities[req.StoneClass]
	}

	stoneLevel := player.ArchitectureModel.Entities[int32(enum.ARCHITECTURE_TYPE_STONE)].Level
	stoneLevelCfg := gameConfig.GetHeritageStatueCfg(stoneLevel)
	if stoneLevelCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	flag := false
	for _, v := range stoneLevelCfg.Attr {
		if v == req.Attr {
			flag = true
			break
		}
	}
	if !flag {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_STONE_ATTR_ATTR_NOT_UNLOCK)
		return
	}

	// 确保 AttrLevel 长度与配置一致（处理配置变更后 attr 数量变化的情况）
	attrCount := len(gameConfig.GetStatueAttrIndexMap()[req.StoneClass])
	if len(stoneEntity.AttrLevel) < attrCount {
		// 配置新增了属性，扩展 AttrLevel
		for i := len(stoneEntity.AttrLevel); i < attrCount; i++ {
			stoneEntity.AttrLevel = append(stoneEntity.AttrLevel, 0)
		}
		player.StoneModel.UpdateAttrLevel(req.StoneClass, stoneEntity.AttrLevel)
	}

	// 获取当前属性等级
	attrIndex := gameConfig.GetStatueAttrIndexMap()[req.StoneClass][req.Attr]
	currentLevel := stoneEntity.AttrLevel[attrIndex]

	// 获取石像等级和对应配置
	architectureDetail := player.ArchitectureModel.Entities[int32(enum.ARCHITECTURE_TYPE_STONE)]
	cfg := gameConfig.GetStatueAttrCfgMap()[architectureDetail.Level][req.StoneClass]
	if cfg == nil {
		platformLogger.InfoWithUser("stone attr cfg not found", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}

	// 判断是否已达当前区间右边界
	maxLevel := cfg.AttrLevel[1]
	if currentLevel >= maxLevel {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_STONE_ATTR_LEVEL_MAX)
		return
	}

	// 计算目标等级（批量升级，不超过上限）
	var upgradeCount int32
	if req.UpType == 1 {
		upgradeCount = 10
	} else {
		upgradeCount = 1
	}
	targetLevel := currentLevel + upgradeCount
	if targetLevel > maxLevel {
		targetLevel = maxLevel
	}

	// 使用反向索引快速查找配置，计算总消耗
	totalCost := make(map[int32]int64)
	for level := currentLevel + 1; level <= targetLevel; level++ {
		cfg := gameConfig.GetStatueAttrLevelMap()[req.StoneClass][level]
		if cfg == nil {
			platformLogger.InfoWithUser("stone attr cfg not found", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
			return
		}
		for _, item := range cfg.Cost {
			totalCost[item.ID] += item.Num
		}
	}

	// 转换消耗列表
	costList := make([]*gameConfig.ItemConfig, 0, len(totalCost))
	for itemId, num := range totalCost {
		costList = append(costList, &gameConfig.ItemConfig{
			ID:  itemId,
			Num: num,
		})
	}

	// 检查并扣除消耗
	flag, err := itemService.CheckItemsCount(player, costList)
	if !flag || err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	err = itemService.RemoveItems(player, costList, enum.ITEM_CHANGE_REASON_ARCHITECTURE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}

	// 更新属性等级
	stoneEntity.AttrLevel[attrIndex] = targetLevel
	player.StoneModel.UpdateAttrLevel(req.StoneClass, stoneEntity.AttrLevel)

	messageSender.SendMessage(player, pb.MESSAGE_ID_STONE_ATTR_UP_RESP, &pb.StoneAttrUpResp{
		Level: targetLevel,
	})
	eventBusService.SubmitStoneAttrLevelUpEvent(player.GetUserId())
}

func StoneAttrResetHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.StoneAttrResetReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_RESET_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	stoneEntity, ok := player.StoneModel.Entities[req.StoneClass]
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_RESET_RESP, pb.ERROR_CODE_STONE_CLASS_ERROR)
		return
	}

	totalCost := make(map[int32]int64)
	for _, value := range gameConfig.GetStatueAttrIndexMap()[req.StoneClass] {
		level := stoneEntity.AttrLevel[value]
		if level > 0 {
			for i := int32(1); i <= player.ArchitectureModel.Entities[int32(enum.ARCHITECTURE_TYPE_STONE)].Level; i++ {
				cfg := gameConfig.GetStatueAttrCfgMap()[i][req.StoneClass]
				if cfg == nil {
					platformLogger.InfoWithUser("stone attr cfg is  nil", player)
					messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_RESET_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
					return
				}
				if level < cfg.AttrLevel[0] {
					break
				}
				maxLevel := min(cfg.AttrLevel[1], level)
				for _, v := range cfg.Cost {
					totalCost[v.ID] += v.Num * int64(maxLevel-cfg.AttrLevel[0]+1)
				}
			}
		}
	}

	addItems := make([]*gameConfig.ItemConfig, 0)
	for itemId, num := range totalCost {
		addItems = append(addItems, &gameConfig.ItemConfig{
			ID:  itemId,
			Num: num,
		})
	}

	err := itemService.AddItems(player, addItems, enum.ITEM_CHANGE_REASON_ARCHITECTURE)
	if err != nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_STONE_ATTR_RESET_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
		return
	}
	attrLevel := make(tool.JSONInt32Slice, 0)
	for i := 0; i < len(gameConfig.GetStatueAttrIndexMap()[req.StoneClass]); i++ {
		attrLevel = append(attrLevel, 0)
	}

	player.StoneModel.UpdateAttrLevel(req.StoneClass, attrLevel)
	messageSender.SendMessage(player, pb.MESSAGE_ID_STONE_ATTR_RESET_RESP, &pb.StoneAttrResetResp{})
}

func CollectionHandle(message proto.Message, player *model.PlayerModel) {
	resp := &pb.CollectionResp{
		CollectionItemInfoList:  make([]*pb.CollectionItemInfo, 0),
		CollectionEntryInfoList: make([]*pb.CollectionEntryInfo, 0),
	}
	for _, v := range player.CollectionModel.CollectionEntity {
		resp.CollectionItemInfoList = append(resp.CollectionItemInfoList, &pb.CollectionItemInfo{
			CollectionId: v.CollectId,
		})
	}
	for _, v := range player.CollectionModel.EntryEntity {
		resp.CollectionEntryInfoList = append(resp.CollectionEntryInfoList, &pb.CollectionEntryInfo{
			EntryId: v.EntryId,
			Level:   v.EntryLevel,
		})
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_RESP, resp)
}

func CollectionItemLevelUpHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CollectionItemLevelUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	collectionCfg := gameConfig.GetCollectionEntityCfg(req.CollectionId)
	if collectionCfg == nil {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_INVALID_REQUEST_PARAM)
		return
	}
	collectionEntity, err := player.CollectionModel.CollectionLevelUp(collectionCfg.Belonging, player, req.UseItemList)
	switch {
	case err == nil:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, &pb.CollectionItemLevelUpResp{
			CollectionItemInfo: &pb.CollectionItemInfo{
				CollectionId: collectionEntity.CollectId,
			},
		})
		operationLogService.OnUserCollectionSystemStarDust(player.GetUserId(), collectionEntity.CollectId, collectionEntity.CollectLevel-1, collectionEntity.CollectLevel)
	case err.Error() == "collection level up is max":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_COLLECTION_LEVEL_MAX)
	case err.Error() == "item id is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
	case err.Error() == "item count is not enough":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
	case err.Error() == "add collection failed":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_ADD_COLLECTION_FAILED)
	case err.Error() == "item target entity is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_TARGET_ENTITY_NOT_EXIST)
	case err.Error() == "item target entity level is not enough":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_TARGET_ENTITY_LEVEL_NOT_ENOUGH)
	case err.Error() == "collection level up is not unlock":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, pb.ERROR_CODE_COLLECTION_LEVEL_UP_NOT_UNLOCKED)
	default:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ITEM_LEVEL_UP_RESP, &pb.CollectionItemLevelUpResp{})
	}
}

func CollectionEntryLevelUpHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.CollectionEntryLevelUpReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	collectionEntity, err := player.CollectionModel.EntryLevelUp(req.EntryId, player)
	switch {
	case err == nil:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, &pb.CollectionEntryLevelUpResp{
			CollectionEntryInfo: &pb.CollectionEntryInfo{
				EntryId: collectionEntity.EntryId,
				Level:   collectionEntity.EntryLevel,
			},
		})
		if collectionEntity.EntryLevel == 1 {
			operationLogService.OnUserCollectionEntryActive(player.GetUserId(), collectionEntity.EntryId)
		} else {
			operationLogService.OnUserCollectionEntryUpLevel(player.GetUserId(), collectionEntity.EntryId, collectionEntity.EntryLevel-1, collectionEntity.EntryLevel)
		}
	case err.Error() == "entry level is max":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_ENTRY_LEVEL_MAX)
	case err.Error() == "entry cfg no exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
	case err.Error() == "item count is not enough":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_ITEM_NOT_ENOUGH)
	case err.Error() == "entry main id is not exist":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_ENTRY_MAIN_ID_NOT_EXIST)
	case err.Error() == "add entry failed":
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, pb.ERROR_CODE_ADD_ENTRY_FAILED)
	default:
		messageSender.SendMessage(player, pb.MESSAGE_ID_COLLECTION_ENTRY_LEVEL_UP_RESP, &pb.CollectionEntryLevelUpResp{})
	}
}
