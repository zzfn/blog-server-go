package kafka

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/segmentio/kafka-go"
	"net/http"
	"os"
	"time"
)

func ArticleHandler(msg kafka.Message) {
	fmt.Printf("Processing article update: %s = %s\n", string(msg.Key), string(msg.Value))
	httpClient := &http.Client{Timeout: 10 * time.Second} // defining the http client here
	secret := os.Getenv("NEXT_SECRET")
	var data []byte
	data = []byte(fmt.Sprintf(`{"tag": ["article","/post/%s"],"secret": "%s"}`, msg.Value, secret))
	body, err := SendRequest(httpClient, "/api/revalidateTag", data)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("Response Body from first API:", string(body))
	data = []byte(fmt.Sprintf(`{"path": ["/post/%s","/api/feed.xml"],"secret": "%s"}`, msg.Value, secret))
	body, err = SendRequest(httpClient, "/api/revalidatePath", data)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("Response Body from first API:", string(body))
}
