package middleware

import (
	"blog-server-go/common"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

func validateAPIKey(c *fiber.Ctx, key string) (bool, error) {
	log.Info("API Key: ", key)
	payload, err := common.ParseToken(key)
	c.Locals("userId", payload.UserID)
	c.Locals("isAdmin", payload.IsAdmin)
	c.Locals("username", payload.Username)
	if err != nil {
		log.Info("API Key is invalid")
		return false, keyauth.ErrMissingOrMalformedAPIKey
	}
	if payload.IsAdmin {
		log.Info("API Key is valid")
		return true, nil
	} else {
		log.Info("API Key is invalid")
		return false, errors.New("API Key is invalid")
	}
}

// AuthMiddleware KeyAuth provides the configuration for API key authentication.
func AdminMiddleware() fiber.Handler {
	return keyauth.New(keyauth.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if errors.Is(err, keyauth.ErrMissingOrMalformedAPIKey) {
				newResp := common.NewResponse(fiber.StatusUnauthorized, err.Error(), nil)
				return c.Status(fiber.StatusUnauthorized).JSON(newResp)
			}
			newResp := common.NewResponse(fiber.StatusUnauthorized, err.Error(), nil)
			return c.Status(fiber.StatusUnauthorized).JSON(newResp)
		},
		Validator: validateAPIKey,
	})
}
