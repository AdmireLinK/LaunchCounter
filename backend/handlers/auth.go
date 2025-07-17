package handlers

import (
	"backend/models"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
	"encoding/base64"
	"crypto/sha256"
	"strings"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler 统一处理用户的注册和登录请求。
// 参数 db 是数据库连接，用于执行数据库操作。
// 参数 config 包含应用的配置信息，如 JWT 密钥。
// 返回一个 Gin 处理函数，用于处理 HTTP 请求。
func AuthHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 定义请求结构体，用于接收客户端发送的 JSON 数据
		var req struct {
			Username string `json:"username" binding:"required"` // 用户名，必填字段
			Password string `json:"password" binding:"required"` // 密码，必填字段
		}

		// 尝试将请求体中的 JSON 数据绑定到 req 结构体
		if err := c.ShouldBindJSON(&req); err != nil {
			// 若绑定失败，返回 400 状态码和错误信息
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
			return
		}

		// 尝试从数据库中获取现有用户信息
		var user models.User
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", req.Username).
			Scan(&user.ID, &user.Password)

		if err == sql.ErrNoRows {
			// 若查询结果为空，说明用户不存在，执行自动注册流程
			// 对用户输入的密码进行哈希处理
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				// 若密码哈希失败，返回 500 状态码和错误信息
				c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
				return
			}

			// 将新用户信息插入到 users 表中
			result, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", 
				req.Username, string(hashedPassword))
			if err != nil {
				// 若插入用户信息失败，返回 500 状态码和错误信息
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
				return
			}

			// 获取新用户的 ID
			userID, _ := result.LastInsertId()
			// 为新用户在 launch_data 表中创建初始记录
			db.Exec(`INSERT INTO launch_data (user_id, total, year_data, month_data, day_data, last_launch)
				VALUES (?, 0, '{}', '{}', '{}', NULL)`, userID)

			// 为新用户生成 JWT 令牌
			token, err := generateJWTToken(int(userID), config)
			if err != nil {
				// 若生成令牌失败，返回 500 状态码和错误信息
				c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
				return
			}
			// 注册成功，返回 200 状态码和生成的 JWT 令牌
			c.JSON(http.StatusOK, gin.H{"token": token})
			return
		} else if err != nil {
			// 若查询数据库过程中出现其他错误，返回 500 状态码和错误信息
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		// 用户存在，执行登录验证流程
		// 比较用户输入的密码和数据库中存储的密码哈希值
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			// 若密码不匹配，返回 401 状态码和错误信息
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "密码错误！如果此账号并非您注册，请您更换一个账号注册；如果此账号为您注册，请输入正确密码",
			})
			return
		}

		// 密码验证通过，为用户生成 JWT 令牌
		token, err := generateJWTToken(user.ID, config)
		if err != nil {
			// 若生成令牌失败，返回 500 状态码和错误信息
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
			return
		}
		// 登录成功，返回 200 状态码和生成的 JWT 令牌
		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}



