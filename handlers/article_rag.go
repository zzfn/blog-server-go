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

// RAGQuestionRequest RAG 问答请求
type RAGQuestionRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"topK,omitempty"` // 返回前 K 篇相关文章
}

// ArticleWithSimilarity 带相似度的文章
type ArticleWithSimilarity struct {
	models.Article
	Similarity float32 `json:"similarity"`
}

// searchArticlesByVector 向量搜索文章（独立函数，供tool调用）
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

// getSearchArticlesTool 获取搜索文章的tool定义
func getSearchArticlesTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "search_articles",
		Desc: "根据关键词搜索博客文章，返回最相关的文章列表。当用户询问任何问题时，你必须先调用此工具搜索相关内容。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     "string",
				Desc:     "搜索关键词或问题",
				Required: true,
			},
		}),
	}
}

// RAGQuestion RAG 问答接口（使用Tool Calling）
func (ah *ArticleHandler) RAGQuestion(c *fiber.Ctx) error {
	var req RAGQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Question == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Question is required"})
	}

	// 设置 SSE 响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// 使用 SetBodyStreamWriter 实现流式响应
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// 初始化消息历史
		messages := []*schema.Message{
			schema.SystemMessage(`你是博客助手。你必须严格遵守以下规则：

规则1：收到用户问题后，你的第一个动作必须是调用 search_articles 工具搜索相关文章
规则2：绝对禁止在没有调用 search_articles 的情况下直接回答用户问题
规则3：如果搜索结果不够，可以调整关键词再次调用 search_articles
规则4：只有在获得搜索结果后，才能基于搜索结果回答问题

记住：你的第一步永远是调用 search_articles 工具！`),
			schema.UserMessage(req.Question),
		}

		// 定义可用的tools
		// 需要至少2个tool，否则eino会把tool_choice从"required"转成
		// {"type":"function","function":{"name":"..."}} 格式，qwen小模型不支持
		tools := []*schema.ToolInfo{
			getSearchArticlesTool(),
			{
				Name: "finish_search",
				Desc: "当你已经获得足够的搜索结果，不需要再搜索时，调用此工具结束搜索阶段",
				ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
					"reason": {Type: "string", Desc: "结束搜索的原因", Required: true},
				}),
			},
		}

		// 用于收集所有搜索到的文章（去重）
		allArticles := make(map[string]map[string]interface{})

		// 阶段一：Tool Calling 循环（强制工具调用），收集搜索结果
		maxIterations := 10
		searchDone := false
		for i := 0; i < maxIterations && !searchDone; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			response, err := ah.LLMService.GenerateWithToolChoice(ctx, messages, tools, schema.ToolChoiceForced)
			cancel()

			if err != nil {
				log.Errorf("Failed to call LLM: %v", err)
				errorJSON, _ := json.Marshal(map[string]string{"error": "Failed to call LLM"})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorJSON)
				w.Flush()
				return
			}

			log.Infof("LLM Response (iteration %d): ToolCalls=%d, Content=%q", i, len(response.ToolCalls), response.Content)

			// 没有 tool 调用（理论上不会发生，因为强制了tool_choice）
			if len(response.ToolCalls) == 0 {
				break
			}

			// 有 tool 调用，添加 assistant 消息并执行
			messages = append(messages, response)

			for _, toolCall := range response.ToolCalls {
				if toolCall.Function.Name == "finish_search" {
					// 搜索阶段结束，添加 tool result 保持消息完整性
					toolResultMsg := schema.ToolMessage(toolCall.ID, "搜索阶段已结束")
					messages = append(messages, toolResultMsg)
					searchDone = true
					break
				}
				if toolCall.Function.Name != "search_articles" {
					toolResultMsg := schema.ToolMessage(toolCall.ID, "未知工具")
					messages = append(messages, toolResultMsg)
					continue
				}

				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					log.Errorf("Failed to parse tool arguments: %v", err)
					toolResultMsg := schema.ToolMessage(toolCall.ID, fmt.Sprintf("参数解析失败: %v", err))
					messages = append(messages, toolResultMsg)
					continue
				}

				query, _ := args["query"].(string)
				topK := 3
				if k, ok := args["top_k"].(float64); ok {
					topK = int(k)
				}

				results, err := ah.searchArticlesByVector(query, topK)
				if err != nil {
					log.Errorf("Failed to search articles: %v", err)
					toolResultMsg := schema.ToolMessage(toolCall.ID, fmt.Sprintf("搜索失败: %v", err))
					messages = append(messages, toolResultMsg)
					continue
				}

				// 构建搜索结果
				var resultBuilder strings.Builder
				resultBuilder.WriteString(fmt.Sprintf("搜索查询: %s\n找到 %d 篇相关文章:\n\n", query, len(results)))

				for idx, article := range results {
					articleKey := string(article.ID)
					if _, exists := allArticles[articleKey]; !exists {
						allArticles[articleKey] = map[string]interface{}{
							"id":         article.ID,
							"title":      article.Title,
							"summary":    article.Summary,
							"similarity": article.Similarity,
						}
					}

					resultBuilder.WriteString(fmt.Sprintf("【文章 %d】\n", idx+1))
					resultBuilder.WriteString(fmt.Sprintf("标题: %s\n", article.Title))
					resultBuilder.WriteString(fmt.Sprintf("相似度: %.2f\n", article.Similarity))
					if article.Summary != "" {
						resultBuilder.WriteString(fmt.Sprintf("摘要: %s\n", article.Summary))
					}
					if article.Content != "" && len(article.Content) < 2000 {
						resultBuilder.WriteString(fmt.Sprintf("内容: %s\n", article.Content))
					} else if article.Content != "" {
						resultBuilder.WriteString(fmt.Sprintf("内容: %s...\n", article.Content[:2000]))
					}
					resultBuilder.WriteString("\n")
				}

				toolResultMsg := schema.ToolMessage(toolCall.ID, resultBuilder.String())
				messages = append(messages, toolResultMsg)
			}
		}

		// 发送收集到的文章
		if len(allArticles) > 0 {
			articlesArray := make([]map[string]interface{}, 0, len(allArticles))
			for _, article := range allArticles {
				articlesArray = append(articlesArray, article)
			}
			articlesJSON, _ := json.Marshal(articlesArray)
			fmt.Fprintf(w, "event: articles\ndata: %s\n\n", articlesJSON)
			w.Flush()
		}

		// 阶段二：真正流式生成最终回答（不带 tools）
		// 追加指令，让模型基于搜索结果回答
		messages = append(messages, schema.UserMessage("请根据以上搜索结果，回答用户的问题。如果搜索结果中没有相关信息，请如实告知。"))

		streamCtx, streamCancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer streamCancel()

		err := ah.LLMService.GenerateStream(streamCtx, messages, func(chunk string) error {
			contentJSON, _ := json.Marshal(map[string]string{"content": chunk})
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", contentJSON)
			return w.Flush()
		})

		if err != nil {
			log.Errorf("Failed to stream response: %v", err)
			errorJSON, _ := json.Marshal(map[string]string{"error": "Failed to stream response"})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorJSON)
			w.Flush()
			return
		}

		// 发送完成事件
		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		w.Flush()
	})

	return nil
}
