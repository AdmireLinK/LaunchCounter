package handlers

import (
	"backend/models"
	"database/sql"
	"encoding/json"
	"net/http"
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

// GetSyncDataHandler 返回一个 Gin 处理函数，用于处理获取用户同步数据的请求。
// 参数 db 是数据库连接，用于执行数据库查询操作。
func GetSyncDataHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Gin 上下文获取用户 ID
		userID := c.GetInt("user_id")
		// 记录日志，表明正在获取指定用户的同步数据
		if config.Env == "dev" {
			log.Printf("成功获取用户 %d 的同步数据", userID)
		}

		// 初始化 LaunchData 结构体，用于存储从数据库获取的发射数据
		var data models.LaunchData
		// 用于存储从数据库获取的 JSON 格式的年度、月度和日度数据
		var yearData, monthData, dayData []byte
		// 用于存储从数据库获取的最后一次发射时间，支持 NULL 值
		var lastLaunch sql.NullTime

		// 添加更健壮的查询逻辑
		// 执行 SQL 查询，从 launch_data 表中获取指定用户的发射数据
		err := db.QueryRow(`
			SELECT total, year_data, month_data, day_data, last_launch
			FROM launch_data
			WHERE user_id = ?
		`, userID).Scan(
			&data.Total,
			&yearData,
			&monthData,
			&dayData,
			&lastLaunch,
		)

		// 检查查询是否出错
		if err != nil {
			// 若查询结果为空，说明用户还没有发射数据
			if err == sql.ErrNoRows {
				log.Printf("用户 %d 未找到发射数据", userID)
				// 创建初始数据
				_, createErr := db.Exec(`
					INSERT INTO launch_data 
					(user_id, total, year_data, month_data, day_data, last_launch)
					VALUES (?, 0, '{}', '{}', '{}', NULL)
				`, userID)
				
				// 检查创建初始数据是否失败
				if createErr != nil {
					log.Printf("创建初始数据失败: %v", createErr)
					// 若失败，返回 500 状态码和错误信息
					c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据失败"})
					return
				}
				
				// 初始化空的发射数据
				data = models.LaunchData{
					Total:      0,
					YearData:   make(map[string]int),
					MonthData:  make(map[string]int),
					DayData:    make(map[string]int),
					LastLaunch: time.Time{},
				}
				// 返回 200 状态码和空的发射数据
				c.JSON(http.StatusOK, gin.H{
					"user_id":     data.UserID,
					"total":       data.Total,
					"year_data":   data.YearData,
					"month_data":  data.MonthData,
					"day_data":    data.DayData,
					"last_launch": data.LastLaunch,
				})
				return
			}
			
			// 若查询过程中出现其他错误，记录日志并返回 500 状态码和错误信息
			log.Printf("数据库查询失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		// 处理时间字段
		// 若 lastLaunch 字段有效，将其值赋给 data.LastLaunch
		if lastLaunch.Valid {
			data.LastLaunch = lastLaunch.Time
		} else {
			// 否则，将 data.LastLaunch 初始化为零值时间
			data.LastLaunch = time.Time{}
		}

		// 解析JSON数据
		// 解析从数据库获取的年度数据 JSON 字节切片到 data.YearData
		if err := json.Unmarshal(yearData, &data.YearData); err != nil {
			log.Printf("解析年度数据失败: %v", err)
			// 若解析失败，初始化 data.YearData 为空映射
			data.YearData = make(map[string]int)
		}
		// 解析从数据库获取的月度数据 JSON 字节切片到 data.MonthData
		if err := json.Unmarshal(monthData, &data.MonthData); err != nil {
			log.Printf("解析月度数据失败: %v", err)
			// 若解析失败，初始化 data.MonthData 为空映射
			data.MonthData = make(map[string]int)
		}
		// 解析从数据库获取的日度数据 JSON 字节切片到 data.DayData
		if err := json.Unmarshal(dayData, &data.DayData); err != nil {
			log.Printf("解析日数据失败: %v", err)
			// 若解析失败，初始化 data.DayData 为空映射
			data.DayData = make(map[string]int)
		}

		// 返回 200 状态码和获取到的发射数据
		c.JSON(http.StatusOK, gin.H{
			"user_id":     data.UserID,
			"total":       data.Total,
			"year_data":   data.YearData,
			"month_data":  data.MonthData,
			"day_data":    data.DayData,
			"last_launch": data.LastLaunch,
		})
	}
}

// PostSyncDataHandler 返回一个 Gin 处理函数，用于处理用户提交同步数据的请求。
// 参数 db 是数据库连接，用于执行数据库更新操作。
func PostSyncDataHandler(db *sql.DB, config *models.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 Gin 上下文获取用户 ID
        userID := c.GetInt("user_id")
        // 记录日志，表明指定用户正在提交同步数据
        log.Printf("用户 %d 提交同步数据", userID)

        // 使用自定义结构体解析 JSON
        // 定义一个临时结构体，用于接收客户端发送的 JSON 数据
        var req struct {
            UserID     int             `json:"user_id"` // 用户 ID
            Total      int             `json:"total"` // 总发射次数
            YearData   map[string]int  `json:"year_data"` // 年度发射数据
            MonthData  map[string]int  `json:"month_data"` // 月度发射数据
            DayData    map[string]int  `json:"day_data"` // 日度发射数据
            LastLaunch string          `json:"last_launch"` // 最后一次发射时间，字符串格式
        }

        // 尝试将请求体中的 JSON 数据绑定到 req 结构体
        if err := c.ShouldBindJSON(&req); err != nil {
            // 若绑定失败，记录错误日志并返回 400 状态码和错误信息
            log.Printf("解析请求体失败: %v", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
            return
        }

        // 手动解析时间
        // 将客户端发送的最后一次发射时间字符串按照 RFC3339 格式解析为 time.Time 类型
        lastLaunch, err := time.Parse(time.RFC3339, req.LastLaunch)
        if err != nil {
            // 若解析失败，记录错误日志并返回 400 状态码和错误信息
            log.Printf("解析时间失败: %v", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "无效的时间格式"})
            return
        }

        // 创建 LaunchData 结构体
        // 将解析后的数据封装到 models.LaunchData 结构体中
        data := models.LaunchData{
            UserID:     req.UserID,
            Total:      req.Total,
            YearData:   req.YearData,
            MonthData:  req.MonthData,
            DayData:    req.DayData,
            LastLaunch: lastLaunch,
        }

        // 准备JSON数据
        // 将年度、月度和日度发射数据转换为 JSON 字节切片，以便存储到数据库
        yearData, _ := json.Marshal(data.YearData)
        monthData, _ := json.Marshal(data.MonthData)
        dayData, _ := json.Marshal(data.DayData)

        // 更新数据库
        // 执行 SQL 更新语句，将用户提交的同步数据更新到 launch_data 表中
        _, err = db.Exec(`
            UPDATE launch_data
            SET total = ?, 
                year_data = ?, 
                month_data = ?, 
                day_data = ?, 
                last_launch = ?
            WHERE user_id = ?
        `, data.Total, yearData, monthData, dayData, data.LastLaunch, userID)

        if err != nil {
            // 若更新失败，记录错误日志并返回 500 状态码和错误信息
            log.Printf("更新数据失败: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据失败"})
            return
        }

        // 记录日志，表明指定用户的数据同步成功
        if config.Env == "dev" {
            log.Printf("用户 %d 数据同步成功", userID)
        }
        // 向该用户的所有客户端广播更新后的数据
        broadcastToUser(userID, data, config)
        // 返回 200 状态码和成功信息
        c.JSON(http.StatusOK, gin.H{"message": "数据同步成功"})
    }
}

// broadcastToUser 函数用于向指定用户的所有客户端广播发射数据。
// 参数 userID 是目标用户的 ID，data 是需要广播的发射数据。
func broadcastToUser(userID int, data models.LaunchData, config *models.Config) {
	// 对客户端列表加读锁，防止在遍历过程中客户端列表被修改
	models.ClientsLock.RLock()
	// 函数结束时释放读锁
	defer models.ClientsLock.RUnlock()

	// 从客户端映射中获取指定用户的所有客户端
	userClients, ok := models.Clients[userID]
	// 若该用户没有客户端连接，则直接返回
	if !ok {
		return
	}

	// 遍历该用户的所有客户端
	for _, client := range userClients {
		select {
		// 尝试将数据发送到客户端的 Send 通道
		case client.Send <- data:
			if config.Env == "dev" {
				log.Printf("成功向用户 %d 推送数据", userID)
			}
		default:
			// 若通道已满，无法发送数据，记录日志
			log.Printf("用户 %d 的通道已满，准备关闭连接", userID)
			// 启动一个 goroutine 来注销该客户端连接
			go unregisterClient(client, config)
		}
	}
}