// AuthMiddleware 是一个中间件生成函数，用于验证请求中的 JWT 令牌。
// 参数 config 包含应用的配置信息，其中 JWTSecretKey 用于验证令牌，Env 用于控制日志输出。
// 返回一个 Gin 处理函数，该函数会在每个请求进入受保护路由时执行。
func AuthMiddleware(config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取 Authorization 字段的值，即 JWT 令牌
		// 通常 JWT 令牌会以 "Bearer <token>" 的格式出现在 Authorization 头中
		tokenString := c.GetHeader("Authorization")
		// 检查令牌是否为空
		if tokenString == "" {
			// 若为空，记录日志，表明请求缺少必要的 Authorization 头
			log.Println("请求缺少Authorization头")
			// 返回 401 状态码和错误信息，提示客户端未提供认证令牌
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			// 终止当前请求的后续处理，不再执行后续的中间件和路由处理函数
			c.Abort()
			return
		}
		
		// 使用统一的 JWT 解析函数解析并验证令牌
		// 传入获取到的令牌字符串、配置中的 JWT 密钥以及配置信息
		claims, err := ParseJWTToken(tokenString, config.JWTSecretKey, config)
		// 检查解析过程中是否出错
		if err != nil {
			// 若出错，记录日志，包含具体的错误信息
			log.Printf("JWT验证失败: %v", err)
			// 返回 401 状态码和错误信息，提示客户端提供的认证令牌无效
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			// 终止当前请求的后续处理，不再执行后续的中间件和路由处理函数
			c.Abort()
			return
		}
		
		// 从解析后的声明中提取用户 ID，并尝试将其转换为 float64 类型
		// 在 JWT 令牌的声明中，user_id 通常以 float64 类型存储
		userID, ok := claims["user_id"].(float64)
		// 检查类型转换是否成功
		if !ok {
			// 若失败，记录日志，包含实际获取到的 user_id 的类型
			log.Printf("用户ID类型错误: %T", claims["user_id"])
			// 返回 401 状态码和错误信息，提示客户端提供的用户 ID 无效
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			// 终止当前请求的后续处理，不再执行后续的中间件和路由处理函数
			c.Abort()
			return
		}
		
		// 记录用户认证成功信息，仅在开发环境下输出日志
		if config.Env == "dev" {
			log.Printf("用户 %d 认证成功", int(userID))
		}
		// 将用户 ID 存储到 Gin 上下文，供后续处理函数使用
		// 后续的处理函数可以通过 c.Get("user_id") 来获取该用户 ID
		c.Set("user_id", int(userID))
		// 继续处理后续的中间件和路由处理函数
		c.Next()
	}
}

// generateJWTToken 用于为指定用户生成 JWT 令牌。
// 参数 userID 是用户的唯一标识，用于在后续请求中识别用户身份。
// 参数 config 包含应用的配置信息，其中 JWTSecretKey 用于对令牌进行签名。
// 返回生成的 JWT 令牌字符串和可能出现的错误。若生成过程正常，错误为 nil。
func generateJWTToken(userID int, config *models.Config) (string, error) {
    // 创建一个新的 JWT 令牌实例，使用 HS256 签名方法，并设置令牌的声明信息
    // HS256 是一种对称加密算法，使用相同的密钥进行签名和验证
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID, // 用户的唯一标识，后续请求可通过该字段识别用户
        "exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 令牌的过期时间，设置为当前时间 7 天后，以 Unix 时间戳表示
        "iat":     time.Now().Unix(), // 令牌的签发时间，记录令牌生成的时刻，以 Unix 时间戳表示
    })

    // 使用配置中的 JWT 密钥对令牌进行签名，生成最终的 JWT 令牌字符串
    // SignedString 方法会将令牌的头部、声明和签名组合成标准的 JWT 格式字符串
    return token.SignedString([]byte(config.JWTSecretKey))
}
// 用户表字段说明:
	// id: 用户唯一标识
	// username: 用户名，唯一
	// password_hash: 用户密码的哈希值

