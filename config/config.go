package config

import (
	"context"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"io"
	"log"
	"os"
	"strconv"
	"time"
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

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", host, user, dbname, password)
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

func SetupElasticsearch() (*elasticsearch.Client, error) {
	esAddress := os.Getenv("ES_ADDRESS")
	esUsername := os.Getenv("ES_USERNAME")
	esPassword := os.Getenv("ES_PASSWORD")
	cfg := elasticsearch.Config{
		Addresses: []string{esAddress},
		Username:  esUsername,
		Password:  esPassword,
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("Error initializing Elasticsearch: %w", err)
	}

	// Ping the Elasticsearch server to get StatusCode
	res, err := esClient.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error pinging Elasticsearch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fiberlog.Errorf("Error closing response body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Elasticsearch returned non-200 status code: %d", res.StatusCode)
	}

	fiberlog.Info("Elasticsearch connected successfully")
	return esClient, nil
}
