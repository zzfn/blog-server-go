package models

import "time"

type Task struct {
	BaseModel
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Amount    float64   `json:"amount"`
	Cycle     string    `json:"cycle"`
	StartDate time.Time `json:"startDate"`
	DueDate   time.Time `json:"dueDate"`
}
