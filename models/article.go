package models

type Article struct {
	BaseModel
	Title     string `json:"title"`
	Content   string `json:"content"`
	ViewCount int    `json:"viewCount"`
	Tag       string `json:"tag"`
	SortOrder int    `json:"sortOrder"`
	IsActive  bool   `json:"isActive"`
	Summary   string `json:"summary" gorm:"-"`
}
