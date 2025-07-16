package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"backend/commands"
	"backend/handlers"
	"backend/models"
	"strings"
	"crypto/sha256"
	"net/http"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db     *sql.DB
	config models.Config
)

func main() {
	// 设置时区
    loc, err := time.LoadLocation("Asia/Shanghai")
    if err == nil {
        time.Local = loc
    } else {
        log.Printf("设置时区失败: %v", err)
    }
	// 加载配置
	models.LoadConfig("config/config.json", &config)
	config.JWTSecretKey = strings.TrimSpace(config.JWTSecretKey)
    // 确保密钥一致
    log.Printf("JWT密钥: '%s' (长度: %d)", config.JWTSecretKey, len(config.JWTSecretKey))
    
    // 计算并打印密钥哈希
    h := sha256.New()
    h.Write([]byte(config.JWTSecretKey))
    log.Printf("JWT密钥哈希: %x", h.Sum(nil))

	// 初始化数据库
	initDB()

	// 启动命令行界面
	go commands.StartCLI(db, &models.Clients, &models.ClientsLock)

    // 设置Gin路由
    router := gin.Default()
    
	// 添加健康检查端点
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "time": time.Now().Format(time.RFC3339),
        })
    })

    // 用户认证相关路由
    router.POST("/register", handlers.RegisterHandler(db, &config))
    router.POST("/login", handlers.LoginHandler(db, &config))
    
    // 需要认证的路由组
    authGroup := router.Group("/")
    authGroup.Use(handlers.AuthMiddleware(&config)) // 应用JWT认证中间件
    {
        authGroup.GET("/sync", handlers.GetSyncDataHandler(db))
        authGroup.POST("/sync", handlers.PostSyncDataHandler(db))
    }
    
    // WebSocket 单独处理，不使用认证中间件
    router.GET("/ws", handlers.WebSocketHandler(db, &config))


	// 添加调试路由
	router.POST("/debug/validate-token", func(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	// 使用 handlers 包中的 ParseJWTToken 函数
	claims, err := handlers.ParseJWTToken(req.Token, config.JWTSecretKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
			"secret_key": config.JWTSecretKey,
			"secret_key_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(config.JWTSecretKey))),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"claims": claims,
		"secret_key": config.JWTSecretKey,
		"secret_key_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(config.JWTSecretKey))),
	})
})

	// 启动服务器
	log.Printf("服务器启动，监听端口 %d", config.ServerPort)
	log.Fatal(router.Run(fmt.Sprintf(":%d", config.ServerPort)))
}

func initDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	handlers.CreateTables(db)
}