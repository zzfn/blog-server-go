package models

type AppUser struct {
	BaseModel
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsAdmin   bool   `json:"isAdmin"`
	AvatarUrl bool   `json:"avatar_url"`
}
