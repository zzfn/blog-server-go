package models

import (
	"time"
)

// FinancialTransaction 理财交易流水模型
type FinancialTransaction struct {
	BaseModel
	TransactionType string      `json:"transactionType"` // 交易类型：存入、取出、收益等
	Amount         float64     `json:"amount"`         // 交易金额
	TransactionTime time.Time   `json:"transactionTime"` // 交易时间
	Status         string      `json:"status"`         // 交易状态：成功、失败、处理中
	AccountID      SnowflakeID `json:"accountId"`      // 关联账户ID
	Description    string      `json:"description"`    // 交易描述
	Reference      string      `json:"reference"`      // 交易参考号
	Category       string      `json:"category"`       // 交易类别：投资、分红、手续费等
	Balance        float64     `json:"balance"`        // 交易后余额
}