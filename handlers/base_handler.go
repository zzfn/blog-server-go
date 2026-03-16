package handlers

import (
	"blog-server-go/kafka"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BaseHandler struct {
	DB            *gorm.DB
	Redis         *redis.Client
	Meili         meilisearch.ServiceManager
	KafkaProducer *kafka.Producer
	WSHandler     *WebSocketHandler
}
