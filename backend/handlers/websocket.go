package handlers

import (
	"backend/models"
	"database/sql"
	"log"
	"net/http"
	"time"
	"github.com/golang-jwt/jwt/v4"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)



// WebSocketHandler 返回一个 Gin 处理函数，用于处理 WebSocket 连接请求。
// 参数 db 是数据库连接，用于查询用户信息。
// 参数 config 包含应用的配置信息，如 JWT 密钥和环境模式等。
func WebSocketHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从查询参数获取 token
        tokenString := c.Query("token")
        if tokenString == "" {
            // 若未提供 token，记录日志并返回 401 未授权响应
            log.Println("WebSocket连接缺少token参数")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
            return
        }

        // 若当前环境不是生产环境，记录收到的 WebSocket 连接请求及 token 信息
        if config.Env != "release" {
            log.Printf("收到WebSocket连接请求，token: %s", tokenString)
        }

        // 验证 token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // 检查 token 的签名方法是否为 HMAC
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                // 若签名方法不符，返回错误信息
                return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
            }
            // 返回 JWT 签名密钥
            return []byte(config.JWTSecretKey), nil
        })

        if err != nil {
            // 若 JWT 验证失败，在开发环境记录错误日志，并返回 401 未授权响应
            if config.Env == "dev" {
                log.Printf("JWT验证失败: %v", err)
            }
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
            return
        }

        // 初始化 WebSocket 升级器，设置允许来自任何来源的连接
        upgrader := websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                // 允许来自任何来源的 WebSocket 连接
                return true
            },
        }

        // 从 token 获取用户 ID
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            // 若无法解析 JWT 声明，记录日志并返回 401 未授权响应
            log.Println("无法解析JWT声明")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌声明"})
            return
        }

        userID, ok := claims["user_id"]
        if !ok {
            // 若令牌缺少 user_id 声明，记录日志并返回 401 未授权响应
            log.Println("令牌缺少user_id声明")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
            return
        }

        // 转换用户ID为整数
        userIDInt, ok := userID.(float64)
        if !ok {
            // 若用户 ID 类型错误，记录日志并返回 401 未授权响应
            log.Printf("用户ID类型错误: %T", userID)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID格式"})
            return
        }

        // 升级 HTTP 连接为 WebSocket 连接
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            // 若升级失败，记录日志并返回
            log.Printf("WebSocket升级失败: %v", err)
            return
        }

        // 获取用户名
        var username string
        err = db.QueryRow("SELECT username FROM users WHERE id = ?", int(userIDInt)).Scan(&username)
        if err != nil {
            // 若获取用户信息失败，记录日志并关闭 WebSocket 连接
            log.Printf("获取用户信息失败: %v", err)
            conn.Close()
            return
        }

        // 创建客户端实例
        client := &models.Client{
            Conn:      conn,       // WebSocket 连接
            UserID:    int(userIDInt), // 用户 ID
            Username:  username,   // 用户名
            IP:        c.ClientIP(), // 客户端 IP 地址
            ConnectAt: time.Now(), // 连接时间
            Send:      make(chan models.LaunchData, 256), // 用于发送数据的通道
        }

        // 注册客户端
        registerClient(client, config)
        // 确保在函数结束时注销客户端
        defer unregisterClient(client, config)

        // 启动写协程，负责向客户端发送数据
        go client.WritePump()
        // 启动读协程，负责从客户端接收数据
        client.ReadPump()

        // 在开发环境记录 WebSocket 连接建立信息
        if config.Env == "dev" {
            log.Printf("用户 %s (%d) WebSocket连接已建立", username, int(userIDInt))
        }
    }
}

// registerClient 函数用于将新的客户端实例注册到全局的客户端映射中。
// 参数 client 是需要注册的客户端实例，包含客户端的连接信息、用户信息等。
// 参数 config 包含应用的配置信息，如环境模式等，用于控制日志输出。
func registerClient(client *models.Client, config *models.Config) {
	// 对全局的客户端映射加写锁，防止在注册过程中其他协程对客户端映射进行读写操作，保证并发安全。
	models.ClientsLock.Lock()
	// 函数结束时自动释放写锁，确保资源正确释放。
	defer models.ClientsLock.Unlock()

	// 将新的客户端实例添加到全局客户端映射中对应用户 ID 的客户端列表里。
	// 若该用户 ID 对应的列表不存在，则创建一个新的列表。
	models.Clients[client.UserID] = append(models.Clients[client.UserID], client)

	// 若当前环境为开发环境，记录新客户端连接的日志，包含用户名、用户 ID 和客户端 IP 地址。
	if config.Env == "dev" {
		log.Printf("用户 %s (%d) 已连接, IP: %s", client.Username, client.UserID, client.IP)
	}
}

// unregisterClient 函数用于将指定客户端实例从全局的客户端映射中注销，
// 并关闭客户端的连接和发送通道。
// 参数 client 是需要注销的客户端实例，包含客户端的连接信息、用户信息等。
// 参数 config 包含应用的配置信息，如环境模式等，用于控制日志输出。
func unregisterClient(client *models.Client, config *models.Config) {
	// 对全局的客户端映射加写锁，防止在注销过程中其他协程对客户端映射进行读写操作，保证并发安全。
	models.ClientsLock.Lock()
	// 函数结束时自动释放写锁，确保资源正确释放。
	defer models.ClientsLock.Unlock()

	// 从全局客户端映射中获取该用户 ID 对应的客户端列表
	userClients := models.Clients[client.UserID]
	// 遍历该用户的客户端列表
	for i, c := range userClients {
		// 找到需要注销的客户端实例
		if c == client {
			// 从列表中移除该客户端实例
			models.Clients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
			// 找到后跳出循环
			break
		}
	}

	// 关闭客户端的发送通道，防止继续向已断开的客户端发送数据
	close(client.Send)
	// 关闭客户端的 WebSocket 连接
	client.Conn.Close()

	// 若当前环境为开发环境，记录客户端断开连接的日志，包含用户名和用户 ID
	if config.Env == "dev" {
		log.Printf("用户 %s (%d) 已断开连接", client.Username, client.UserID)
	}
}