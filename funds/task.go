package funds

import (
	"encoding/json"
	"strconv"

	"github.com/cryptoSelect/fundsTask/auth"
	"github.com/cryptoSelect/fundsTask/utils"
	"github.com/cryptoSelect/fundsTask/utils/logger"

	"github.com/cryptoSelect/public/database"
	publicModels "github.com/cryptoSelect/public/models"
)

func StartTask(_ *auth.TokenPair) {
	logger.Log.Info("Starting coin info task with 4-hour interval", nil)

	// 创建币种服务
	coinService := NewCoinService()

	// 启动定时器
	go runCoinInfoTimer(coinService)
}

// runCoinInfoTimer 运行币种信息定时器
func runCoinInfoTimer(service *CoinService) {
	for {
		// 根据配置模式决定是否延时
		if utils.ShouldDelay() {
			utils.WaitForNext4HourMark()
		}

		// 执行任务
		processCoinInfoTask(service)
	}
}

// processCoinInfoTask 处理币种信息任务
func processCoinInfoTask(service *CoinService) {
	logger.Log.Info("Processing coin info task", nil)

	// 创建认证服务并获取有效 Token
	authService := auth.NewAuthService()
	tokenPair, err := authService.GetTokens()
	if err != nil {
		logger.Log.Error("Failed to get tokens for coin info task", map[string]interface{}{"error": err})
		return
	}

	// 查询币种信息
	coinService := NewCoinService()
	coinResp, err := coinService.QueryCoins(tokenPair.AccountToken)
	if err != nil {
		logger.Log.Error("Coin query failed", map[string]interface{}{"error": err})
		return
	}

	logger.Log.Info("Coin query successful", map[string]interface{}{
		"code":   coinResp.Code,
		"req_id": coinResp.ReqID,
	})

	// 解析响应数据
	if coinResp.Code != 200 {
		logger.Log.Error("Invalid response code", map[string]interface{}{
			"code": coinResp.Code,
			"msg":  coinResp.Msg,
		})
		return
	}

	// 将 interface{} 转换为 CoinData
	dataBytes, err := json.Marshal(coinResp.Data)
	if err != nil {
		logger.Log.Error("Failed to marshal response data", map[string]interface{}{"error": err})
		return
	}

	var coinData map[string]interface{}
	err = json.Unmarshal(dataBytes, &coinData)
	if err != nil {
		logger.Log.Error("Failed to unmarshal coin data", map[string]interface{}{"error": err})
		return
	}

	logger.Log.Info("Parsed coin data", map[string]interface{}{
		"total": coinData["total"],
		"count": len(coinData["list"].([]interface{})),
	})

	// 保存到数据库
	if err := saveCoinInfoToDB(coinData["list"].([]interface{})); err != nil {
		logger.Log.Error("Failed to save coin info to database", map[string]interface{}{"error": err})
		return
	}

	logger.Log.Info("Coin info task completed successfully", map[string]interface{}{
		"count": len(coinData["list"].([]interface{})),
	})
}

// saveCoinInfoToDB 保存币种信息到数据库
func saveCoinInfoToDB(coinList []interface{}) error {
	for _, coin := range coinList {
		coinMap := coin.(map[string]interface{})
		// 解析 VSTokenID
		vsTokenID, err := strconv.ParseInt(coinMap["vsTokenId"].(string), 10, 64)
		if err != nil {
			logger.Log.Error("Failed to parse VSTokenID", map[string]interface{}{
				"vsTokenId": coinMap["vsTokenId"],
				"error":     err,
			})
			continue
		}

		// 解析 MarketCap
		marketCap, err := strconv.ParseFloat(coinMap["marketCap"].(string), 64)
		if err != nil {
			logger.Log.Error("Failed to parse MarketCap", map[string]interface{}{
				"symbol":    coinMap["symbol"],
				"marketCap": coinMap["marketCap"],
				"error":     err,
			})
			continue
		}

		// 创建 VsCoinInfo 记录
		vsCoinInfo := publicModels.VsCoinInfo{
			VSTokenID: vsTokenID,
			Name:      coinMap["name"].(string),
			Symbol:    coinMap["symbol"].(string),
			MarketCap: marketCap,
		}

		// 使用 Upsert 方式保存（如果存在则更新，不存在则创建）
		result := database.DB.Where("vs_token_id = ?", vsTokenID).
			Assign(&publicModels.VsCoinInfo{
				Name:      coinMap["name"].(string),
				Symbol:    coinMap["symbol"].(string),
				MarketCap: marketCap,
			}).
			FirstOrCreate(&vsCoinInfo)

		if result.Error != nil {
			logger.Log.Error("Failed to save coin record", map[string]interface{}{
				"symbol": coinMap["symbol"],
				"error":  result.Error,
			})
			continue
		}

		logger.Log.Debug("Coin info saved", map[string]interface{}{
			"vs_token_id": vsTokenID,
			"symbol":      coinMap["symbol"],
			"name":        coinMap["name"],
			"market_cap":  marketCap,
		})
	}

	return nil
}
