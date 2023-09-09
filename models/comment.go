package models

import (
	"blog-server-go/common"
	"time"
)

type Comment struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	ObjectID   string    `json:"objectId"`
	ObjectType string    `json:"objectType"`
	Content    string    `json:"content"`
	IP         string    `json:"ip"`
	Address    string    `json:"address"`
	CreatedBy  string    `json:"createdBy"`
	UpdatedBy  string    `json:"updatedBy"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	IsDeleted  bool      `json:"isDeleted"`
	Replies    []Reply   `json:"replies" gorm:"foreignKey:CommentID"`
}

type Reply struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	CommentID string    `json:"commentId"`
	Content   string    `json:"content"`
	IP        string    `json:"ip"`
	Address   string    `json:"address"`
	CreatedBy string    `json:"createdBy"`
	UpdatedBy string    `json:"updatedBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsDeleted bool      `json:"isDeleted"`
}

func (model *Comment) BeforeCreate() (err error) {
	model.ID, err = common.GenerateID()
	return
}
func (model *Reply) BeforeCreate() (err error) {
	model.ID, err = common.GenerateID()
	return
}
