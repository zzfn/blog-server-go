package middleware

import (
	"blog-server-go/common"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
)

func ResponseMiddleware(c *fiber.Ctx) error {
	// 先调用下一个中间件或路由处理函数
	if err := c.Next(); err != nil {
		var be *common.BusinessException
		if errors.As(err, &be) {
			// 这是一个业务异常
			newResp := common.NewResponse(be.Code, be.Message, nil)
			return c.JSON(newResp)
		} else {
			newResp := common.NewResponse(500, "Internal Server Error", nil)
			return c.JSON(newResp)
		}
	}

	// 获取原始响应
	rawBody := c.Response().Body()
	// 创建新的响应体
	newResp := common.NewResponse(200, "Success", json.RawMessage(rawBody))

	// 使用新的响应体覆盖原始响应
	return c.JSON(newResp)
}