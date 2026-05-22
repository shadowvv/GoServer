package robotMonitor

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drop/GoServer/server/service/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const defaultLatencySamples = 5000

type ConfigSummary struct {
	TotalRobots atomic.Int32
}

type SystemSample struct {
	NumCPU        int
	NumGoroutine  int
	AppMemAllocMB float64
	AppMemSysMB   float64
	AppMemPercent float64
	SysMemTotalGB float64
	SysMemUsedPct float64
	CPUUsedPct    float64
	GCCount       uint32
}

type SystemStats struct {
	mu      sync.Mutex
	peak    *SystemSample
	summary *ConfigSummary

	reporterMu   sync.Mutex
	reporterStop chan struct{}
	reporterDone chan struct{}

	totalMessagesSent         atomic.Int64
	totalOperationsCompleted  atomic.Int64
	periodMessagesSent        atomic.Int64
	periodOperationsCompleted atomic.Int64
	lastReportTime            time.Time

	latencySamples    []time.Duration
	latencyIdx        int
	latencyMaxSamples int
	latencyCount      int64
	totalLatency      time.Duration
	minLatency        time.Duration
	maxLatency        time.Duration
	minLatencyMsgID   uint32
	maxLatencyMsgID   uint32
	minLatencyRobotID string
	maxLatencyRobotID string
}

func NewSystemStats(summary *ConfigSummary) *SystemStats {
	if summary == nil {
		summary = &ConfigSummary{}
	}

	return &SystemStats{
		summary:           summary,
		peak:              &SystemSample{},
		lastReportTime:    time.Now(),
		latencyMaxSamples: defaultLatencySamples,
		latencySamples:    make([]time.Duration, 0, defaultLatencySamples),
	}
}

func (s *SystemStats) AddRobot() {
	s.summary.TotalRobots.Add(1)
}

func (s *SystemStats) RemoveRobot() {
	for {
		cur := s.summary.TotalRobots.Load()
		if cur <= 0 {
			return
		}
		if s.summary.TotalRobots.CompareAndSwap(cur, cur-1) {
			return
		}
	}
}

func (s *SystemStats) RecordMessageSent() {
	s.totalMessagesSent.Add(1)
	s.periodMessagesSent.Add(1)
}

func (s *SystemStats) RecordOperationCompleted() {
	s.totalOperationsCompleted.Add(1)
	s.periodOperationsCompleted.Add(1)
}

func (s *SystemStats) RecordOperationLatency(latency time.Duration, msgID uint32, robotID string) {
	if latency <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.latencyCount++
	s.totalLatency += latency

	if s.minLatency == 0 || latency < s.minLatency {
		s.minLatency = latency
		s.minLatencyMsgID = msgID
		s.minLatencyRobotID = robotID
	}
	if latency > s.maxLatency {
		s.maxLatency = latency
		s.maxLatencyMsgID = msgID
		s.maxLatencyRobotID = robotID
	}

	if len(s.latencySamples) < s.latencyMaxSamples {
		s.latencySamples = append(s.latencySamples, latency)
		return
	}
	s.latencySamples[s.latencyIdx] = latency
	s.latencyIdx = (s.latencyIdx + 1) % s.latencyMaxSamples
}

func (s *SystemStats) StartReporter(interval time.Duration) {
	s.reporterMu.Lock()
	defer s.reporterMu.Unlock()

	if s.reporterStop != nil {
		return
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	s.reporterStop = stop
	s.reporterDone = done

	go s.runReporter(interval, stop, done)
}

func (s *SystemStats) StopReporter() {
	s.reporterMu.Lock()
	stop := s.reporterStop
	done := s.reporterDone
	if stop != nil {
		close(stop)
		s.reporterStop = nil
	}
	s.reporterMu.Unlock()

	if done != nil {
		<-done
		s.reporterMu.Lock()
		if s.reporterDone == done {
			s.reporterDone = nil
		}
		s.reporterMu.Unlock()
	}
}

func (s *SystemStats) runReporter(interval time.Duration, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			logger.InfoWithSprintf("%s", s.Report())
		}
	}
}

func collectSystemSample() *SystemSample {
	var rt runtime.MemStats
	runtime.ReadMemStats(&rt)

	sample := &SystemSample{
		NumCPU:        runtime.NumCPU(),
		NumGoroutine:  runtime.NumGoroutine(),
		AppMemAllocMB: float64(rt.Alloc) / 1024.0 / 1024.0,
		AppMemSysMB:   float64(rt.Sys) / 1024.0 / 1024.0,
		GCCount:       rt.NumGC,
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		sample.SysMemTotalGB = float64(vm.Total) / 1024.0 / 1024.0 / 1024.0
		sample.SysMemUsedPct = vm.UsedPercent
		if vm.Total > 0 {
			sample.AppMemPercent = float64(rt.Sys) / float64(vm.Total) * 100.0
		}
	}

	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		sample.CPUUsedPct = pcts[0]
	}

	return sample
}

