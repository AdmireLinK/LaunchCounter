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
	ServerPort    int      `json:"server_port"`
	DBHost        string   `json:"db_host"`
	DBPort        int      `json:"db_port"`
	DBUser        string   `json:"db_user"`
	DBPassword    string   `json:"db_password"`
	DBName        string   `json:"db_name"`
	JWTSecretKey  string   `json:"jwt_secret_key"`
	Env           string   `json:"env"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type LaunchData struct {
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
// LoadConfig 函数用于从指定文件加载配置信息到 Config 结构体中。
// 如果配置文件中 JWT 密钥为空，会生成一个新的密钥，并将更新后的配置保存回文件。
// 参数 filename 是配置文件的路径，config 是指向 Config 结构体的指针，用于存储配置信息。
func LoadConfig(filename string, config *Config) {
	// 尝试读取现有配置文件
	// 使用 os.Open 打开指定的配置文件，如果文件存在且打开成功，则继续处理
	if file, err := os.Open(filename); err == nil {
		// 确保文件在函数结束时关闭，避免资源泄漏
		defer file.Close()
		// 使用 json.NewDecoder 从文件中解码 JSON 数据到 config 结构体
		// 若解码失败，config 会保留默认值
		json.NewDecoder(file).Decode(config)
	}

	// 检查 JWT 密钥是否为空
	// 如果 JWT 密钥为空字符串，需要生成一个新的密钥以保证安全性
	if config.JWTSecretKey == "" {
		// 生成一个 32 字节的随机密钥
		// 使用 crypto/rand 包生成安全的随机字节序列，适用于加密场景
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			// 若随机字节生成失败，抛出恐慌并记录错误信息
			// 因为 JWT 密钥生成失败会影响系统的认证功能，程序无法继续安全运行
			panic("无法生成JWT密钥: " + err.Error())
		}
		// 将生成的随机字节序列使用 Base64 标准编码转换为字符串
		// 方便存储和传输
		config.JWTSecretKey = base64.StdEncoding.EncodeToString(key)
	}

	// 保存配置信息到文件
	// 使用 os.Create 创建或覆盖指定的配置文件，如果创建成功，则继续处理
	if file, err := os.Create(filename); err == nil {
		// 确保文件在函数结束时关闭，避免资源泄漏
		defer file.Close()
		// 使用 json.NewEncoder 将 config 结构体编码为 JSON 数据并写入文件
		// 若写入失败，可能需要手动检查文件权限等问题
		json.NewEncoder(file).Encode(config)
	}
}

// 获取在线客户端信息
// GetOnlineClients 获取当前在线客户端的信息。
// 该函数会返回一个映射，键为格式化后的用户 ID（格式为 "user_<用户ID>"），
// 值为该用户对应的客户端信息列表。每个客户端信息包含 IP 地址、连接时间和连接时长。
func GetOnlineClients() map[string][]map[string]interface{} {
	// 加读锁，防止在读取 Clients 时被其他协程修改，保证并发安全
	ClientsLock.RLock()
	// 函数结束时释放读锁，确保资源正确释放
	defer ClientsLock.RUnlock()

	// 初始化结果映射，用于存储最终的客户端信息
	result := make(map[string][]map[string]interface{})
	// 遍历 Clients 映射，其中 key 为用户 ID，value 为该用户对应的客户端列表
	for userID, clients := range Clients {
		// 初始化该用户的客户端信息列表
		userClients := make([]map[string]interface{}, 0)
		// 遍历该用户的客户端列表
		for _, client := range clients {
			// 将每个客户端的 IP 地址、连接时间和连接时长添加到用户客户端信息列表中
			userClients = append(userClients, map[string]interface{}{
				"ip":         client.IP, // 客户端的 IP 地址
				// 格式化连接时间为 "2006-01-02 15:04:05" 格式，这是 Go 语言中时间格式化的标准模板
				"connect_at": client.ConnectAt.Format("2006-01-02 15:04:05"),
				// 计算并格式化连接时长，精确到秒
				"duration":   time.Since(client.ConnectAt).Round(time.Second).String(),
			})
		}
		// 将该用户的客户端信息列表添加到结果映射中，键为格式化后的用户 ID
		result[fmt.Sprintf("user_%d", userID)] = userClients
	}
	return result
}

// WritePump 是 Client 结构体的方法，用于持续从 Client 的 Send 通道读取数据，
// 并将数据以 JSON 格式通过 WebSocket 连接发送给客户端。
// 当通道关闭或发送过程中出现错误时，会关闭 WebSocket 连接。
func (c *Client) WritePump() {
	// 使用 defer 确保在函数退出时关闭 WebSocket 连接，避免资源泄漏
	defer func() {
		c.Conn.Close()
	}()

	// 进入无限循环，持续监听 Send 通道，等待数据发送
	for {
		select {
		// 从 c.Send 通道接收数据，ok 表示通道是否正常打开
		case data, ok := <-c.Send:
			// 检查通道是否已关闭
			if !ok {
				// 通道已关闭，发送 WebSocket 关闭消息告知客户端连接即将关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 序列化数据
			// 将接收到的 LaunchData 类型数据使用 json.Marshal 方法转换为 JSON 字节切片
			jsonData, err := json.Marshal(data)
			if err != nil {
				// 序列化失败，记录错误日志并跳过本次发送，继续等待下一次数据
				log.Printf("序列化数据失败: %v", err)
				continue
			}

			// 发送JSON数据
			// 使用 WriteMessage 方法将 JSON 数据以文本消息的形式通过 WebSocket 连接发送给客户端
			if err := c.Conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				// 发送失败，记录错误日志并退出函数，结束写操作
				log.Printf("发送消息失败: %v", err)
				return
			}
		}
	}
}

// 添加 WebSocket 读协程
// ReadPump 是 Client 结构体的方法，用于持续从 WebSocket 连接读取客户端发送的消息。
// 当读取过程中出现错误或者连接关闭时，会自动关闭 WebSocket 连接。
// 由于本项目不需要处理来自客户端的消息，该方法仅关注错误处理。
func (c *Client) ReadPump() {
	// 使用 defer 确保在函数退出时关闭 WebSocket 连接，避免资源泄漏
	defer func() {
		c.Conn.Close()
	}()

	// 进入无限循环，持续从 WebSocket 连接读取消息
	for {
		// ReadMessage 从 WebSocket 连接读取下一条消息。
		// 第一个返回值是消息类型（如文本消息、二进制消息等），
		// 第二个返回值是消息数据的字节切片，
		// 第三个返回值是可能出现的错误。
		// 由于本项目不需要处理消息内容，这里忽略前两个返回值
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			// 判断是否为意外关闭错误，CloseGoingAway 表示客户端正常关闭，CloseAbnormalClosure 表示异常关闭
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录意外关闭的错误信息，方便后续排查问题
				log.Printf("WebSocket错误: %v", err)
			}
			// 出现错误，跳出循环，结束读取操作
			break
		}
		// 本项目不需要处理来自客户端的消息，继续下一次读取
	}
}