package models

import (
	"blog-server-go/common"
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        SnowflakeID `json:"id"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	IsDeleted bool        `json:"isDeleted"`
	CreatedBy SnowflakeID `json:"createdBy"`
	UpdatedBy SnowflakeID `json:"updatedBy"`
}

func (model *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if len(model.ID) == 0 {
		var newID string
		newID, err = common.GenerateID()
		if err != nil {
			return err
		}
		model.ID = SnowflakeID(newID)
	}
	return nil
}
