package models

type Article struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ViewCount int    `json:"viewCount"`
	Tag       string `json:"tag"`
	SortOrder int    `json:"sortOrder"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	IsDeleted bool   `json:"isDeleted"`
	CreatedBy string `json:"createdBy"`
	UpdatedBy string `json:"updatedBy"`
}
