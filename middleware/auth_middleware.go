package middleware

import (
	"blog-server-go/common"
	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	token := common.ExtractToken(auth)
	payload, _ := common.ParseToken(token)
	c.Locals("userId", payload.UserID)
	c.Locals("isAdmin", payload.IsAdmin)
	c.Locals("username", payload.Username)
	return c.Next()
}
