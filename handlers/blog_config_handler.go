package handlers

import (
	"blog-server-go/common"
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type BlogConfigHandler struct {
	BaseHandler
}

func (bch *BlogConfigHandler) GetSiteConfig(c *fiber.Ctx) error {
	if bch.Redis == nil {
		return &common.BusinessException{
			Code:    5000,
			Message: "Redis client not initialized",
		}
	}
	var ctx = context.Background()
	blogConfig, _ := bch.Redis.HGetAll(ctx, "blog_config").Result()
	return c.JSON(blogConfig)
}

func (bch *BlogConfigHandler) SaveSiteConfig(c *fiber.Ctx) error {
	var ctx = context.Background()
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Error(err)
		return c.Status(400).SendString("Failed to parse request body")
	}
	log.Info("Saving site config", body)
	err := bch.Redis.HMSet(ctx, "blog_config", body)
	if err != nil {
		return c.JSON(nil)
	}
	return c.JSON(nil)
}
