package models

type AppUser struct {
	BaseModel
	Username            string `json:"username" gorm:"uniqueIndex"`
	Password            string `json:"-"` // never expose password hashes
	Email               string `json:"email" gorm:"uniqueIndex"`
	IsAdmin             bool   `json:"isAdmin"`
	AvatarUrl           string `json:"avatarUrl"`
	Nickname            string `json:"nickname"`
	DiscourseExternalID string `json:"discourseExternalId" gorm:"uniqueIndex"`
	DiscourseGroups     string `json:"discourseGroups"`
}
