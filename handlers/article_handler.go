package handlers

import (
	"archive/zip"
	"blog-server-go/common"
	"blog-server-go/kafka"
	"blog-server-go/models"
	"blog-server-go/services"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ArticleHandler 处理与文章相关的请求
type ArticleHandler struct {
	BaseHandler
	LLMService *services.LLMService
}

// GetArticles 获取所有文章
func (ah *ArticleHandler) GetArticles(c *fiber.Ctx) error {
	var articles []models.Article

	orderStr := c.Query("order", "created_at desc")
	limitStr := c.Query("limit")
	isActive := c.Query("isActive", "true")
	tagStr := c.Query("tag")
	isRss := c.Query("rss")
	log.Info("isRss", isRss)
	fields := "id,title,tag,created_at,updated_at,sort_order"
	if isRss == "true" {
		fields += ",CONTENT"
	}
	if isActive == "true" {
		fields += ",is_active"
	}

	query := ah.DB.Select(fields).Where("is_deleted", false).Where("is_active", isActive).Order("sort_order desc").Order(orderStr)

	if tagStr != "" {
		query = query.Where("tag = ?", tagStr)
	}
	// 如果提供了 limit 参数，则应用它
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid limit value")
		}
		query = query.Limit(limit)
	}
	result := query.Find(&articles)
	if result.Error != nil {
		log.Error(result.Error)
		return c.Status(fiber.StatusInternalServerError).SendString(result.Error.Error())
	}

	// 从Redis获取所有文章的summary
	var ctx = context.Background()
	allSummaries, _ := ah.Redis.HGetAll(ctx, "articleSummary").Result()

	// 为每篇文章设置summary
	for i := range articles {
		articles[i].Summary = allSummaries[string(articles[i].ID)]
	}

	return c.JSON(articles)
}

