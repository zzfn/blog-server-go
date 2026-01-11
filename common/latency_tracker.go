package common

import (
	"sort"
	"sync"
	"time"
)

// LatencyTracker 延迟追踪器
type LatencyTracker struct {
	mu            sync.RWMutex
	latencies     []time.Duration
	maxSize       int
	totalRequests int64
	startTime     time.Time
}

// NewLatencyTracker 创建延迟追踪器
func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		latencies: make([]time.Duration, 0, maxSize),
		maxSize:   maxSize,
		startTime: time.Now(),
	}
}

// Record 记录一次请求延迟
func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.totalRequests++
	if len(lt.latencies) >= lt.maxSize {
		// 滑动窗口：移除最旧的记录
		lt.latencies = lt.latencies[1:]
	}
	lt.latencies = append(lt.latencies, latency)
}

// GetAverage 获取平均延迟
func (lt *LatencyTracker) GetAverage() time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, l := range lt.latencies {
		sum += l
	}
	return sum / time.Duration(len(lt.latencies))
}

// GetPercentile 获取延迟百分位
func (lt *LatencyTracker) GetPercentile(p float64) time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if len(lt.latencies) == 0 {
		return 0
	}

	// 复制一份避免修改原始数据
	sorted := make([]time.Duration, len(lt.latencies))
	copy(sorted, lt.latencies)

	// 排序
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// 计算百分位索引
	index := int(float64(len(sorted)) * p / 100)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// GetCount 获取记录数量
func (lt *LatencyTracker) GetCount() int {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return len(lt.latencies)
}

// GetTotalRequests 获取总请求数
func (lt *LatencyTracker) GetTotalRequests() int64 {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.totalRequests
}

// GetUptime 获取运行时间
func (lt *LatencyTracker) GetUptime() time.Duration {
	return time.Since(lt.startTime)
}

// GetQPS 获取每秒请求数
func (lt *LatencyTracker) GetQPS() float64 {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	uptime := time.Since(lt.startTime).Seconds()
	if uptime <= 0 {
		return 0
	}
	return float64(lt.totalRequests) / uptime
}
