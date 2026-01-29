package funds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cryptoSelect/fundsTask/auth"
	"github.com/cryptoSelect/fundsTask/utils"
	"github.com/cryptoSelect/fundsTask/utils/logger"

	"github.com/cryptoSelect/public/database"
	publicModels "github.com/cryptoSelect/public/models"
)

const (
	TradeInflowURL = "https://api.valuescan.io/api/trade/getCoinTradeInflow"
)

// TradeInflowResponse 资金流向查询响应
type TradeInflowResponse struct {
	Code     int         `json:"code"`
	Data     interface{} `json:"data"`
	Msg      string      `json:"msg"`
	ReqID    string      `json:"reqId"`
	UserRole string      `json:"userRole"`
}

// TradeInflowData 资金流向数据结构
type TradeInflowData struct {
	Total  int               `json:"total"`
	List   []TradeInflowInfo `json:"list"`
	Extend string            `json:"extend"`
}

// TradeInflowInfo 资金流向信息
type TradeInflowInfo struct {
	Symbol                    string  `json:"symbol"`
	TimeParticleEnum          int     `json:"timeParticleEnum"`
	Time                      string  `json:"time"`
	Stop                      bool    `json:"stop"`
	StopTradeInflow           float64 `json:"stopTradeInflow"`
	StopTradeAmount           float64 `json:"stopTradeAmount"`
	StopTradeInflowChange     float64 `json:"stopTradeInflowChange"`
	StopTradeAmountChange     float64 `json:"stopTradeAmountChange"`
	Contract                  bool    `json:"contract"`
	ContractTradeInflow       float64 `json:"contractTradeInflow"`
	ContractTradeAmount       float64 `json:"contractTradeAmount"`
	ContractTradeInflowChange float64 `json:"contractTradeInflowChange"`
	ContractTradeAmountChange float64 `json:"contractTradeAmountChange"`
	StopTradeIn               float64 `json:"stopTradeIn"`
	StopTradeOut              float64 `json:"stopTradeOut"`
	ContractTradeIn           float64 `json:"contractTradeIn"`
	ContractTradeOut          float64 `json:"contractTradeOut"`
}

// TradeInflowService 资金流向服务
type TradeInflowService struct {
	client *http.Client
}