// 添加统一的 JWT 解析函数
// ParseJWTToken 用于解析并验证 JWT 令牌。
// 参数 tokenString 是待解析的 JWT 令牌字符串。
// 参数 secretKey 是用于验证令牌签名的密钥。
// 参数 config 包含应用的配置信息，用于控制开发环境的调试日志输出。
// 返回解析后的 JWT 声明信息和可能出现的错误。若解析验证成功，错误为 nil。
func ParseJWTToken(tokenString, secretKey string, config *models.Config) (jwt.MapClaims, error) {
	// 开发环境输出调试信息
	// 若当前环境为开发环境，记录原始令牌和密钥长度，方便调试
	if config.Env == "dev" {
		log.Printf("原始令牌: %s", tokenString)
		log.Printf("密钥长度: %d", len(secretKey))
	}
	
	// 开发环境记录密钥哈希
	// 若当前环境为开发环境，计算并记录密钥的 SHA-256 哈希值
	if config.Env == "dev" {
		h := sha256.New()
		h.Write([]byte(secretKey))
		log.Printf("密钥哈希: %x", h.Sum(nil))
	}
	
	// 打印令牌的头部和声明部分
	// 将 JWT 令牌按 '.' 分割成三部分，标准的 JWT 由头部、声明和签名三部分组成，以 '.' 分隔
	parts := strings.Split(tokenString, ".")
	// 检查分割后的部分数量是否为 3，若不是则令牌格式无效
	if len(parts) != 3 {
		return nil, fmt.Errorf("令牌格式无效")
	}
	
	// 解码头部
	// 使用 Base64 原始 URL 编码对令牌头部进行解码，JWT 的头部和声明部分通常使用 Base64 编码
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		// 若解码失败，返回错误信息
		return nil, fmt.Errorf("解码头部失败: %v", err)
	}
	// 开发环境输出声明详情
	// 若当前环境为开发环境，记录解码后的令牌头部信息
	if config.Env == "dev" {
		log.Printf("令牌头部: %s", string(headerBytes))
	}
	
	// 解码声明部分
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// 若解码失败，返回错误信息
		return nil, fmt.Errorf("解码声明失败: %v", err)
	}
	// 开发环境输出声明详情
	// 若当前环境为开发环境，记录解码后的令牌声明信息
	if config.Env == "dev" {
		log.Printf("令牌声明: %s", string(claimsBytes))
	}
	
	// 解析并验证令牌
	// 使用 jwt.Parse 函数解析 JWT 令牌，并传入密钥验证函数
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 检查令牌的签名方法是否为 HMAC
		// JWT 支持多种签名方法，这里只接受 HMAC 方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// 若不是 HMAC 方法，返回错误信息
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		// 返回用于验证签名的密钥
		return []byte(secretKey), nil
	})
	
	if err != nil {
		// 若解析过程中出现错误，返回错误信息
		return nil, fmt.Errorf("令牌解析失败: %v", err)
	}
	
	// 检查令牌的声明是否为 jwt.MapClaims 类型，并且令牌是否有效
	// jwt.MapClaims 是一个 map[string]interface{} 类型，用于存储令牌的声明信息
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 若验证通过，返回解析后的声明信息
		return claims, nil
	}
	
	// 若令牌无效，返回错误信息
	return nil, fmt.Errorf("令牌无效")
}

// CreateTables 函数用于在数据库中创建必要的表。
// 若表已存在，则不会重复创建；若创建过程中出现错误，会打印错误信息并终止程序。
// 参数 db 是数据库连接，用于执行 SQL 语句。
func CreateTables(db *sql.DB) {
	// 创建用户表
	// 使用 db.Exec 方法执行 SQL 语句，若表不存在则创建 users 表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			-- 用户唯一标识，自增整数类型，作为主键，用于唯一标识每个用户
			id INT AUTO_INCREMENT PRIMARY KEY,
			-- 用户名，最大长度 50 个字符，唯一且不能为空，用于用户登录和识别
			username VARCHAR(50) UNIQUE NOT NULL,
			-- 用户密码的哈希值，最大长度 255 个字符，不能为空，用于安全存储用户密码
			password_hash VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		-- 使用 InnoDB 存储引擎，支持事务和外键约束
		-- 默认字符集为 utf8mb4，支持存储多语言字符
	`)
	if err != nil {
		// 若创建用户表失败，打印错误信息并终止程序
		log.Fatalf("创建用户表失败: %v", err)
	}

	// 创建发射数据表
	// 使用 db.Exec 方法执行 SQL 语句，若表不存在则创建 launch_data 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS launch_data (
			-- 用户 ID，作为主键，关联 users 表中的用户
			user_id INT PRIMARY KEY,
			-- 总发射次数，整数类型，不能为空，默认值为 0，记录用户的总发射次数
			total INT NOT NULL DEFAULT 0,
			-- 年度发射数据，JSON 类型，存储用户每年的发射次数信息
			year_data JSON,
			-- 月度发射数据，JSON 类型，存储用户每月的发射次数信息
			month_data JSON,
			-- 每日发射数据，JSON 类型，存储用户每天的发射次数信息
			day_data JSON,
			-- 最后一次发射时间，时间戳类型，可为空，记录用户最后一次发射的时间
			last_launch TIMESTAMP NULL,
			-- 外键约束，关联 users 表的 id 字段，当用户记录删除时，级联删除此表中的相关记录
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		-- 使用 InnoDB 存储引擎，支持事务和外键约束
		-- 默认字符集为 utf8mb4，支持存储多语言字符
	`)
	if err != nil {
		// 若创建发射数据表失败，打印错误信息并终止程序
		log.Fatalf("创建发射数据表失败: %v", err)
	}
}