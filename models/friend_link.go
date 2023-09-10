package models

import "time"

type FriendLink struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Url         string    `json:"url"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsDeleted   bool      `json:"isDeleted"`
}
