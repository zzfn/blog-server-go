package models

// Comment 评论模型（已废弃 - 现在使用 Discourse 评论系统）
// 保留此结构是为了向后兼容，不再使用本地存储
// Deprecated: 使用 Discourse API 获取评论
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

// Reply 回复模型（已废弃 - 现在使用 Discourse 评论系统）
// 保留此结构是为了向后兼容，不再使用本地存储
// Deprecated: 使用 Discourse API 获取回复
type Reply struct {
	BaseModel
	CommentID string  `json:"commentId"`
	Content   string  `json:"content"`
	IP        string  `json:"ip"`
	Address   string  `json:"address"`
	Username  string  `json:"username"`
	AppUser   AppUser `json:"appUser" gorm:"foreignKey:Username;references:Username"`
}
