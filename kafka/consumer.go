package kafka

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

type MessageHandlerFunc func(message kafka.Message)

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandlerFunc
}

func NewConsumer(topic string) *Consumer {
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
	return &Consumer{reader: reader}
}

func (c *Consumer) Start() {
	go func() {
		for {
			msg, err := c.reader.ReadMessage(context.Background())
			if err != nil {
				log.Fatalf("Failed to read message: %v", err)
			}
			c.handler(msg)
			fmt.Printf("message at offset %d: %s = %s\n", msg.Offset, string(msg.Key), string(msg.Value))
		}
	}()
}

func (c *Consumer) Close() {
	c.reader.Close()
}
