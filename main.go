package main

import (
	"blog-server-go/common"
	"blog-server-go/config"
	"blog-server-go/handlers"
	"blog-server-go/kafka"
	"blog-server-go/middleware"
	"blog-server-go/routes"
	"context"
	"database/sql"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func RegisterHooks(lc fx.Lifecycle, app *fiber.App, sqlDB *sql.DB, redisClient *redis.Client, kafkaConsumer *kafka.Consumer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(":8000"); err != nil {
					log.Fatalf("Failed to start Fiber app: %v", err)
				}
			}()
			// Kafka消费者启动逻辑
			go kafkaConsumer.Start()
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

			kafkaConsumer.Close()
			return app.Shutdown()
		},
	})
}

func NewFiberApp() *fiber.App {
	app := fiber.New()
	app.Use(cors.New())
	app.Use(middleware.ResponseMiddleware)
	app.Use(middleware.LoggingMiddleware)
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

func NewWebSocketHandler(db *gorm.DB, redis *redis.Client, es *elasticsearch.Client) *handlers.WebSocketHandler {
	return &handlers.WebSocketHandler{
		BaseHandler: handlers.BaseHandler{
			DB:    db,
			Redis: redis,
			ES:    es,
		},
	}
}
func NewBaseHandler(db *gorm.DB, redisClient *redis.Client, esClient *elasticsearch.Client) handlers.BaseHandler {
	return handlers.BaseHandler{DB: db, Redis: redisClient, ES: esClient}
}

func RegisterRoutes(app *fiber.App, baseHandler handlers.BaseHandler) {
	articleHandler := handlers.ArticleHandler{BaseHandler: baseHandler}
	commentsHandler := handlers.CommentsHandler{BaseHandler: baseHandler}
	appUserHandler := handlers.AppUserHandler{BaseHandler: baseHandler}
	webSocketHandler := handlers.WebSocketHandler{BaseHandler: baseHandler}
	friendLinksHandler := handlers.FriendLinksHandler{BaseHandler: baseHandler}
	allHandlers := &routes.Handlers{
		ArticleHandler:    articleHandler,
		CommentsHandler:   commentsHandler,
		WebSocketHandler:  webSocketHandler,
		AppUserHandler:    appUserHandler,
		FriendLinkHandler: friendLinksHandler,
	}
	routes.SetupRoutes(app, allHandlers)
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
			NewWebSocketHandler,
			NewBaseHandler,
			kafka.NewProducer, // 假设这是你初始化Kafka生产者的函数
			kafka.NewConsumer, // 假设这是你初始化Kafka消费者的函数
		),
		// Invokes
		fx.Invoke(RegisterHooks),
		fx.Invoke(RegisterRoutes),
	)

	app.Run()
}
