package handlers

import (
	"blog-server-go/common"
	"blog-server-go/kafka"
	"blog-server-go/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
	"io"
	"strconv"
)

// ArticleHandler 处理与文章相关的请求
type ArticleHandler struct {
	BaseHandler
}

// GetArticles 获取所有文章
func (ah *ArticleHandler) GetArticles(c *fiber.Ctx) error {
	var articles []models.Article

	orderStr := c.Query("order", "created_at desc")
	limitStr := c.Query("limit")
	isRss := c.Query("rss")
	log.Info("isRss", isRss)
	fields := "ID,TITLE,TAG,CREATED_AT,updated_at"
	if isRss == "true" {
		fields += ",CONTENT"
	}

	query := ah.DB.Select(fields).Where("is_deleted", false).Where("is_active", true).Order(orderStr)

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
		log.Errorf("Failed to retrieve article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.JSON(article)
}
func (ah *ArticleHandler) UpdateArticleViews(c *fiber.Ctx) error {
	id := c.Params("id")
	article := models.Article{
		BaseModel: models.BaseModel{ID: models.SnowflakeID(id)},
	}
	result := ah.DB.Model(&article).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	if result.Error != nil {
		log.Errorf("Failed to update article views: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	return c.JSON(result.RowsAffected)
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
	result := ah.DB.Take(&existingArticle, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Errorf("Failed to retrieve article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	// 更新文章内容
	result = ah.DB.Model(&existingArticle).Updates(inputArticle)
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
