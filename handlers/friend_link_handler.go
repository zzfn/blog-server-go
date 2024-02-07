package handlers

import (
	"blog-server-go/common"
	"blog-server-go/kafka"
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type FriendLinksHandler struct {
	BaseHandler
}

// GetFriendLinks retrieves all friend links from the database
func (flh *FriendLinksHandler) GetFriendLinks(c *fiber.Ctx) error {
	var friendLinks []models.FriendLink
	if err := flh.DB.Where("is_deleted", false).Where("is_active", true).Order("created_at").Find(&friendLinks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch friend links"})
	}

	return c.JSON(friendLinks)
}

// SaveFriendLink saves a new friend link to the database
func (flh *FriendLinksHandler) SaveFriendLink(c *fiber.Ctx) error {
	var input models.FriendLink
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}

	// Save to DB
	if err := flh.DB.Create(&input).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save friend link"})
	}
	flh.KafkaProducer.ProduceMessage(kafka.FriendUpdateTopic, "id", "Friend link created")
	return c.Status(201).JSON(input)
}
