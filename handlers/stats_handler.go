package handlers

import (
	"blog-server-go/models"
	"blog-server-go/middleware"
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type StatsHandler struct {
	BaseHandler
}

// StatsOverview 统计概览响应结构
type StatsOverview struct {
	BasicStats       BasicStats       `json:"basicStats"`
	ServiceInfo      ServiceInfo      `json:"serviceInfo"`
	CacheInfo        CacheInfo        `json:"cacheInfo"`
	PerformanceInfo  PerformanceInfo  `json:"performanceInfo"`
	RequestStats     RequestStats     `json:"requestStats"`
	DatabaseStats    DatabaseStats    `json:"databaseStats"`
}

// BasicStats 基础统计
type BasicStats struct {
	OnlineUsers  int64 `json:"onlineUsers"`
	TotalViews   int64 `json:"totalViews"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	GoVersion     string `json:"goVersion"`
	KernelVersion string `json:"kernelVersion"`
}

// CacheInfo 缓存信息
type CacheInfo struct {
	Status       string `json:"status"`
	Latency      string `json:"latency"`
	CachedKeys   int64  `json:"cachedKeys"`
	CacheBackend string `json:"cacheBackend"`
	MemoryUsage  string `json:"memoryUsage"`
}

// PerformanceInfo 性能信息
type PerformanceInfo struct {
	MemoryUsage      string `json:"memoryUsage"`
	Goroutines       int    `json:"goroutines"`
	GCSTWTime        string `json:"gcstwTime"`
	AverageLatency   string `json:"averageLatency"`
}

// RequestStats 请求统计
type RequestStats struct {
	TotalRequests int64   `json:"totalRequests"`
	QPS           float64 `json:"qps"`
	Uptime        string  `json:"uptime"`
	P50Latency    string  `json:"p50Latency"`
	P95Latency    string  `json:"p95Latency"`
	P99Latency    string  `json:"p99Latency"`
}

// DatabaseStats 数据库统计
type DatabaseStats struct {
	Status       string `json:"status"`
	MaxOpenConns int    `json:"maxOpenConns"`
	OpenConns    int    `json:"openConns"`
	InUse        int    `json:"inUse"`
	Idle         int    `json:"idle"`
}

// GetOverview 获取统计概览
func (sh *StatsHandler) GetOverview(c *fiber.Ctx) error {
	ctx := context.Background()

	// 并发获取各项统计数据
	basicStats := sh.getBasicStats(ctx)
	serviceInfo := sh.getServiceInfo()
	cacheInfo := sh.getCacheInfo(ctx)
	performanceInfo := sh.getPerformanceInfo()
	requestStats := sh.getRequestStats()
	databaseStats := sh.getDatabaseStats()

	overview := StatsOverview{
		BasicStats:      basicStats,
		ServiceInfo:     serviceInfo,
		CacheInfo:       cacheInfo,
		PerformanceInfo: performanceInfo,
		RequestStats:    requestStats,
		DatabaseStats:   databaseStats,
	}

	return c.JSON(overview)
}

// getBasicStats 获取基础统计数据
func (sh *StatsHandler) getBasicStats(ctx context.Context) BasicStats {
	// 获取在线人数
	onlineUsers, _ := sh.Redis.ZCard(ctx, "online_users").Result()

	// 获取总阅读数
	var totalViews int64
	sh.DB.Model(&models.Article{}).Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)

	return BasicStats{
		OnlineUsers: onlineUsers,
		TotalViews:  totalViews,
	}
}

// getServiceInfo 获取服务信息
func (sh *StatsHandler) getServiceInfo() ServiceInfo {
	// 检查服务状态（数据库连接）
	status := "正常"
	if sqlDB, err := sh.DB.DB(); err != nil || sqlDB.Ping() != nil {
		status = "异常"
	}

	// 获取 Go 版本
	goVersion := runtime.Version()

	return ServiceInfo{
		Status:        status,
		Version:       "1.0.0", // 可以从 git tag 或构建信息获取
		GoVersion:     goVersion,
		KernelVersion: getKernelVersion(),
	}
}

// getCacheInfo 获取缓存信息
func (sh *StatsHandler) getCacheInfo(ctx context.Context) CacheInfo {
	// 测量 Redis 延迟
	start := time.Now()
	_, err := sh.Redis.Ping(ctx).Result()
	latency := time.Since(start).Milliseconds()

	status := "正常"
	if err != nil {
		status = "异常"
	}

	// 获取缓存 key 数量
	var cachedKeys int64
	for _, pattern := range []string{"article*", "online*", "session*"} {
		keys, _ := sh.Redis.Keys(ctx, pattern).Result()
		cachedKeys += int64(len(keys))
	}

	// 获取 Redis 内存使用
	memoryUsage := "N/A"
	if info, err := sh.Redis.Info(ctx, "memory").Result(); err == nil {
		// 解析 INFO memory 输出获取 used_memory
		for _, line := range strings.Split(info, "\n") {
			if strings.HasPrefix(line, "used_memory:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					if bytes, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
						memoryUsage = formatBytes(uint64(bytes))
					}
				}
				break
			}
		}
	}

	return CacheInfo{
		Status:       status,
		Latency:      formatDuration(latency),
		CachedKeys:   cachedKeys,
		CacheBackend: "Redis",
		MemoryUsage:  memoryUsage,
	}
}

// getPerformanceInfo 获取性能信息
func (sh *StatsHandler) getPerformanceInfo() PerformanceInfo {
	// 获取内存信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := formatBytes(m.Alloc)

	// 获取 Goroutine 数量
	goroutines := runtime.NumGoroutine()

	// 获取 GC 信息
	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)
	gcSTWTime := formatDuration(gcStats.PauseTotal.Nanoseconds() / 1e6)

	// 获取平均请求延迟
	latencyTracker := middleware.GetLatencyTracker()
	avgLatency := latencyTracker.GetAverage()
	averageLatency := "0ms"
	if avgLatency > 0 {
		averageLatency = formatDuration(avgLatency.Milliseconds())
	}

	return PerformanceInfo{
		MemoryUsage:    memoryUsage,
		Goroutines:     goroutines,
		GCSTWTime:      gcSTWTime,
		AverageLatency: averageLatency,
	}
}

// getRequestStats 获取请求统计
func (sh *StatsHandler) getRequestStats() RequestStats {
	latencyTracker := middleware.GetLatencyTracker()

	// 获取运行时间
	uptime := latencyTracker.GetUptime()
	uptimeStr := formatUptime(uptime)

	// 获取百分位延迟
	p50 := latencyTracker.GetPercentile(50)
	p95 := latencyTracker.GetPercentile(95)
	p99 := latencyTracker.GetPercentile(99)

	return RequestStats{
		TotalRequests: latencyTracker.GetTotalRequests(),
		QPS:           latencyTracker.GetQPS(),
		Uptime:        uptimeStr,
		P50Latency:    formatDuration(p50.Milliseconds()),
		P95Latency:    formatDuration(p95.Milliseconds()),
		P99Latency:    formatDuration(p99.Milliseconds()),
	}
}

// getDatabaseStats 获取数据库统计
func (sh *StatsHandler) getDatabaseStats() DatabaseStats {
	sqlDB, err := sh.DB.DB()
	if err != nil {
		return DatabaseStats{
			Status: "异常",
		}
	}

	// 获取连接池统计
	stats := sqlDB.Stats()

	return DatabaseStats{
		Status:       "正常",
		MaxOpenConns: stats.MaxOpenConnections,
		OpenConns:    stats.OpenConnections,
		InUse:        stats.InUse,
		Idle:         stats.Idle,
	}
}

// formatUptime 格式化运行时间
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d分", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d小时", int(d.Hours()))
	}
	return fmt.Sprintf("%d天", int(d.Hours()/24))
}

// formatBytes 格式化字节数
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return "< 1KB"
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%d %s", b/div, units[exp])
}

// formatDuration 格式化时长
func formatDuration(ms int64) string {
	return fmt.Sprintf("%dms", ms)
}

// getKernelVersion 获取内核版本（简化版）
func getKernelVersion() string {
	return "Linux" // 或通过执行 uname 命令获取
}