package middleware

import (
	"blog-server-go/common"
	"sync"

	"github.com/gofiber/fiber/v2"
)

type TokenValidator func(token string, payload common.Payload) (bool, error)

var (
	tokenValidatorMu sync.RWMutex
	tokenValidator   TokenValidator
)

func SetTokenValidator(validator TokenValidator) {
	tokenValidatorMu.Lock()
	defer tokenValidatorMu.Unlock()
	tokenValidator = validator
}

func AuthMiddleware(c *fiber.Ctx) error {
	auth := c.Get("Authorization")
	token := common.ExtractToken(auth)
	if token == "" {
		token = c.Cookies("blog_token")
	}
	if token == "" {
		return c.Next()
	}

	payload, err := common.ParseToken(token)
	if err != nil {
		return c.Next()
	}

	tokenValidatorMu.RLock()
	validator := tokenValidator
	tokenValidatorMu.RUnlock()
	if validator != nil {
		ok, err := validator(token, payload)
		if err != nil || !ok {
			return c.Next()
		}
	}

	c.Locals("userId", payload.UserID)
	c.Locals("isAdmin", payload.IsAdmin)
	c.Locals("username", payload.Username)
	return c.Next()
}
