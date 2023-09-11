package handlers

import (
	"blog-server-go/common"
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type CommentsHandler struct {
	BaseHandler
}

func (ch *CommentsHandler) GetComments(c *fiber.Ctx) error {
	objectId := c.Query("objectId")
	objectType := c.Query("objectType")
	var comments []models.Comment

	query := ch.DB.Preload("Replies").Where("is_deleted", false)

	if objectId != "" {
		query = query.Where("object_id", objectId)
	}

	if objectType != "" {
		query = query.Where("object_type", objectType)
	}

	if err := query.Find(&comments).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch comments"})
	}

	return c.JSON(comments)
}
func (ch *CommentsHandler) CreateComment(c *fiber.Ctx) error {
	var input models.Comment
	ip := common.GetConnectingIp(c)
	input.IP = ip
	input.Address, _ = common.GetIpAddressInfo(ch.Redis, ip)
	// Parse and validate request body
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	log.Info(&input)
	// Save to DB
	if err := ch.DB.Create(&input).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save comment"})
	}

	return c.Status(201).JSON(input)
}
func (ch *CommentsHandler) CreateReply(c *fiber.Ctx) error {
	var input models.Reply
	ip := common.GetConnectingIp(c)
	input.IP = ip
	input.Address, _ = common.GetIpAddressInfo(ch.Redis, ip)
	if err := c.BodyParser(&input); err != nil {
		log.Error(err.Error())
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	// Save to DB
	if err := ch.DB.Create(&input).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save comment"})
	}

	return c.Status(201).JSON(input)
}
