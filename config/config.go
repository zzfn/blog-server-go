package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func SetupDatabase() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASSWORD")
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // 慢 SQL 阈值
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // 彩色打印
		},
	)

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s connect_timeout=5 timezone=Asia/Shanghai", host, user, dbname, password)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数形式的表名
		},
	})
	if err != nil {
		return nil, err
	}
	fiberlog.Info("Database connected successfully")
	return db, nil
}

var ctx = context.Background()

func SetupRedis() (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB")) // 默认是0

	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	}

	client := redis.NewClient(options)
	_, err := client.Ping(ctx).Result()

	if err != nil {
		return nil, err
	}
	fiberlog.Info("Redis connected successfully")
	return client, nil
}

func SetupMeilisearch() (meilisearch.ServiceManager, error) {
	meiliAddress := os.Getenv("MEILI_ADDRESS")
	if meiliAddress == "" {
		meiliAddress = os.Getenv("MEILI_HOST")
	}
	meiliAPIKey := os.Getenv("MEILI_API_KEY")

	if meiliAddress == "" {
		return nil, fmt.Errorf("MEILI_ADDRESS is required")
	}

	meiliClient := meilisearch.New(meiliAddress, meilisearch.WithAPIKey(meiliAPIKey))

	health, err := meiliClient.Health()
	if err != nil {
		return nil, fmt.Errorf("error pinging Meilisearch: %w", err)
	}
	if health.Status != "available" {
		return nil, fmt.Errorf("Meilisearch is not available: %s", health.Status)
	}

	fiberlog.Info("Meilisearch connected successfully")
	return meiliClient, nil
}
