package models

import (
	"blog-server-go/common"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
	"time"
)

type Comment struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	ObjectID   string    `json:"objectId"`
	ObjectType string    `json:"objectType"`
	Content    string    `json:"content"`
	IP         string    `json:"ip"`
	Address    string    `json:"address"`
	UserID     string    `json:"userID"`
	CreatedBy  int64     `json:"createdBy"`
	UpdatedBy  int64     `json:"updatedBy"`
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
	UserID    string    `json:"userID"`
	CreatedBy int64     `json:"createdBy"`
	UpdatedBy int64     `json:"updatedBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsDeleted bool      `json:"isDeleted"`
}

func (model *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	model.ID, err = common.GenerateID()
	log.Info("ID", model.ID)
	return
}
func (model *Reply) BeforeCreate(tx *gorm.DB) (err error) {
	model.ID, err = common.GenerateID()
	return
}
