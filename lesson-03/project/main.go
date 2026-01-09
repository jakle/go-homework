package main

import (
	"gin-examples/project/database"
	"log"

	"gin-examples/project/config"
	"gin-examples/project/handlers"
	"gin-examples/project/middleware"
	"gin-examples/project/services"
	"gin-examples/project/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := database.NewDB(cfg)
	//db, err := gorm.Open(sqlite.Open("users.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 初始化迁移
	migration := database.NewMigration(db.DB)

	// 自动迁移
	if err := migration.Run(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 创建默认管理员用户（可选）
	//if err := migration.CreateAdminUser(); err != nil {
	//	log.Printf("Warning: Failed to create admin user: %v", err)
	//}

	// 数据库健康检查
	if err := db.HealthCheck(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	// 初始化服务
	userService := services.NewUserService(db.DB, cfg)
	userHandler := handlers.NewUserHandler(userService, []byte(cfg.JWT.Secret))

	// 创建 Gin 引擎
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		utils.Success(c, gin.H{
			"status": "ok",
		})
	})

	// 公开路由
	public := r.Group("/api/v1")
	{
		public.POST("/users/register", userHandler.Register)
		public.POST("/users/login", userHandler.Login)
	}

	// 需要认证的路由
	protected := r.Group("/api/v1")
	protected.Use(middleware.Auth([]byte(cfg.JWT.Secret)))
	{
		protected.GET("/users/me", userHandler.GetProfile)
		protected.PUT("/users/me", userHandler.UpdateProfile)
		protected.POST("/users/UploadAvatar", userHandler.UploadAvatar)
	}

	// 需要管理员权限的路由
	admin := r.Group("/api/v1")
	admin.Use(middleware.Auth([]byte(cfg.JWT.Secret)))
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/users", userHandler.GetUserList)
		admin.PUT("/users/:id/role", userHandler.UpdateUserRole)
	}

	// 启动服务器
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
