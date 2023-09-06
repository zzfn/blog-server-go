package main

import (
	"blog-server-go/common"
	"blog-server-go/config"
	"blog-server-go/handlers" // 假设这里包含 ArticleHandler
	"blog-server-go/middleware"
	"blog-server-go/routes"
	"context"
	"database/sql"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func RegisterHooks(lc fx.Lifecycle, app *fiber.App, sqlDB *sql.DB, redisClient *redis.Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(":3000"); err != nil {
					log.Fatalf("Failed to start Fiber app: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 关闭数据库连接
			if err := sqlDB.Close(); err != nil {
				log.Errorf("Error closing database: %v", err)
			}

			// 关闭Redis客户端
			if err := redisClient.Close(); err != nil {
				log.Errorf("Error closing Redis client: %v", err)
			}
			return app.Shutdown()
		},
	})
}

func NewFiberApp() *fiber.App {
	app := fiber.New()
	app.Use(middleware.ResponseMiddleware)
	return app
}

func NewDatabaseConnection() (*gorm.DB, *sql.DB) {
	db, err := config.SetupDatabase()
	common.HandleError(err, "Failed to connect to database:")
	sqlDB, err := db.DB()
	common.HandleError(err, "Failed to get DB object:")
	return db, sqlDB
}

func NewRedisClient() *redis.Client {
	redisClient, err := config.SetupRedis()
	common.HandleError(err, "Error setting up Redis:")
	return redisClient
}

func NewArticleHandler(db *gorm.DB, redisClient *redis.Client, esClient *elasticsearch.Client) handlers.ArticleHandler {
	return handlers.ArticleHandler{
		DB:    db,
		Redis: redisClient,
		ES:    esClient,
	}
}

func RegisterRoutes(app *fiber.App, handler handlers.ArticleHandler) {
	routes.SetupArticleRoutes(app, handler)
}
func NewElasticsearchClient() (*elasticsearch.Client, error) {
	esClient, err := config.SetupElasticsearch()
	common.HandleError(err, "Error setting up Elasticsearch:")
	return esClient, err
}
func main() {
	app := fx.New(
		// Provides
		fx.Provide(
			NewFiberApp,
			NewDatabaseConnection,
			NewRedisClient,
			NewElasticsearchClient,
			NewArticleHandler,
		),
		// Invokes
		fx.Invoke(RegisterHooks),
		fx.Invoke(RegisterRoutes),
	)

	app.Run()
}