func (s *SystemStats) updatePeak(cur *SystemSample) {
	if cur.NumGoroutine > s.peak.NumGoroutine {
		s.peak.NumGoroutine = cur.NumGoroutine
	}
	if cur.AppMemAllocMB > s.peak.AppMemAllocMB {
		s.peak.AppMemAllocMB = cur.AppMemAllocMB
	}
	if cur.AppMemSysMB > s.peak.AppMemSysMB {
		s.peak.AppMemSysMB = cur.AppMemSysMB
	}
	if cur.AppMemPercent > s.peak.AppMemPercent {
		s.peak.AppMemPercent = cur.AppMemPercent
	}
	if cur.SysMemUsedPct > s.peak.SysMemUsedPct {
		s.peak.SysMemUsedPct = cur.SysMemUsedPct
	}
	if cur.CPUUsedPct > s.peak.CPUUsedPct {
		s.peak.CPUUsedPct = cur.CPUUsedPct
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (s *SystemStats) Report() string {
	cur := collectSystemSample()

	now := time.Now()
	periodMessagesSent := s.periodMessagesSent.Swap(0)
	periodOpsCompleted := s.periodOperationsCompleted.Swap(0)
	totalMessagesSent := s.totalMessagesSent.Load()
	totalOpsCompleted := s.totalOperationsCompleted.Load()

	s.mu.Lock()
	s.updatePeak(cur)
	peak := s.peak
	summary := s.summary
	elapsed := now.Sub(s.lastReportTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	s.lastReportTime = now

	latencyCount := s.latencyCount
	totalLatency := s.totalLatency
	minLatency := s.minLatency
	maxLatency := s.maxLatency
	minMsgID := s.minLatencyMsgID
	maxMsgID := s.maxLatencyMsgID
	minRobotID := s.minLatencyRobotID
	maxRobotID := s.maxLatencyRobotID
	latencySamples := append([]time.Duration(nil), s.latencySamples...)
	s.mu.Unlock()

	avgLatency := time.Duration(0)
	if latencyCount > 0 {
		avgLatency = totalLatency / time.Duration(latencyCount)
	}
	p50 := percentile(latencySamples, 0.50)
	p95 := percentile(latencySamples, 0.95)
	p99 := percentile(latencySamples, 0.99)

	minLatencyLine := "n/a"
	maxLatencyLine := "n/a"
	if latencyCount > 0 {
		minLatencyLine = fmt.Sprintf("%v (msg=%d, robot=%s)", minLatency, minMsgID, minRobotID)
		maxLatencyLine = fmt.Sprintf("%v (msg=%d, robot=%s)", maxLatency, maxMsgID, maxRobotID)
	}

	return fmt.Sprintf(
		"\n========== System Resource (current | peak) ==========\n"+
			"Config: robots=%d\n"+
			"Throughput(period): sent=%d msgs, completed=%d ops, msg/s=%.2f, ops/s=%.2f\n"+
			"Throughput(total): sent=%d msgs, completed=%d ops\n"+
			"Latency(all): avg=%v, p50=%v, p95=%v, p99=%v\n"+
			"Latency(min): %s\n"+
			"Latency(max): %s\n"+
			"CPU Cores: %d\n"+
			"CPU Usage: %.2f%% | Peak: %.2f%%\n"+
			"Goroutines: %d | Peak: %d\n"+
			"App Memory (Alloc): %.2f MB | Peak: %.2f MB\n"+
			"App Memory (Sys): %.2f MB | Peak: %.2f MB\n"+
			"App Memory Ratio: %.2f%% | Peak: %.2f%%\n"+
			"System Total Memory: %.2f GB\n"+
			"System Memory Usage: %.2f%% | Peak: %.2f%%\n"+
			"GC Count: %d\n",
		summary.TotalRobots.Load(),
		periodMessagesSent, periodOpsCompleted,
		float64(periodMessagesSent)/elapsed, float64(periodOpsCompleted)/elapsed,
		totalMessagesSent, totalOpsCompleted,
		avgLatency, p50, p95, p99,
		minLatencyLine, maxLatencyLine,
		cur.NumCPU,
		cur.CPUUsedPct, peak.CPUUsedPct,
		cur.NumGoroutine, peak.NumGoroutine,
		cur.AppMemAllocMB, peak.AppMemAllocMB,
		cur.AppMemSysMB, peak.AppMemSysMB,
		cur.AppMemPercent, peak.AppMemPercent,
		cur.SysMemTotalGB,
		cur.SysMemUsedPct, peak.SysMemUsedPct,
		cur.GCCount,
	)
}
