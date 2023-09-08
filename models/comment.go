package models

type Comment struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	ContentId uint    `json:"content_id"`
	Content   string  `json:"content"`
	CreatedBy string  `json:"createdBy"`
	Replies   []Reply `json:"replies" gorm:"foreignKey:CommentID"`
}

type Reply struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	CommentID uint    `json:"comment_id"`
	Content   string  `json:"content"`
	CreatedBy string  `json:"createdBy"`
	Children  []Reply `json:"children" gorm:"foreignKey:ParentID"`
	ParentID  uint    `json:"parent_id"`
}