func (ah *ArticleHandler) GetArticleByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article
	result := ah.DB.Take(&article, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Errorf("Failed to retrieve article: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	// 从Redis获取summary
	var ctx = context.Background()
	summary, _ := ah.Redis.HGet(ctx, "articleSummary", id).Result()
	article.Summary = summary

	return c.JSON(article)
}

func (ah *ArticleHandler) UpdateArticleViews(c *fiber.Ctx) error {
	id := c.Params("id")
	article := models.Article{
		BaseModel: models.BaseModel{ID: models.SnowflakeID(id)},
	}
	ah.DB.Select("view_count").Take(&article, id)
	ip := common.GetConnectingIp(c)
	var ctx = context.Background()
	if ah.Redis.ZScore(ctx, "article_views:"+id, ip).Val() > float64(time.Now().Add(-time.Minute*30).UnixNano()/int64(time.Millisecond)) {
		return c.JSON(article.ViewCount)
	}
	ah.Redis.ZAdd(ctx, "article_views:"+id, redis.Z{Score: float64(time.Now().UnixNano() / int64(time.Millisecond)), Member: ip})
	result := ah.DB.Model(&article).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	if result.Error != nil {
		log.Errorf("Failed to update article views: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.JSON(article.ViewCount)
}

// CreateArticle 创建文章
func (ah *ArticleHandler) CreateArticle(c *fiber.Ctx) error {
	var article models.Article
	if err := c.BodyParser(&article); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}

	result := ah.DB.Create(&article)
	if result.Error != nil {
		log.Errorf("Failed to save article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	ah.KafkaProducer.ProduceMessage(kafka.ArticleUpdateTopic, "id", string(article.ID))
	return c.Status(fiber.StatusCreated).JSON(article)
}

// UpdateArticle 更新文章
func (ah *ArticleHandler) UpdateArticle(c *fiber.Ctx) error {
	var inputArticle models.Article

	id := c.Params("id")

	if err := c.BodyParser(&inputArticle); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无法解析JSON",
		})
	}

	var existingArticle models.Article
	ah.Redis.HDel(context.Background(), "articleSummary", id)
	result := ah.DB.Take(&existingArticle, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Errorf("Failed to retrieve article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	// 更新文章内容
	result = ah.DB.Model(&existingArticle).Updates(map[string]interface{}{
		"title":     inputArticle.Title,
		"content":   inputArticle.Content,
		"tag":       inputArticle.Tag,
		"is_active": inputArticle.IsActive,
	})
	ah.KafkaProducer.ProduceMessage(kafka.ArticleUpdateTopic, "id", id)
	if result.Error != nil {
		log.Errorf("Failed to update article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	return c.JSON(existingArticle)
}

func (ah *ArticleHandler) SearchInES(c *fiber.Ctx) error {
	var keyword = c.Query("keyword")
	ctx := context.Background()
	_, err := ah.Redis.ZIncrBy(ctx, "searchKeywords", 1, keyword).Result()
	common.HandleError(err, "Error incrementing keyword score:")

	var buf map[string]interface{}
	buf = map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{"match_phrase": map[string]interface{}{"content": keyword}},
					{"match_phrase": map[string]interface{}{"title": keyword}},
					{"match_phrase": map[string]interface{}{"tag": keyword}},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"content": map[string]interface{}{},
				"title":   map[string]interface{}{},
				"tag":     map[string]interface{}{},
			},
		},
	}
	var b []byte
	b, err = json.Marshal(buf)
	if err != nil {
		log.Fatalf("Error marshaling query: %s", err)
	}

	var res *esapi.Response
	res, err = ah.ES.Search(
		ah.ES.Search.WithIndex("blog"),
		ah.ES.Search.WithBody(bytes.NewReader(b)),
		ah.ES.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if res.IsError() {
		log.Fatalf("Error: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error parsing the response body: " + err.Error())
	}

	// 提取 hits 对象
	hits, ok := r["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		log.Error("Error: Hits not found")
	}

	// 准备用于返回的 articles 切片
	var articles []map[string]interface{}

	// 遍历每一个 hit 并提取需要的字段
	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})["_source"].(map[string]interface{})
		highlight := hit.(map[string]interface{})["highlight"].(map[string]interface{})
		docId := hit.(map[string]interface{})["_id"]

		// 创建一个新的 article map 来存储字段和高亮信息
		article := make(map[string]interface{})
		if !ok {
			log.Error("Could not convert _id to string")
			continue // 或返回错误
		}
		article["id"] = docId
		article["title"] = hitMap["title"]
		article["content"] = hitMap["content"]
		article["tag"] = hitMap["tag"]

		// 如果存在高亮信息，则用高亮信息替换原字段
		if title, ok := highlight["title"].([]interface{}); ok {
			article["title"] = title[0]
		}
		if content, ok := highlight["content"].([]interface{}); ok {
			article["content"] = content[0]
		}
		if tag, ok := highlight["tag"].([]interface{}); ok {
			article["tag"] = tag[0]
		}

		articles = append(articles, article)
	}

	return c.JSON(articles)
}
func (ah *ArticleHandler) SyncSQLToES(c *fiber.Ctx) error {
	// 删除已存在的索引
	res, err := ah.ES.Indices.Delete([]string{"blog"})
	if err != nil {
		log.Fatalf("Failed deleting index: %v", err)
	}
	defer res.Body.Close()
	// 创建新索引
	res, err = ah.ES.Indices.Create("blog")
	if err != nil {
		log.Fatalf("Failed creating index: %v", err)
	}
	defer res.Body.Close()

	// 设置映射
	propertiesMapping := map[string]interface{}{
		"title": map[string]interface{}{
			"type":            "text",
			"analyzer":        "smartcn",
			"search_analyzer": "smartcn",
		},
		// ... 添加其他字段映射
	}
	mapping := map[string]interface{}{"properties": propertiesMapping}
	b, _ := json.Marshal(mapping)
	res, err = ah.ES.Indices.PutMapping([]string{"blog"}, bytes.NewReader(b))
	if err != nil {
		log.Fatalf("Failed setting mappings: %v", err)
	}
	defer res.Body.Close()

	query := ah.DB.Where("is_deleted", false).Where("is_active", true)
	var articles []models.Article

	_ = query.Find(&articles)

	// 执行批量索引操作
	var buf bytes.Buffer
	for _, article := range articles {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_id":    article.ID,
				"_index": "blog",
			},
		}
		data, _ := json.Marshal(meta)
		buf.Write(data)
		buf.WriteByte('\n')

		body, _ := json.Marshal(article)
		buf.Write(body)
		buf.WriteByte('\n')
	}

	res, err = ah.ES.Bulk(&buf)
	if err != nil {
		log.Fatalf("Failed bulk indexing: %v", err)
	}
	defer res.Body.Close()

	return c.JSON(nil)
}

