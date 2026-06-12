package gameController

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/platform/dispatcherService"
	"github.com/drop/GoServer/server/logic/platform/gamePlatform"

	"github.com/drop/GoServer/server/service/dbService"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/drop/GoServer/server/service/serviceInterface"

	"github.com/drop/GoServer/server/logic/pet"
	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/loginMutexService"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"

	"github.com/drop/GoServer/server/logic/vipCard"

	"github.com/drop/GoServer/server/logic/raid"

	"github.com/drop/GoServer/server/logic/cityAge"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/lumber"
	"github.com/drop/GoServer/server/logic/trial"
	"github.com/drop/GoServer/server/service/logger"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	"github.com/drop/GoServer/server/logic/task"

	"github.com/drop/GoServer/server/logic/hero"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func init() {
	RegisterController("login", &LoginController{})
}

type LoginController struct {
}

var _ LogicControllerInterface = (*LoginController)(nil)

type loginParam struct {
	IsNewPlayer  bool
	IsLoadPlayer bool
}

func (l *LoginController) RegisterLogicMessage() {
	RegisterLoginMessageHandler(enum.MSG_TYPE_LOGIN, pb.MESSAGE_ID_LOGIN_REQ, &pb.LoginReq{}, LoginReqHandle)

	RegisterPlayerInnerTask(enum.INNER_MEG_AFTER_LOGIN, afterLoginHandler)
}

func LoginReqHandle(message proto.Message, user logicCommon.UserBaseInterface) {
	req, ok := message.(*pb.LoginReq)
	if !ok {
		platformLogger.ErrorWithUser("message error", user, nil)
		messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PB_CONV_ERROR)
		return
	}
	loginUser, ok := user.(*model.LoginUser)
	if !ok {
		platformLogger.ErrorWithUser("user is not login user", user, nil)
		messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_PLAYER_CONV_ERROR)
		return
	}
	if req.Account == "" {
		platformLogger.ErrorWithUser("account is empty", user, nil)
		messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ACCOUNT_ERROR)
		return
	}
	loginUser.Account = req.Account
	loginUser.ServerId = req.ServerId
	loginUser.NodeId = nodeConfig.NodeId

	func() {
		defer func() {
			if r := recover(); r != nil {
				loginMutexService.ExitMutex(loginUser.Account, user.GetSession().GetID())
				logger.ErrorBySprintf("login err player:%s panic:%+v", loginUser.Account, r)
			}
		}()
		if ok = loginMutexService.EnterMutex(loginUser.Account, user.GetSession().GetID()); !ok {
			platformLogger.ErrorWithUser("account already login", user, nil)
			messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ALREADY_ERROR)
			return
		}

		platformLogger.InfoWithUser("account begin login", user)
		IsNewPlayer := false
		IsLoadPlayer := false

		gameSessionManger := sessionManager.(*logicSessionManager.GameSessionManager)
		if gameSessionManger == nil {
			platformLogger.ErrorWithUser("session manager is nil", user, nil)
			messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			return
		}
		basePlayer := gameSessionManger.GetPlayerBasicInfoByUserId(loginUser.UserId)
		if basePlayer != nil {
			player, ok := basePlayer.(*model.PlayerModel)
			if !ok {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				platformLogger.ErrorWithUser("transfer to playerModel error", user, nil)
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			platformLogger.InfoWithUser("player already in server", user)
			if player.IsOnline() {
				// 相同的连接又发了一次登录
				if player.GetSession().GetID() == user.GetSession().GetID() {
					loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
					platformLogger.InfoWithUser("player already login", user)
					return
				}
				// 关闭老的连接
				messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REPLACE_PLAYER_ERROR)
			}
			// 绑定新连接
			err := rebindPlayer(loginUser, player)
			if err != nil {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				platformLogger.ErrorWithUser("rebind player error", user, err)
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_REBIND_PLAYER_ERROR)
				return
			}
			// 更新副本状态
			raid.ResetCurrentStage(player.PlayerInstanceModel.CurrentRaidInfo, player.StaticData.GetDailyPrivilegeDrop(), player)
		} else {
			platformLogger.InfoWithUser("player not in server", user)

			// 检查玩家是否存在
			userEntity, err := checkPlayerExist(user.GetUserAccount(), user.GetUserServerId())
			if err != nil {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				platformLogger.ErrorWithUser("get user error", user, err)
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_CHECK_PLAYER_ERROR)
				return
			}
			var player *model.PlayerModel
			if userEntity != nil {
				platformLogger.InfoWithUser("begin load player from db", user)
				IsLoadPlayer = true
				player, err = LoadPlayer(userEntity, true)
				if err != nil {
					loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
					platformLogger.ErrorWithUser("load player error", user, err)
					messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_LOAD_PLAYER_ERROR)
					return
				}
			} else {
				platformLogger.InfoWithUser("begin create player", user)
				player, err = createPlayer(loginUser)
				IsNewPlayer = true
				IsLoadPlayer = true
				if err != nil {
					loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
					platformLogger.ErrorWithUser("create player error", user, err)
					messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_CREATE_PLAYER_ERROR)
					return
				}
			}
			// 先绑定 session，确保后续操作可以发送消息
			gameSessionManger.BindWithSession(loginUser, player)

			// 处理副本
			mainInstance := player.PlayerInstanceModel.InstanceEntities[int32(enum.MAIN_INSTANCE_ID)]
			if mainInstance == nil {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			extraDrop := int32(0)
			count, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_MAIN_REWARD)
			if err == nil && count > 0 {
				extraDrop = player.StaticData.GetDailyPrivilegeDrop()
			}
			current, next, err := raid.BuildAllMainInstanceData(player.GetUserId(), mainInstance.StageId, mainInstance.CurrentSubStageId, mainInstance.MaxStageId, mainInstance.MaxSubStageId, mainInstance.Info, extraDrop)
			if err != nil {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
				return
			}
			player.PlayerInstanceModel.CurrentRaidInfo = current
			player.PlayerInstanceModel.CurrentMainInstanceInfo = current
			player.PlayerInstanceModel.NextMainInstanceInfo = next

			err = raid.LoginEnterScene(player)
			if err != nil {
				loginMutexService.ExitMutex(req.Account, user.GetSession().GetID())
				platformLogger.ErrorWithUser("enter scene error", user, err)
				messageSender.CloseSessionWithError(user, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
				return
			}
			basePlayer = player
		}
		dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MEG_AFTER_LOGIN, basePlayer.GetUserId(), &loginParam{IsLoadPlayer: IsLoadPlayer, IsNewPlayer: IsNewPlayer}, 0, 0, nil)
	}()
}

