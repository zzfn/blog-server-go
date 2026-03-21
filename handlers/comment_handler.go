package handlers

import (
	"blog-server-go/common"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type CommentsHandler struct {
	BaseHandler
	DiscourseClient *common.DiscourseAPIClient
}

// discourseCommentRequest Discourse 评论请求格式
type discourseCommentRequest struct {
	Content   string `json:"content"`
	ObjectID  string `json:"objectId"`
	ObjectType string `json:"objectType"`
}

// discourseReplyRequest Discourse 回复请求格式
type discourseReplyRequest struct {
	Content  string `json:"content"`
	CommentID string `json:"commentId"` // Discourse 帖子 ID
}

// GetComments 获取评论（从 Discourse 获取）
func (ch *CommentsHandler) GetComments(c *fiber.Ctx) error {
	objectID := c.Query("objectId")
	objectType := c.Query("objectType")

	// 只处理文章类型的评论
	if objectType != "article" {
		return c.JSON([]interface{}{})
	}

	// 将 objectId 转换为 Discourse topic ID
	topicID, err := strconv.ParseInt(objectID, 10, 64)
	if err != nil {
		log.Errorf("invalid objectId: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid objectId"})
	}

	// 从 Discourse 获取帖子
	posts, err := ch.DiscourseClient.GetTopicPosts(topicID)
	if err != nil {
		log.Errorf("failed to get discourse posts: %v", err)
		// 如果 Discourse API 失败，返回空数组而不是错误
		return c.JSON([]interface{}{})
	}

	// 转换为本地评论格式
	comments := ch.DiscourseClient.ConvertToCommentFormat(posts)

	return c.JSON(comments)
}

// CreateComment 创建新评论（发布到 Discourse）
func (ch *CommentsHandler) CreateComment(c *fiber.Ctx) error {
	var input discourseCommentRequest

	// 解析请求体
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	// 验证内容
	if strings.TrimSpace(input.Content) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Content is required"})
	}

	// 验证 object type
	if input.ObjectType != "article" {
		return c.Status(400).JSON(fiber.Map{"error": "Only article comments are supported"})
	}

	// 转换 objectId 为 Discourse topic ID
	topicID, err := strconv.ParseInt(input.ObjectID, 10, 64)
	if err != nil {
		log.Errorf("invalid objectId: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid objectId"})
	}

	// 获取当前用户（从认证中间件设置）
	username := c.Locals("username")
	if username == nil || username == "" {
		return c.Status(401).JSON(fiber.Map{"error": "User not authenticated"})
	}

	// 使用 Discourse Connect 登录的用户名
	discourseUsername := username.(string)

	// 创建 Discourse API 客户端实例（使用用户身份）
	apiClient := common.NewDiscourseAPIClient()
	apiClient.APIUser = discourseUsername

	// 发布到 Discourse
	result, err := apiClient.CreatePost("", input.Content, topicID, nil)
	if err != nil {
		log.Errorf("failed to create discourse post: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create comment"})
	}

	log.Infof("Created discourse post: topicID=%d, postID=%d, postNumber=%d",
		result.TopicID, result.ID, result.PostNumber)

	return c.Status(201).JSON(fiber.Map{
		"id":                 strconv.FormatInt(result.ID, 10),
		"discoursePostId":    result.ID,
		"discoursePostNumber": result.PostNumber,
		"topicId":            result.TopicID,
	})
}

// CreateReply 创建回复（发布到 Discourse）
func (ch *CommentsHandler) CreateReply(c *fiber.Ctx) error {
	var input discourseReplyRequest

	// 解析请求体
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	// 验证内容
	if strings.TrimSpace(input.Content) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Content is required"})
	}

	// 解析评论 ID（Discourse 帖子 ID）
	commentID, err := strconv.ParseInt(input.CommentID, 10, 64)
	if err != nil {
		log.Errorf("invalid commentId: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid commentId"})
	}

	// 获取当前用户
	username := c.Locals("username")
	if username == nil || username == "" {
		return c.Status(401).JSON(fiber.Map{"error": "User not authenticated"})
	}

	// 使用 Discourse Connect 登录的用户名
	discourseUsername := username.(string)

	// 创建 Discourse API 客户端实例（使用用户身份）
	apiClient := common.NewDiscourseAPIClient()
	apiClient.APIUser = discourseUsername

	// 发布回复到 Discourse（replyToPostNumber 参数）
	postNumber := int(commentID) // Discourse 使用 post number 作为回复目标
	result, err := apiClient.CreatePost("", input.Content, 0, &postNumber)
	if err != nil {
		log.Errorf("failed to create discourse reply: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create reply"})
	}

	log.Infof("Created discourse reply: topicID=%d, postID=%d, postNumber=%d, replyTo=%d",
		result.TopicID, result.ID, result.PostNumber, postNumber)

	return c.Status(201).JSON(fiber.Map{
		"id":                 strconv.FormatInt(result.ID, 10),
		"discoursePostId":    result.ID,
		"discoursePostNumber": result.PostNumber,
		"topicId":            result.TopicID,
	})
}