func (ah *ArticleHandler) GetArticleSummary(c *fiber.Ctx) error {
	id := c.Params("id")
	var ctx = context.Background()
	summary, _ := ah.Redis.HGet(ctx, "articleSummary", id).Result()

	// 如果 summary 为空，则调用 LLM 生成
	if summary == "" {
		var article models.Article
		result := ah.DB.Take(&article, id)
		if result.Error != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
		}

		// 使用 Eino 生成摘要
		messages := []*schema.Message{
			schema.UserMessage(fmt.Sprintf("请为以下文章生成一段简洁的摘要，以平均阅读速度300字/分钟为基准，根据摘要字数计算预计阅读时间,将摘要与阅读时间合并为一段，格式为\"[摘要内容](预计阅读时间:X分钟)\",控制在100字以内:\n\n%s", article.Content)),
		}

		content, err := ah.LLMService.GenerateText(ctx, messages)
		if err != nil {
			log.Errorf("Failed to generate summary: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate summary"})
		}

		summary = content
		// 保存到 Redis
		if err := ah.Redis.HSet(ctx, "articleSummary", id, summary).Err(); err != nil {
			log.Error("Failed to save summary to Redis:", err)
		}
	}

	return c.JSON(summary)
}

func (ah *ArticleHandler) UpdateArticleSummary(c *fiber.Ctx) error {
	// 解析入参 summary 存入redis
	// 入参 {summary:"xxxxxx"}
	id := c.Params("id")
	var ctx = context.Background()

	var inputArticle struct {
		Summary string `json:"summary"`
	}
	if err := c.BodyParser(&inputArticle); err != nil {
		common.HandleError(err, "Error incrementing keyword score:")

		return c.Status(fiber.StatusBadRequest).JSON(common.NewResponse(fiber.StatusBadRequest, "Invalid request body", nil))
	}
	if err := ah.Redis.HSet(ctx, "articleSummary", id, inputArticle.Summary).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(common.NewResponse(fiber.StatusInternalServerError, "Failed to update article summary", nil))
	}
	return c.JSON("Article summary updated successfully")
}

// ExportArticleMarkdown handles exporting article as markdown file
func (ah *ArticleHandler) ExportArticleMarkdown(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article
	result := ah.DB.Take(&article, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Error("Article not found:", id)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Error("Database error:", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve article"})
	}

	// 验证文章标题
	if article.Title == "" {
		log.Error("Article title is empty for ID:", id)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid article title"})
	}

	// 处理文件名编码
	encodedFilename := url.QueryEscape(article.Title + ".md")

	// 设置响应头
	c.Response().Header.Set("Content-Type", "application/octet-stream")
	c.Response().Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedFilename))

	// 返回二进制数据
	return c.Send([]byte(article.Content))
}

