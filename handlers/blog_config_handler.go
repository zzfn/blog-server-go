package handlers

import (
	"blog-server-go/common"
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type BlogConfigHandler struct {
	BaseHandler
}

func (bch *BlogConfigHandler) GetSiteConfig(c *fiber.Ctx) error {
	var ctx = context.Background()
	blogConfig, _ := bch.Redis.HGetAll(ctx, "blog_config").Result()

	// Create a new map to hold the deserialized values
	deserializedConfig := make(map[string]interface{})

	// Iterate over the blogConfig map and deserialize each value
	for key, value := range blogConfig {
		var deserializedValue interface{}
		err := json.Unmarshal([]byte(value), &deserializedValue)
		if err != nil {
			log.Error(err)
			return c.Status(500).SendString("Failed to deserialize site config")
		}
		deserializedConfig[key] = deserializedValue
	}

	return c.JSON(deserializedConfig)
}
func (bch *BlogConfigHandler) SaveSiteConfig(c *fiber.Ctx) error {
	var ctx = context.Background()
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		log.Error(err)
		return c.Status(400).SendString("Failed to parse request body")
	}
	log.Info("Saving site config", body)

	for key, value := range body {
		jsonString, err := json.Marshal(value)
		if err != nil {
			log.Error(err)
			return &common.BusinessException{
				Code:    5000,
				Message: "Failed to convert site config to JSON",
			}
		}

		// Store the JSON string in Redis using HSET
		err = bch.Redis.HSet(ctx, "blog_config", key, jsonString).Err()
		if err != nil {
			log.Error(err)
			return &common.BusinessException{
				Code:    5000,
				Message: "Failed to save site config",
			}
		}
	}

	return c.JSON(body)
}
