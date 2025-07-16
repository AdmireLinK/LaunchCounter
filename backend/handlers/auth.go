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

func RegisterHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
			return
		}

		// 检查用户名是否存在
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", req.Username).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
			return
		}

		// 密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}

		// 创建用户
		result, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", req.Username, string(hashedPassword))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户ID失败"})
			return
		}

		// 创建初始发射数据
		_, err = db.Exec(`
			INSERT INTO launch_data (user_id, total, year_data, month_data, day_data, last_launch)
			VALUES (?, 0, '{}', '{}', '{}', NULL)
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建发射数据失败"})
			return
		}

		// 生成JWT
		token, err := generateJWTToken(int(userID), config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

func LoginHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
			return
		}

		// 获取用户信息
		var user models.User
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", req.Username).
			Scan(&user.ID, &user.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名不存在"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}

        // 生成令牌
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
            "user_id": user.ID,
            "exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
        })
        
        tokenString, err := token.SignedString([]byte(config.JWTSecretKey))
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
            return
        }
        
        c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

func AuthMiddleware(config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			log.Println("请求缺少Authorization头")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}
		
		// 使用统一的 JWT 解析函数
		claims, err := ParseJWTToken(tokenString, config.JWTSecretKey)
		if err != nil {
			log.Printf("JWT验证失败: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			c.Abort()
			return
		}
		
		userID, ok := claims["user_id"].(float64)
		if !ok {
			log.Printf("用户ID类型错误: %T", claims["user_id"])
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
			c.Abort()
			return
		}
		
		log.Printf("用户 %d 认证成功", int(userID))
		c.Set("user_id", int(userID))
		c.Next()
	}
}

func generateJWTToken(userID int, config *models.Config) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7天有效期
        "iat":     time.Now().Unix(), // 添加签发时间
    })

	return token.SignedString([]byte(config.JWTSecretKey))
}
	// 用户表字段说明:
	// id: 用户唯一标识
	// username: 用户名，唯一
	// password_hash: 用户密码的哈希值

// 添加统一的 JWT 解析函数
func ParseJWTToken(tokenString, secretKey string) (jwt.MapClaims, error) {
	log.Printf("原始令牌: %s", tokenString)
	log.Printf("密钥: '%s'", secretKey)
	log.Printf("密钥长度: %d", len(secretKey))
	
	// 打印密钥哈希
	h := sha256.New()
	h.Write([]byte(secretKey))
	log.Printf("密钥哈希: %x", h.Sum(nil))
	
	// 打印令牌的头部和声明部分
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("令牌格式无效")
	}
	
	// 解码头部
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("解码头部失败: %v", err)
	}
	log.Printf("令牌头部: %s", string(headerBytes))
	
	// 解码声明
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("解码声明失败: %v", err)
	}
	log.Printf("令牌声明: %s", string(claimsBytes))
	
	// 解析并验证令牌
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("令牌解析失败: %v", err)
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("令牌无效")
}

func CreateTables(db *sql.DB) {
	// 用户表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	if err != nil {
		log.Fatalf("创建用户表失败: %v", err)
	}

	// 发射数据表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS launch_data (
			user_id INT PRIMARY KEY,
			total INT NOT NULL DEFAULT 0,
			year_data JSON,
			month_data JSON,
			day_data JSON,
			last_launch TIMESTAMP NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`)
	if err != nil {
		log.Fatalf("创建发射数据表失败: %v", err)
	}
}