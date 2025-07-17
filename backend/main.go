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

// main 是程序的入口函数，负责初始化各项配置、启动数据库、命令行界面和 HTTP 服务器。
func main() {
	// 设置时区
	// 尝试加载亚洲/上海时区，如果成功则将本地时区设置为该时区，失败则记录错误信息。
    loc, err := time.LoadLocation("Asia/Shanghai")
    if err == nil {
        time.Local = loc
    } else {
        log.Printf("设置时区失败: %v", err)
    }
	// 加载配置
	// 从 config/config.json 文件中加载配置信息到全局变量 config 中。
	models.LoadConfig("config/config.json", &config)

	// 设置Gin运行模式
	if config.Env == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	// 去除 JWT 密钥两端的空白字符
	// 防止因配置文件中可能存在的空白字符影响 JWT 验证
	config.JWTSecretKey = strings.TrimSpace(config.JWTSecretKey)
    // 开发环境输出密钥信息
	if config.Env == "dev" {
		log.Printf("JWT密钥: '%s' (长度: %d)", config.JWTSecretKey, len(config.JWTSecretKey))
		// 计算并打印密钥哈希
		h := sha256.New()
		h.Write([]byte(config.JWTSecretKey))
		log.Printf("JWT密钥哈希: %x", h.Sum(nil))
	}

	// 初始化数据库
	// 调用 initDB 函数连接数据库并创建必要的表。
	initDB()

	// 启动命令行界面
	// 在一个新的 goroutine 中启动命令行界面，传入数据库连接、客户端列表和客户端锁。
	go commands.StartCLI(db, &models.Clients, &models.ClientsLock)

    // 设置Gin路由
    // 创建一个默认的 Gin 引擎，包含日志和恢复中间件。
    router := gin.Default()
	// 设置信任的代理，nil 表示不信任任何代理，直接获取客户端真实 IP
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// 添加健康检查端点
    // 注册一个 GET 请求的健康检查端点，返回服务器状态和当前时间。
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "time": time.Now().Format(time.RFC3339),
        })
    })

    // 用户认证相关路由
    // 注册用户注册和登录的 POST 请求路由，分别调用对应的处理函数。
    router.POST("/register", handlers.RegisterHandler(db, &config))
    router.POST("/login", handlers.LoginHandler(db, &config))
    
    // 需要认证的路由组
    // 创建一个路由组，应用 JWT 认证中间件，只有通过认证的请求才能访问该组内的路由。
    authGroup := router.Group("/")
    authGroup.Use(handlers.AuthMiddleware(&config)) // 应用JWT认证中间件
    {
        // 注册同步数据的 GET 和 POST 请求路由，分别调用对应的处理函数。
        authGroup.GET("/sync", handlers.GetSyncDataHandler(db, &config))
        authGroup.POST("/sync", handlers.PostSyncDataHandler(db, &config))
    }
    
    // WebSocket 单独处理，不使用认证中间件
    // 注册 WebSocket 连接的 GET 请求路由，调用对应的处理函数。
    router.GET("/ws", handlers.WebSocketHandler(db, &config))

	// 添加调试路由
	// 注册一个调试用的 POST 请求路由，用于验证 JWT 令牌的有效性。
	router.POST("/debug/validate-token", func(c *gin.Context) {
		// 定义请求结构体，用于解析 JSON 请求体中的 Token 字段。
		var req struct {
			Token string `json:"token"`
		}
		
		// 尝试将请求体绑定到 req 结构体，如果失败则返回错误响应。
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
			return
		}

		// 使用 handlers 包中的 ParseJWTToken 函数解析 JWT 令牌
		claims, err := handlers.ParseJWTToken(req.Token, config.JWTSecretKey, &config)
		if err != nil {
			// 解析失败，返回验证结果为 false 及错误信息
			c.JSON(http.StatusOK, gin.H{
				"valid": false,
				"error": err.Error(),
				"secret_key": config.JWTSecretKey,
				"secret_key_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(config.JWTSecretKey))),
			})
			return
		}
		
		// 解析成功，返回验证结果为 true 及令牌声明信息
		c.JSON(http.StatusOK, gin.H{
			"valid": true,
			"claims": claims,
			"secret_key": config.JWTSecretKey,
			"secret_key_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(config.JWTSecretKey))),
		})
	})

	// 启动服务器
	// 打印服务器启动信息，指定监听端口，并启动 HTTP 服务器，若启动失败则记录错误信息。
	log.Printf("服务器启动，监听端口 %d", config.ServerPort)
	log.Fatal(router.Run(fmt.Sprintf(":%d", config.ServerPort)))
}

// initDB 函数用于初始化数据库连接，验证连接有效性，并创建必要的数据库表。
func initDB() {
	// 构建 MySQL 数据库连接的数据源名称（DSN）
	// 格式为 username:password@tcp(host:port)/database_name?parseTime=true
	// parseTime=true 表示将数据库中的时间类型自动解析为 Go 的 time.Time 类型
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	var err error
	// 使用 sql.Open 函数创建一个数据库连接池，此时并未实际连接数据库
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		// 若创建连接池失败，打印错误信息并终止程序
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 使用 db.Ping 方法尝试与数据库建立实际连接，验证连接是否有效
	if err := db.Ping(); err != nil {
		// 若连接测试失败，打印错误信息并终止程序
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	// 调用 handlers 包中的 CreateTables 函数，在数据库中创建必要的表
	handlers.CreateTables(db)
}