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
// StartCLI 启动后端管理控制台的命令行界面，允许管理员执行用户管理等操作。
// 参数 db 是数据库连接，用于执行与用户相关的数据库操作。
// 参数 clients 是指向在线客户端映射的指针，键为用户 ID，值为客户端实例切片。
// 参数 lock 是读写锁，用于保证对在线客户端映射的并发安全访问。
func StartCLI(db *sql.DB, clients *map[int][]*models.Client, lock *sync.RWMutex) {
	// 创建一个新的扫描器，用于从标准输入读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	// 打印启动信息，提示用户输入 'help' 查看可用命令
	fmt.Println("后端管理控制台已启动 (输入 'help' 查看命令)")

	// 进入无限循环，持续等待用户输入命令
	for {
		// 打印命令提示符
		fmt.Print("> ")
		// 尝试从标准输入读取一行内容，如果读取失败则退出循环
		if !scanner.Scan() {
			break
		}

		// 去除输入内容两端的空白字符
		input := strings.TrimSpace(scanner.Text())
		// 如果输入为空，则跳过本次循环，继续等待下一次输入
		if input == "" {
			continue
		}

		// 将输入内容按空白字符分割成多个部分
		parts := strings.Fields(input)
		// 获取输入的第一个部分作为命令
		command := parts[0]

		// 根据不同的命令执行相应的操作
		switch command {
		case "exit", "quit":
			// 打印退出信息并返回，结束命令行界面
			fmt.Println("退出管理控制台")
			return
		case "help":
			// 调用 printHelp 函数显示帮助信息
			printHelp()
		case "list":
			// 调用 listUsers 函数列出所有用户
			listUsers(db)
		case "create":
			// 检查输入参数是否足够
			if len(parts) < 3 {
				// 若参数不足，打印使用说明
				fmt.Println("用法: create <用户名> <密码>")
			} else {
				// 调用 createUser 函数创建新用户
				createUser(db, parts[1], parts[2])
			}
		case "delete":
			// 检查输入参数是否足够
			if len(parts) < 2 {
				// 若参数不足，打印使用说明
				fmt.Println("用法: delete <用户名>")
			} else {
				// 调用 deleteUser 函数删除指定用户
				deleteUser(db, parts[1])
			}
		case "passwd":
			// 检查输入参数是否足够
			if len(parts) < 3 {
				// 若参数不足，打印使用说明
				fmt.Println("用法: passwd <用户名> <新密码>")
			} else {
				// 调用 changePassword 函数更改指定用户的密码
				changePassword(db, parts[1], parts[2])
			}
		case "online":
			// 调用 showOnlineUsers 函数显示当前在线用户
			showOnlineUsers(clients, lock)
		case "clients":
			// 检查输入参数是否足够
			if len(parts) < 2 {
				// 若参数不足，打印使用说明
				fmt.Println("用法: clients <用户名>")
			} else {
				// 调用 showUserClients 函数显示指定用户的在线客户端
				showUserClients(db, parts[1], clients, lock)
			}
		default:
			// 若输入的命令未知，提示用户输入 'help' 查看可用命令
			fmt.Println("未知命令，输入 'help' 查看可用命令")
		}
	}
}
// printHelp 函数用于打印后端管理控制台的可用命令列表，帮助用户了解控制台支持的操作。
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

// listUsers 函数用于从数据库中查询所有用户信息，并将其打印输出。
// 参数 db 是数据库连接，用于执行 SQL 查询语句。
func listUsers(db *sql.DB) {
	// 执行 SQL 查询语句，从 users 表中选取用户的 ID 和用户名
	rows, err := db.Query("SELECT id, username FROM users")
	// 检查查询是否出错
	if err != nil {
		// 若出错，记录错误日志并返回，终止函数执行
		log.Println("查询用户失败:", err)
		return
	}
	// 确保在函数结束时关闭查询结果集，释放资源
	defer rows.Close()

	// 打印用户列表标题
	fmt.Println("用户列表:")
	// 打印表头，包含 ID 和用户名两列
	fmt.Println("ID\t用户名")
	// 遍历查询结果集的每一行
	for rows.Next() {
		// 定义变量用于存储当前行的用户 ID 和用户名
		var id int
		var username string
		// 将当前行的数据扫描到定义的变量中
		if err := rows.Scan(&id, &username); err != nil {
			// 若扫描出错，记录错误日志并跳过当前行，继续处理下一行
			log.Println("读取用户失败:", err)
			continue
		}
		// 打印当前行的用户 ID 和用户名
		fmt.Printf("%d\t%s\n", id, username)
	}
}

