package routes

import (
	"blog-server-go/handlers"
	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	ArticleHandler handlers.ArticleHandler
	CommentHandler handlers.CommentHandler
}

func SetupRoutes(app *fiber.App, h *Handlers) {
	// API Versioning
	v1 := app.Group("/v1")

	// Articles
	articles := v1.Group("/articles")
	articles.Get("/", h.ArticleHandler.GetArticles)
	articles.Get("/search", h.ArticleHandler.SearchArticles)

	// Comments
	comments := v1.Group("/comments")
	comments.Get("/", h.CommentHandler.GetAllComments)
}
