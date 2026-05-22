package robotMonitor

import (
	"time"

	"github.com/drop/GoServer/server/service/logger"
)

type PlatformMonitor struct {
	SystemStats *SystemStats
}

func NewPlatformMonitor() *PlatformMonitor {
	summary := &ConfigSummary{}
	summary.TotalRobots.Store(0)
	return &PlatformMonitor{
		SystemStats: NewSystemStats(summary),
	}
}

func (m *PlatformMonitor) StartReporter(interval time.Duration) {
	m.SystemStats.StartReporter(interval)
}

func (m *PlatformMonitor) StopPeriodicReporter() {
	m.SystemStats.StopReporter()
}

func (m *PlatformMonitor) PrintFinalSummary(startTime time.Time) {
	logger.InfoWithSprintf("%s", m.SystemStats.Report())
	logger.InfoWithSprintf("total run time: %v\n", time.Since(startTime))
	logger.InfoWithSprintf("program exited")
}
