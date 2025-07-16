package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"log"

	"github.com/gorilla/websocket"
)

type Config struct {
	ServerPort   int    `json:"server_port"`
	DBHost       string `json:"db_host"`
	DBPort       int    `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	JWTSecretKey string `json:"jwt_secret_key"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type LaunchData struct {
	Version    int             `json:"version"`
	UserID     int             `json:"user_id"`
	Total      int             `json:"total"`
	YearData   map[string]int  `json:"year_data"`
	MonthData  map[string]int  `json:"month_data"`
	DayData    map[string]int  `json:"day_data"`
	LastLaunch time.Time       `json:"last_launch"`
}

type Client struct {
	Conn       *websocket.Conn
	UserID     int
	Username   string
	IP         string
	ConnectAt  time.Time
	Send       chan LaunchData
}

var (
	Clients     = make(map[int][]*Client)
	ClientsLock sync.RWMutex
)

// 加载配置（如果JWT密钥为空则生成）
func LoadConfig(filename string, config *Config) {
	// 尝试读取现有配置
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(config)
	}

	// 如果JWT密钥为空，生成新密钥
	if config.JWTSecretKey == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			panic("无法生成JWT密钥: " + err.Error())
		}
		config.JWTSecretKey = base64.StdEncoding.EncodeToString(key)
	}

	// 保存配置（包含生成的密钥）
	if file, err := os.Create(filename); err == nil {
		defer file.Close()
		json.NewEncoder(file).Encode(config)
	}
}

// 获取在线客户端信息
func GetOnlineClients() map[string][]map[string]interface{} {
	ClientsLock.RLock()
	defer ClientsLock.RUnlock()

	result := make(map[string][]map[string]interface{})
	for userID, clients := range Clients {
		userClients := make([]map[string]interface{}, 0)
		for _, client := range clients {
			userClients = append(userClients, map[string]interface{}{
				"ip":         client.IP,
				"connect_at": client.ConnectAt.Format("2006-01-02 15:04:05"),
				"duration":   time.Since(client.ConnectAt).Round(time.Second).String(),
			})
		}
		result[fmt.Sprintf("user_%d", userID)] = userClients
	}
	return result
}

// 添加 WebSocket 写协程
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.Send:
			if !ok {
				// 通道已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 序列化数据
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("序列化数据失败: %v", err)
				continue
			}

			// 发送JSON数据
			if err := c.Conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Printf("发送消息失败: %v", err)
				return
			}
		}
	}
}

// 添加 WebSocket 读协程
func (c *Client) ReadPump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}
		// 本项目不需要处理来自客户端的消息
	}
}