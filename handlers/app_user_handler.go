package handlers

import (
	"blog-server-go/common"
	"blog-server-go/models"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AppUserHandler struct {
	BaseHandler
}

const discourseNoncePrefix = "discourse_sso_nonce:"

func discourseConnectBaseURL() string {
	return strings.TrimRight(os.Getenv("DISCOURSE_BASE_URL"), "/")
}

func discourseConnectSecret() string {
	secret := strings.TrimSpace(os.Getenv("DISCOURSE_CONNECT_PROVIDER_SECRET"))
	if secret != "" {
		return secret
	}
	return strings.TrimSpace(os.Getenv("DISCOURSE_CONNECT_SECRET"))
}

func discourseConnectCallbackURL() string {
	callbackURL := strings.TrimSpace(os.Getenv("DISCOURSE_CONNECT_CALLBACK_URL"))
	if callbackURL != "" {
		return callbackURL
	}
	return discourseConnectBaseURL()
}

func frontendRedirectBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("APP_FRONTEND_URL")), "/")
}

func buildFrontendRedirect(targetPath string, params map[string]string) string {
	baseURL := frontendRedirectBaseURL()
	if baseURL == "" {
		baseURL = "/"
	}

	redirectURL, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	if targetPath != "" {
		redirectURL.Path = strings.TrimRight(redirectURL.Path, "/") + targetPath
	}

	query := redirectURL.Query()
	for key, value := range params {
		if value != "" {
			query.Set(key, value)
		}
	}
	redirectURL.RawQuery = query.Encode()
	return redirectURL.String()
}

func (auh *AppUserHandler) discourseConnectConfig() (string, string, string, error) {
	baseURL := discourseConnectBaseURL()
	secret := discourseConnectSecret()
	callbackURL := strings.TrimSpace(os.Getenv("DISCOURSE_CONNECT_CALLBACK_URL"))

	switch {
	case baseURL == "":
		return "", "", "", fmt.Errorf("DISCOURSE_BASE_URL is required")
	case secret == "":
		return "", "", "", fmt.Errorf("DISCOURSE_CONNECT_PROVIDER_SECRET is required")
	case callbackURL == "":
		return "", "", "", fmt.Errorf("DISCOURSE_CONNECT_CALLBACK_URL is required")
	default:
		return baseURL, secret, callbackURL, nil
	}
}

func (auh *AppUserHandler) setAuthCookie(c *fiber.Ctx, token string) {
	secure := strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	sameSite := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_COOKIE_SAME_SITE")))
	if sameSite == "" {
		sameSite = "lax"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "blog_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func (auh *AppUserHandler) clearAuthCookie(c *fiber.Ctx) {
	secure := strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	sameSite := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_COOKIE_SAME_SITE")))
	if sameSite == "" {
		sameSite = "lax"
	}

	c.Cookie(&fiber.Cookie{
		Name:     "blog_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN")),
		Expires:  time.Unix(0, 0),
	})
}

func (auh *AppUserHandler) issueUserToken(ctx context.Context, user *models.AppUser) (string, error) {
	token, err := common.GenerateToken(string(user.ID), user.IsAdmin, user.Username)
	if err != nil {
		return "", err
	}

	oldToken, err := auh.Redis.HGet(ctx, "username_to_token", string(user.ID)).Result()
	if err == nil && oldToken != "" {
		auh.Redis.HDel(ctx, "token_to_username", oldToken)
	}

	auh.Redis.HSet(ctx, "username_to_token", string(user.ID), token)
	auh.Redis.HSet(ctx, "token_to_username", token, string(user.ID))
	return token, nil
}

func (auh *AppUserHandler) upsertDiscourseUser(values url.Values) (*models.AppUser, error) {
	externalID := strings.TrimSpace(values.Get("external_id"))
	username := strings.TrimSpace(values.Get("username"))
	email := strings.TrimSpace(values.Get("email"))
	name := strings.TrimSpace(values.Get("name"))
	avatarURL := strings.TrimSpace(values.Get("avatar_url"))
	groups := strings.TrimSpace(values.Get("groups"))

	if externalID == "" {
		return nil, fmt.Errorf("missing external_id")
	}
	if username == "" {
		return nil, fmt.Errorf("missing username")
	}

	isAdmin := strings.EqualFold(values.Get("admin"), "true")
	nickname := name
	if nickname == "" {
		nickname = username
	}

	var user models.AppUser
	result := auh.DB.Where("discourse_external_id = ?", externalID).First(&user)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) && email != "" {
		emailLookup := auh.DB.Where("email = ?", email).First(&user)
		if emailLookup.Error != nil && !errors.Is(emailLookup.Error, gorm.ErrRecordNotFound) {
			return nil, emailLookup.Error
		}
		result = emailLookup
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		user = models.AppUser{
			Username:            username,
			Email:               email,
			IsAdmin:             isAdmin,
			AvatarUrl:           avatarURL,
			Nickname:            nickname,
			DiscourseExternalID: externalID,
			DiscourseGroups:     groups,
		}

		if err := auh.DB.Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}

	updates := map[string]interface{}{
		"username":              username,
		"email":                 email,
		"is_admin":              isAdmin,
		"avatar_url":            avatarURL,
		"nickname":              nickname,
		"discourse_external_id": externalID,
		"discourse_groups":      groups,
	}
	if err := auh.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := auh.DB.First(&user, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Register 新用户注册
func (auh *AppUserHandler) Register(c *fiber.Ctx) error {
	var input models.AppUser
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}
	input.Password = string(hashedPassword)

	if err := auh.DB.Create(&input).Error; err != nil {
		log.Error(err) // 记录数据库错误
		return c.Status(500).JSON(fiber.Map{"error": "Failed to register user: " + err.Error()})
	}
	return c.Status(201).JSON(input)
}

