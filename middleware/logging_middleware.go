package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func LoggingMiddleware(c *fiber.Ctx) error {
	// 记录请求的开始时间
	start := time.Now()

	// 等待请求处理完成
	err := c.Next()

	// 计算耗时
	duration := time.Since(start)
	log.Println(c.IPs(), c.GetReqHeaders())
	// 打印请求方法、路径和耗时
	log.Printf("[%s] %s - %v - IP: %s\n", c.Method(), c.Path(), duration, c.IP())

	return err
}
