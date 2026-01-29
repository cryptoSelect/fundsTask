package main

import (
	"github.com/cryptoSelect/fundsTask/auth"
	"github.com/cryptoSelect/fundsTask/config"
	"github.com/cryptoSelect/fundsTask/funds"
	"github.com/cryptoSelect/fundsTask/utils/logger"
	"github.com/cryptoSelect/public/database"
	publicModels "github.com/cryptoSelect/public/models"
)

func main() {
	// 初始化配置
	config.Init()

	// 初始化日志
	logger.Init(config.Cfg.Mode)

	// 初始化数据库
	database.InitDB(
		config.Cfg.Database.Host,
		config.Cfg.Database.User,
		config.Cfg.Database.Password,
		config.Cfg.Database.DBName,
		config.Cfg.Database.Port,
	)

	// 自动迁移数据库表
	err := database.AutoMigrate(
		&publicModels.CoinTradeInflowDto{},
		&publicModels.VsCoinInfo{},
	)
	if err != nil {
		logger.Log.Error("Database migration failed", map[string]interface{}{"error": err})
		return
	}

	logger.Log.Info("Database migration completed successfully")

	logger.Log.Info("Application starting", map[string]interface{}{
		"mode": config.Cfg.Mode,
		"app":  "FundsTask",
	})

	// 创建认证服务
	authService := auth.NewAuthService()

	// 执行登录
	logger.Log.Info("Starting login process")
	tokenPair, err := authService.GetTokens()
	if err != nil {
		logger.Log.Error("Login failed", map[string]interface{}{"error": err})
		return
	}

	// 输出登录结果
	logger.Log.Info("Login successful", map[string]interface{}{
		"token_valid":       tokenPair.IsValid(),
		"refresh_valid":     tokenPair.IsRefreshValid(),
		"account_token_len": len(tokenPair.AccountToken),
		"refresh_token_len": len(tokenPair.RefreshToken),
	})

	// 启动币种信息定时任务
	go funds.StartTask(nil)

	// 启动资金流向定时任务
	go funds.StartTradeInflowTask(tokenPair)

	// 保持主程序运行
	select {}

}