// ExportAllArticlesMarkdown handles exporting all articles as a zip file
func (ah *ArticleHandler) ExportAllArticlesMarkdown(c *fiber.Ctx) error {
	var articles []models.Article
	result := ah.DB.Find(&articles)
	if result.Error != nil {
		log.Error("Database error:", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve articles"})
	}

	if len(articles) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No articles found"})
	}

	// 创建一个内存中的 zip 文件
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// 将每篇文章添加到 zip 文件中
	for _, article := range articles {
		// 处理文件名
		filename := article.Title
		if filename == "" {
			filename = fmt.Sprintf("article_%d", article.ID)
		}
		filename = filename + ".md"

		// 创建 zip 文件中的文件
		writer, err := zipWriter.Create(filename)
		if err != nil {
			log.Error("Error creating zip entry:", err)
			zipWriter.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create zip file"})
		}

		// 写入文章内容
		_, err = writer.Write([]byte(article.Content))
		if err != nil {
			log.Error("Error writing to zip:", err)
			zipWriter.Close()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write to zip file"})
		}
	}

	// 关闭 zip writer
	err := zipWriter.Close()
	if err != nil {
		log.Error("Error closing zip writer:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to close zip file"})
	}

	// 设置响应头
	c.Response().Header.Set("Content-Type", "application/zip")
	c.Response().Header.Set("Content-Disposition", "attachment; filename*=UTF-8''articles.zip")

	// 返回 zip 文件
	return c.Send(zipBuffer.Bytes())
}

func (ah *ArticleHandler) SyncToDify(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article
	result := ah.DB.Take(&article, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Error("Article not found:", id)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Error("Database error:", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve article"})
	}

	// 准备文章内容
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: %s\n", article.Title))
	buf.WriteString(fmt.Sprintf("date: %s\n", article.CreatedAt.Format("2006-01-02 15:04:05")))
	if article.Tag != "" {
		buf.WriteString(fmt.Sprintf("tags: [%s]\n", article.Tag))
	}
	buf.WriteString("---\n\n")
	buf.WriteString(article.Content)

	// 准备 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加 data 字段
	dataJson := map[string]interface{}{
		"indexing_technique": "high_quality",
		"doc_type":           "web_page",
		"doc_metadata": map[string]interface{}{
			"title": article.Title,
			"url":   id,
		},
		"process_rule": map[string]interface{}{
			"rules": map[string]interface{}{
				"pre_processing_rules": []map[string]interface{}{
					{"id": "remove_extra_spaces", "enabled": true},
					{"id": "remove_urls_emails", "enabled": true},
				},
				"segmentation": map[string]interface{}{
					"separator":  "###",
					"max_tokens": 500,
				},
			},
			"mode": "custom",
		},
	}
	dataField, err := writer.CreateFormField("data")
	if err != nil {
		log.Error("Failed to create form field:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create form"})
	}
	if err := json.NewEncoder(dataField).Encode(dataJson); err != nil {
		log.Error("Failed to encode data:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to encode data"})
	}

	// 添加文件
	fileField, err := writer.CreateFormFile("file", article.Title+".md")
	if err != nil {
		log.Error("Failed to create form file:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create form file"})
	}
	if _, err := io.Copy(fileField, bytes.NewReader(buf.Bytes())); err != nil {
		log.Error("Failed to write file content:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to write file content"})
	}
	writer.Close()

	// 创建请求
	url := fmt.Sprintf("%s/v1/datasets/%s/document/create-by-file",
		"http://dify.ooxo.cc",
		//os.Getenv("DIFY_BASE_URL"),=\[[[[
		//os.Getenv("DIFY_DATASET_ID"))
		"d0cb86b7-d79d-4f1c-b434-ce906577d99b")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		log.Error("Failed to create request:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create request"})
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+os.Getenv("DIFY_API_KEY"))

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send request:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to send request"})
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read response"})
	}

	log.Error("Dify API Response:", string(respBody))

	// 重新创建reader供后续使用
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":    "Dify API error",
			"response": string(respBody),
		})
	}

	// 更新同步状态
	if err := ah.DB.Save(&article).Error; err != nil {
		log.Error("Failed to update sync status:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update sync status",
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Article synced to Dify successfully",
		"response": json.RawMessage(respBody),
	})
}

