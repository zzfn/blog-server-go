package models

type Comment struct {
	ID        uint `gorm:"primaryKey"`
	ContentID uint
	Content   string
	UserID    uint
	Replies   []Reply `gorm:"foreignKey:CommentID"`
}

type Reply struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint
	Content   string
	UserID    uint
	Children  []Reply `gorm:"foreignKey:ParentID"`
	ParentID  uint
}
