package kafka

import (
	"bytes"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"io"
	"net/http"
	"os"
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

func CreateMultiConsumer(topicHandlers map[string]MessageHandlerFunc) []*Consumer {
	var consumers []*Consumer
	for topic, handler := range topicHandlers {
		consumer := NewConsumer(topic)
		consumer.handler = handler
		consumers = append(consumers, consumer)
	}
	return consumers
}
