package kafka

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func SendRequest(client *http.Client, apiPath string, requestData []byte) ([]byte, error) {
	nextPublicBaseUrl := os.Getenv("NEXT_PUBLIC_BASE_URL")
	log.Info("nextPublicBaseUrl", nextPublicBaseUrl)
	req, err := http.NewRequest("POST", nextPublicBaseUrl+apiPath, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}

	return body, nil
}

func CreateMultiConsumer(topicHandlers map[string]MessageHandlerFunc, db *gorm.DB, redis *redis.Client) []*Consumer {
	var consumers []*Consumer
	for topic, handler := range topicHandlers {
		consumer := NewConsumer(topic, db, redis)
		consumer.handler = handler
		consumers = append(consumers, consumer)
	}
	return consumers
}
