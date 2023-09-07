package routes

import (
	"blog-server-go/handlers"
	"github.com/gofiber/fiber/v2"
)

// SetupArticleRoutes 设置与文章相关的路由
func SetupArticleRoutes(app *fiber.App, handler handlers.ArticleHandler) {

	v1 := app.Group("v1")
	post := v1.Group("post")
	post.Get("list", handler.GetArticles)
	post.Get("search", handler.SearchArticles)
}
