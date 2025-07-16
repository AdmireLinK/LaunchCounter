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

func GetSyncDataHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("user_id")
		log.Printf("获取用户 %d 的同步数据", userID)

		var data models.LaunchData
		var yearData, monthData, dayData []byte
		var lastLaunch sql.NullTime

		// 添加更健壮的查询逻辑
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

		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("用户 %d 未找到发射数据", userID)
				// 创建初始数据
				_, createErr := db.Exec(`
					INSERT INTO launch_data 
					(user_id, total, year_data, month_data, day_data, last_launch)
					VALUES (?, 0, '{}', '{}', '{}', NULL)
				`, userID)
				
				if createErr != nil {
					log.Printf("创建初始数据失败: %v", createErr)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据失败"})
					return
				}
				
				// 返回空数据
				data = models.LaunchData{
					Total:      0,
					YearData:   make(map[string]int),
					MonthData:  make(map[string]int),
					DayData:    make(map[string]int),
					LastLaunch: time.Time{},
				}
				c.JSON(http.StatusOK, gin.H{
			"user_id": data.UserID,
			"total": data.Total,
			"year_data": data.YearData,
			"month_data": data.MonthData,
			"day_data": data.DayData,
			"last_launch": data.LastLaunch,

		})
				return
			}
			
			log.Printf("数据库查询失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败"})
			return
		}

		// 处理时间字段
		if lastLaunch.Valid {
			data.LastLaunch = lastLaunch.Time
		} else {
			data.LastLaunch = time.Time{}
		}

		// 解析JSON数据
		if err := json.Unmarshal(yearData, &data.YearData); err != nil {
			log.Printf("解析年度数据失败: %v", err)
			data.YearData = make(map[string]int)
		}
		if err := json.Unmarshal(monthData, &data.MonthData); err != nil {
			log.Printf("解析月度数据失败: %v", err)
			data.MonthData = make(map[string]int)
		}
		if err := json.Unmarshal(dayData, &data.DayData); err != nil {
			log.Printf("解析日数据失败: %v", err)
			data.DayData = make(map[string]int)
		}

		log.Printf("成功获取用户 %d 的同步数据", userID)
		c.JSON(http.StatusOK, gin.H{
			"user_id": data.UserID,
			"total": data.Total,
			"year_data": data.YearData,
			"month_data": data.MonthData,
			"day_data": data.DayData,
			"last_launch": data.LastLaunch,
		})
	}
}

func PostSyncDataHandler(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetInt("user_id")
        log.Printf("用户 %d 提交同步数据", userID)

        // 使用自定义结构体解析 JSON
        var req struct {
            UserID     int             `json:"user_id"`
            Total      int             `json:"total"`
            YearData   map[string]int  `json:"year_data"`
            MonthData  map[string]int  `json:"month_data"`
            DayData    map[string]int  `json:"day_data"`
            LastLaunch string          `json:"last_launch"`
        }

        if err := c.ShouldBindJSON(&req); err != nil {
            log.Printf("解析请求体失败: %v", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
            return
        }

        // 手动解析时间
        lastLaunch, err := time.Parse(time.RFC3339, req.LastLaunch)
        if err != nil {
            log.Printf("解析时间失败: %v", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "无效的时间格式"})
            return
        }

        // 创建 LaunchData 结构体
        data := models.LaunchData{
            UserID:     req.UserID,
            Total:      req.Total,
            YearData:   req.YearData,
            MonthData:  req.MonthData,
            DayData:    req.DayData,
            LastLaunch: lastLaunch,
        }

        // 准备JSON数据
        yearData, _ := json.Marshal(data.YearData)
        monthData, _ := json.Marshal(data.MonthData)
        dayData, _ := json.Marshal(data.DayData)

        // 更新数据库
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
            log.Printf("更新数据失败: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据失败"})
            return
        }

        log.Printf("用户 %d 数据同步成功", userID)
        broadcastToUser(userID, data)
        c.JSON(http.StatusOK, gin.H{"message": "数据同步成功"})
    }
}

func broadcastToUser(userID int, data models.LaunchData) {
	models.ClientsLock.RLock()
	defer models.ClientsLock.RUnlock()

	userClients, ok := models.Clients[userID]
	if !ok {
		return
	}

	for _, client := range userClients {
		select {
		case client.Send <- data:
			log.Printf("成功向用户 %d 推送数据", userID)
		default:
			log.Printf("用户 %d 的通道已满，准备关闭连接", userID)
			// 通道满时关闭连接
			go unregisterClient(client)
		}
	}
}