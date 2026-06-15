package gameController

import (
	"context"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	unlockSvc "github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterController("Lotter", &LotteryController{})
}

type LotteryController struct {
}

var _ LogicControllerInterface = (*LotteryController)(nil)

func (l *LotteryController) RegisterLogicMessage() {
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_REQ, &pb.GetLotteryHistoryDetailReq{}, GetLotteryHistoryDetailHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_GET_LOTTERY_INFO_REQ, &pb.GetLotteryInfoReq{}, GetLotteryInfoHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_LOTTERY_REQ, &pb.LotteryReq{}, LotteryHandle, enum.FUNCTION_ID_NONE)
	RegisterPlayerMessageHandler(enum.MSG_TYPE_PLAYER, pb.MESSAGE_ID_NEW_LUCKY_REWARD_REQ, &pb.NewLuckyRewardReq{}, NewLuckyRewardHandle, enum.FUNCTION_ID_NONE)
}

func GetLotteryHistoryDetailHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get lottery history detail req", player)

	req, ok := message.(*pb.GetLotteryHistoryDetailReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	if player.LotteryModel == nil || player.LotteryModel.LotteryHistoryDetail == nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, &pb.GetLotteryHistoryDetailResp{
			HistoryDetails: nil,
		})
		return
	}
	res := make([]*pb.LotteryHistoryDetail, 0)
	for _, v := range player.LotteryModel.LotteryHistoryDetail[lotteryId] {
		res = append(res, &pb.LotteryHistoryDetail{
			Id:     v.Id,
			ItemId: v.ItemId,
			Count:  v.Count,
		})
	}
	resp := &pb.GetLotteryHistoryDetailResp{
		HistoryDetails: res,
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, resp)
}

func GetLotteryInfoHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("get lottery info req", player)

	req, ok := message.(*pb.GetLotteryInfoReq)
	if !ok {

		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	if player.LotteryModel == nil || player.LotteryModel.LotteryEntities == nil {
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, &pb.GetLotteryInfoResp{
			LotteryInfo: &pb.LotteryInfo{
				LotteryCount:         0,
				LastBasicGuaranteNum: 0,
				IsDirty:              false,
			},
			FirstDropFree: 0,
		})
		return
	}
	lotteryEntity := player.LotteryModel.GetLotteryEntityById(req.LotteryId)
	// 确定共享保底数据的实体（如果ActModID存在，多个卡池共用一份保底数据）
	dataLotteryId := player.LotteryModel.GetSharedLotteryEntityId(lotteryId)
	dataEntity := player.LotteryModel.GetLotteryEntityById(dataLotteryId)
	if dataEntity == nil && lotteryEntity == nil {
		// 共享实体和自身实体都不存在，返回全0
		messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, &pb.GetLotteryInfoResp{
			LotteryInfo: &pb.LotteryInfo{
				LotteryCount:         0,
				LastBasicGuaranteNum: 0,
				IsDirty:              false,
			},
			FirstDropFree: 0,
		})
		return
	}
	if dataEntity == nil {
		dataEntity = lotteryEntity
	}
	firstDropFree := int32(0)
	if lotteryEntity != nil {
		if lotteryEntity.FirstDropFree > 0 {
			if !tool.IsSameDayByMilli(lotteryEntity.LastFirstDropFreeTime, tool.UnixNowMilli()) {
				lotteryEntity.FirstDropFree--
				player.LotteryModel.UpdateFirstDropFree(lotteryId, lotteryEntity.FirstDropFree)
			}
		}
		firstDropFree = lotteryEntity.FirstDropFree
	}
	lotteryLuckyEvent := player.LotteryModel.GetLotteryLuckyEvent()
	resp := &pb.GetLotteryInfoResp{
		LotteryInfo: &pb.LotteryInfo{
			LotteryCount:         dataEntity.AllCount,
			LastBasicGuaranteNum: dataEntity.LastBasicGuaranteeNum,
			IsDirty:              dataEntity.IsDirty,
		},
		FirstDropFree: firstDropFree,
	}
	if lotteryLuckyEvent != nil {
		if cfg.ActModID != 0 {
			resp.LotteryLuckyEvent = &pb.LotteryLuckyEvent{
				LuckyNum:   lotteryLuckyEvent.LuckyNum,
				CreateTime: lotteryLuckyEvent.CreateTime,
			}
		}
		resp.NewLuckyCount = lotteryLuckyEvent.NewLuckyCount
		resp.NewIsReward = lotteryLuckyEvent.IsNewLuckyReward
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_GET_LOTTERY_INFO_RESP, resp)
}

