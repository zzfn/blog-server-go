package handlers

import (
	"context"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"sync"
)

type WebSocketHandler struct {
	BaseHandler
}

var (
	clientsMutex sync.Mutex
	clients      = make(map[*websocket.Conn]bool) // 使用map作为set来存储所有客户端的连接
)

func (wsh *WebSocketHandler) HandleConnection(conn *websocket.Conn, userId string) {
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()
	var ctx = context.Background()
	wsh.Redis.ZIncrBy(ctx, "online_users", 1, userId)
	notifyAllClients(ctx, wsh.Redis)

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()

		newScore := wsh.Redis.ZIncrBy(ctx, "online_users", -1, userId).Val()
		if newScore <= 0 {
			wsh.Redis.ZRem(ctx, "online_users", userId)
		}
		notifyAllClients(ctx, wsh.Redis)
		err := conn.Close()
		if err != nil {
			// Handle or log the error
		}
	}()
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			break // Break out of the loop if there's an error reading the message
		}
		// You can handle or broadcast the message here if needed
		// For now, I'll just echo back the message
		if err := conn.WriteMessage(messageType, p); err != nil {
			// Handle or log the error
			break
		}
	}
}

var websocketUpgrade = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func (wsh *WebSocketHandler) UpgradeToWebSocket(c *fiber.Ctx) error {
	userId := c.Query("userId") // 例如, 从查询参数获取

	err := websocketUpgrade.Upgrade(c.Context(), func(conn *websocket.Conn) {
		wsh.HandleConnection(conn, userId)
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to upgrade to WebSocket"})
	}
	return nil
}
func notifyAllClients(ctx context.Context, rdb *redis.Client) {
	onlineCount := rdb.ZCard(ctx, "online_users").Val()
	message := []byte(fmt.Sprintf("Online users: %d", onlineCount))

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			// Handle or log error
		}
	}
}
