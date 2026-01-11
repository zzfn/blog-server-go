package middleware

import (
	"blog-server-go/common"
	"time"

	"github.com/gofiber/fiber/v2"
)

// 全局延迟追踪器实例
var latencyTracker = common.NewLatencyTracker(1000) // 保留最近1000次请求的延迟

// LatencyMiddleware 延迟追踪中间件
func LatencyMiddleware(c *fiber.Ctx) error {
	start := time.Now()

	// 继续处理请求
	err := c.Next()

	// 记录请求延迟
	latency := time.Since(start)
	latencyTracker.Record(latency)

	return err
}

// GetLatencyTracker 获取延迟追踪器实例
func GetLatencyTracker() *common.LatencyTracker {
	return latencyTracker
}
