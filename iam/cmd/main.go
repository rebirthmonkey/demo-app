package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	redis "github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"io"
	"log"
	"net"
	"os"
)

func initConfig() {
	// 定义一个命令行参数 -c 用于指定配置文件
	configPath := flag.String("c", "./configs/config.yaml", "config file path")
	flag.Parse()

	// 使用 Viper 读取配置文件
	viper.SetConfigFile(*configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
}

func setupLogger() *logrus.Logger {
	logFilePath := viper.GetString("log.filePath")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logrus.Fatalf("Failed to open log file %s for output: %s", logFilePath, err)
	}

	logger := logrus.New()
	logger.Out = logFile
	logger.Formatter = &logrus.JSONFormatter{} // 设置为JSON格式

	return logger
}

// 连接到 MySQL
func connectMySQL() *sql.DB {
	dbUser := viper.GetString("mysql.user")
	dbPass := viper.GetString("mysql.password")
	dbHost := viper.GetString("mysql.host")
	dbPort := viper.GetString("mysql.port")
	dbName := viper.GetString("mysql.dbname")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return db
}

// 查询 user 表中的所有 user 的 name
func queryUserName(db *sql.DB) []string {
	rows, err := db.Query("SELECT name FROM user")
	if err != nil {
		panic(err)
	}

	var users []string
	for rows.Next() {
		var user string
		err := rows.Scan(&user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}
	return users
}

// 连接到 Redis
func connectRedis() *redis.Client {
	redisAddr := viper.GetString("redis.addr")
	redisPass := viper.GetString("redis.password")
	redisDB := viper.GetInt("redis.db")

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})

	return client
}

// 查询 groupset 集合中的所有 groups
func queryGroups(client *redis.Client) []string {
	var groups []string
	var ctx = context.Background()
	groups, err := client.SMembers(ctx, "groupset").Result()
	if err != nil {
		panic(err)
	}
	return groups
}

func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// 查询 user 表中的特定 user 的 name 和 password
func queryAuth(db *sql.DB, name string, password string) bool {
	row := db.QueryRow("SELECT name FROM user WHERE name = ? AND password = ?", name, password)
	var user string
	err := row.Scan(&user)
	if err != nil {
		if err == sql.ErrNoRows {
			// 没有匹配的行，返回 false
			return false
		}
		panic(err)
	}
	// 找到匹配的行，返回 true
	return true
}

func main() {
	initConfig()

	logger := setupLogger()
	gin.DefaultWriter = io.MultiWriter(logger.Out, os.Stdout) // 使用Logger的输出

	mysqlDB := connectMySQL()
	redisDB := connectRedis()

	// 确保在程序退出前关闭数据库连接
	defer mysqlDB.Close()
	defer redisDB.Close()

	// 设置 Gin 路由
	r := gin.Default()

	//// 配置CORS
	//r.Use(cors.Default())

	r.Use(gin.Logger())

	// 设置Prometheus metrics
	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":    "Hello World!",
			"IP Address": getIPAddress(),
		})
	})

	r.GET("/users", func(c *gin.Context) {
		var users = queryUserName(mysqlDB)

		c.JSON(200, gin.H{
			"users":      users,
			"IP Address": getIPAddress(),
		})
	})

	r.GET("/groups", func(c *gin.Context) {
		var groups = queryGroups(redisDB)

		c.JSON(200, gin.H{
			"groups":     groups,
			"IP Address": getIPAddress(),
		})
	})

	r.GET("/auth", func(c *gin.Context) {
		user := c.Query("user")
		pwd := c.Query("pwd")

		isAuthenticated := queryAuth(mysqlDB, user, pwd)

		c.JSON(200, gin.H{
			"authenticated": isAuthenticated,
		})
	})

	// 启动 Gin 服务
	r.Run("0.0.0.0:8888") // 默认在 0.0.0.0:8080 上启动服务
}
