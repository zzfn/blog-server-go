package kafka

import (
	"blog-server-go/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"io"
	"net/http"
	"os"
	"time"
)

func ArticleHandler(msg kafka.Message, db *gorm.DB, redis *redis.Client) {
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

	var ctx = context.Background()
	id := string(msg.Value)
	var article models.Article
	result := db.Take(&article, id)
	if result.Error != nil {
		log.Error("Error retrieving article:", result.Error)
		return
	}

	// 准备请求体
	reqBody := map[string]interface{}{
		"inputs": map[string]string{
			"blog_content": article.Content,
		},
		"response_mode": "blocking",
		"user":          "system",
	}

	jsonBody, err := json.Marshal(reqBody)

	if err != nil {
		log.Error("Error marshaling request body:", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", os.Getenv("DIFY_WORKFLOW_URL"), bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Error creating request:", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("DIFY_API_KEY"))

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error calling Dify API:", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Error("Error reading response:", err)
	}
	log.Error("Dify API Response:", string(respBody))

	// 重新创建reader供后续使用
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Error("Dify API error")
	}

	// 解析响应
	var difyResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&difyResult); err != nil {
		log.Error("Failed to parse response")
	}

	// 提取摘要并保存到 Redis
	if data, ok := difyResult["data"].(map[string]interface{}); ok {
		if outputs, ok := data["outputs"].(map[string]interface{}); ok {
			if text, ok := outputs["text"].(string); ok {
				log.Info("Dify API Response:", text)
				err = redis.HSet(ctx, "articleSummary", id, text).Err()
				if err != nil {
					log.Error("Failed to save summary to Redis:", err)
				}
			}
		}
	}
}
