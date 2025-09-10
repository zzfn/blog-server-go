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
		"model": "google/gemini-2.5-flash",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("请为以下文章生成一段简洁的摘要，以平均阅读速度300字/分钟为基准，根据摘要字数计算预计阅读时间,将摘要与阅读时间合并为一段，格式为\"[摘要内容]（预计阅读时间：X分钟）\",控制在100字以内：\n\n%s", article.Content),
			},
		},
		"max_tokens": 150,
	}

	jsonBody, err := json.Marshal(reqBody)

	if err != nil {
		log.Error("Error marshaling request body:", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Error("Error creating request:", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENROUTER_API_KEY"))
	req.Header.Set("HTTP-Referer", "https://zzfzzf.com")
	req.Header.Set("X-Title", "Blog Article Summarizer")

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error calling OpenRouter API:", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Error("Error reading response:", err)
	}
	log.Error("OpenRouter API Response:", string(respBody))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		log.Error("OpenRouter API error")
		return
	}

	// 解析响应
	var openrouterResult map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(respBody)).Decode(&openrouterResult); err != nil {
		log.Error("Failed to parse response")
		return
	}

	// 提取摘要并保存到 Redis
	if choices, ok := openrouterResult["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					log.Info("OpenRouter API Response:", content)
					err = redis.HSet(ctx, "articleSummary", id, content).Err()
					if err != nil {
						log.Error("Failed to save summary to Redis:", err)
					}
				}
			}
		}
	}
}
