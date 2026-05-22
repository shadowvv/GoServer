package platform

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
)

// 命令行参数
type CmdArgs struct {
	NodeId      int    // 节点id
	NodeType    string // 节点类型
	Environment string // 环境
	ConfigName  string // 配置名称
	ChannelId   int    // 渠道id
}

func ParseCmdArgs() *CmdArgs {
	args := &CmdArgs{}

	flag.IntVar(&args.NodeId, "nodeId", 0, "node id (1~9)")
	flag.StringVar(&args.NodeType, "nodeType", "", "node type: web|gateway|game|social|rank")
	flag.StringVar(&args.Environment, "env", "", "environment: local|dev|test|stage|prod")
	flag.StringVar(&args.ConfigName, "configName", "", "config name")
	flag.IntVar(&args.ChannelId, "channelId", 0, "channel id")

	flag.Parse()
	return args
}

// 初始化基础服务
func BootBasicService() *nodeConfig.PlatformConfig {

	// 节点配置信息读取
	cmd := ParseCmdArgs()
	if cmd.NodeId != 0 {
		nodeConfig.InitNodeInfo(int32(cmd.NodeId), cmd.NodeType, enum.Environment(cmd.Environment), cmd.ConfigName, int32(cmd.ChannelId))
	} else {
		config := &nodeConfig.NodeConfig{}
		err := tool.LoadYaml("config/nodeConfig.yaml", config)
		if err != nil {
			log.Fatalf("[platform] Load node config error:%v", err)
		}
		nodeConfig.InitNodeInfo(config.NodeId, config.NodeType, enum.Environment(config.Environment), config.ConfigName, config.ChannelId)
	}

	// 读取节点对应类型和环境的配置
	allPlatformConfig := &nodeConfig.AllPlatformConfig{}
	err := tool.LoadYaml("config/platformConfig.yaml", allPlatformConfig)
	if err != nil {
		log.Fatalf("[platform] Load platform config error:%v", err)
	}
	configs, ok := allPlatformConfig.Configs[nodeConfig.NodeType]
	if !ok {
		log.Fatalf("[platform] No config for node type %s", nodeConfig.NodeType)
	}
	cfg, ok := configs[nodeConfig.ConfigName]
	if !ok {
		log.Fatalf("[platform] No config for node type %s and environment %s", nodeConfig.NodeType, nodeConfig.ConfigName)
	}

	// 日志初始化
	err = logger.InitLoggerByConfig(cfg.LoggerConfig)
	if err != nil {
		log.Fatalf("[platform] Init logger error:%v", err)
	}

	logger.InfoWithSprintf("[platform] Init node nodeId:%d,nodeType:%s,environment:%s,channelId:%d", nodeConfig.NodeId, nodeConfig.NodeType, nodeConfig.Env, nodeConfig.ChannelId)
	return cfg
}

func BootBackendService() *nodeConfig.PlatformConfig {

	// 节点信息读取
	cmd := ParseCmdArgs()
	if cmd.NodeId != 0 {
		nodeConfig.InitNodeInfo(int32(cmd.NodeId), cmd.NodeType, enum.Environment(cmd.Environment), cmd.ConfigName, int32(cmd.ChannelId))
	} else {
		config := &nodeConfig.NodeConfig{}
		err := tool.LoadYaml("config/nodeConfig.yaml", config)
		if err != nil {
			log.Fatalf("[platform] Load node config error:%v", err)
		}
		nodeConfig.InitNodeInfo(config.NodeId, config.NodeType, enum.Environment(config.Environment), config.ConfigName, config.ChannelId)
	}

	// 读取节点对应类型和环境的配置
	allPlatformConfig := &nodeConfig.AllPlatformConfig{}
	err := tool.LoadYaml("config/backendConfig.yaml", allPlatformConfig)
	if err != nil {
		log.Fatalf("[platform] Load platform config error:%v", err)
	}
	configs, ok := allPlatformConfig.Configs[nodeConfig.NodeType]
	if !ok {
		log.Fatalf("[platform] No config for node type %s", nodeConfig.NodeType)
	}
	cfg, ok := configs[nodeConfig.ConfigName]
	if !ok {
		log.Fatalf("[platform] No config for node type %s and environment %s", nodeConfig.NodeType, nodeConfig.Env)
	}

	// 日志初始化
	err = logger.InitLoggerByConfig(cfg.LoggerConfig)
	if err != nil {
		log.Fatalf("[platform] Init logger error:%v", err)
	}

	logger.InfoWithSprintf("[platform] Init node nodeId:%d,nodeType:%s,environment:%s,channelId:%d", nodeConfig.NodeId, nodeConfig.NodeType, nodeConfig.Env, nodeConfig.ChannelId)
	return cfg
}

func ListenSignal(hooker logicCommon.SignalHooker) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(
		sigCh,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	go func() {
		for sig := range sigCh {
			logger.InfoWithSprintf("[signal] receive: %v", sig)

			switch sig {
			case syscall.SIGHUP:
				gameConfig.ReloadAllConfig()
				hooker.AfterAllConfigReload()
			case syscall.SIGQUIT:
				hooker.KickAllPlayer()
			case syscall.SIGTERM, syscall.SIGINT:
				gracefulShutdown()
				return
			default:
				return
			}
		}
	}()
}

var dumpOnce sync.Mutex

func dumpRuntimeInfo() {
	if !dumpOnce.TryLock() {
		return
	}
	defer dumpOnce.Unlock()

	now := time.Now()
	ts := now.Format("20060102_150405")
	pid := os.Getpid()

	// 1️⃣ runtime dump
	runtimeFile := fmt.Sprintf("./runtime_%s_%d.dump", ts, pid)
	file, err := os.OpenFile(runtimeFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(file, format+"\n", args...)
	}

	write("===== RUNTIME DUMP BEGIN =====")
	write("time=%s pid=%d", now.Format(time.RFC3339), pid)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	write(
		"Alloc=%dMB TotalAlloc=%dMB Sys=%dMB NumGC=%d",
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC,
	)

	write("goroutines=%d", runtime.NumGoroutine())
	write("===== RUNTIME DUMP END =====")

	// 2️⃣ goroutine pprof
	pprofFile := fmt.Sprintf("./goroutine_%s_%d.pprof", ts, pid)
	f, err := os.Create(pprofFile)
	if err == nil {
		defer f.Close()
		_ = pprof.Lookup("goroutine").WriteTo(f, 2)
	}
}

func gracefulShutdown() {
	os.Exit(0)
}