func (ah *ArticleHandler) SyncAllToDify(c *fiber.Ctx) error {
	// 获取所有文章
	var articles []models.Article
	query := ah.DB.Where("is_deleted", false).Where("is_active", true)
	result := query.Find(&articles)
	if result.Error != nil {
		log.Error("Failed to fetch articles:", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch articles",
		})
	}

	// 创建一个通道来控制并发数量
	semaphore := make(chan struct{}, 5) // 最多5个并发
	var wg sync.WaitGroup

	// 创建结果通道
	type syncResult struct {
		ID      models.SnowflakeID `json:"id"`
		Title   string             `json:"title"`
		Success bool               `json:"success"`
		Error   string             `json:"error,omitempty"`
	}
	results := make(chan syncResult, len(articles))

	// 遍历所有文章进行同步
	for _, article := range articles {
		wg.Add(1)
		go func(article models.Article) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 准备文章内容
			var buf bytes.Buffer
			buf.WriteString("---\n")
			buf.WriteString(fmt.Sprintf("title: %s\n", article.Title))
			buf.WriteString(fmt.Sprintf("date: %s\n", article.CreatedAt.Format("2006-01-02 15:04:05")))
			if article.Tag != "" {
				buf.WriteString(fmt.Sprintf("tags: [%s]\n", article.Tag))
			}
			buf.WriteString("---\n\n")
			buf.WriteString(article.Content)

			// 准备 multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			// 添加 data 字段
			dataJson := map[string]interface{}{
				"indexing_technique": "high_quality",
				"doc_type":           "web_page",
				"doc_metadata": map[string]interface{}{
					"title": article.Title,
					"url":   article.ID,
				},
				"process_rule": map[string]interface{}{
					"rules": map[string]interface{}{
						"pre_processing_rules": []map[string]interface{}{
							{"id": "remove_extra_spaces", "enabled": true},
							{"id": "remove_urls_emails", "enabled": true},
						},
						"segmentation": map[string]interface{}{
							"separator":  "###",
							"max_tokens": 500,
						},
					},
					"mode": "custom",
				},
			}

			dataField, err := writer.CreateFormField("data")
			if err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to create form field"}
				return
			}
			if err := json.NewEncoder(dataField).Encode(dataJson); err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to encode data"}
				return
			}

			// 添加文件
			fileField, err := writer.CreateFormFile("file", article.Title+".md")
			if err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to create form file"}
				return
			}
			if _, err := io.Copy(fileField, bytes.NewReader(buf.Bytes())); err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to write file content"}
				return
			}
			writer.Close()

			// 创建请求
			url := fmt.Sprintf("http://dify.ooxo.cc/v1/datasets/%s/document/create-by-file",
				"d0cb86b7-d79d-4f1c-b434-ce906577d99b")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "POST", url, body)
			if err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to create request"}
				return
			}

			// 设置请求头
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("Authorization", "Bearer dataset-9gyYzoiNz1DPATdFy60JBVYF")

			// 发送请求
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to send request"}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: fmt.Sprintf("API error: %s", string(respBody))}
				return
			}

			if err := ah.DB.Save(&article).Error; err != nil {
				results <- syncResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to update sync status"}
				return
			}

			results <- syncResult{ID: article.ID, Title: article.Title, Success: true}
		}(article)
	}

	// 等待所有同步完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var syncResults []syncResult
	for result := range results {
		syncResults = append(syncResults, result)
	}

	// 统计结果
	successCount := 0
	failureCount := 0
	for _, result := range syncResults {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	return c.JSON(fiber.Map{
		"total":   len(articles),
		"success": successCount,
		"failure": failureCount,
		"results": syncResults,
	})
}

