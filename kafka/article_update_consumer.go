package kafka

import (
	"bytes"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/segmentio/kafka-go"
	"io"
	"net/http"
	"os"
	"time"
)

func SendRequest(client *http.Client, apiPath string, requestData []byte) ([]byte, error) {
	nextPublicBaseUrl := os.Getenv("NEXT_PUBLIC_BASE_URL")
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

func ArticleUpdateHandler(msg kafka.Message) {
	fmt.Printf("Processing article update: %s = %s\n", string(msg.Key), string(msg.Value))
	httpClient := &http.Client{Timeout: 10 * time.Second} // defining the http client here
	secret := os.Getenv("NEXT_SECRET")
	data := []byte(fmt.Sprintf(`{"tag": ["article"],"secret": "%s"}`, secret))
	body, err := SendRequest(httpClient, "/api/revalidateTag", data)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("Response Body from first API:", string(body))
	data = []byte(fmt.Sprintf(`{"path": ["/post/%s"],"secret": "%s"}`, msg.Value, secret))
	body, err = SendRequest(httpClient, "/api/revalidatePath", data)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("Response Body from first API:", string(body))
}

func NewArticleUpdateConsumer() *Consumer {
	consumer := NewConsumer(ArticleUpdateTopic)
	consumer.handler = ArticleUpdateHandler
	return consumer
}
