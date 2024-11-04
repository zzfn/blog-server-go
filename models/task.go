package models

type Task struct {
	BaseModel
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Amount   float64 `json:"amount"`
	Cycle    string  `json:"cycle"`
	Nickname string  `json:"nickname"`
}
