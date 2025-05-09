package main

import (
	"blog-server-go/common"
	"blog-server-go/config"
	"blog-server-go/handlers"
	"blog-server-go/kafka"
	"blog-server-go/middleware"
	"blog-server-go/routes"
	"blog-server-go/tasks"
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func StartServices(app *fiber.App, kafkaConsumer []*kafka.Consumer) {
	// Start Kafka consumer
	for _, consumer := range kafkaConsumer {
		consumer.Start()
	}
	if err := app.Listen(":8000"); err != nil {
		log.Fatalf("Failed to start Fiber app: %v", err)
	}
}

func ShutdownServices(app *fiber.App, sqlDB *sql.DB, redisClient *redis.Client, kafkaConsumer []*kafka.Consumer) {
	if err := sqlDB.Close(); err != nil {
		log.Errorf("Error closing database: %v", err)
	}

	if err := redisClient.Close(); err != nil {
		log.Errorf("Error closing Redis client: %v", err)
	}

	for _, consumer := range kafkaConsumer {
		consumer.Close()
	}
	_ = app.Shutdown()
}

func NewFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 20 * 1024 * 1024})
	app.Use(cors.New())
	app.Use(middleware.LoggingMiddleware)
	app.Use(middleware.AuthMiddleware)
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

func NewWebSocketHandler(db *gorm.DB, redis *redis.Client, es *elasticsearch.Client) *handlers.WebSocketHandler {
	return &handlers.WebSocketHandler{
		BaseHandler: handlers.BaseHandler{
			DB:    db,
			Redis: redis,
			ES:    es,
		},
	}
}

// 注入BaseHandler
func NewBaseHandler(db *gorm.DB, redisClient *redis.Client, esClient *elasticsearch.Client, kafkaProducer *kafka.Producer, wsHandler *handlers.WebSocketHandler) handlers.BaseHandler {
	return handlers.BaseHandler{DB: db, Redis: redisClient, ES: esClient, KafkaProducer: kafkaProducer, WSHandler: wsHandler}
}

func RegisterRoutes(app *fiber.App, baseHandler handlers.BaseHandler) {
	articleHandler := handlers.ArticleHandler{BaseHandler: baseHandler}
	commentsHandler := handlers.CommentsHandler{BaseHandler: baseHandler}
	appUserHandler := handlers.AppUserHandler{BaseHandler: baseHandler}
	webSocketHandler := handlers.WebSocketHandler{BaseHandler: baseHandler}
	friendLinksHandler := handlers.FriendLinksHandler{BaseHandler: baseHandler}
	blogConfigHandler := handlers.BlogConfigHandler{BaseHandler: baseHandler}
	taskHandler := handlers.TaskHandler{BaseHandler: baseHandler}
	financialTransactionHandler := handlers.FinancialTransactionHandler{BaseHandler: baseHandler}
	allHandlers := &routes.Handlers{
		ArticleHandler:              articleHandler,
		CommentsHandler:             commentsHandler,
		WebSocketHandler:            webSocketHandler,
		AppUserHandler:              appUserHandler,
		FriendLinkHandler:           friendLinksHandler,
		BlogConfigHandler:           blogConfigHandler,
		TaskHandler:                 taskHandler,
		FinancialTransactionHandler: financialTransactionHandler,
	}
	routes.SetupRoutes(app, allHandlers)
}

func NewElasticsearchClient() (*elasticsearch.Client, error) {
	esClient, err := config.SetupElasticsearch()
	common.HandleError(err, "Error setting up Elasticsearch:")
	return esClient, err
}
func main() {
	// 初始化Fiber app
	app := NewFiberApp()

	// 初始化数据库连接

	db, sqlDB := NewDatabaseConnection()
	defer sqlDB.Close()

	// 初始化Redis客户端
	redisClient := NewRedisClient()
	defer redisClient.Close()

	// 初始化Elasticsearch客户端
	esClient, err := NewElasticsearchClient()
	if err != nil {
		log.Fatalf("Error initializing Elasticsearch client: %v", err)
	}

	// 初始化WebSocketHandler
	wsHandler := NewWebSocketHandler(db, redisClient, esClient)

	// 初始化BaseHandler
	kafkaProducer := kafka.NewProducer()
	baseHandler := NewBaseHandler(db, redisClient, esClient, kafkaProducer, wsHandler)
	topicHandlers := map[string]kafka.MessageHandlerFunc{
		kafka.ArticleUpdateTopic:    kafka.ArticleHandler,
		kafka.FriendUpdateTopic:     kafka.FriendHandler,
		kafka.RevalidateUpdateTopic: kafka.RevalidateHandler,
	}
	// 初始化Kafka消费者
	kafkaConsumer := kafka.CreateMultiConsumer(topicHandlers, db, redisClient)

	// 注册路由
	RegisterRoutes(app, baseHandler)
	// 开始定时任务
	go tasks.StartCronJobs()
	// 启动服务
	StartServices(app, kafkaConsumer)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		ShutdownServices(app, sqlDB, redisClient, kafkaConsumer)
		os.Exit(0)
	}()
}
