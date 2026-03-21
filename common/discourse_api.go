package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// DiscourseAPIClient Discourse API 客户端
type DiscourseAPIClient struct {
	BaseURL string
	APIKey  string
	APIUser string // Discourse 用户名（用于 API 调用）
}

// DiscoursePost Discourse 帖子结构
type DiscoursePost struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Username        string    `json:"username"`
	Raw             string    `json:"raw"`
	CreatedAt       time.Time `json:"created_at"`
	Cooked          string    `json:"cooked"`
	PostNumber      int       `json:"post_number"`
	TopicID         int64     `json:"topic_id"`
	TopicSlug       string    `json:"topic_slug"`
	DisplayUsername string    `json:"display_username"`
	AvatarTemplate  string    `json:"avatar_template"`
	UserTitle       string    `json:"user_title"`
	Replies         []DiscoursePost
}

// DiscourseTopicPostsResponse Discourse 主题帖子列表响应
type DiscourseTopicPostsResponse struct {
	PostStream struct {
		Posts []DiscoursePost `json:"posts"`
	} `json:"post_stream"`
}

// CreatePostRequest 创建帖子请求
type CreatePostRequest struct {
	Title    string `json:"title,omitempty"`
	Raw      string `json:"raw"`
	TopicID  int64  `json:"topic_id,omitempty"`
	ReplyToPostID *int64 `json:"reply_to_post_number,omitempty"`
}

// CreatePostResponse 创建帖子响应
type CreatePostResponse struct {
	PostType   string `json:"post_type"`
	PostNumber int    `json:"post_number"`
	ID         int64  `json:"id"`
	TopicID    int64  `json:"topic_id"`
}

// NewDiscourseAPIClient 创建 Discourse API 客户端
func NewDiscourseAPIClient() *DiscourseAPIClient {
	baseURL := strings.TrimSpace(os.Getenv("DISCOURSE_BASE_URL"))
	apiKey := strings.TrimSpace(os.Getenv("DISCOURSE_API_KEY"))
	apiUser := strings.TrimSpace(os.Getenv("DISCOURSE_API_USER"))

	if apiKey == "" {
		apiUser = "system" // 默认使用系统用户
	}

	return &DiscourseAPIClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		APIUser: apiUser,
	}
}

// GetTopicPosts 获取主题的所有帖子（不包括第一楼）
func (c *DiscourseAPIClient) GetTopicPosts(topicID int64) ([]DiscoursePost, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("DISCOURSE_BASE_URL is not configured")
	}

	url := fmt.Sprintf("%s/t/%d/posts.json?include_raw=true", c.BaseURL, topicID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Api-Key", c.APIKey)
		req.Header.Set("Api-Username", c.APIUser)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discourse API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response DiscourseTopicPostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 过滤掉第一楼（通常是文章内容），只保留评论
	posts := make([]DiscoursePost, 0)
	for _, post := range response.PostStream.Posts {
		if post.PostNumber > 1 {
			posts = append(posts, post)
		}
	}

	// 构建回复关系树
	c.buildRepliesTree(posts)

	return posts, nil
}

// buildRepliesTree 构建回复关系树
func (c *DiscourseAPIClient) buildRepliesTree(posts []DiscoursePost) {
	// 创建帖子 ID 到帖子指针的映射
	postMap := make(map[int64]*DiscoursePost)
	for i := range posts {
		postMap[posts[i].ID] = &posts[i]
		posts[i].Replies = make([]DiscoursePost, 0)
	}

	// TODO: 这里需要根据 Discourse API 返回的 reply_to_post_number 来构建树
	// 目前先保持简单，只获取一级回复
}

// CreatePost 创建新帖子或回复
func (c *DiscourseAPIClient) CreatePost(title, raw string, topicID int64, replyToPostNumber *int) (*CreatePostResponse, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("DISCOURSE_BASE_URL is not configured")
	}

	if c.APIKey == "" {
		return nil, fmt.Errorf("DISCOURSE_API_KEY is not configured")
	}

	endpoint := "/posts.json"
	url := c.BaseURL + endpoint

	requestBody := CreatePostRequest{
		Raw: raw,
	}

	if title != "" {
		requestBody.Title = title
	}

	if topicID > 0 {
		requestBody.TopicID = topicID
	}

	if replyToPostNumber != nil {
		replyToPostID := int64(*replyToPostNumber)
		requestBody.ReplyToPostID = &replyToPostID
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Api-Username", c.APIUser)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discourse API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CreatePostResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ConvertToCommentFormat 转换为本地评论格式
func (c *DiscourseAPIClient) ConvertToCommentFormat(posts []DiscoursePost) []map[string]interface{} {
	comments := make([]map[string]interface{}, 0, len(posts))

	for _, post := range posts {
		avatarURL := c.expandAvatarTemplate(post.AvatarTemplate)

		comment := map[string]interface{}{
			"id":        strconv.FormatInt(post.ID, 10),
			"content":   post.Cooked,
			"raw":       post.Raw,
			"username":  post.Username,
			"createdAt": post.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"discoursePostId": post.ID,
			"discoursePostNumber": post.PostNumber,
		}

		if avatarURL != "" {
			comment["avatarUrl"] = avatarURL
		}

		if post.Name != "" {
			comment["name"] = post.Name
		}

		if post.UserTitle != "" {
			comment["userTitle"] = post.UserTitle
		}

		// 递归处理回复
		if len(post.Replies) > 0 {
			comment["replies"] = c.ConvertToCommentFormat(post.Replies)
		}

		comments = append(comments, comment)
	}

	return comments
}

// expandAvatarTemplate 扩展头像模板为完整 URL
func (c *DiscourseAPIClient) expandAvatarTemplate(template string) string {
	if template == "" {
		return ""
	}

	// Discourse 头像模板格式: /user_avatar/{domain}/{username}/{size}/{version}.png
	// 替换 {size} 为实际尺寸
	if strings.Contains(template, "{size}") {
		template = strings.ReplaceAll(template, "{size}", "120")
	}

	if strings.HasPrefix(template, "http") {
		return template
	}

	return c.BaseURL + template
}
