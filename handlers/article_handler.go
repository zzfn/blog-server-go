package handlers

import (
	"blog-server-go/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
)

// ArticleHandler 处理与文章相关的请求
type ArticleHandler struct {
	DB *gorm.DB
}

// GetArticles 获取所有文章
func (ah *ArticleHandler) GetArticles(c *fiber.Ctx) error {
	var articles []models.Article
	result := ah.DB.Find(&articles)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(result.Error.Error())
	}
	log.Info("查询成功")
	return c.JSON(articles)
}
