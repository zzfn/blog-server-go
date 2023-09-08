package handlers

import (
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
)

type CommentsHandler struct {
	BaseHandler
}

func (ch *CommentsHandler) GetAllComments(c *fiber.Ctx) error {
	var comments []models.Comment
	if err := ch.DB.Preload("Replies").Find(&comments).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch comments"})
	}

	return c.JSON(comments)
}
