package routes

import (
	"blog-server-go/handlers"
	"blog-server-go/middleware"

	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	ArticleHandler    handlers.ArticleHandler
	CommentsHandler   handlers.CommentsHandler
	WebSocketHandler  handlers.WebSocketHandler
	FriendLinkHandler handlers.FriendLinksHandler
	AppUserHandler    handlers.AppUserHandler
	FileHandler       handlers.FileHandler
	BlogConfigHandler handlers.BlogConfigHandler
	TaskHandler       handlers.TaskHandler
}

func SetupRoutes(app *fiber.App, h *Handlers) {

	// API Versioning
	v1 := app.Group("/v1")
	v1.Get("/ws", h.WebSocketHandler.UpgradeToWebSocket)

	// Articles
	articles := v1.Group("/articles")
	articles.Get("/", h.ArticleHandler.GetArticles)
	articles.Post("/", middleware.AdminMiddleware(), h.ArticleHandler.CreateArticle)   // 新建文章
	articles.Put("/:id", middleware.AdminMiddleware(), h.ArticleHandler.UpdateArticle) // 更新文章
	articles.Get("/search/es", h.ArticleHandler.SearchInES)
	articles.Get("/search/sync", h.ArticleHandler.SyncSQLToES)
	articles.Get("/:id", h.ArticleHandler.GetArticleByID)
	articles.Put("/:id/views", h.ArticleHandler.UpdateArticleViews)
	// 摘要
	articles.Get("/summary/:id", h.ArticleHandler.GetArticleSummary)
	articles.Post("/summary/:id", h.ArticleHandler.UpdateArticleSummary)
	articles.Get("/export/markdown/:id", h.ArticleHandler.ExportArticleMarkdown)
	articles.Get("/sync2dify/:id", h.ArticleHandler.SyncToDify)
	articles.Get("/sync/all2Dify", h.ArticleHandler.SyncAllToDify)

	// Comments
	comments := v1.Group("/comments")
	comments.Get("/", h.CommentsHandler.GetComments)
	comments.Post("/", h.CommentsHandler.CreateComment)
	// Replies
	replies := v1.Group("/replies")
	replies.Post("/", h.CommentsHandler.CreateReply)
	// Friend Links
	friendLinks := v1.Group("/friend-links")
	friendLinks.Post("/", h.FriendLinkHandler.SaveFriendLink) // Save a friend link
	friendLinks.Get("/", h.FriendLinkHandler.GetFriendLinks)
	//App User
	appUsers := v1.Group("/app-users")                         // 修改为 app-users
	appUsers.Post("/github/login", h.AppUserHandler.Github)    // github login
	appUsers.Post("/register", h.AppUserHandler.Register)      // 注册新用户
	appUsers.Post("/login", h.AppUserHandler.Login)            // 用户登录
	appUsers.Post("/logout", h.AppUserHandler.Logout)          // 用户注销
	appUsers.Get("/me", h.AppUserHandler.GetAuthenticatedUser) // 获取当前登录的用户信息
	// file
	files := v1.Group("/files")
	files.Post("/upload", middleware.AdminMiddleware(), h.FileHandler.UploadFile)
	files.Get("/list", middleware.AdminMiddleware(), h.FileHandler.ListFile)
	// config
	config := v1.Group("/config")
	config.Get("/site", h.BlogConfigHandler.GetSiteConfig)
	config.Post("/site", h.BlogConfigHandler.SaveSiteConfig)
	// task
	task := v1.Group("/task")
	task.Get("/", h.TaskHandler.GetTaskList)
	task.Post("/", h.TaskHandler.SaveTaskList)
	task.Put("/", h.TaskHandler.UpdateTaskList)
	task.Delete("/:id", h.TaskHandler.DeleteTask)
}