func (auh *AppUserHandler) DiscourseLogin(c *fiber.Ctx) error {
	baseURL, secret, callbackURL, err := auh.discourseConnectConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	requestParams := map[string]string{}
	if prompt := strings.TrimSpace(c.Query("prompt")); prompt != "" {
		requestParams["prompt"] = prompt
	}
	if require2FA := strings.TrimSpace(c.Query("require_2fa")); require2FA != "" {
		requestParams["require_2fa"] = require2FA
	}

	nonce, loginURL, err := common.BuildDiscourseConnectURL(baseURL, secret, callbackURL, requestParams)
	if err != nil {
		log.Errorf("failed to build discourse connect login URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize discourse login"})
	}

	ctx := context.Background()
	if err := auh.Redis.Set(ctx, discourseNoncePrefix+nonce, "1", 10*time.Minute).Err(); err != nil {
		log.Errorf("failed to persist discourse connect nonce: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize discourse login"})
	}

	return c.Redirect(loginURL, fiber.StatusFound)
}

func (auh *AppUserHandler) DiscourseCallback(c *fiber.Ctx) error {
	_, secret, _, err := auh.discourseConnectConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	payload := c.Query("sso")
	sig := c.Query("sig")
	if payload == "" || sig == "" {
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "missing_discourse_payload",
		}), fiber.StatusFound)
	}

	values, err := common.ParseDiscourseConnectPayload(secret, payload, sig)
	if err != nil {
		log.Errorf("failed to validate discourse connect payload: %v", err)
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "invalid_discourse_payload",
		}), fiber.StatusFound)
	}

	nonce := strings.TrimSpace(values.Get("nonce"))
	if nonce == "" {
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "missing_nonce",
		}), fiber.StatusFound)
	}

	ctx := context.Background()
	nonceKey := discourseNoncePrefix + nonce
	if _, err := auh.Redis.GetDel(ctx, nonceKey).Result(); err != nil {
		log.Errorf("invalid or expired discourse connect nonce: %v", err)
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "invalid_nonce",
		}), fiber.StatusFound)
	}

	if strings.EqualFold(values.Get("failed"), "true") {
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "failed",
		}), fiber.StatusFound)
	}

	user, err := auh.upsertDiscourseUser(values)
	if err != nil {
		log.Errorf("failed to upsert discourse user: %v", err)
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "user_sync_failed",
		}), fiber.StatusFound)
	}

	token, err := auh.issueUserToken(ctx, user)
	if err != nil {
		log.Errorf("failed to issue user token: %v", err)
		return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
			"status": "error",
			"error":  "token_issue_failed",
		}), fiber.StatusFound)
	}

	auh.setAuthCookie(c, token)
	return c.Redirect(buildFrontendRedirect("/auth/callback", map[string]string{
		"status": "success",
	}), fiber.StatusFound)
}

// Login 用户登录
func (auh *AppUserHandler) Login(c *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	user := models.AppUser{}
	if err := auh.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid password"})
	}
	token, err := auh.issueUserToken(context.Background(), &user)
	if err != nil {
		log.Errorf("failed to issue local login token: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to issue token"})
	}
	auh.setAuthCookie(c, token)
	return c.JSON(token)
}

// Logout 用户注销
func (auh *AppUserHandler) Logout(c *fiber.Ctx) error {
	ctx := context.Background()
	token := common.ExtractToken(c.Get("Authorization"))
	if token == "" {
		token = strings.TrimSpace(c.Cookies("blog_token"))
	}

	if token != "" {
		userID, err := auh.Redis.HGet(ctx, "token_to_username", token).Result()
		if err == nil && userID != "" {
			auh.Redis.HDel(ctx, "username_to_token", userID)
		}
		auh.Redis.HDel(ctx, "token_to_username", token)
	}
	auh.clearAuthCookie(c)
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// GetAuthenticatedUser 获取当前登录的用户信息
func (auh *AppUserHandler) GetAuthenticatedUser(c *fiber.Ctx) error {
	if c.Locals("userId") == nil || c.Locals("userId") == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}
	var user models.AppUser
	result := auh.DB.Take(&user, c.Locals("userId"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Article not found"})
		}
		log.Errorf("Failed to retrieve article: %v", result.Error) // 使用你的日志库记录错误
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.JSON(fiber.Map{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"isAdmin":             user.IsAdmin,
		"avatarUrl":           user.AvatarUrl,
		"nickname":            user.Nickname,
		"discourseExternalId": user.DiscourseExternalID,
		"discourseGroups":     user.DiscourseGroups,
	})
}
