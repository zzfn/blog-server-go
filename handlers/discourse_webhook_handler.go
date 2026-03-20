package handlers

import (
	"blog-server-go/kafka"
	"blog-server-go/models"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
)

type DiscourseWebhookHandler struct {
	BaseHandler
}

type discourseWebhookPayload struct {
	Post *discoursePostPayload `json:"post"`
	Tags []string              `json:"tags"`
}

type discoursePostPayload struct {
	ID             int64   `json:"id"`
	PostNumber     int     `json:"post_number"`
	TopicID        int64   `json:"topic_id"`
	TopicSlug      string  `json:"topic_slug"`
	TopicTitle     string  `json:"topic_title"`
	TopicArchetype string  `json:"topic_archetype"`
	Raw            string  `json:"raw"`
	DeletedAt      *string `json:"deleted_at"`
	Hidden         bool    `json:"hidden"`
	CategoryID     *int64  `json:"category_id"`
}

func (dh *DiscourseWebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "empty request body"})
	}

	if err := verifyDiscourseSignature(c.Get("X-Discourse-Event-Signature"), body); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	event := strings.TrimSpace(c.Get("X-Discourse-Event"))
	if event == "" {
		event = strings.TrimSpace(c.Get("X-Discourse-Event-Type"))
	}

	var payload discourseWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid discourse webhook payload"})
	}

	if payload.Post == nil {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "post payload is required"})
	}

	post := payload.Post
	if post.PostNumber != 1 {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "only first post is synced"})
	}
	if post.TopicID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing topic_id"})
	}
	if strings.TrimSpace(post.TopicTitle) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing topic_title"})
	}
	if strings.TrimSpace(post.Raw) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing raw content"})
	}
	if post.TopicArchetype == "private_message" {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "private message is not synced"})
	}
	if post.Hidden || post.DeletedAt != nil {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "hidden or deleted post is not synced"})
	}

	instance := c.Get("X-Discourse-Instance")
	if strings.TrimSpace(instance) == "" {
		instance = os.Getenv("DISCOURSE_BASE_URL")
	}

	article, created, err := dh.upsertDiscourseArticle(post, payload.Tags, instance)
	if err != nil {
		log.Errorf("failed to sync discourse webhook: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sync discourse article"})
	}

	if dh.KafkaProducer != nil {
		dh.KafkaProducer.ProduceMessage(kafka.ArticleUpdateTopic, "id", string(article.ID))
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"action":  map[bool]string{true: "created", false: "updated"}[created],
		"event":   event,
		"id":      article.ID,
		"title":   article.Title,
		"topicId": article.DiscourseTopicID,
	})
}

func (dh *DiscourseWebhookHandler) upsertDiscourseArticle(post *discoursePostPayload, tags []string, instance string) (*models.Article, bool, error) {
	var article models.Article
	result := dh.DB.Where("source = ? AND discourse_topic_id = ?", "discourse", post.TopicID).First(&article)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, false, result.Error
	}

	topicURL := buildDiscourseTopicURL(instance, post.TopicSlug, post.TopicID)
	tagValue := strings.Join(tags, ",")
	created := result.Error == gorm.ErrRecordNotFound

	if created {
		article = models.Article{
			Title:             strings.TrimSpace(post.TopicTitle),
			Content:           post.Raw,
			Tag:               tagValue,
			IsActive:          true,
			Source:            "discourse",
			DiscourseTopicID:  post.TopicID,
			DiscoursePostID:   post.ID,
			DiscourseTopicURL: topicURL,
		}
		if err := dh.DB.Omit("embedding").Create(&article).Error; err != nil {
			return nil, false, err
		}
		return &article, true, nil
	}

	updates := map[string]interface{}{
		"title":               strings.TrimSpace(post.TopicTitle),
		"content":             post.Raw,
		"tag":                 tagValue,
		"is_active":           true,
		"source":              "discourse",
		"discourse_post_id":   post.ID,
		"discourse_topic_url": topicURL,
	}
	if err := dh.DB.Model(&article).Updates(updates).Error; err != nil {
		return nil, false, err
	}
	if err := dh.DB.Where("id = ?", article.ID).First(&article).Error; err != nil {
		return nil, false, err
	}
	return &article, false, nil
}

func verifyDiscourseSignature(signature string, body []byte) error {
	secret := strings.TrimSpace(os.Getenv("DISCOURSE_WEBHOOK_SECRET"))
	if secret == "" {
		return nil
	}
	if signature == "" {
		return fmt.Errorf("missing discourse webhook signature")
	}

	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(body)
	expected := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid discourse webhook signature")
	}
	return nil
}

func buildDiscourseTopicURL(instance, slug string, topicID int64) string {
	instance = strings.TrimRight(strings.TrimSpace(instance), "/")
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	if instance == "" || topicID == 0 {
		return ""
	}
	if slug == "" {
		return fmt.Sprintf("%s/t/%d", instance, topicID)
	}
	return fmt.Sprintf("%s/t/%s/%d", instance, slug, topicID)
}
