package routes

import (
	"blog-server-go/handlers"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupArticleRoutes 设置与文章相关的路由
func SetupArticleRoutes(app *fiber.App, db *gorm.DB) {

	articleHandler := handlers.ArticleHandler{DB: db}
	app.Get("/articles", articleHandler.GetArticles)
}
