package models

type AppUser struct {
	BaseModel
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsAdmin   bool   `json:"isAdmin"`
	AvatarUrl string `json:"avatarUrl"`
	Nickname  string `json:"nickname"`
}
