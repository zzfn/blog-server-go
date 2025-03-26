package kafka

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

func FriendHandler(msg kafka.Message, db *gorm.DB, redis *redis.Client) {
	fmt.Printf("Processing friend update: %s = %s\n", string(msg.Key), string(msg.Value))
	httpClient := &http.Client{Timeout: 10 * time.Second} // defining the http client here
	secret := os.Getenv("NEXT_SECRET")
	data := []byte(fmt.Sprintf(`{"path": ["/friends"],"secret": "%s"}`, secret))
	body, err := SendRequest(httpClient, "/api/revalidatePath", data)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("Response Body from first API:", string(body))
}
