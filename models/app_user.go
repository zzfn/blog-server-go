package models

type AppUser struct {
	BaseModel
	Username            string `json:"username"`
	Password            string `json:"-"` // never expose password hashes
	Email               string `json:"email"`
	IsAdmin             bool   `json:"isAdmin"`
	AvatarUrl           string `json:"avatarUrl"`
	Nickname            string `json:"nickname"`
	DiscourseExternalID string `json:"discourseExternalId" gorm:"index"`
	DiscourseGroups     string `json:"discourseGroups"`
}
