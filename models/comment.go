package models

type Comment struct {
	BaseModel
	ObjectID   string  `json:"objectId"`
	ObjectType string  `json:"objectType"`
	Content    string  `json:"content"`
	IP         string  `json:"ip"`
	Address    string  `json:"address"`
	UserID     string  `json:"userID"`
	Replies    []Reply `json:"replies" gorm:"foreignKey:CommentID"`
}

type Reply struct {
	BaseModel
	CommentID string `json:"commentId"`
	Content   string `json:"content"`
	IP        string `json:"ip"`
	Address   string `json:"address"`
	UserID    string `json:"userID"`
}
