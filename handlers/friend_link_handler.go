package handlers

import (
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type FriendLinkHandler struct {
	BaseHandler
}

// GetFriendLinks retrieves all friend links from the database
func (flh *FriendLinkHandler) GetFriendLinks(c *fiber.Ctx) error {
	var friendLinks []models.FriendLink
	if err := flh.DB.Find(&friendLinks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch friend links"})
	}

	return c.JSON(friendLinks)
}

// SaveFriendLink saves a new friend link to the database
func (flh *FriendLinkHandler) SaveFriendLink(c *fiber.Ctx) error {
	var input models.FriendLink
	// Parse and validate request body
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	// Save to DB
	if err := flh.DB.Create(&input).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save friend link"})
	}

	return c.Status(201).JSON(input)
}
