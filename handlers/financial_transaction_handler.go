package handlers

import (
	"blog-server-go/common"
	"blog-server-go/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type FinancialTransactionHandler struct {
	BaseHandler
}

// CreateTransaction 创建新的理财交易记录
func (fth *FinancialTransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	var transaction models.FinancialTransaction
	if err := c.BodyParser(&transaction); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}
	log.Info(transaction)
	result := fth.DB.Create(&transaction)
	if result.Error != nil {
		log.Errorf("创建交易记录失败: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.Status(fiber.StatusCreated).JSON(transaction)
}

// GetTransactions 获取交易记录列表
func (fth *FinancialTransactionHandler) GetTransactions(c *fiber.Ctx) error {
	var transactions []models.FinancialTransaction
	result := fth.DB.Order("transaction_time DESC").Find(&transactions)
	if result.Error != nil {
		log.Errorf("获取交易记录列表失败: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.JSON(transactions)
}

// GetTransactionByID 根据ID获取单个交易记录
func (fth *FinancialTransactionHandler) GetTransactionByID(c *fiber.Ctx) error {
	id, err := common.ParseString(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的ID"})
	}

	var transaction models.FinancialTransaction
	result := fth.DB.First(&transaction, id)
	if result.Error != nil {
		log.Errorf("获取交易记录失败: %v", result.Error)
		return c.Status(404).JSON(fiber.Map{"error": "交易记录不存在"})
	}

	return c.JSON(transaction)
}

// UpdateTransaction 更新交易记录
func (fth *FinancialTransactionHandler) UpdateTransaction(c *fiber.Ctx) error {
	var transaction models.FinancialTransaction
	if err := c.BodyParser(&transaction); err != nil {
		log.Error(err)
		return &common.BusinessException{
			Code:    5000,
			Message: "无法解析JSON",
		}
	}
	log.Info(transaction)
	result := fth.DB.Updates(&transaction)
	if result.Error != nil {
		log.Errorf("更新交易记录失败: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}
	return c.Status(fiber.StatusOK).JSON(transaction)
}

// DeleteTransaction 删除交易记录
func (fth *FinancialTransactionHandler) DeleteTransaction(c *fiber.Ctx) error {
	id, err := common.ParseString(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的ID"})
	}
	result := fth.DB.Delete(&models.FinancialTransaction{}, id)
	if result.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": "删除交易记录失败"})
	}
	return c.JSON(fiber.Map{"message": "交易记录删除成功"})
}

// GetTransactionsByAccount 根据账户ID获取交易记录
func (fth *FinancialTransactionHandler) GetTransactionsByAccount(c *fiber.Ctx) error {
	accountID, err := common.ParseString(c.Params("accountId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "无效的账户ID"})
	}

	var transactions []models.FinancialTransaction
	result := fth.DB.Where("account_id = ?", accountID).Order("transaction_time DESC").Find(&transactions)
	if result.Error != nil {
		log.Errorf("获取账户交易记录失败: %v", result.Error)
		return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
	}

	return c.JSON(transactions)
}