// GenerateEmbedding 调用本地 llama.cpp embedding API 生成向量
func (ah *ArticleHandler) GenerateEmbedding(text string) ([]float32, error) {
	// 准备请求体
	reqBody := map[string]interface{}{
		"model": "text-embedding-v3",
		"input": []string{text},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", "http://embed.ooxo.cc/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	// 解析响应
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

// VectorizeArticle 对单篇文章进行向量化
func (ah *ArticleHandler) VectorizeArticle(c *fiber.Ctx) error {
	id := c.Params("id")
	var article models.Article
	result := ah.DB.Take(&article, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Errorf("Database error: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve article"})
	}

	// 合并标题和内容/摘要生成向量
	text := article.Title
	if article.Content != "" {
		text = fmt.Sprintf("标题：%s\n内容：%s", article.Title, article.Content)
	} else if article.Summary != "" {
		text = fmt.Sprintf("标题：%s\n摘要：%s", article.Title, article.Summary)
	}
	embedding, err := ah.GenerateEmbedding(text)
	if err != nil {
		log.Errorf("Failed to generate embedding for article %s: %v", article.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to generate embedding: %v", err)})
	}

	// 保存向量到数据库 - 使用原生 SQL 更新
	vectorStr := pgvector.NewVector(embedding).String()
	log.Infof("Vector string: %s", vectorStr)
	if err := ah.DB.Exec("UPDATE article SET embedding = ? WHERE id = ?", vectorStr, article.ID).Error; err != nil {
		log.Errorf("Failed to save embedding: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save embedding"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
		"message": "Article vectorized successfully",
	})
}

// VectorizeAllArticles 批量向量化所有文章
func (ah *ArticleHandler) VectorizeAllArticles(c *fiber.Ctx) error {
	// 获取所有文章
	var articles []models.Article
	query := ah.DB.Where("is_deleted", false).Where("is_active", true)
	result := query.Find(&articles)
	if result.Error != nil {
		log.Errorf("Failed to fetch articles: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch articles",
		})
	}

	if len(articles) == 0 {
		return c.JSON(fiber.Map{
			"total":   0,
			"success": 0,
			"failure": 0,
			"message": "No articles to vectorize",
		})
	}

	// 创建结果通道
	type vectorizeResult struct {
		ID      models.SnowflakeID `json:"id"`
		Title   string             `json:"title"`
		Success bool               `json:"success"`
		Error   string             `json:"error,omitempty"`
	}
	results := make(chan vectorizeResult, len(articles))

	// 创建一个通道来控制并发数量
	semaphore := make(chan struct{}, 3) // 最多3个并发，避免API限流
	var wg sync.WaitGroup

	// 遍历所有文章进行向量化
	for _, article := range articles {
		wg.Add(1)
		go func(article models.Article) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 合并标题和内容/摘要生成向量
			text := article.Title
			if article.Content != "" {
				text = fmt.Sprintf("标题：%s\n内容：%s", article.Title, article.Content)
			} else if article.Summary != "" {
				text = fmt.Sprintf("标题：%s\n摘要：%s", article.Title, article.Summary)
			}
			embedding, err := ah.GenerateEmbedding(text)
			if err != nil {
				results <- vectorizeResult{ID: article.ID, Title: article.Title, Success: false, Error: err.Error()}
				return
			}

			// 保存向量到数据库 - 使用原生 SQL 更新
			vectorStr := pgvector.NewVector(embedding).String()
			if err := ah.DB.Exec("UPDATE article SET embedding = ? WHERE id = ?", vectorStr, article.ID).Error; err != nil {
				results <- vectorizeResult{ID: article.ID, Title: article.Title, Success: false, Error: "Failed to save embedding"}
				return
			}

			results <- vectorizeResult{ID: article.ID, Title: article.Title, Success: true}
		}(article)
	}

	// 等待所有向量化完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var vectorizeResults []vectorizeResult
	for result := range results {
		vectorizeResults = append(vectorizeResults, result)
	}

	// 统计结果
	successCount := 0
	failureCount := 0
	for _, result := range vectorizeResults {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	return c.JSON(fiber.Map{
		"total":   len(articles),
		"success": successCount,
		"failure": failureCount,
		"results": vectorizeResults,
	})
}

