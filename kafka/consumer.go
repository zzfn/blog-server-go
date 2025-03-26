package kafka

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type MessageHandlerFunc func(message kafka.Message, db *gorm.DB, redis *redis.Client)

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandlerFunc
	db      *gorm.DB
	redis   *redis.Client
}

func NewConsumer(topic string, db *gorm.DB, redis *redis.Client) *Consumer {
	brokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	groupID := os.Getenv("KAFKA_GROUP_ID")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddress},
		GroupID:   groupID,
		Topic:     topic,
		Partition: 0,
		MinBytes:  10,
		MaxBytes:  10e6,
	})
	return &Consumer{
		reader: reader,
		db:     db,
		redis:  redis,
	}
}

func (c *Consumer) Start() {
	go func() {
		for {
			log.Info("Waiting for message from topic:", c.reader.Config().Topic) // 打印 topic
			msg, err := c.reader.ReadMessage(context.Background())
			if err != nil {
				log.Fatalf("Failed to read message: %v", err)
			}
			c.handler(msg, c.db, c.redis)
			log.Error("message at offset %d: %s = %s\n\n", msg.Offset, string(msg.Key), string(msg.Value))
		}
	}()
}

func (c *Consumer) Close() {
	c.reader.Close()
}
