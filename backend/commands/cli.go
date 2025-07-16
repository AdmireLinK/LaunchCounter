package commands

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"backend/models"
	"golang.org/x/crypto/bcrypt"
)

// 启动命令行界面
func StartCLI(db *sql.DB, clients *map[int][]*models.Client, lock *sync.RWMutex) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("后端管理控制台已启动 (输入 'help' 查看命令)")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "exit", "quit":
			fmt.Println("退出管理控制台")
			return
		case "help":
			printHelp()
		case "list":
			listUsers(db)
		case "create":
			if len(parts) < 3 {
				fmt.Println("用法: create <用户名> <密码>")
			} else {
				createUser(db, parts[1], parts[2])
			}
		case "delete":
			if len(parts) < 2 {
				fmt.Println("用法: delete <用户名>")
			} else {
				deleteUser(db, parts[1])
			}
		case "passwd":
			if len(parts) < 3 {
				fmt.Println("用法: passwd <用户名> <新密码>")
			} else {
				changePassword(db, parts[1], parts[2])
			}
		case "online":
			showOnlineUsers(clients, lock)
		case "clients":
			if len(parts) < 2 {
				fmt.Println("用法: clients <用户名>")
			} else {
				showUserClients(db, parts[1], clients, lock)
			}
		default:
			fmt.Println("未知命令，输入 'help' 查看可用命令")
		}
	}
}

func printHelp() {
	fmt.Println("可用命令:")
	fmt.Println("  help               - 显示帮助信息")
	fmt.Println("  list               - 列出所有用户")
	fmt.Println("  create <user> <pw> - 创建新用户")
	fmt.Println("  delete <user>      - 删除用户")
	fmt.Println("  passwd <user> <pw> - 更改用户密码")
	fmt.Println("  online             - 显示在线用户")
	fmt.Println("  clients <user>     - 显示用户在线客户端")
	fmt.Println("  exit               - 退出管理控制台")
}

func listUsers(db *sql.DB) {
	rows, err := db.Query("SELECT id, username FROM users")
	if err != nil {
		log.Println("查询用户失败:", err)
		return
	}
	defer rows.Close()

	fmt.Println("用户列表:")
	fmt.Println("ID\t用户名")
	for rows.Next() {
		var id int
		var username string
		if err := rows.Scan(&id, &username); err != nil {
			log.Println("读取用户失败:", err)
			continue
		}
		fmt.Printf("%d\t%s\n", id, username)
	}
}

func createUser(db *sql.DB, username, password string) {
	// 检查用户名是否已存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		fmt.Println("检查用户失败:", err)
		return
	}

	if exists {
		fmt.Println("错误: 用户名已存在")
		return
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码加密失败:", err)
		return
	}

	// 创建用户
	result, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		fmt.Println("创建用户失败:", err)
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		fmt.Println("获取用户ID失败:", err)
		return
	}

	// 创建初始发射数据
	_, err = db.Exec(`
		INSERT INTO launch_data (user_id, total, year_data, month_data, day_data, last_launch)
		VALUES (?, 0, '{}', '{}', '{}', NULL)
	`, userID)
	if err != nil {
		fmt.Println("创建发射数据失败:", err)
		return
	}

	fmt.Printf("用户 %s 创建成功, ID: %d\n", username, userID)
}

func deleteUser(db *sql.DB, username string) {
	// 获取用户ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("错误: 用户不存在")
			return
		}
		fmt.Println("查询用户失败:", err)
		return
	}

	// 删除用户
	_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		fmt.Println("删除用户失败:", err)
		return
	}

	fmt.Printf("用户 %s (ID: %d) 已删除\n", username, userID)
}

func changePassword(db *sql.DB, username, newPassword string) {
	// 获取用户ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("错误: 用户不存在")
			return
		}
		fmt.Println("查询用户失败:", err)
		return
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("密码加密失败:", err)
		return
	}

	// 更新密码
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hashedPassword, userID)
	if err != nil {
		fmt.Println("更新密码失败:", err)
		return
	}

	fmt.Printf("用户 %s (ID: %d) 密码已更新\n", username, userID)
}

func showOnlineUsers(clients *map[int][]*models.Client, lock *sync.RWMutex) {
	lock.RLock()
	defer lock.RUnlock()

	if len(*clients) == 0 {
		fmt.Println("当前没有在线用户")
		return
	}

	fmt.Println("在线用户:")
	fmt.Println("用户ID\t客户端数量")
	for userID, clientList := range *clients {
		fmt.Printf("%d\t%d\n", userID, len(clientList))
	}
}

func showUserClients(db *sql.DB, username string, clients *map[int][]*models.Client, lock *sync.RWMutex) {
	// 获取用户ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("错误: 用户不存在")
			return
		}
		fmt.Println("查询用户失败:", err)
		return
	}

	lock.RLock()
	defer lock.RUnlock()

	clientList, exists := (*clients)[userID]
	if !exists || len(clientList) == 0 {
		fmt.Printf("用户 %s 没有在线客户端\n", username)
		return
	}

	fmt.Printf("用户 %s (ID: %d) 的在线客户端:\n", username, userID)
	fmt.Println("IP地址\t\t连接时间\t\t\t连接时长")
	for _, client := range clientList {
		duration := time.Since(client.ConnectAt).Round(time.Second)
		fmt.Printf("%s\t%s\t%s\n", 
			client.IP, 
			client.ConnectAt.Format("2006-01-02 15:04:05"),
			duration)
	}
}