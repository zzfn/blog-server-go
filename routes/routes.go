package routes

import (
	"blog-server-go/handlers"
	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	ArticleHandler    handlers.ArticleHandler
	CommentsHandler   handlers.CommentsHandler
	WebSocketHandler  handlers.WebSocketHandler
	FriendLinkHandler handlers.FriendLinkHandler
}

func SetupRoutes(app *fiber.App, h *Handlers) {

	// API Versioning
	v1 := app.Group("/v1")
	v1.Get("/ws", h.WebSocketHandler.UpgradeToWebSocket)

	// Articles
	articles := v1.Group("/articles")
	articles.Get("/", h.ArticleHandler.GetArticles)
	articles.Get("/search/es", h.ArticleHandler.SearchInES)
	articles.Get("/:id", h.ArticleHandler.GetArticleByID)
	articles.Put("/:id/views", h.ArticleHandler.UpdateArticleViews)

	// Comments
	comments := v1.Group("/comments")
	comments.Get("/", h.CommentsHandler.GetComments)
	comments.Post("/", h.CommentsHandler.CreateComment)
	replies := v1.Group("/replies")
	replies.Post("/", h.CommentsHandler.CreateReply)
	// Friend Links
	friendLinks := v1.Group("/friend-links")
	friendLinks.Post("/", h.FriendLinkHandler.SaveFriendLink) // Save a friend link
	friendLinks.Get("/", h.FriendLinkHandler.GetFriendLinks)
}
