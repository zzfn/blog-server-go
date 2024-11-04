package handlers

import (
	"blog-server-go/common"
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type TaskHandler struct {
	BaseHandler
}

func (th *TaskHandler) GetTaskList(c *fiber.Ctx) error {
	var task models.Task
	if err := c.BodyParser(&task); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}

	result := th.DB.Create(&task)
	if result.Error != nil {
		log.Errorf("Failed to save article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.Status(fiber.StatusCreated).JSON(task)
}
func (th *TaskHandler) SaveTask(c *fiber.Ctx) error {
	return nil
}
