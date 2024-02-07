package kafka

import (
	"context"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

const (
	ArticleUpdateTopic = "ARTICLE_UPDATE_TOPIC"
	FriendUpdateTopic  = "FRIEND_UPDATE_TOPIC"
)

func NewProducer() *Producer {
	brokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Balancer: &kafka.LeastBytes{},
	}
	return &Producer{writer: w}
}

func (p *Producer) Start() {
	// 在这里你可以添加一些启动逻辑，如果有的话
}

func (p *Producer) Close() {
	p.writer.Close()
}

func (p *Producer) ProduceMessage(topic, key, message string) {
	p.writer.Topic = topic
	err := p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: []byte(message),
		},
	)
	if err != nil {
		log.Fatalf("Failed to write message: %v", err)
	}
}
