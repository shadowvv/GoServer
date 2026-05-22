package robotPlatform

import (
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "github.com/drop/GoServer/server/robot/robotModuleController"

	"github.com/drop/GoServer/server/robot/robotConfig"
	"github.com/drop/GoServer/server/robot/robotLogic"
	"github.com/drop/GoServer/server/robot/robotMonitor"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"github.com/drop/GoServer/server/service/logger"
)

const (
	configPath                = "config/robot.yaml"
	robotStartupDelayJitterMs = 1000
	periodicReportInterval    = 5 * time.Second
)

type RobotPlatform struct {
	robotsMu sync.RWMutex
	cfg      *robotConfig.RobotConfig
	monitor  *robotMonitor.PlatformMonitor
	robots   []*robotLogic.Robot
}

func NewRobotPlatform() (*RobotPlatform, error) {
	// 初始化controller
	if err := robotRouter.RegisterAllRobotMessages(); err != nil {
		return nil, err
	}
	// 加载构建配置
	cfg, err := robotConfig.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	// 初始化日志
	err = logger.InitLoggerByConfig(&cfg.Logger)
	if err != nil {
		return nil, err
	}
	// 构建成真正的operation
	if err = cfg.BuildRealOperations(); err != nil {
		return nil, err
	}

	return &RobotPlatform{
		cfg:     cfg,
		monitor: robotMonitor.NewPlatformMonitor(),
		robots:  make([]*robotLogic.Robot, 0),
	}, nil
}

func (a *RobotPlatform) StartAllRobots() {
	logger.InfoWithSprintf("start creating robots...")
	a.startRobotGroup()
	logger.InfoWithSprintf("started robots: %d", len(a.snapshotRobots()))

	a.monitor.StartReporter(periodicReportInterval)
}

func (a *RobotPlatform) startRobotGroup() {
	total := a.cfg.RuntimeCount()
	for i := 0; i < total; i++ {
		go func(index int) {
			a.startSingleRobot(index)
		}(i)
	}
}

func (a *RobotPlatform) startSingleRobot(index int) {
	login := a.cfg.Main.Login
	run := a.cfg.Main.RunConfig

	robotName := buildRobotName(a.cfg.RuntimeNamePrefix(), index)
	account := buildRobotAccount(login.Account, index)

	robotInstance := robotLogic.NewRobot(
		robotName,
		account,
		login.ServerID,
		login.LoginURL,
		login.Channel,
		login.Version,
		login.Language,
		login.DeviceID,
		login.AppID,
		login.Sign,
		a.cfg,
		a.cfg.Main.RunConfig.Mode,
		run.Interval,
		run.Duration,
		a.monitor,
	)
	interval := time.Duration(rand.Intn(robotStartupDelayJitterMs)) * time.Millisecond
	logger.InfoWithSprintf("phase=robot_start status=scheduled robot=%s account=%s index=%d jitterMs=%d", robotName, account, index, interval.Milliseconds())
	time.Sleep(interval)

	startedAt := time.Now()
	logger.InfoWithSprintf("phase=robot_start status=start robot=%s account=%s index=%d", robotName, account, index)
	if err := robotInstance.Start(); err != nil {
		logger.ErrorBySprintf("phase=robot_start status=failed robot=%s account=%s index=%d costMs=%d err=%v", robotName, account, index, time.Since(startedAt).Milliseconds(), err)
		return
	}
	a.addRobot(robotInstance)
	logger.InfoWithSprintf("phase=robot_start status=success robot=%s account=%s index=%d costMs=%d", robotName, account, index, time.Since(startedAt).Milliseconds())
}

func buildRobotName(baseName string, index int) string {
	return baseName + "_" + strconv.Itoa(index)
}

func buildRobotAccount(baseAccount string, index int) string {
	return robotUtils.GenerateAccount(baseAccount, index)
}

func (a *RobotPlatform) addRobot(r *robotLogic.Robot) {
	a.robotsMu.Lock()
	a.robots = append(a.robots, r)
	a.robotsMu.Unlock()
}

func (a *RobotPlatform) snapshotRobots() []*robotLogic.Robot {
	a.robotsMu.RLock()
	defer a.robotsMu.RUnlock()
	return append([]*robotLogic.Robot(nil), a.robots...)
}

func (a *RobotPlatform) WaitForExitSignal() {
	logger.InfoWithSprintf("program started, waiting for Ctrl+C...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	a.monitor.StopPeriodicReporter()
}

func (a *RobotPlatform) StopAllRobots() {
	logger.InfoWithSprintf("\nstopping all robots...")

	var wg sync.WaitGroup
	for _, robotInstance := range a.snapshotRobots() {
		wg.Add(1)
		go func(r *robotLogic.Robot) {
			defer wg.Done()
			r.Stop()
		}(robotInstance)
	}
	wg.Wait()
}

func (a *RobotPlatform) PrintFinalSummary(startTime time.Time) {
	a.monitor.PrintFinalSummary(startTime)
}
