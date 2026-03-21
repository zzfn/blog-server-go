package middleware

import (
	"blog-server-go/common"

	"github.com/gofiber/fiber/v2"
)

func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		token := common.ExtractToken(auth)
		if token == "" {
			token = c.Cookies("blog_token")
		}
		if token == "" {
			newResp := common.NewResponse(fiber.StatusUnauthorized, "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(newResp)
		}

		payload, err := common.ParseToken(token)
		if err != nil {
			newResp := common.NewResponse(fiber.StatusUnauthorized, "Unauthorized", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(newResp)
		}

		tokenValidatorMu.RLock()
		validator := tokenValidator
		tokenValidatorMu.RUnlock()
		if validator != nil {
			ok, err := validator(token, payload)
			if err != nil || !ok {
				newResp := common.NewResponse(fiber.StatusUnauthorized, "Unauthorized", nil)
				return c.Status(fiber.StatusUnauthorized).JSON(newResp)
			}
		}

		if !payload.IsAdmin {
			newResp := common.NewResponse(fiber.StatusUnauthorized, "Admin access required", nil)
			return c.Status(fiber.StatusUnauthorized).JSON(newResp)
		}

		c.Locals("userId", payload.UserID)
		c.Locals("isAdmin", payload.IsAdmin)
		c.Locals("username", payload.Username)
		return c.Next()
	}
}
