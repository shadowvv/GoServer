package backendPlatform

import (
	"encoding/json"
	"net/http"

	"github.com/drop/GoServer/server/logic/activityService"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/platform"
	"github.com/drop/GoServer/server/logic/platform/dbPool"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/webService"
	"go.uber.org/zap"
)

var httpServiceInstance *webService.HttpWebService                 // http服务
var dbPoolManager *dbPool.DBPoolManager                            // 数据库连接池
var serverInfoService *gameServerInfoService.GameServerInfoService // 游戏服务器信息
var activityInfoService *activityService.GameActivityService       // 活动信息

func BootBackEndPlatform() {
	cfg := platform.BootBackendService()

	data, _ := json.MarshalIndent(cfg.HttpConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Boot http platform config:%s", string(data))
	httpServiceInstance = webService.NewHttpWebService(cfg.HttpConfig)
	if httpServiceInstance == nil {
		logger.ErrorWithZapFields("[platform] Init web service error")
		panic("[platform] Init web service error")
	}

	data, _ = json.MarshalIndent(cfg.ServerDBConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init server db config:%s", string(data))
	serverDB, err := dbService.InitMySQL(cfg.ServerDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetServerDB(serverDB)
	logDB, err := dbService.InitMySQL(cfg.LogDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetLogDB(logDB)
	backendDb, err := dbService.InitMySQL(cfg.BackEndDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init server db error", zap.Error(err))
		panic("[platform] Init server db error")
	}
	easyDB.SetBackendDB(backendDb)
	data, _ = json.MarshalIndent(cfg.GameDBConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init game db config:%s", string(data))
	gameDB, err := dbService.InitMySQL(cfg.GameDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init game db error", zap.Error(err))
		panic("[platform] Init game db error")
	}
	dbPoolManager = dbPool.NewDBPoolManager(gameDB)
	easyDB.SetGameDBPool(dbPoolManager)
	rankDB, err := dbService.InitMySQL(cfg.RankDBConfig, logger.Logger)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init game db error", zap.Error(err))
		panic("[platform] Init game db error")
	}
	easyDB.SetRankDB(rankDB)
	data, _ = json.MarshalIndent(cfg.RedisConfig, "", "  ")
	logger.InfoWithSprintf("[platform] Init redis db config:%s", string(data))
	err = dbService.InitRedis(cfg.RedisConfig)
	if err != nil {
		logger.ErrorWithZapFields("[platform] Init redis error", zap.Error(err))
		panic("[platform] Init redis error")
	}

	gameConfig.LoadAllConfig()
	logger.InfoWithSprintf("[platform] load all config success !!!")
	// 游戏服务器信息服务
	serverInfoService = gameServerInfoService.NewGameServerInfoService()
	logger.InfoWithSprintf("[platform] init server info service")
	unlock := unlockService.NewUnlockService(serverInfoService)
	activityInfoService = activityService.NewGameActivityService(serverInfoService, unlock)
	logger.InfoWithSprintf("[platform] init activity info service")

	logger.InfoWithSprintf("[platform] Boot platform success !!!")
}

func StartHttpService() {
	logger.InfoWithSprintf("[platform] start http service")
	go httpServiceInstance.Start()
}

func RegisterHttpMessage(path string, handler http.HandlerFunc) {
	httpServiceInstance.RegisterRoutes(path, handler)
	logger.InfoWithSprintf("[platform] register http message: %s", path)
}

// RegisterHttpMessageWithMaxBody 注册一条单独设置 Body 上限的路由，适用于导入这种大请求体接口。
// maxBytes <= 0 表示不限制 Body 大小。
func RegisterHttpMessageWithMaxBody(path string, handler http.HandlerFunc, maxBytes int64) {
	httpServiceInstance.RegisterRoutesWithMaxBody(path, handler, maxBytes)
	logger.InfoWithSprintf("[platform] register http message: %s, maxBody=%d", path, maxBytes)
}
