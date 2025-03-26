package kafka

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

func RevalidateHandler(msg kafka.Message, db *gorm.DB, redis *redis.Client) {
	fmt.Printf("Processing revalidate update: %s = %s\n", string(msg.Key), string(msg.Value))
	httpClient := &http.Client{Timeout: 10 * time.Second} // defining the http client here
	secret := os.Getenv("NEXT_SECRET")
	var data []byte
	if string(msg.Key) == "tag" {
		tags := strings.Split(string(msg.Value), ",")
		jsonData, err := json.Marshal(tags)
		data = []byte(fmt.Sprintf(`{"tag": %s,"secret": "%s"}`, string(jsonData), secret))
		body, err := SendRequest(httpClient, "/api/revalidateTag", data)
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Info("Response Body from first API:", string(body))
	}
	if string(msg.Key) == "path" {
		paths := strings.Split(string(msg.Value), ",")
		jsonData, err := json.Marshal(paths)
		data = []byte(fmt.Sprintf(`{"path": %s,"secret": "%s"}`, string(jsonData), secret))
		body, err := SendRequest(httpClient, "/api/revalidatePath", data)
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Info("Response Body from first API:", string(body))
	}
}