// NewTradeInflowService 创建资金流向服务
func NewTradeInflowService() *TradeInflowService {
	return &TradeInflowService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTradeInflow 获取资金流向数据
func (s *TradeInflowService) GetTradeInflow(accessToken, vsTokenID string) (*TradeInflowResponse, error) {
	// 验证 Token 有效性
	authService := auth.NewAuthService()
	validToken, err := authService.ValidateAndRefreshToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate/refresh token: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s?keyword=%s", TradeInflowURL, vsTokenID)

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accessToken", validToken)

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var tradeInflowResp TradeInflowResponse
	err = json.Unmarshal(body, &tradeInflowResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.Log.Info("Trade inflow response received", map[string]interface{}{
		"vs_token_id": vsTokenID,
		"code":        tradeInflowResp.Code,
		"req_id":      tradeInflowResp.ReqID,
	})

	return &tradeInflowResp, nil
}

// StartTradeInflowTask 启动资金流向定时任务
func StartTradeInflowTask(tokenPair *auth.TokenPair) {
	logger.Log.Info("Starting trade inflow task", nil)

	// 创建资金流向服务
	tradeInflowService := NewTradeInflowService()

	// 启动定时器
	go runTradeInflowTimer(tradeInflowService, tokenPair.AccountToken)
}

// runTradeInflowTimer 运行资金流向定时器
func runTradeInflowTimer(service *TradeInflowService, accessToken string) {
	for {
		// 根据配置模式决定是否延时
		if utils.ShouldDelay() {
			utils.WaitForNext5MinuteMark()
		}

		// 执行任务
		processTradeInflow(service, accessToken)
	}
}

// processTradeInflow 处理资金流向数据
func processTradeInflow(service *TradeInflowService, accessToken string) {
	logger.Log.Info("Processing trade inflow data", nil)

	// 查询数据库中所有的 VSTokenID
	vsTokenIDs, err := getVSTokenIDsFromDB()
	if err != nil {
		logger.Log.Error("Failed to get VSTokenIDs from database", map[string]interface{}{"error": err})
		return
	}

	logger.Log.Info("Found VSTokenIDs in database", map[string]interface{}{
		"count": len(vsTokenIDs),
	})

	// 遍历每个 VSTokenID 查询资金流向
	successCount := 0
	currentToken := accessToken // 跟踪当前有效的 Token

	for _, vsTokenID := range vsTokenIDs {
		if err := queryAndSaveTradeInflow(service, &currentToken, vsTokenID); err != nil {
			logger.Log.Error("Failed to process trade inflow for token", map[string]interface{}{
				"vs_token_id": vsTokenID,
				"error":       err,
			})
			continue
		}
		successCount++
	}

	logger.Log.Info("Trade inflow processing completed", map[string]interface{}{
		"total":   len(vsTokenIDs),
		"success": successCount,
		"failed":  len(vsTokenIDs) - successCount,
	})
}

// getVSTokenIDsFromDB 从数据库获取所有 VSTokenID
func getVSTokenIDsFromDB() ([]int64, error) {
	var vsTokenIDs []int64

	// 查询所有 VSTokenID
	err := database.DB.Model(&publicModels.VsCoinInfo{}).
		Pluck("vs_token_id", &vsTokenIDs).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to query VSTokenIDs: %w", err)
	}

	return vsTokenIDs, nil
}

// queryAndSaveTradeInflow 查询并保存资金流向数据
func queryAndSaveTradeInflow(service *TradeInflowService, accessToken *string, vsTokenID int64) error {
	// 转换 VSTokenID 为字符串
	vsTokenIDStr := strconv.FormatInt(vsTokenID, 10)

	// 查询资金流向数据（会自动验证和刷新 Token）
	resp, err := service.GetTradeInflow(*accessToken, vsTokenIDStr)
	if err != nil {
		return fmt.Errorf("failed to get trade inflow: %w", err)
	}

	// 检查响应状态
	if resp.Code != 200 {
		return fmt.Errorf("invalid response code: %d, msg: %s", resp.Code, resp.Msg)
	}

	// 解析数据
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	var tradeData map[string]interface{}
	err = json.Unmarshal(dataBytes, &tradeData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal trade data: %w", err)
	}

	// 调试：打印完整的响应数据结构
	logger.Log.Info("Trade inflow API response structure", map[string]interface{}{
		"vs_token_id":   vsTokenID,
		"response_data": tradeData,
	})

	// 检查数据结构
	if tradeData["coinTradeInflowDtoList"] == nil {
		logger.Log.Info("No trade inflow data found", map[string]interface{}{
			"vs_token_id":    vsTokenID,
			"available_keys": getMapKeys(tradeData),
		})
		return nil
	}

	// 处理资金流向列表
	tradeList := tradeData["coinTradeInflowDtoList"].([]interface{})
	symbol := getString(tradeData["symbol"]) // 从外层获取 symbol

	logger.Log.Info("Found trade inflow data", map[string]interface{}{
		"vs_token_id": vsTokenID,
		"symbol":      symbol,
		"list_length": len(tradeList),
	})

	return saveTradeInflowToDB(tradeList, symbol)
}

// getMapKeys 获取 map 的所有键，用于调试
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// saveTradeInflowToDB 保存资金流向数据到数据库
func saveTradeInflowToDB(tradeList []interface{}, symbol string) error {
	for _, trade := range tradeList {
		tradeMap := trade.(map[string]interface{})

		// 创建 CoinTradeInflowDto 记录
		tradeInflow := publicModels.CoinTradeInflowDto{
			Symbol:                    symbol, // 从外层获取 symbol
			TimeParticleEnum:          getInt(tradeMap["timeParticleEnum"]),
			Time:                      getString(tradeMap["time"]),
			Stop:                      getBool(tradeMap["stop"]),
			StopTradeInflow:           getFloat64(tradeMap["stopTradeInflow"]),
			StopTradeAmount:           getFloat64(tradeMap["stopTradeAmount"]),
			StopTradeInflowChange:     getFloat64(tradeMap["stopTradeInflowChange"]),
			StopTradeAmountChange:     getFloat64(tradeMap["stopTradeAmountChange"]),
			Contract:                  getBool(tradeMap["contract"]),
			ContractTradeInflow:       getFloat64(tradeMap["contractTradeInflow"]),
			ContractTradeAmount:       getFloat64(tradeMap["contractTradeAmount"]),
			ContractTradeInflowChange: getFloat64(tradeMap["contractTradeInflowChange"]),
			ContractTradeAmountChange: getFloat64(tradeMap["contractTradeAmountChange"]),
			StopTradeIn:               getFloat64(tradeMap["stopTradeIn"]),
			StopTradeOut:              getFloat64(tradeMap["stopTradeOut"]),
			ContractTradeIn:           getFloat64(tradeMap["contractTradeIn"]),
			ContractTradeOut:          getFloat64(tradeMap["contractTradeOut"]),
		}

		// 保存到数据库（使用 Upsert 方式）
		result := database.DB.Where("symbol = ? AND time = ?", tradeInflow.Symbol, tradeInflow.Time).
			Assign(&tradeInflow).
			FirstOrCreate(&tradeInflow)

		if result.Error != nil {
			logger.Log.Error("Failed to save trade inflow record", map[string]interface{}{
				"symbol": tradeInflow.Symbol,
				"time":   tradeInflow.Time,
				"error":  result.Error,
			})
			continue
		}

		logger.Log.Debug("Trade inflow saved", map[string]interface{}{
			"symbol": tradeInflow.Symbol,
			"time":   tradeInflow.Time,
		})
	}

	return nil
}

// 辅助函数：安全地从 interface{} 获取各种类型的数据
func getString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func getInt(value interface{}) int {
	if value == nil {
		return 0
	}
	if num, ok := value.(float64); ok {
		return int(num)
	}
	if num, ok := value.(int); ok {
		return num
	}
	if str, ok := value.(string); ok {
		if i, err := strconv.Atoi(str); err == nil {
			return i
		}
	}
	return 0
}

func getFloat64(value interface{}) float64 {
	if value == nil {
		return 0
	}
	if num, ok := value.(float64); ok {
		return num
	}
	if num, ok := value.(int); ok {
		return float64(num)
	}
	if str, ok := value.(string); ok {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f
		}
	}
	return 0
}

func getBool(value interface{}) bool {
	if value == nil {
		return false
	}
	if b, ok := value.(bool); ok {
		return b
	}
	if str, ok := value.(string); ok {
		if b, err := strconv.ParseBool(str); err == nil {
			return b
		}
	}
	return false
}
