package models

type AppUser struct {
	BaseModel
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsAdmin   bool   `json:"isAdmin"`
	AvatarUrl string `json:"avatar_url"`
	Nickname  string `json:"nickname"`
}