// createUser 函数用于在数据库中创建新用户。
// 参数 db 是数据库连接，用于执行 SQL 语句。
// 参数 username 是要创建的用户的用户名。
// 参数 password 是要创建的用户的密码。
func createUser(db *sql.DB, username, password string) {
	// 检查用户名是否已存在
	// 使用 EXISTS 子查询判断 users 表中是否已存在该用户名
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		// 若查询出错，打印错误信息并返回，终止用户创建流程
		fmt.Println("检查用户失败:", err)
		return
	}

	if exists {
		// 若用户名已存在，打印错误信息并返回，终止用户创建流程
		fmt.Println("错误: 用户名已存在")
		return
	}

	// 密码哈希
	// 使用 bcrypt 算法对用户输入的密码进行哈希处理，使用默认的计算成本
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// 若密码哈希失败，打印错误信息并返回，终止用户创建流程
		fmt.Println("密码加密失败:", err)
		return
	}

	// 创建用户
	// 将用户名和哈希后的密码插入到 users 表中
	result, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		// 若插入用户信息失败，打印错误信息并返回，终止用户创建流程
		fmt.Println("创建用户失败:", err)
		return
	}

	// 获取新创建用户的 ID
	userID, err := result.LastInsertId()
	if err != nil {
		// 若获取用户 ID 失败，打印错误信息并返回，终止用户创建流程
		fmt.Println("获取用户ID失败:", err)
		return
	}

	// 创建初始发射数据
	// 为新用户在 launch_data 表中创建一条初始记录，设置总发射次数为 0，各时间段发射数据为空 JSON 对象，最后发射时间为 NULL
	_, err = db.Exec(`
		INSERT INTO launch_data (user_id, total, year_data, month_data, day_data, last_launch)
		VALUES (?, 0, '{}', '{}', '{}', NULL)
	`, userID)
	if err != nil {
		// 若创建发射数据失败，打印错误信息并返回，终止用户创建流程
		fmt.Println("创建发射数据失败:", err)
		return
	}

	// 打印用户创建成功信息，包含用户名和用户 ID
	fmt.Printf("用户 %s 创建成功, ID: %d\n", username, userID)
}

// deleteUser 函数用于从数据库中删除指定用户名的用户。
// 参数 db 是数据库连接，用于执行 SQL 语句。
// 参数 username 是要删除的用户的用户名。
func deleteUser(db *sql.DB, username string) {
	// 获取用户ID
	// 执行 SQL 查询，根据用户名从 users 表中获取对应的用户 ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		// 检查是否因为用户不存在而导致查询失败
		if err == sql.ErrNoRows {
			// 若用户不存在，打印错误信息并返回，终止删除流程
			fmt.Println("错误: 用户不存在")
			return
		}
		// 若出现其他查询错误，打印错误信息并返回，终止删除流程
		fmt.Println("查询用户失败:", err)
		return
	}

	// 删除用户
	// 执行 SQL 删除语句，根据用户 ID 从 users 表中删除对应的用户记录
	_, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		// 若删除操作失败，打印错误信息并返回，终止删除流程
		fmt.Println("删除用户失败:", err)
		return
	}

	// 打印用户删除成功信息，包含用户名和用户 ID
	fmt.Printf("用户 %s (ID: %d) 已删除\n", username, userID)
}

