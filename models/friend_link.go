package models

type FriendLink struct {
	BaseModel
	Name        string `json:"name"`
	Logo        string `json:"logo"`
	Url         string `json:"url"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
}
