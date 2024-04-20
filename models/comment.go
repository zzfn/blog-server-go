package models

type Comment struct {
	BaseModel
	ObjectID   string  `json:"objectId"`
	ObjectType string  `json:"objectType"`
	Content    string  `json:"content"`
	IP         string  `json:"ip"`
	Address    string  `json:"address"`
	Username   string  `json:"username"`
	Replies    []Reply `json:"replies" gorm:"foreignKey:CommentID"`
	AppUser    AppUser `json:"appUser" gorm:"foreignKey:Username;references:Username"`
}

type Reply struct {
	BaseModel
	CommentID string  `json:"commentId"`
	Content   string  `json:"content"`
	IP        string  `json:"ip"`
	Address   string  `json:"address"`
	Username  string  `json:"username"`
	AppUser   AppUser `json:"appUser" gorm:"foreignKey:Username;references:Username"`
}