// changePassword 函数用于更改指定用户的密码。
// 参数 db 是数据库连接，用于执行 SQL 语句。
// 参数 username 是要更改密码的用户的用户名。
// 参数 newPassword 是用户的新密码。
func changePassword(db *sql.DB, username, newPassword string) {
	// 获取用户ID
	// 执行 SQL 查询，根据用户名从 users 表中获取对应的用户 ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		// 检查是否因为用户不存在而导致查询失败
		if err == sql.ErrNoRows {
			// 若用户不存在，打印错误信息并返回，终止密码更改流程
			fmt.Println("错误: 用户不存在")
			return
		}
		// 若出现其他查询错误，打印错误信息并返回，终止密码更改流程
		fmt.Println("查询用户失败:", err)
		return
	}

	// 密码哈希
	// 使用 bcrypt 算法对用户输入的新密码进行哈希处理，使用默认的计算成本
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		// 若密码哈希失败，打印错误信息并返回，终止密码更改流程
		fmt.Println("密码加密失败:", err)
		return
	}

	// 更新密码
	// 执行 SQL 更新语句，将指定用户的密码哈希值更新为新生成的哈希值
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hashedPassword, userID)
	if err != nil {
		// 若更新操作失败，打印错误信息并返回，终止密码更改流程
		fmt.Println("更新密码失败:", err)
		return
	}

	// 打印密码更新成功信息，包含用户名和用户 ID
	fmt.Printf("用户 %s (ID: %d) 密码已更新\n", username, userID)
}

// showOnlineUsers 函数用于显示当前在线用户及其对应的客户端数量。
// 参数 clients 是指向在线客户端映射的指针，键为用户 ID，值为客户端实例切片。
// 参数 lock 是读写锁，用于保证对在线客户端映射的并发安全访问。
func showOnlineUsers(clients *map[int][]*models.Client, lock *sync.RWMutex) {
	// 加读锁，防止在读取在线客户端信息时，其他协程对客户端映射进行写操作
	lock.RLock()
	// 函数返回时自动释放读锁，确保资源正确释放
	defer lock.RUnlock()

	// 检查在线客户端映射是否为空
	if len(*clients) == 0 {
		// 若为空，打印提示信息并返回
		fmt.Println("当前没有在线用户")
		return
	}

	// 打印在线用户列表标题
	fmt.Println("在线用户:")
	// 打印表头，包含用户 ID 和客户端数量两列
	fmt.Println("用户ID\t客户端数量")
	// 遍历在线客户端映射
	for userID, clientList := range *clients {
		// 打印每个用户的 ID 及其对应的客户端数量
		fmt.Printf("%d\t%d\n", userID, len(clientList))
	}
}

// showUserClients 函数用于显示指定用户的在线客户端信息。
// 参数 db 是数据库连接，用于查询用户信息。
// 参数 username 是要查询的用户的用户名。
// 参数 clients 是指向在线客户端映射的指针，键为用户 ID，值为客户端实例切片。
// 参数 lock 是读写锁，用于保证对在线客户端映射的并发安全访问。
func showUserClients(db *sql.DB, username string, clients *map[int][]*models.Client, lock *sync.RWMutex) {
	// 获取用户ID
	// 执行 SQL 查询，根据用户名从 users 表中获取对应的用户 ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		// 检查是否因为用户不存在而导致查询失败
		if err == sql.ErrNoRows {
			// 若用户不存在，打印错误信息并返回，终止查询流程
			fmt.Println("错误: 用户不存在")
			return
		}
		// 若出现其他查询错误，打印错误信息并返回，终止查询流程
		fmt.Println("查询用户失败:", err)
		return
	}

	// 加读锁，防止在读取在线客户端信息时，其他协程对客户端映射进行写操作
	lock.RLock()
	// 函数返回时自动释放读锁，确保资源正确释放
	defer lock.RUnlock()

	// 从在线客户端映射中获取指定用户 ID 对应的客户端列表
	clientList, exists := (*clients)[userID]
	// 检查该用户是否有在线客户端
	if !exists || len(clientList) == 0 {
		// 若没有在线客户端，打印提示信息并返回
		fmt.Printf("用户 %s 没有在线客户端\n", username)
		return
	}

	// 打印指定用户的在线客户端信息标题，包含用户名和用户 ID
	fmt.Printf("用户 %s (ID: %d) 的在线客户端:\n", username, userID)
	// 打印表头，包含 IP 地址、连接时间和连接时长三列
	fmt.Println("IP地址\t\t连接时间\t\t\t连接时长")
	// 遍历该用户的在线客户端列表
	for _, client := range clientList {
		// 计算客户端的连接时长，四舍五入到秒
		duration := time.Since(client.ConnectAt).Round(time.Second)
		// 打印每个客户端的 IP 地址、连接时间和连接时长
		fmt.Printf("%s\t%s\t%s\n", 
			client.IP, 
			client.ConnectAt.Format("2006-01-02 15:04:05"),
			duration)
	}
}