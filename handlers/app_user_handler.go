package handlers

import (
	"blog-server-go/common"
	"blog-server-go/models"
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"golang.org/x/crypto/bcrypt"
)

type AppUserHandler struct {
	BaseHandler
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

	newToken, _ := common.GenerateID()
	var ctx = context.Background()
	oldToken, err := auh.Redis.HGet(ctx, "username_to_token", string(user.ID)).Result()
	if err == nil && oldToken != "" {
		// 从 "Token → 用户名" 哈希表中删除这个 Token
		auh.Redis.HDel(ctx, "token_to_username", oldToken)
	}
	auh.Redis.HSet(ctx, "username_to_token", string(user.ID), newToken)
	auh.Redis.HSet(ctx, "token_to_username", newToken, string(user.ID))
	return c.JSON(newToken)
}

// Logout 用户注销
func (auh *AppUserHandler) Logout(c *fiber.Ctx) error {

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// GetAuthenticatedUser 获取当前登录的用户信息
func (auh *AppUserHandler) GetAuthenticatedUser(c *fiber.Ctx) error {
	user := models.AppUser{
		// Fill with mock or actual data
	}
	return c.JSON(user)
}
