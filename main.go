package main

import (
	"blog-server-go/config"
	"blog-server-go/middleware"
	"blog-server-go/routes"
	"fmt"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// 初始化 Fiber 应用
	app := fiber.New()
	app.Use(middleware.ResponseMiddleware)

	// 初始化数据库连接
	db, err := config.SetupDatabase()
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println("Failed to get DB object:", err)
		return
	}
	defer sqlDB.Close()

	// 引入路由
	routes.SetupArticleRoutes(app, db)

	// 启动 Fiber 应用
	err = app.Listen(":3000")
	if err != nil {
		fmt.Println("Failed to start Fiber app:", err)
	}
}