func afterLoginHandler(messageTask serviceInterface.InnerTaskInterface) (any, error) {
	p := sessionManager.GetPlayerBasicInfoByUserId(messageTask.GetReqId())
	if p == nil {
		logger.ErrorBySprintf("player is not in server %d", messageTask.GetReqId())
		return nil, nil
	}
	player, ok := p.(*model.PlayerModel)
	if !ok {
		logger.ErrorBySprintf("player model transfer error %d", messageTask.GetReqId())
		return nil, nil
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				loginMutexService.ExitMutex(player.GetUserAccount(), player.GetSession().GetID())
				logger.ErrorBySprintf("afterLogin err player:%s panic:%+v", player.GetUserAccount(), r)
				messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
			}
		}()

		innerTask, ok := messageTask.(*dispatcherService.InnerTask)
		if !ok {
			logger.ErrorBySprintf("message error")
			return
		}
		if innerTask.ReqParameter == nil {
			logger.ErrorBySprintf("message error")
			return
		}
		param, ok := innerTask.ReqParameter.(*loginParam)
		if !ok {
			logger.ErrorBySprintf("message error")
			return
		}
		ok = afterLogin(player, param)
		if !ok {
			return
		}
		pushLoginMessage(player)
		enum.PublishLogin(dbService.RDB, player.GetUserId(), player.GetUserAccount(), 0)

		//RetryDeliverFieldItemHandler(player)
	}()
	return nil, nil
}

func afterLogin(player *model.PlayerModel, param *loginParam) bool {
	if param.IsNewPlayer {
		cfg := gameConfig.GetBaseCfg()
		if cfg != nil {
			_ = itemService.AddItems(player, cfg.Item, enum.ITEM_CHANGE_REASON_CREATE_PLAYER)
		}
	}
	// 登录之后先心跳一次，处理跨天
	for _, pModel := range player.PlayerModels {
		if param.IsLoadPlayer {
			pModel.Heartbeat(player.User.GetLastOfflineTime(), tool.UnixNowMilli(), tool.GetNatureDayDistance(tool.UnixNowMilli(), player.User.GetLastOfflineTime()), false)
		}
	}
	player.LastHeartbeatTime = tool.UnixNowMilli()

	// 处理副本结束，传出副本
	if player.PlayerInstanceModel.CurrentRaidInfo.InstanceID != enum.MAIN_INSTANCE_ID && player.PlayerInstanceModel.CurrentRaidInfo.IsOver {
		raid.OnLeaveRaid(player.PlayerInstanceModel.CurrentRaidInfo)
		err := raid.EnterScene(player, player.PlayerInstanceModel.CurrentMainInstanceInfo)
		if err != nil {
			messageSender.SendErrorMessage(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_LOGIN_ENTER_SCENE_ERROR)
			return false
		}
		dispatcher.DispatchInnerMessageTask(enum.INNER_MSG_TYPE_PLAYER, enum.INNER_MEG_AFTER_LOGIN, player.GetUserId(), param, 0, 0, nil)
		return false
	}
	// 功能解锁
	for funcId, cfg := range gameConfig.GetAllSystemUnlockCfg() {
		status := int32(0)
		isShow := true
		isOpen := true
		for _, show := range cfg.ShowId {
			if !unlockService.CheckUnlock(show, player) {
				isShow = false
				break
			}
		}
		for _, unlock := range cfg.UnlockId {
			if !unlockService.CheckUnlock(unlock, player) {
				isOpen = false
				break
			}
		}
		if isShow {
			status = 1
		}
		if isOpen {
			status = 2
		}

		if current, ok := player.PlayerFunctionModel.FunctionStatus[enum.FunctionIdEnum(funcId)]; ok {
			if status == current {
				continue
			}
			player.PlayerFunctionModel.FunctionStatus[enum.FunctionIdEnum(funcId)] = status
		} else {
			if status == 0 {
				continue
			}
			player.PlayerFunctionModel.FunctionStatus[enum.FunctionIdEnum(funcId)] = status
		}
	}

	// 处理邮件系统（检查全服邮件并推送红点）
	mailSvc := GetMailService()
	if mailSvc != nil {
		if err := mailSvc.OnPlayerLogin(player.GetUserId()); err != nil {
			platformLogger.ErrorWithUser("mail service OnPlayerLogin failed", player, err)
		}
	}

	// 通行证过期未领奖励补发（登录时检测该玩家所在服已结束活动）
	passService.ProcessExpiredPassMailsForPlayer(player)
	// 更新通行证进度（登录天数类型）
	passService.GetAllPassInfo(player)
	passService.UpdateAllPassProgressBySystem(player, 201) // 201=登录天数

	// 月卡周卡
	for _, item := range player.PlayerShopModel.PassEntities {
		cfg := gameConfig.GetWeeklyPassCfg(item.ShopItemId)
		if cfg == nil {
			continue
		}
		if item.AcceptCount >= cfg.ValidityPeriod {
			continue
		}
		shopItemCfg := gameConfig.GetStillShopCfg(item.ShopItemId)
		if shopItemCfg == nil {
			continue
		}
		if shopItemCfg.TypeId != int32(enum.ShopItemTypeWeekly) {
			continue
		}
		for _, unlock := range shopItemCfg.UnlockBuy {
			if !unlockService.CheckUnlock(unlock, player) {
				continue
			}
		}
		buyItem := player.PlayerShopModel.GetShopItemInfo(item.ShopItemId)
		if buyItem == nil {
			continue
		}
		if tool.GetNatureDayDistance(tool.UnixNowMilli(), buyItem.LastBuyTime) < 1 {
			continue
		}

		canAcceptDay := tool.GetNatureDayDistance(tool.UnixNowMilli(), item.LastAcceptTime)
		if canAcceptDay < 2 {
			continue
		}
		realAcceptCount := int32(0)
		if item.AcceptCount+canAcceptDay > cfg.ValidityPeriod {
			realAcceptCount = cfg.ValidityPeriod - item.AcceptCount
		} else {
			realAcceptCount = canAcceptDay - 1
		}

		itemList := make([]*gameConfig.ItemConfig, 0)
		for i := int32(0); i < realAcceptCount; i++ {
			day := item.AcceptCount + i + 1
			if day > int32(len(cfg.DropId)) {
				logger.ErrorBySprintf("week drop error weekId:%d,day:%d,playerId:%d", cfg.Id, day, player.GetUserId())
				break
			}
			list := gameConfig.Drop(cfg.DropId[day-1])
			itemList = append(itemList, list...)
		}
		_, err := mailService.SendRewardMailByTemplateID(player.GetUserId(), 8001, itemList, nil, nil)
		if err != nil {
			logger.ErrorBySprintf("send mail error mailId:%d,playerId:%d", 8001, player.GetUserId())
		}
		player.PlayerShopModel.UpdateShopPassData(item, tool.UnixNowMilli()-tool.DAY_MILLI, item.AcceptCount+realAcceptCount)
	}
	player.PlayerShopModel.PushedItemId = make(map[int32]struct{})

	// 玩家活动处理
	player.PlayerActivityModel.CheckNewActivity()

	// 签到处理
	if player.PlayerSignModel != nil {
		player.PlayerSignModel.SyncAllDaySignOnLogin(tool.UnixNowMilli())
	}

	player.PlayerArenaModel.UpdateScoreInterval = model.UPDATE_INIT_INTERVAL
	player.BuildPlayerCacheInfo()
	err := logicCommon.UpdatePlayerRedisInfo(player.PlayerCacheInfo)
	if err != nil {
		logger.ErrorBySprintf("update player cache info error:%v,playerId:%d", err, player.GetUserId())
	}
	return true
}