func LotteryHandle(message proto.Message, player *model.PlayerModel) {
	platformLogger.InfoWithUser("lottery req", player)

	req, ok := message.(*pb.LotteryReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}

	if player.LotteryModel == nil {
		platformLogger.ErrorWithUser("lottery model nil", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_LOAD_INFO_ERROR)
		return
	}

	lotteryId := req.LotteryId
	cfg := gameConfig.GetSummonPoolCfg(lotteryId)
	if cfg == nil {
		platformLogger.InfoWithUser("lottery req param error", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_LOTTERY_LOTTERY_ID_ERROR)
		return
	}
	if int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)) != 0 && !unlockService.CheckSystemUnlock(int32(player.LotteryModel.GetLotterySystemId(cfg.Gashatype)), player) {
		platformLogger.InfoWithUser("lottery system not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_GET_LOTTERY_HISTORY_DETAIL_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}
	lotteryType := req.LotteryType
	lotteryNum := int32(0)
	lotteryDetail := player.LotteryModel.GetLotteryEntityById(lotteryId)
	if lotteryDetail == nil {
		// 初始化抽奖信息
		newEntity := &model.LotteryEntity{
			UserId:                  player.GetUserId(),
			Id:                      lotteryId,
			AllCount:                0,
			LastBasicGuaranteeNum:   0,
			LastSpecialGuaranteeNum: 0,
			SpecialGuaranteeCount:   0,
			LastOnceEffectiveNum:    0,
			OnceEffectiveCount:      0,
			LastChangeTime:          tool.UnixNowMilli(),
			FirstDropFree:           0,
		}
		if cfg.ActModID != 0 {
			ok, actVersion := player.PlayerActivityModel.CheckActivityOpen(cfg.ActModID)
			if ok {
				newEntity.ActVersion = actVersion
			} else {
				platformLogger.InfoWithUser("lottery req activity not open", player)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
				return
			}
		}
		if err := player.LotteryModel.AddLotteryEntity(newEntity); err != nil {
			platformLogger.ErrorWithUser("add lottery entity error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_LOAD_INFO_ERROR)
			return
		}
		lotteryDetail = newEntity
	}

	if lotteryType == 1 {
		lotteryNum = 1
	} else if lotteryType == 2 {
		lotteryNum = 10
	} else {
		platformLogger.ErrorWithUser("lottery req param error", player, nil)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_TYPE_ERROR)
		return
	}

	res := make([]*pb.ItemBasicInfo, 0)

	//判断背包容量
	if lotteryId == 1 {
		if len(player.HeroDetailsModel.Entities)+int(lotteryNum) > hero.HeroBagMaxNum {
			platformLogger.InfoWithUser("lottery req bag full", player)
			messageSender.SendMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, &pb.LotteryResp{
				Result: pb.LotteryResult_LOTTERY_RESULT_HERO_BAG_FULL,
			})
			return
		}
	}

	// 判断抽奖道具是否足够
	if lotteryNum == 1 && cfg.FirstDropFree-lotteryDetail.FirstDropFree < 0 || lotteryNum == 10 {
		if flag, err := itemService.CheckItemCount(player, &gameConfig.ItemConfig{ID: gameConfig.GetSummonPoolCfg(req.LotteryId).DropToken, Num: int64(lotteryNum)}); !flag || err != nil {
			platformLogger.InfoWithUser("lottery req lack item", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_ITEM_NOT_ENOUGH)
			return
		}
	}

	// todo 判断卡池时间
	if cfg.ActModID != 0 {
		ok, actVersion := player.PlayerActivityModel.CheckActivityOpen(cfg.ActModID)
		if !ok {
			platformLogger.InfoWithUser("lottery req activity not open", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_ACTIVITY_NOT_OPEN)
			return
		}
		if lotteryDetail.ActVersion != "" && lotteryDetail.ActVersion != actVersion {
			// todo 重置卡池数据
		}
	}

	if cfg.UnlockID != 0 && !unlockService.CheckUnlock(cfg.UnlockID, player) {
		platformLogger.InfoWithUser("lottery req unlock not open", player)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_UNLOCK_NOT_OPEN)
		return
	}

	// 确定共享保底数据的实体ID（如果ActModID存在，多个卡池共用同一份保底数据）
	dataLotteryId := player.LotteryModel.GetSharedLotteryEntityId(lotteryId)
	dataDetail := player.LotteryModel.GetLotteryEntityById(dataLotteryId)
	if dataDetail == nil && dataLotteryId != lotteryId {
		// 共享实体不存在，需要创建
		sharedEntity := &model.LotteryEntity{
			UserId:                  player.GetUserId(),
			Id:                      dataLotteryId,
			AllCount:                0,
			LastBasicGuaranteeNum:   0,
			LastSpecialGuaranteeNum: 0,
			SpecialGuaranteeCount:   0,
			LastOnceEffectiveNum:    0,
			OnceEffectiveCount:      0,
			LastChangeTime:          tool.UnixNowMilli(),
			FirstDropFree:           0,
		}
		if err := player.LotteryModel.AddLotteryEntity(sharedEntity); err != nil {
			platformLogger.ErrorWithUser("add shared lottery entity error", player, err)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_LOTTERY_LOAD_INFO_ERROR)
			return
		}
		dataDetail = sharedEntity
	} else if dataDetail == nil {
		dataDetail = lotteryDetail
	}

	vipCards, err := vipCard.Service.GetAllFunctionValues(player)
	if err != nil {
		platformLogger.ErrorWithUser("GetVipCardInfoList failed", player, err)
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	var lossValue int64 = 0
	if _, ok := vipCards[enum.VIP_PRIVILEGE_LOTTERY_GUAR]; ok {
		lossValue = vipCards[enum.VIP_PRIVILEGE_LOTTERY_GUAR]
	}

	for i := int32(0); i < lotteryNum; i++ {
		// 抽奖逻辑 - 使用共享实体进行保底计数
		player.LotteryModel.UpdateAllCount(dataLotteryId, dataDetail.AllCount+1)

		if len(cfg.LuckyGuarantees) > 0 {
			// 奖励式保底机制（与单次保底、循环保底不共存）
			luckyGuaranteesCfg := cfg.LuckyGuarantees[0]
			if luckyGuaranteesCfg == nil {
				platformLogger.ErrorWithUser("luckyGuaranteesCfg is nil", player, nil)
				continue
			}
			if dataDetail.IsDirty && luckyGuaranteesCfg.Num != 0 && dataDetail.AllCount-dataDetail.LastBasicGuaranteeNum >= luckyGuaranteesCfg.Num {
				// 已歪且达到保底阈值，强制从奖励式保底池抽取
				player.LotteryModel.LuckyGuaranteeLottery(cfg, &res, dataLotteryId)
			} else if cfg.Guarantees != nil && dataDetail.AllCount-dataDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
				player.LotteryModel.BasicLotteryGuarantee(cfg, &res, false, false, dataLotteryId)
			} else {
				// 普通抽取，抽取次数决定卡池
				player.LotteryModel.BasicLottery(cfg, &res, false, false, dataLotteryId)
			}
		} else {
			// 原有逻辑：单次保底/循环保底
			// 单次保底全部触发 或 未配置
			if dataDetail.OnceEffectiveCount >= int32(len(cfg.Guarantees1)) {
				//循环保底
				if len(cfg.Guarantees2) > 0 {
					if dataDetail.AllCount-dataDetail.LastSpecialGuaranteeNum >= cfg.Guarantees2[dataDetail.SpecialGuaranteeCount].Num {

						dropGroupId := gameConfig.WeightedRandomChoice(cfg.Guarantees2[dataDetail.SpecialGuaranteeCount].DropGroupIdList, cfg.Guarantees2Weight[dataDetail.SpecialGuaranteeCount])
						itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)

						player.LotteryModel.UpdateLastBasicGuaranteeNum(dataLotteryId, dataDetail.AllCount)
						for _, v := range itemIdList {
							itemCfg := gameConfig.GetItemCfg(v.ID)
							if itemCfg != nil {
								if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) {
									player.LotteryModel.AddLotteryLog(&model.LotteryLog{UserId: dataDetail.UserId, Id: dataLotteryId, ItemId: v.ID, Count: dataDetail.AllCount, AddTime: tool.UnixNowMilli()})
								}
							}
						}
						player.LotteryModel.UpdateLastSpecialGuaranteeNum(dataLotteryId, dataDetail.AllCount)
						player.LotteryModel.CheckAndUpSpecialGuaranteeNum(cfg, dataLotteryId)
						player.LotteryModel.UpdateLastChengeTime(dataLotteryId, tool.UnixNowMilli())

						for _, itemInfo := range itemIdList {
							res = append(res, &pb.ItemBasicInfo{
								ItemId: itemInfo.ID,
								Count:  itemInfo.Num,
							})
						}

					} else {
						// 普通保底
						if cfg.Guarantees != nil && dataDetail.AllCount-dataDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
							player.LotteryModel.BasicLotteryGuarantee(cfg, &res, false, true, dataLotteryId)
						} else {
							// 普通抽取，抽取次数决定卡池
							player.LotteryModel.BasicLottery(cfg, &res, false, true, dataLotteryId)
						}
					}
				} else {
					if cfg.Guarantees != nil && dataDetail.AllCount-dataDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
						player.LotteryModel.BasicLotteryGuarantee(cfg, &res, false, false, dataLotteryId)
					} else {
						// 普通抽取，抽取次数决定卡池
						player.LotteryModel.BasicLottery(cfg, &res, false, false, dataLotteryId)
					}
				}
			} else {
				// 单次保底未触发
				if dataDetail.AllCount-dataDetail.LastOnceEffectiveNum >= cfg.Guarantees1[dataDetail.OnceEffectiveCount].Num {
					// 触发单次保底
					dropGroupId := gameConfig.WeightedRandomChoice(cfg.Guarantees1[dataDetail.OnceEffectiveCount].DropGroupIdList, cfg.Guarantees1Weight[dataDetail.OnceEffectiveCount])
					itemIdList := gameConfig.DropGroupItems(dropGroupId, nil)
					player.LotteryModel.UpdateLastBasicGuaranteeNum(dataLotteryId, dataDetail.AllCount)
					for _, v := range itemIdList {
						itemCfg := gameConfig.GetItemCfg(v.ID)
						if itemCfg != nil {
							if itemCfg.ShowGroup == int32(enum.ITEM_TYPE_HERO) {
								player.LotteryModel.AddLotteryLog(&model.LotteryLog{UserId: dataDetail.UserId, Id: dataLotteryId, ItemId: v.ID, Count: dataDetail.AllCount, AddTime: tool.UnixNowMilli()})
							}
						}
					}
					player.LotteryModel.UpdateLastSpecialGuaranteeNum(dataLotteryId, dataDetail.AllCount)
					player.LotteryModel.UpdateLastOnceEffectiveNum(dataLotteryId, dataDetail.AllCount)
					player.LotteryModel.UpdateOnceEffectiveCount(dataLotteryId, dataDetail.OnceEffectiveCount+1)
					player.LotteryModel.UpdateLastChengeTime(dataLotteryId, tool.UnixNowMilli())
					for _, itemInfo := range itemIdList {
						res = append(res, &pb.ItemBasicInfo{
							ItemId: itemInfo.ID,
							Count:  itemInfo.Num,
						})
					}
				} else {
					// 普通保底
					if cfg.Guarantees != nil && dataDetail.AllCount-dataDetail.LastBasicGuaranteeNum >= cfg.Guarantees.Num-int32(lossValue) {
						player.LotteryModel.BasicLotteryGuarantee(cfg, &res, true, false, dataLotteryId)
					} else {
						player.LotteryModel.BasicLottery(cfg, &res, true, false, dataLotteryId)
					}
				}
			}
		}
	}
	lotteryLuckyEvent := player.LotteryModel.GetLotteryLuckyEvent()
	if lotteryNum == 1 && lotteryDetail.FirstDropFree < cfg.FirstDropFree {
		player.LotteryModel.UpdateFirstDropFree(lotteryId, lotteryDetail.FirstDropFree+1)
		player.LotteryModel.UpdateLastFirstDropFreeTime(lotteryId, tool.UnixNowMilli())
	} else {
		removeItems := &gameConfig.ItemConfig{ID: gameConfig.GetSummonPoolCfg(req.LotteryId).DropToken, Num: int64(lotteryNum)}
		// 抽奖类型：英雄卡池          数量：十抽          卡池类型：限定卡池
		if cfg.Gashatype == 1 && lotteryNum == 10 && cfg.ActModID != 0 {
			luckyEventCfgTime := gameConfig.GetConstantCfg(gameConfig.CONSTANT_limitedGachaLuckyEventTime)
			// 事件存在，折数存在，且未过期
			if lotteryLuckyEvent != nil && lotteryLuckyEvent.LuckyNum > 0 && (tool.UnixNowMilli()-lotteryLuckyEvent.CreateTime) < int64(luckyEventCfgTime.Value[0]*1000) {
				removeItems.Num = removeItems.Num * int64(lotteryLuckyEvent.LuckyNum) / 10
			}
			err := player.LotteryModel.CreateLotteryLuckyEvent()
			if err != nil {
				platformLogger.ErrorWithUser("create lottery lucky event error", player, err)
				messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_CREATE_LOTTERY_LUCKY_EVENT_ERROR)
				return
			}
		}

		err := itemService.RemoveItem(player, removeItems, enum.ITEM_CHANGE_REASON_DRAW_LOTTERY)
		if err != nil {
			platformLogger.InfoWithUser("扣除材料失败", player)
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, pb.ERROR_CODE_REMOVE_ITEM_ERROR)
			return
		}
	}
	if cfg.Gashatype == 1 && !lotteryLuckyEvent.IsNewLuckyReward {
		player.LotteryModel.UpdateNewLuckyCount(lotteryLuckyEvent.NewLuckyCount + lotteryNum)
	}
	items := make([]*gameConfig.ItemConfig, 0)
	for _, v := range res {
		items = append(items, &gameConfig.ItemConfig{
			ID:  v.ItemId,
			Num: v.Count,
		})
	}
	err = itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_LOTTERY)
	if err != nil {
		platformLogger.ErrorWithUser("add item error", player, err)
		return
	}
	resp := &pb.LotteryResp{
		Result:   pb.LotteryResult_LOTTERY_RESULT_SUCCESS,
		InfoList: res,
		LotteryInfo: &pb.LotteryInfo{
			LotteryCount:         dataDetail.AllCount,
			LastBasicGuaranteNum: dataDetail.LastBasicGuaranteeNum,
			IsDirty:              dataDetail.IsDirty,
		},
		FirstDropFree: player.LotteryModel.LotteryEntities[req.LotteryId].FirstDropFree,
		NewLuckyCount: lotteryLuckyEvent.NewLuckyCount,
	}
	if cfg.ActModID != 0 {
		resp.LotteryLuckyEvent = &pb.LotteryLuckyEvent{
			CreateTime: lotteryLuckyEvent.CreateTime,
			LuckyNum:   lotteryLuckyEvent.LuckyNum,
		}
	}
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOTTERY_RESP, resp)

	eventBusService.SubmitLuckyLotteryEvent(player.GetUserId(), enum.AllLotteryType[cfg.Gashatype], lotteryNum, items)

	// 上报抽奖日志
	operationLogService.OnUserHeroLottery(player.GetUserId(), lotteryNum)

	// 记录今日抽卡数据到 Redis
	ctx := context.Background()
	if cfg.Gashatype == 2 {
		player.StaticData.UpdateCollectionLotteryDrawCount(player.StaticData.GetCollectionLotteryDrawCount() + lotteryNum)
		err := unlockSvc.DailyCache.RecordCollectionLottery(ctx, player.GetUserId(), lotteryNum)
		if err != nil {
			platformLogger.ErrorWithUser("record collection lottery to redis error", player, err)
		}
	}
	for _, item := range items {
		err := unlockSvc.DailyCache.RecordLottery(ctx, player.GetUserId(), lotteryId, item.ID, int32(item.Num))
		if err != nil {
			platformLogger.ErrorWithUser("record lottery to redis error", player, err)
		}
	}
}

func NewLuckyRewardHandle(message proto.Message, player *model.PlayerModel) {
	req, ok := message.(*pb.NewLuckyRewardReq)
	if !ok {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	newLuckyNumCfg := gameConfig.GetConstantCfg(gameConfig.CONSTANT_beginnerBenefitsNumbers)
	if newLuckyNumCfg == nil || len(newLuckyNumCfg.Value) < 0 {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_CFG_NOT_FOUND)
		return
	}
	flag := player.LotteryModel.RewardNewLucky(req.ItemId)
	if !flag {
		messageSender.SendErrorMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, pb.ERROR_CODE_NEW_LUCKY_REWARD_ERROR)
		return
	}
	itemService.AddItem(player, &gameConfig.ItemConfig{ID: req.ItemId, Num: 1}, enum.ITEM_CHANGE_REASON_NEW_LUCKY_REWARD)
	messageSender.SendMessage(player, pb.MESSAGE_ID_NEW_LUCKY_REWARD_RESP, &pb.NewLuckyRewardResp{})
}
