package models

type AppUser struct {
	BaseModel
	Username string `json:"username"`
	Password string `json:"password"` // 注意：密码在API响应中不应该返回，这里仅作为存储的模型
	IsAdmin  bool   `json:"isAdmin"`
}