func pushLoginMessage(player *model.PlayerModel) {
	stacks, err := invService.GetItemList(player.GetUserId())
	if err != nil {
		loginMutexService.ExitMutex(player.User.GetAccount(), player.GetSession().GetID())
		platformLogger.ErrorWithUser("invService GetItemLists is nil", player, err)
		messageSender.CloseSessionWithError(player, pb.MESSAGE_ID_LOGIN_RESP, pb.ERROR_CODE_SYSTEM_ERROR)
		return
	}

	inventoryData := &pb.InventoryData{
		Items: make([]*pb.ItemInfo, 0),
	}
	for _, s := range stacks {
		inventoryData.Items = append(inventoryData.Items, &pb.ItemInfo{ItemId: s.ItemId, ItemNum: s.ItemNum, ItemUid: 0})
	}

	openInfo := make([]*pb.FunctionOpenInfo, 0)
	unlockTime := int64(0)
	for funcId, status := range player.PlayerFunctionModel.FunctionStatus {
		rewardCommited := int32(0)
		if entity := player.PlayerFunctionModel.Get(int32(funcId)); entity != nil {
			rewardCommited = entity.RewardCommited
			unlockTime = entity.UnlockTime
		}
		openInfo = append(openInfo, &pb.FunctionOpenInfo{
			FuncId:         int32(funcId),
			Status:         status,
			RewardCommited: rewardCommited,
			UnlockTime:     unlockTime,
		})
	}

	// 组装特权卡信息
	vipCards, err := vipCard.Service.GetVipCardInfoList(player)
	if err != nil {
		platformLogger.ErrorWithUser("GetVipCardInfoList failed", player, err)
		vipCards = nil
	}

	// 获取副本信息
	sceneInfo := raid.BuildRaidPB(player, player.PlayerInstanceModel.CurrentRaidInfo)

	allActivity := player.PlayerActivityModel.GetAllOpenActivity()

	// 特权奖励领取信息（用于客户端显示“是否已领/上次领取时间”）
	vipCardReward := make([]*pb.ClaimPrivilegeRewardInfo, 0)
	cfgMap := gameConfig.GetAllPrivilegeRewardCfg()
	if len(cfgMap) > 0 {
		keys := make([]int, 0, len(cfgMap))
		for rewardType := range cfgMap {
			keys = append(keys, int(rewardType))
		}
		sort.Ints(keys)
		for _, k := range keys {
			rt := int32(k)
			last := int64(0)
			if player.PrivilegeRewardModel != nil {
				if ent := player.PrivilegeRewardModel.Rewards[rt]; ent != nil {
					last = ent.LastClaimTime
				}
			}
			vipCardReward = append(vipCardReward, &pb.ClaimPrivilegeRewardInfo{
				RewardType:    rt,
				LastClaimTime: last,
			})
		}
	}

	// 登录时把所有已触发剧情的次数下发给客户端：storyId -> count
	storyMap := map[int32]int32{}
	if player.StoryTriggerModel != nil {
		storyMap = player.StoryTriggerModel.GetAllStoryTriggers()
	}

	// Pet star book: petId -> history max star.
	petStarBook := make(map[int32]int32)
	if player.PetAffinityModel != nil && player.PetAffinityModel.BookEntities != nil {
		for petID, ent := range player.PetAffinityModel.BookEntities {
			if ent == nil {
				continue
			}
			if petID <= 0 {
				continue
			}
			if ent.MaxStar < 0 {
				continue
			}
			petStarBook[petID] = ent.MaxStar
		}
	}
	serverOpenTime := int64(0)
	serverInfo := gamePlatform.GetServerInfoService().GetServerInfo(player.GetUserServerId())
	if serverInfo != nil {
		serverOpenTime = serverInfo.ServerTime
	}

	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(player.GetUserId())

	dailyTempBattleSpeedUpCount := player.StaticData.GetBattleSpeedUpTimes()
	if dailyTempBattleSpeedUpCount > 0 {
		dailyTempBattleSpeedUpCount = player.StaticData.GetDailyBattleSpeedUpTimes()
	} else {
		dailyTempBattleSpeedUpCount = -1
	}
	res := &pb.LoginResp{
		BasicInfo: &pb.PlayerBasicInfo{
			UserId:      player.GetUserId(),
			ServerId:    player.GetUserServerId(),
			NickName:    player.User.GetNickname(),
			HeadId:      player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHead),
			HeadFrameId: player.AppearanceModel.GetWearAppearance(enum.AvatarTypeHeadFrame),
			TitleId:     player.AppearanceModel.GetWearAppearance(enum.AvatarTypeTitle),
			Level:       player.ArchitectureModel.GetMainLevel(),
			BubbleId:    player.AppearanceModel.GetWearAppearance(enum.AvatarTypeBubble),
			ImageId:     player.AppearanceModel.GetWearAppearance(enum.AvatarTypeImage),
		},
		Inventory:           inventoryData,
		HeroBagDetailsMap:   hero.GetHeroBagInfo(player),
		Formations:          hero.GetHeroFormationInfo(player),
		ChangeNicknameTimes: player.StaticData.GetChangeNicknameTimes(),
		SceneInfo:           sceneInfo,
		TaskInfoList:        task.PlayerLoginLoadTask(player),
		OpenInfos:           openInfo,
		BountyinfoList:      task.PlayerLoginLoadBounty(player),
		VipCards:            vipCards,
		ActivityInfos:       allActivity,
		PassCardTask:        task.PlayerLoginLoadPassTask(player),
		VipCardReward:       vipCardReward,
		StoryMap:            storyMap,
		SystemInfo: &pb.SystemInfo{
			Timestamp:      tool.UnixNowMilli(),
			ServerOpenTime: serverOpenTime,
		},
		AccountBind:   model.GetAllChannelBind(player.GetUserId()),
		MaxStageId:    player.PlayerInstanceModel.GetMainInstanceMaxStageId(),
		IsCreate:      player.User.GetLastLoginTime() == 0,
		IsRecharged:   player.User.GetChargeCount() != 0,
		PetBagDetails: pet.GetPetBagInfo(player),
		PetStarBook:   petStarBook,
		AllianceInfo: &pb.PlayerAllianceInfo{
			AllianceId:   allianceInfo.AllianceId,
			AllianceName: allianceInfo.AllianceName,
		},
		BuyDispatchFormationNum:   player.StaticData.GetBuyDispatchFormationNum(),
		DailyTempBattleSpeedCount: dailyTempBattleSpeedUpCount,
	}

	player.User.UpdateLastLoginTime(tool.UnixNowMilli())
	player.UpdatePlayerBasicInfoToRedis()
	messageSender.SendMessage(player, pb.MESSAGE_ID_LOGIN_RESP, res)
	loginMutexService.ExitMutex(player.User.GetAccount(), player.GetSession().GetID())
	platformLogger.InfoWithUser("login success", player)
}

