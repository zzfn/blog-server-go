package handlers

import (
	"blog-server-go/models"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/pgvector/pgvector-go"
)

// expandQueryPrompt 查询扩展的 system prompt
const expandQueryPrompt = `你是一个搜索查询扩展助手。用户会给你一个问题，你需要生成多个搜索关键词来帮助找到相关文章。

要求：
1. 生成 2-4 个搜索关键词/短语
2. 包含原始问题的核心词、同义词、相关概念
3. 只返回 JSON 数组，不要其他内容

示例：
用户问题：Go 怎么处理错误
返回：["Go 错误处理", "golang error handling", "Go panic recover"]

用户问题：React 性能优化
返回：["React 性能优化", "React memo useMemo", "前端渲染优化"]`

// RAGQuestionRequest RAG 问答请求
type RAGQuestionRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"topK,omitempty"` // 每个关键词返回前 K 篇相关文章
}

// ArticleWithSimilarity 带相似度的文章
type ArticleWithSimilarity struct {
	models.Article
	Similarity float32 `json:"similarity"`
}

// searchArticlesByVector 向量搜索文章
func (ah *ArticleHandler) searchArticlesByVector(query string, topK int) ([]ArticleWithSimilarity, error) {
	if topK <= 0 {
		topK = 3
	}

	// 生成查询向量
	queryEmbedding, err := ah.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 向量相似度搜索
	var results []ArticleWithSimilarity
	sqlQuery := `
		SELECT id, created_at, updated_at, is_deleted, created_by, updated_by,
		       title, content, view_count, tag, sort_order, is_active,
		       1 - (embedding <=> ?) as similarity
		FROM article
		WHERE is_deleted = false AND is_active = true AND embedding IS NOT NULL
		ORDER BY embedding <=> ? ASC
		LIMIT ?
	`
	if err := ah.DB.Raw(sqlQuery, pgvector.NewVector(queryEmbedding), pgvector.NewVector(queryEmbedding), topK).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
}

// expandQuery 使用 LLM 扩展搜索关键词
func (ah *ArticleHandler) expandQuery(ctx context.Context, question string) ([]string, error) {
	messages := []*schema.Message{
		schema.SystemMessage(expandQueryPrompt),
		schema.UserMessage(question),
	}

	content, err := ah.LLMService.GenerateText(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to expand query: %w", err)
	}

	// 解析 JSON 数组
	content = strings.TrimSpace(content)
	var queries []string
	if err := json.Unmarshal([]byte(content), &queries); err != nil {
		log.Warnf("LLM 返回的关键词格式异常，回退使用原始问题: %s", content)
		return []string{question}, nil
	}

	return queries, nil
}

// RAGQuestion RAG 问答接口（查询扩展 + 向量搜索 + 流式回答）
func (ah *ArticleHandler) RAGQuestion(c *fiber.Ctx) error {
	var req RAGQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Question == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Question is required"})
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 3
	}

	// 第一步：LLM 扩展搜索关键词
	expandCtx, expandCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer expandCancel()

	queries, err := ah.expandQuery(expandCtx, req.Question)
	if err != nil {
		log.Errorf("查询扩展失败，回退使用原始问题: %v", err)
		queries = []string{req.Question}
	}
	log.Infof("查询扩展结果: %v", queries)

	// 第二步：用所有关键词批量向量搜索，合并去重
	allArticles := make(map[string]ArticleWithSimilarity)
	for _, query := range queries {
		results, err := ah.searchArticlesByVector(query, topK)
		if err != nil {
			log.Errorf("搜索失败 (query=%s): %v", query, err)
			continue
		}
		for _, article := range results {
			key := string(article.ID)
			// 保留相似度最高的结果
			if existing, exists := allArticles[key]; !exists || article.Similarity > existing.Similarity {
				allArticles[key] = article
			}
		}
	}

	// 构建搜索结果上下文
	var contextBuilder strings.Builder
	for _, article := range allArticles {
		contextBuilder.WriteString(fmt.Sprintf("【%s】\n", article.Title))
		if len(article.Content) < 2000 {
			contextBuilder.WriteString(article.Content)
		} else {
			contextBuilder.WriteString(article.Content[:2000] + "...")
		}
		contextBuilder.WriteString("\n\n")
	}

	// 设置 SSE 响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// 发送搜索到的文章列表
		if len(allArticles) > 0 {
			articlesArray := make([]map[string]interface{}, 0, len(allArticles))
			for _, article := range allArticles {
				articlesArray = append(articlesArray, map[string]interface{}{
					"id":         article.ID,
					"title":      article.Title,
					"summary":    article.Summary,
					"similarity": article.Similarity,
				})
			}
			articlesJSON, _ := json.Marshal(articlesArray)
			fmt.Fprintf(w, "event: articles\ndata: %s\n\n", articlesJSON)
			w.Flush()
		}

		// 第三步：基于搜索结果流式生成回答
		messages := []*schema.Message{
			schema.SystemMessage("你是博客助手。请根据以下参考文章内容回答用户的问题。如果参考内容中没有相关信息，请如实告知。\n\n" + contextBuilder.String()),
			schema.UserMessage(req.Question),
		}

		streamCtx, streamCancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer streamCancel()

		err := ah.LLMService.GenerateStream(streamCtx, messages, func(chunk string) error {
			contentJSON, _ := json.Marshal(map[string]string{"content": chunk})
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", contentJSON)
			return w.Flush()
		})

		if err != nil {
			log.Errorf("流式生成失败: %v", err)
			errorJSON, _ := json.Marshal(map[string]string{"error": "Failed to stream response"})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorJSON)
			w.Flush()
			return
		}

		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		w.Flush()
	})

	return nil
}
