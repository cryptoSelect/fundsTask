package funds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cryptoSelect/fundsTask/auth"
	"github.com/cryptoSelect/fundsTask/utils/logger"
)

const (
	CoinQueryURL = "https://api.valuescan.io/api/vs-token/queryCoin"
)

// CoinInfo 币种信息
type CoinInfo struct {
	VSTokenID string `json:"vsTokenId"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	MarketCap string `json:"marketCap"`
}

// CoinQueryResponse 币种查询响应
type CoinQueryResponse struct {
	Code     int         `json:"code"`
	Data     interface{} `json:"data"` // 使用 interface{} 来处理不同类型的 data
	Msg      string      `json:"msg"`
	ReqID    string      `json:"reqId"`
	UserRole string      `json:"userRole"`
}

// CoinData 成功响应的数据结构
type CoinData struct {
	Total  int        `json:"total"`
	List   []CoinInfo `json:"list"`
	Extend string     `json:"extend"`
}

// CoinService 币种服务
type CoinService struct {
	client *http.Client
}

// GetCoinData 从 CoinQueryResponse 中提取币种数据
func (resp *CoinQueryResponse) GetCoinData() (*CoinData, error) {
	if resp.Code != 200 {
		return nil, fmt.Errorf("response indicates error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	// 检查 data 是否为空字符串
	if resp.Data == "" || resp.Data == nil {
		return nil, fmt.Errorf("empty data in response")
	}

	// 将 interface{} 转换为 CoinData
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var coinData CoinData
	err = json.Unmarshal(dataBytes, &coinData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal coin data: %w", err)
	}

	return &coinData, nil
}

// NewCoinService 创建币种服务实例
func NewCoinService() *CoinService {
	return &CoinService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QueryCoins 查询币种信息
func (s *CoinService) QueryCoins(accessToken string) (*CoinQueryResponse, error) {
	// 创建请求体
	requestBody := map[string]interface{}{
		"search":    "",
		"isBinance": true,
		"page":      1,
		"pageSize":  100,
	}
	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Log.Error("Failed to marshal coin query request body", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", CoinQueryURL, bytes.NewBuffer(reqBody))
	if err != nil {
		logger.Log.Error("Failed to create coin query request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// 发送请求
	logger.Log.Debug("Sending coin query request", map[string]interface{}{
		"url":          CoinQueryURL,
		"auth_header":  "Bearer ***", // 隐藏敏感信息
		"request_body": string(reqBody),
	})

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Log.Error("Failed to send coin query request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read coin query response", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Coin query HTTP error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var coinResp CoinQueryResponse
	err = json.Unmarshal(body, &coinResp)
	if err != nil {
		logger.Log.Error("Failed to unmarshal coin query response", map[string]interface{}{
			"error": err,
			"body":  string(body),
		})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查业务状态码
	if coinResp.Code != 200 {
		logger.Log.Error("Coin query failed", map[string]interface{}{
			"code": coinResp.Code,
			"msg":  coinResp.Msg,
		})
		return nil, fmt.Errorf("query coins failed: code=%d, msg=%s", coinResp.Code, coinResp.Msg)
	}

	logger.Log.Info("Coin query successful", map[string]interface{}{
		"total":     "N/A", // 将在 GetCoinData 中获取
		"user_role": coinResp.UserRole,
		"req_id":    coinResp.ReqID,
	})

	return &coinResp, nil
}

// GetCoinsWithAuth 使用认证服务获取币种信息（便捷方法）
func GetCoinsWithAuth(authService *auth.AuthService) (*CoinQueryResponse, error) {
	// 获取令牌
	tokenPair, err := authService.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	// 创建币种服务并查询
	coinService := NewCoinService()
	return coinService.QueryCoins(tokenPair.AccountToken)
}
