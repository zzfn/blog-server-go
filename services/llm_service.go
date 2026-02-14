package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	openaicomp "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/gofiber/fiber/v2/log"
)

// loggingTransport 用于调试 HTTP 请求
type loggingTransport struct {
	transport http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		log.Infof("[LLM Request] %s %s\nBody: %s", req.Method, req.URL, string(body))
	}
	resp, err := t.transport.RoundTrip(req)
	if err == nil && resp.Body != nil {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
		log.Infof("[LLM Response] Status: %d\nBody: %s", resp.StatusCode, string(respBody))
	}
	return resp, err
}

// LLMService LLM 服务封装
type LLMService struct {
	chatModel model.ChatModel
}

// NewLLMService 创建 LLM 服务实例
func NewLLMService() *LLMService {
	config := &openaicomp.ChatModelConfig{
		BaseURL: "https://llm.ooxo.cc/v1",
		APIKey:  os.Getenv("OPENROUTER_API_KEY"),
		Model:   "qwen2.5-1.5b-instruct",
		HTTPClient: &http.Client{
			Transport: &loggingTransport{transport: http.DefaultTransport},
		},
	}

	chatModel, err := openaicomp.NewChatModel(context.Background(), config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create chat model: %v", err))
	}

	return &LLMService{
		chatModel: chatModel,
	}
}

// GenerateText 生成文本（非流式）
func (s *LLMService) GenerateText(ctx context.Context, messages []*schema.Message) (string, error) {
	resp, err := s.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	return resp.Content, nil
}

// GenerateStream 生成流式响应
func (s *LLMService) GenerateStream(ctx context.Context, messages []*schema.Message, callback func(chunk string) error, opts ...model.Option) error {
	stream, err := s.chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("stream error: %w", err)
		}

		if chunk.Content != "" {
			if err := callback(chunk.Content); err != nil {
				return fmt.Errorf("callback error: %w", err)
			}
		}
	}

	return nil
}

// GenerateWithTools 使用工具调用生成（非流式）
func (s *LLMService) GenerateWithTools(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.Message, error) {
	opts := []model.Option{
		model.WithTools(tools),
	}

	resp, err := s.chatModel.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate with tools: %w", err)
	}

	return resp, nil
}

// GenerateWithToolChoice 使用工具调用生成，并强制模型必须调用工具
func (s *LLMService) GenerateWithToolChoice(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, choice schema.ToolChoice) (*schema.Message, error) {
	opts := []model.Option{
		model.WithTools(tools),
		model.WithToolChoice(choice),
	}

	resp, err := s.chatModel.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate with tools: %w", err)
	}

	return resp, nil
}
