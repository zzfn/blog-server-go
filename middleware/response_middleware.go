package middleware

import (
	"blog-server-go/common"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func ResponseMiddleware(c *fiber.Ctx) error {
	// 检查是否为WebSocket的握手请求
	if c.Get("Upgrade") == "websocket" && c.Get("Connection") == "Upgrade" {
		return c.Next()
	}
	//检查是否是下载的响应
	if strings.Contains(c.Path(), "/export/markdown/") {
		return c.Next()
	}
	// 先调用下一个中间件或路由处理函数
	if err := c.Next(); err != nil {
		var be *common.BusinessException
		if errors.As(err, &be) {
			// 这是一个业务异常
			newResp := common.NewResponse(be.Code, be.Message, nil)
			return c.JSON(newResp)
		} else {
			log.Error(err)
			newResp := common.NewResponse(5000, "Internal Server Error", nil)
			return c.JSON(newResp)
		}
	}

	// 获取原始响应
	rawBody := c.Response().Body()
	// 创建新的响应体
	newResp := common.NewResponse(2000, "Success", json.RawMessage(rawBody))

	// 使用新的响应体覆盖原始响应
	return c.JSON(newResp)
}
