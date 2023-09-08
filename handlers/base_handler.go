package handlers

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BaseHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
	ES    *elasticsearch.Client
}