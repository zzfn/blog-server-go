package models

type Article struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Tag       string `json:"tag"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
