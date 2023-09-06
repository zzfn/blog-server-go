package config

import (
	"context"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"io"
	"os"
	"strconv"
)

func SetupDatabase() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASSWORD")

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", host, user, dbname, password)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数形式的表名
		},
	})
	if err != nil {
		return nil, err
	}
	log.Info("Database connected successfully")
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
	log.Info("Redis connected successfully")
	return client, nil
}

func SetupElasticsearch() (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://192.168.100.198:30015",
		},
		Username: "elastic",
		Password: "seFQV1U8KxzBUj*5fFB1",
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
			log.Errorf("Error closing response body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Elasticsearch returned non-200 status code: %d", res.StatusCode)
	}

	log.Info("Elasticsearch connected successfully")
	return esClient, nil
}
