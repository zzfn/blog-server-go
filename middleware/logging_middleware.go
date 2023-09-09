package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Person struct {
	XForwardedFor  string `reqHeader:"x-forwarded-for"`
	XRealIp        string `reqHeader:"x-real-ip"`
	CfConnectingIp string `reqHeader:"cf-connecting-ip"`
}

func LoggingMiddleware(c *fiber.Ctx) error {
	// 记录请求的开始时间
	start := time.Now()

	// 等待请求处理完成
	err := c.Next()

	// 计算耗时
	duration := time.Since(start)
	p := new(Person)

	if err := c.ReqHeaderParser(p); err != nil {
		return err
	}
	log.Printf("Client IP: %s | Forwarded IPs: %v | X-Forwarded-For: %s | X-Real-Ip: %s",
		c.IP(), c.IPs(), p.XForwardedFor, p.XRealIp)
	// 打印请求方法、路径和耗时
	log.Printf("[%s] %s - %v - IP: %s\n", c.Method(), c.Path(), duration, p.CfConnectingIp)

	return err
}
