package routes

import (
	"blog-server-go/handlers"
	"github.com/gofiber/fiber/v2"
)

// SetupArticleRoutes 设置与文章相关的路由
func SetupArticleRoutes(app *fiber.App, handler handlers.ArticleHandler) {

	article := app.Group("/post")
	article.Get("/list", handler.GetArticles)
	article.Get("/search", handler.SearchArticles)
}
