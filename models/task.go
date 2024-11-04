package models

import "time"

type Task struct {
	BaseModel
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Cycle       string    `json:"cycle"`
	Amount      float64   `json:"amount"`
	StartDate   time.Time `json:"startDate"`
	DueDate     time.Time `json:"dueDate"`
}
