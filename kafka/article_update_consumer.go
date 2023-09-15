package kafka

import (
	"bytes"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/segmentio/kafka-go"
	"io"
	"net/http"
	"os"
)

func ArticleUpdateHandler(msg kafka.Message) {
	fmt.Printf("Processing article update: %s = %s\n", string(msg.Key), string(msg.Value))
	client := &http.Client{}
	secret := os.Getenv("NEXT_SECRET")
	nextPublicBaseUrl := os.Getenv("NEXT_PUBLIC_BASE_URL")
	data := []byte(`{"tag": ["article"],"secret": "` + secret + `"}`)
	req, err := http.NewRequest("POST", nextPublicBaseUrl+"/api/revalidateTag", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}
	log.Info("Request Body", string(body))
}

func NewArticleUpdateConsumer() *Consumer {
	consumer := NewConsumer(ArticleUpdateTopic)
	consumer.handler = ArticleUpdateHandler
	return consumer
}