func rebindPlayer(user logicCommon.UserBaseInterface, oldPlayer *model.PlayerModel) error {
	platformLogger.InfoWithUser(" rebind player", user)
	gameSessionManager := sessionManager.(*logicSessionManager.GameSessionManager)
	gameSessionManager.ReplaceSession(user.GetSession(), oldPlayer)
	return nil
}

func checkPlayerExist(account string, serverId int32) (*model.UserEntity, error) {
	userEntity, err := easyDB.GetPlayerEntityByWhere[model.UserEntity](map[string]interface{}{"account": account, "server_id": serverId})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return userEntity, nil
}

func createPlayer(user *model.LoginUser) (*model.PlayerModel, error) {
	player := model.NewPlayerModel()
	err := CreatePlayer(player, user, user.UserId, user.GetUserServerId())
	if err != nil {
		return nil, err
	}
	dbService.RDB.Incr(context.Background(), enum.GetRegisterConst(user.GetUserServerId()))
	return player, nil
}

func LoadPlayer(userEntity *model.UserEntity, isLogin bool) (*model.PlayerModel, error) {
	player := model.NewPlayerModel()
	err := loadPlayerFromDB(player, userEntity, isLogin)
	if err != nil {
		return nil, err
	}
	return player, nil
}

// create和load没有放到对应的model，是为了避免包的循环引用问题
func loadPlayerFromDB(player *model.PlayerModel, userEntity *model.UserEntity, isLogin bool) error {
	userModel := model.NewUserModel(userEntity, player)
	player.User = userModel
	player.AppendPlayerModel(userModel)

	staticModel, err := model.LoadStaticDataModel(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			player.StaticData, err = model.CreateStaticDataModel(userEntity.UserId)
		}
		if err != nil {
			logger.ErrorWithZapFields("CreateStaticDataModel error")
			return err
		}
	} else {
		player.StaticData = staticModel
	}
	player.AppendPlayerModel(player.StaticData)

	inventoryModel, err := model.LoadInventoryModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load inventory model error")
		return err
	}
	player.InventoryModel = inventoryModel
	player.PlayerModels = append(player.PlayerModels, player.InventoryModel)

	// 剧情触发次数模型：单独表 player_story_trigger
	storyTriggerModel, err := model.LoadStoryTriggerModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("LoadStoryTriggerModel error")
		return err
	}
	player.StoryTriggerModel = storyTriggerModel
	player.AppendPlayerModel(storyTriggerModel)

	heroModel, err := model.LoadHeroBags(userEntity.UserId, player, isLogin)
	if err != nil {
		logger.ErrorWithZapFields("load hero model error")
		return err
	}
	player.HeroDetailsModel = heroModel
	player.AppendPlayerModel(heroModel)
	player.HeroAttrModels = append(player.HeroAttrModels, heroModel)

	taskModel, err := model.LoadTaskModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load task model error")
		return err
	}
	player.TaskModel = taskModel
	player.AppendPlayerModel(taskModel)

	cityAgeModel, ok := cityAge.Service.GetOrLoadModel(player, false)
	if !ok {
		logger.ErrorWithZapFields("load city age model error")
		return errors.New("load city age model error")
	}
	player.HeroAttrModels = append(player.HeroAttrModels, cityAgeModel)

	equipModel, err := model.LoadEquipmentBags(userEntity.UserId, player)
	if err != nil {
		logger.ErrorWithZapFields("load equipment model error")
	}
	player.EquipmentModel = equipModel
	player.AppendPlayerModel(equipModel)
	player.HeroAttrModels = append(player.HeroAttrModels, equipModel)

	player.LotteryModel, err = model.LoadLotteryModel(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load lottery model error")
			return err
		}
	}
	player.AppendPlayerModel(player.LotteryModel)

	player.LoopBoxModel, err = model.LoadLoopBoxModel(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load loopBox model error")
			return err
		}
	}
	player.AppendPlayerModel(player.LoopBoxModel)

	player.PassTaskModel, err = model.LoadPassCardTask(userEntity.UserId, player)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load loopBox model error")
			return err
		}
	}
	player.AppendPlayerModel(player.PassTaskModel)

	player.AllianceDailyActivityModel, err = model.LoadAllianceDailyActivityModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load alliance daily activity model error")
		return err
	}
	player.AppendPlayerModel(player.AllianceDailyActivityModel)

	if player.AlbumRewardModel == nil {
		player.AlbumRewardModel = model.NewAlbumRewardModel(userEntity.UserId, model.LoadAlbumRewardScore(userEntity.UserId))
		player.AppendPlayerModel(player.AlbumRewardModel)
	}
	if player.HeroAlbumModel == nil {
		player.HeroAlbumModel = model.NewHeroAlbumCollectionModel(userEntity.UserId, model.LoadAlbum(userEntity.UserId))
		player.AppendPlayerModel(player.HeroAlbumModel)
	}
	if player.HeroFormationModel == nil {
		player.HeroFormationModel = model.NewHeroFormationCollectionModel(userEntity.UserId, model.LoadHeroFormations(userEntity.UserId))
		player.AppendPlayerModel(player.HeroFormationModel)
	}
	if player.EquipmentModel == nil {
		player.EquipmentModel = model.NewEquipmentCollectionModel(userEntity.UserId, make(map[int64]*model.EquipmentEntity), player)
		player.AppendPlayerModel(player.EquipmentModel)
		player.HeroAttrModels = append(player.HeroAttrModels, player.EquipmentModel)
	}

	sceneDataModel, err := model.LoadPlayerInstanceModel(player)
	if err != nil {
		return err
	}
	player.PlayerInstanceModel = sceneDataModel
	player.AppendPlayerModel(player.PlayerInstanceModel)

	playerShopModel, err := model.LoadPlayerShopModel(userEntity.UserId, player)
	if err != nil {
		return err
	}
	player.PlayerShopModel = playerShopModel
	player.AppendPlayerModel(player.PlayerShopModel)

	expeditionModel, err := model.LoadExpeditionModel(player)
	if err != nil {
		return err
	}
	player.ExpeditionModel = expeditionModel
	player.AppendPlayerModel(player.ExpeditionModel)

	accessoryMdoel, err := model.LoadAccessory(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load loopBox model error")
			return err
		}
	}
	player.AccessoryModel = accessoryMdoel
	player.AppendPlayerModel(accessoryMdoel)
	player.HeroAttrModels = append(player.HeroAttrModels, accessoryMdoel)

	// Load pet model.
	petModel, err := model.LoadPetModel(userEntity.UserId, isLogin)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load pet model error")
			return err
		}
	}
	player.PetModel = petModel
	player.AppendPlayerModel(player.PetModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.PetModel)

	// 宠物缘分模型：激活/升级状态（对所有英雄加成）
	petAffinityModel, err := model.LoadPetAffinityModel(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load petAffinity model error")
			return err
		}
	}
	petAffinityModel.BindPetModel(player.PetModel)
	player.PetAffinityModel = petAffinityModel
	player.AppendPlayerModel(player.PetAffinityModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.PetAffinityModel)

	petRecruitModel, err := model.LoadOrCreatePetRecruitModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load pet recruit model error")
		return err
	}
	player.PetRecruitModel = petRecruitModel
	player.AppendPlayerModel(player.PetRecruitModel)

	accessoryLuckyModel, err := model.LoadAccessoryLucky(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load loopBox model error")
			return err
		}
	}
	player.AccessoryLuckyModel = accessoryLuckyModel
	player.AppendPlayerModel(accessoryLuckyModel)

	bountyModel, err := model.LoadBounty(userEntity.UserId, player)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		} else {
			logger.ErrorWithZapFields("load bounty model error")
			return err
		}
	}
	player.BountyModel = bountyModel
	player.AppendPlayerModel(player.BountyModel)

	taskActiveRewardModel, err := model.LoadTaskActiveRewardModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load taskActiveRewardModel model error")
		return err
	}
	player.TaskActiveRewardModel = taskActiveRewardModel
	player.AppendPlayerModel(player.TaskActiveRewardModel)

	playerActivityModel, err := model.LoadPlayerActivityModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load player activity model error")
		return err
	}
	player.PlayerActivityModel = playerActivityModel
	player.AppendPlayerModel(player.PlayerActivityModel)

	turnTableModel, err := model.LoadTurnTableModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load turn table model error")
		return err
	}
	player.TurnTableModel = turnTableModel
	player.PlayerModels = append(player.PlayerModels, player.TurnTableModel)

	signModel, err := model.LoadPlayerSignModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load player sign model error")
		return err
	}
	player.PlayerSignModel = signModel
	player.AppendPlayerModel(player.PlayerSignModel)

	playerFunctionModel, err := model.LoadPlayerFunctionModel(userEntity.UserId, player)
	if err != nil {
		logger.ErrorBySprintf("load playerFunctionModel model error")
		return err
	}
	player.PlayerFunctionModel = playerFunctionModel
	player.AppendPlayerModel(playerFunctionModel)

	idleModel, err := model.LoadIdleModel(userEntity.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			idleModel, err = model.CreateIdleModel(userEntity.UserId)
			if err != nil {
				logger.ErrorWithZapFields("create idle model error")
				return err
			}
		} else {
			logger.ErrorWithZapFields("load idle model error")
			return err
		}
	}
	player.IdleModel = idleModel
	player.AppendPlayerModel(idleModel)

	playerAdventureModel, err := model.LoadPlayerAdventureModel(userEntity.UserId, player)
	if err != nil {
		logger.ErrorWithZapFields("load player adventure model error")
		return err
	}
	player.PlayerAdventureModel = playerAdventureModel
	player.AppendPlayerModel(playerAdventureModel)

	// Load vip card model.
	vipCardModel, err := model.LoadVipCardModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load vipCard model error")
		return err
	}
	player.VipCardModel = vipCardModel
	player.AppendPlayerModel(vipCardModel)

	// Load pass model.
	passModel, err := model.LoadPassModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load vipCard model error")
		return err
	}
	player.PassModel = passModel
	player.AppendPlayerModel(passModel)

	// 鐗规潈濂栧姳
	privilegeRewardModel, err := model.LoadPrivilegeRewardModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load privilegeReward model error")
		return err
	}
	player.PrivilegeRewardModel = privilegeRewardModel
	player.AppendPlayerModel(privilegeRewardModel)

	arenaModel, err := model.LoadPlayerArenaModel(player)
	if err != nil {
		logger.ErrorWithZapFields("load arena model error")
		return err
	}
	player.PlayerArenaModel = arenaModel
	player.AppendPlayerModel(player.PlayerArenaModel)

	gloryArenaModel, err := model.LoadPlayerGloryArenaModel(player)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			gloryArenaModel, err = model.CreatePlayerGloryArenaModel(player)
		}
		if err != nil {
			logger.ErrorWithZapFields("load glory arena model error")
			return err
		}
	}
	player.PlayerGloryArenaModel = gloryArenaModel
	player.PlayerModels = append(player.PlayerModels, player.PlayerGloryArenaModel)

	tokenShopModel, err := model.LoadPlayerTokenShop(player.GetUserId())
	if err != nil {
		logger.ErrorWithZapFields("load tokenShop model error")
		return err
	}
	player.PlayerTokenShopModel = tokenShopModel
	player.AppendPlayerModel(player.PlayerTokenShopModel)

	// 建筑信息
	player.ArchitectureModel, err = model.LoadArchitecture(userEntity.UserId, player)
	if err != nil {
		logger.ErrorBySprintf("load architecture model error")
		return err
	}
	player.AppendPlayerModel(player.ArchitectureModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.ArchitectureModel)

	player.LumberModel, err = model.LoadLumberModel(userEntity.UserId)
	if err != nil {
		logger.ErrorBySprintf("load city production model error")
		return err
	}
	player.AppendPlayerModel(player.LumberModel)

	player.FurnitureModel, err = model.LoadFurnitureModel(userEntity.UserId)
	if err != nil {
		logger.ErrorBySprintf("load city furniture model error")
		return err
	}
	player.AppendPlayerModel(player.FurnitureModel)

	trialModel, err := model.LoadTrialModel(userEntity.UserId)
	if err != nil {
		logger.ErrorBySprintf("load trial model error")
		return err
	}
	player.TrialModel = trialModel
	player.AppendPlayerModel(player.TrialModel)

	player.ArchitectureModel.OnUpgradeCallback = func(p *model.PlayerModel, archType int32, oldLevel int32, senderMsg bool) {
		lumber.Service.OnBuildingUpgradeComplete(p, archType, oldLevel)
	}
	player.ArchitectureModel.EnsureUnlockedArchitectureEntities()

	// 七日试炼：过期进度邮件补发、已结束活动清理（见 trial.OnPlayerLoad）。
	if trial.Service != nil {
		trial.Service.OnPlayerLoad(player)
	}

	player.CollectionModel, err = model.LoadCollectionModel(userEntity.UserId)
	if err != nil {
		logger.ErrorBySprintf("load collection model error")
		return err
	}
	player.AppendPlayerModel(player.CollectionModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.CollectionModel)

	player.StoneModel, err = model.LoadStone(userEntity.UserId, player)
	if err != nil {
		logger.ErrorBySprintf("load stone model error")
		return err
	}
	player.AppendPlayerModel(player.StoneModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.StoneModel)

	player.AppearanceModel, err = model.LoadAppearanceModel(userEntity.UserId, player)
	if err != nil {
		logger.ErrorBySprintf("load appearance model error")
		return err
	}
	player.AppendPlayerModel(player.AppearanceModel)
	player.HeroAttrModels = append(player.HeroAttrModels, player.AppearanceModel)

	player.SignInModel, err = model.LoadSignInModel(userEntity.UserId)
	if err != nil {
		logger.ErrorWithZapFields("load sign in model error")
		return err
	}
	player.AppendPlayerModel(player.SignInModel)

	return nil
}

