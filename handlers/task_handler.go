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

func (th *TaskHandler) SaveTaskList(c *fiber.Ctx) error {
	var task models.Task
	if err := c.BodyParser(&task); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}
	log.Info(task)
	result := th.DB.Create(&task)
	if result.Error != nil {
		log.Errorf("Failed to save task: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.Status(fiber.StatusCreated).JSON(task)
}
func (th *TaskHandler) UpdateTaskList(c *fiber.Ctx) error {
	var task models.Task
	if err := c.BodyParser(&task); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}
	log.Info(task)
	result := th.DB.Updates(&task)
	if result.Error != nil {
		log.Errorf("Failed to save task: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.Status(fiber.StatusOK).JSON(task)
}
func (th *TaskHandler) GetTaskList(c *fiber.Ctx) error {
	var tasks []models.Task
	result := th.DB.Order("created_at DESC").Find(&tasks)
	if result.Error != nil {
		log.Errorf("Failed to get tasks: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.JSON(tasks)
}
func (th *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	id, err := common.ParseString(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}
	result := th.DB.Delete(&models.Task{}, id)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete task"})
	}
	return c.JSON(fiber.Map{"message": "Task deleted successfully"})
}
