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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求
	},
}

func WebSocketHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从查询参数获取 token
        tokenString := c.Query("token")
        if tokenString == "" {
            log.Println("WebSocket连接缺少token参数")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
            return
        }


        
        log.Printf("收到WebSocket连接请求，token: %s", tokenString)

        // 验证 token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
            }
            return []byte(config.JWTSecretKey), nil
        })


        // 删除重复的JWT解析代码

        // 从 token 获取用户 ID
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            log.Println("无法解析JWT声明")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌声明"})
            return
        }

        userID, ok := claims["user_id"]
        if !ok {
            log.Println("令牌缺少user_id声明")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
            return
        }
        
        // 转换用户ID为整数
        userIDInt, ok := userID.(float64)
        if !ok {
            log.Printf("用户ID类型错误: %T", userID)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID格式"})
            return
        }

        // 升级为 WebSocket
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            log.Printf("WebSocket升级失败: %v", err)
            return
        }

        // 获取用户名
        var username string
        err = db.QueryRow("SELECT username FROM users WHERE id = ?", int(userIDInt)).Scan(&username)
        if err != nil {
            log.Printf("获取用户信息失败: %v", err)
            conn.Close()
            return
        }

        client := &models.Client{
            Conn:      conn,
            UserID:    int(userIDInt),
            Username:  username,
            IP:        c.ClientIP(),
            ConnectAt: time.Now(),
            Send:      make(chan models.LaunchData, 256),
        }

        registerClient(client)
        defer unregisterClient(client)

        go client.WritePump()
        client.ReadPump()
        
        log.Printf("用户 %s (%d) WebSocket连接已建立", username, int(userIDInt))
    }
}

func registerClient(client *models.Client) {
	models.ClientsLock.Lock()
	defer models.ClientsLock.Unlock()

	models.Clients[client.UserID] = append(models.Clients[client.UserID], client)
	log.Printf("用户 %s (%d) 已连接, IP: %s", client.Username, client.UserID, client.IP)
}

func unregisterClient(client *models.Client) {
	models.ClientsLock.Lock()
	defer models.ClientsLock.Unlock()

	userClients := models.Clients[client.UserID]
	for i, c := range userClients {
		if c == client {
			models.Clients[client.UserID] = append(userClients[:i], userClients[i+1:]...)
			break
		}
	}

	close(client.Send)
	client.Conn.Close()
	log.Printf("用户 %s (%d) 已断开连接", client.Username, client.UserID)
}