func CreatePlayer(player *model.PlayerModel, user *model.LoginUser, userId int64, serverId int32) error {
	userModel, err := model.CreateUserModel(user.Account, userId, serverId, player)
	if err != nil {
		logger.ErrorWithZapFields("create user model error")
		return err
	}
	player.User = userModel
	player.AppendPlayerModel(userModel)

	staticDataModel, err := model.CreateStaticDataModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create static data model error")
		return err
	}
	player.StaticData = staticDataModel
	player.AppendPlayerModel(staticDataModel)

	player.InventoryModel = model.CreateInventoryModel(userId)
	player.AppendPlayerModel(player.InventoryModel)

	// Init story trigger model.
	player.StoryTriggerModel = model.NewStoryTriggerModel(userId)
	player.AppendPlayerModel(player.StoryTriggerModel)

	heroModel, err := model.CreateHeroModel(userId, player)
	if err != nil {
		logger.ErrorWithZapFields("create hero model error")
		return err
	}
	player.HeroDetailsModel = heroModel
	player.AppendPlayerModel(player.HeroDetailsModel)
	player.AppendHeroAttrModel(player.HeroDetailsModel)

	equipModel, err := model.CreateEquipmentModel(userId, player)
	if err != nil {
		logger.ErrorWithZapFields("create equipment model error")
	}
	player.EquipmentModel = equipModel
	if player.EquipmentModel != nil {
		player.AppendPlayerModel(player.EquipmentModel)
		player.AppendHeroAttrModel(player.EquipmentModel)
	}

	albumRewardModel, err := model.CreateAlbumRewardModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create album reward model error")
		return err
	}
	player.AlbumRewardModel = albumRewardModel
	player.AppendPlayerModel(albumRewardModel)

	if player.HeroAlbumModel == nil {
		player.HeroAlbumModel = model.NewHeroAlbumCollectionModel(userId, make(map[int64]*model.HeroAlbumEntity))
		player.AppendPlayerModel(player.HeroAlbumModel)
	}
	if player.HeroFormationModel == nil {
		player.HeroFormationModel = model.NewHeroFormationCollectionModel(userId, make(map[int32]map[int32]*model.HeroFormationEntity))
		player.AppendPlayerModel(player.HeroFormationModel)
	}
	if player.TaskModel == nil {
		player.TaskModel = model.NewTaskModel(userId, make(map[int32]map[int32]map[int32]*model.TaskEntity), make(map[int32]*model.TaskEntity), player)
		player.AppendPlayerModel(player.TaskModel)
	}
	cityAgeModel, ok := cityAge.Service.GetOrLoadModel(player, true)
	if !ok {
		logger.ErrorWithZapFields("create city age model error")
		return errors.New("create city age model error")
	}
	player.AppendHeroAttrModel(cityAgeModel)
	if player.EquipmentModel == nil {
		player.EquipmentModel = model.NewEquipmentCollectionModel(userId, make(map[int64]*model.EquipmentEntity), player)
		player.AppendPlayerModel(player.EquipmentModel)
		player.AppendHeroAttrModel(player.EquipmentModel)
	}

	player.PlayerInstanceModel = model.CreatePlayerInstanceModel(player)
	player.AppendPlayerModel(player.PlayerInstanceModel)

	player.PlayerShopModel = model.NewPlayerShopModel(userId, player, make(map[int32]*model.PlayerShopItemEntity), make(map[int32]*model.PlayerPassEntity))
	player.AppendPlayerModel(player.PlayerShopModel)

	player.ExpeditionModel, err = model.CreateExpeditionModel(player)
	if err != nil {
		logger.ErrorWithZapFields("create expedition model error")
		return err
	}
	player.AppendPlayerModel(player.ExpeditionModel)

	player.LotteryModel = model.NewLotteryModel(userId, make(map[int32]*model.LotteryEntity), &model.LotteryLuckyEventEntity{}, make(map[int32][]*model.LotteryLog), make(map[int32]map[int32]bool), make(map[int32]map[int32]bool))
	player.AppendPlayerModel(player.LotteryModel)

	player.LoopBoxModel, err = model.CreatLoopBoxModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create loop box model error")
		return err
	}
	player.AppendPlayerModel(player.LoopBoxModel)

	player.AccessoryModel = model.NewAccessoryModel(userId, make(map[int32]*model.AccessoryEntity))
	player.AppendPlayerModel(player.AccessoryModel)
	player.AppendHeroAttrModel(player.AccessoryModel)

	// Create pet model.
	player.PetModel, err = model.CreatePetModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create pet model error")
		return err
	}
	player.AppendPlayerModel(player.PetModel)
	player.AppendHeroAttrModel(player.PetModel)

	// Create pet affinity model.
	player.PetAffinityModel, err = model.CreatePetAffinityModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create petAffinity model error")
		return err
	}
	player.PetAffinityModel.BindPetModel(player.PetModel)
	player.AppendPlayerModel(player.PetAffinityModel)
	player.AppendHeroAttrModel(player.PetAffinityModel)

	player.PetRecruitModel, err = model.CreatePetRecruitModel(player, userId)
	if err != nil {
		logger.ErrorWithZapFields("create pet recruit model error")
		return err
	}
	player.AppendPlayerModel(player.PetRecruitModel)

	player.AccessoryLuckyModel = model.NewAccessoryLuckyModel(make(map[int32]*model.AccessoryLuckyEntity), userId)
	player.AppendPlayerModel(player.AccessoryLuckyModel)

	player.PassTaskModel = model.NewPassCardTaskModel(make(map[int32]*model.PassCardTaskEntity), player, userId, enum.TaskAffiliationPassCard*10000, new([]int32))
	player.AppendPlayerModel(player.PassTaskModel)

	slot := make([]int32, 0)
	player.BountyModel = model.NewBountyModel(make(map[int32]*model.BountyEntity), userId, player, &slot, enum.TaskAffiliationBounty*10000)
	player.AppendPlayerModel(player.BountyModel)

	player.TaskActiveRewardModel = model.NewTaskActiveRewardModel(&model.TaskActiveRewardEntity{}, userId)
	err = player.TaskActiveRewardModel.AddTaskActiveRewardEntity(player.GetUserId())
	if err != nil {
		logger.ErrorWithZapFields("create task active reward model error")
		return err
	}
	player.AppendPlayerModel(player.TaskActiveRewardModel)

	player.ArchitectureModel = model.NewArchitectureModel(make(map[int32]*model.ArchitectureEntity), userId, player)
	player.AppendPlayerModel(player.ArchitectureModel)
	err = player.ArchitectureModel.AddArchitectureEntity(int32(enum.ARCHITECTURE_TYPE_MAIN), 0)
	if err != nil {
		logger.ErrorBySprintf("create Architecture model error")
		return err
	}
	player.AppendHeroAttrModel(player.ArchitectureModel)

	player.LumberModel = model.NewLumberModel(make(map[int32]*model.LumberEntity), userId)
	player.AppendPlayerModel(player.LumberModel)

	player.FurnitureModel = model.NewFurnitureModel(make(map[int32]map[int32]*model.FurnitureEntity), userId)
	player.AppendPlayerModel(player.FurnitureModel)

	player.ArchitectureModel.OnUpgradeCallback = func(p *model.PlayerModel, archType int32, oldLevel int32, senderMsg bool) {
		lumber.Service.OnBuildingUpgradeComplete(p, archType, oldLevel)
	}

	player.StoneModel = model.NewStoneModel(make(map[int32]*model.StoneEntity), userId, player)
	player.AppendPlayerModel(player.StoneModel)
	player.AppendHeroAttrModel(player.StoneModel)

	baseCfg := gameConfig.GetBaseCfg()
	if baseCfg == nil {
		logger.ErrorWithZapFields("base cfg is nil")
		return errors.New("base cfg is nil")
	}
	heroOwnId := hero.HeroIdGenerator.NextId()
	_, err = player.HeroDetailsModel.AddHero(player, int64(baseCfg.Hero), heroOwnId)
	if err != nil {
		logger.ErrorWithZapFields("add default hero error")
		return err
	}
	if err = player.HeroFormationModel.AddHeroFormation(int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN), 1, &model.HeroFormationEntity{
		UserID:        userId,
		FormationType: int32(pb.HeroFormationType_HERO_FORMATION_TYPE_MAIN),
		FormationID:   1,
		HeroOwnIDList: []int64{heroOwnId},
		IsActive:      true,
	}); err != nil {
		logger.ErrorWithZapFields("add default formation error")
		return err
	}

	player.PlayerActivityModel = model.NewPlayerActivityModel(player)
	player.AppendPlayerModel(player.PlayerActivityModel)

	player.TurnTableModel = model.NewTurnTableModel(make(map[int32]*model.TurnTableEntity), make(map[model.TurnTableStateKey]*model.TurnTableStateEntity), userId, player)
	player.AppendPlayerModel(player.TurnTableModel)

	player.TrialModel = model.NewTrialModel(userId)
	player.AppendPlayerModel(player.TrialModel)

	player.PlayerSignModel = model.NewPlayerSignModel(player)
	player.AppendPlayerModel(player.PlayerSignModel)

	player.PlayerArenaModel, err = model.CreatePlayerArenaModel(player)
	if err != nil {
		logger.ErrorWithZapFields("create player arena model error")
		return err
	}
	player.AppendPlayerModel(player.PlayerArenaModel)

	gloryArenaModel, err := model.CreatePlayerGloryArenaModel(player)
	if err != nil {
		logger.ErrorWithZapFields("create player glory arena model error")
		return err
	}
	player.PlayerGloryArenaModel = gloryArenaModel
	player.AppendPlayerModel(player.PlayerGloryArenaModel)

	player.PlayerTokenShopModel, err = model.CreatePlayerTokenShop(userId)
	if err != nil {
		logger.ErrorWithZapFields("create player token shop model error")
		return err
	}
	player.AppendPlayerModel(player.PlayerTokenShopModel)

	player.PlayerFunctionModel = model.NewPlayerFunctionModel(userId, player, make(map[int32]*model.PlayerFunctionEntity))
	player.AppendPlayerModel(player.PlayerFunctionModel)

	idleModel, err := model.CreateIdleModel(userId)
	if err != nil {
		logger.ErrorWithZapFields("create idle model error")
		return err
	}
	player.IdleModel = idleModel
	player.AppendPlayerModel(idleModel)

	player.PlayerAdventureModel, err = model.CreatePlayerAdventureModel(userId, player)
	if err != nil {
		logger.ErrorWithZapFields("create player adventure model error")
		return err
	}
	player.AppendPlayerModel(player.PlayerAdventureModel)

	player.VipCardModel = model.NewVipCardModel(userId)
	player.AppendPlayerModel(player.VipCardModel)

	player.CollectionModel = model.NewCollectionModel(userId, make(map[int32]*model.CollectionEntity), make(map[int32]*model.CollectionEntryEntity), make(map[int32]*model.CollectionEntity))
	player.AppendPlayerModel(player.CollectionModel)
	player.AppendHeroAttrModel(player.CollectionModel)

	player.AppearanceModel = model.NewAppearanceModel(userId, make(map[int32]*model.AppearanceEntity), player, make(map[enum.AvatarType]int32))
	player.AppendPlayerModel(player.AppearanceModel)
	player.AppendHeroAttrModel(player.AppearanceModel)

	signInModel, err := model.LoadSignInModel(player.GetUserId())
	if err != nil {
		logger.ErrorWithZapFields("load signin model error")
		return err
	}
	player.SignInModel = signInModel
	player.AppendPlayerModel(player.SignInModel)

	player.AllianceDailyActivityModel, err = model.LoadAllianceDailyActivityModel(player.GetUserId())
	if err != nil {
		logger.ErrorWithZapFields("load alliance daily activity model error")
		return err
	}
	player.AppendPlayerModel(player.AllianceDailyActivityModel)

	enum.PublishRegister(dbService.RDB, player.GetUserId(), player.GetUserAccount(), int64(getPlayerInfoCountByAccount(player.GetUserAccount())))

	return nil
}

