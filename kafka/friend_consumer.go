package kafka

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/segmentio/kafka-go"
	"net/http"
	"os"
	"time"
)

func FriendHandler(msg kafka.Message) {
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