func LoadRecentPlayerBasicInfoFromDB() {
	infos, err := easyDB.GetPlayerEntitiesByRaw[model.UserEntity](enum.SELECT_RECENT_PLAYER_SQL)
	if err != nil {
		logger.ErrorWithZapFields("[arena] load player error", zap.Error(err))
		return
	}

	logger.InfoWithSprintf("[arena] load recent players success, count=%d", len(infos))
	workerNum := 100
	wg := sync.WaitGroup{}
	taskChan := make(chan *model.UserEntity, workerNum)

	for i := 0; i < workerNum; i++ {

		wg.Add(1)
		go func() {

			defer wg.Done()

			ctx := context.Background()

			for info := range taskChan {
				func() {
					defer func() {
						if err := recover(); err != nil {
							logger.ErrorWithZapFields("[arena] load player panic", zap.Any("err", err))
						}
					}()

					player, err := LoadPlayer(info, false)
					if err != nil {
						logger.ErrorWithZapFields("[arena] load player error", zap.Error(err))
						return
					}
					player.BuildPlayerCacheInfo()

					version := player.PlayerArenaModel.GetVersion()
					score := player.PlayerArenaModel.GetScore()
					key := enum.GetArenaScoreInfoKey(info.ServerId, version)
					member := fmt.Sprintf("%d", info.UserId)
					dbService.RDB.ZAdd(ctx, key, &redis.Z{
						Score:  float64(score),
						Member: member,
					})

					// 只设置一次TTL
					dbService.RDB.ExpireNX(ctx, key, logicCommon.PLAYER_REDIS_TIMEOUT)

					// 更新基础信息缓存
					err = logicCommon.UpdatePlayerRedisInfo(player.PlayerCacheInfo)
					if err != nil {
						logger.ErrorWithZapFields("[arena] update player basic info error", zap.Error(err))
						return
					}

				}()
			}
		}()
	}

	for _, info := range infos {
		taskChan <- info
	}

	close(taskChan)

	wg.Wait()

	logger.InfoWithSprintf("[arena] load arena rank finished")
}

func getPlayerInfoCountByAccount(account string) int32 {
	info, err := easyDB.GetPlayerEntitiesByWhere[model.UserEntity](map[string]interface{}{"account": account})
	if err != nil {
		return 0
	}
	return int32(len(info))
}
