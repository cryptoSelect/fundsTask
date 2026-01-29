package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cryptoSelect/fundsTask/config"
	"github.com/cryptoSelect/fundsTask/utils/logger"
)

const (
	LoginURL = "https://api.valuescan.io/api/authority/login"
)

// LoginRequest 登录请求结构
type LoginRequest struct {
	PhoneOrEmail  string `json:"phoneOrEmail"`
	Code          string `json:"code"`
	EndpointEnum  int    `json:"endpointEnum"`
	LoginTypeEnum int    `json:"loginTypeEnum"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Code     int       `json:"code"`
	Data     LoginData `json:"data"`
	Msg      string    `json:"msg"`
	ReqID    string    `json:"reqId"`
	UserRole string    `json:"userRole"`
}

// LoginData 登录数据
type LoginData struct {
	AccountToken string `json:"account_token"`
	RefreshToken string `json:"refresh_token"`
}

// TokenPair 令牌对，用于存储获取到的令牌
type TokenPair struct {
	AccountToken     string `json:"account_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresAt        int64  `json:"expires_at,omitempty"`         // 过期时间戳（可选）
	RefreshExpiresAt int64  `json:"refresh_expires_at,omitempty"` // 刷新令牌过期时间戳（可选）
}

// AuthService 认证服务
type AuthService struct {
	client *http.Client
}

// NewTokenPair 创建新的令牌对
func NewTokenPair(accountToken, refreshToken string) *TokenPair {
	return &TokenPair{
		AccountToken: accountToken,
		RefreshToken: refreshToken,
	}
}

// IsValid 检查令牌是否有效（基于过期时间）
func (tp *TokenPair) IsValid() bool {
	if tp.ExpiresAt == 0 {
		return true // 如果没有设置过期时间，默认有效
	}

	now := time.Now().Unix()
	return now < tp.ExpiresAt
}

// IsRefreshValid 检查刷新令牌是否有效
func (tp *TokenPair) IsRefreshValid() bool {
	if tp.RefreshExpiresAt == 0 {
		return true // 如果没有设置过期时间，默认有效
	}

	now := time.Now().Unix()
	return now < tp.RefreshExpiresAt
}

// NewAuthService 创建认证服务实例
func NewAuthService() *AuthService {
	return &AuthService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login 执行登录
func (s *AuthService) Login() (*LoginResponse, error) {
	logger.Log.Debug("Starting login process", map[string]interface{}{
		"email": config.Cfg.Login.PhoneOrEmail,
	})

	// 构建请求体
	loginReq := LoginRequest{
		PhoneOrEmail:  config.Cfg.Login.PhoneOrEmail,
		Code:          config.Cfg.Login.Code,
		EndpointEnum:  1,
		LoginTypeEnum: 2,
	}

	// 序列化请求体
	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		logger.Log.Error("Failed to marshal login request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", LoginURL, bytes.NewBuffer(reqBody))
	if err != nil {
		logger.Log.Error("Failed to create login request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// 发送请求
	logger.Log.Debug("Sending login request", map[string]interface{}{"url": LoginURL})
	resp, err := s.client.Do(req)
	if err != nil {
		logger.Log.Error("Failed to send login request", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read login response", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Login HTTP error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        string(body),
		})
		return nil, fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var loginResp LoginResponse
	err = json.Unmarshal(body, &loginResp)
	if err != nil {
		logger.Log.Error("Failed to unmarshal login response", map[string]interface{}{"error": err})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查业务状态码
	if loginResp.Code != 200 {
		logger.Log.Error("Login failed", map[string]interface{}{
			"code": loginResp.Code,
			"msg":  loginResp.Msg,
		})
		return nil, fmt.Errorf("login failed: code=%d, msg=%s", loginResp.Code, loginResp.Msg)
	}

	logger.Log.Info("Login successful", map[string]interface{}{
		"user_role": loginResp.UserRole,
		"req_id":    loginResp.ReqID,
	})

	return &loginResp, nil
}

// GetTokens 获取令牌（便捷方法）
func (s *AuthService) GetTokens() (*TokenPair, error) {
	resp, err := s.Login()
	if err != nil {
		return nil, err
	}

	return NewTokenPair(resp.Data.AccountToken, resp.Data.RefreshToken), nil
}

// GetTokensWithExpiry 获取令牌并设置过期时间（基于JWT解析）
func (s *AuthService) GetTokensWithExpiry() (*TokenPair, error) {
	resp, err := s.Login()
	if err != nil {
		return nil, err
	}

	tokenPair := NewTokenPair(resp.Data.AccountToken, resp.Data.RefreshToken)

	// 尝试从JWT令牌中解析过期时间
	if exp := extractExpiryFromJWT(resp.Data.AccountToken); exp > 0 {
		tokenPair.ExpiresAt = exp
	}

	if refreshExp := extractExpiryFromJWT(resp.Data.RefreshToken); refreshExp > 0 {
		tokenPair.RefreshExpiresAt = refreshExp
	}

	return tokenPair, nil
}

// ValidateAndRefreshToken 验证并刷新 Token
func (s *AuthService) ValidateAndRefreshToken(accessToken string) (string, error) {
	// 检查 Token 是否有效
	tokenPair := &TokenPair{
		AccountToken: accessToken,
	}

	if tokenPair.IsValid() {
		logger.Log.Debug("Token is still valid", nil)
		return accessToken, nil
	}

	logger.Log.Info("Token expired, attempting re-login", nil)

	// 重新登录获取新 Token
	newTokenPair, err := s.GetTokens()
	if err != nil {
		return "", fmt.Errorf("failed to re-login: %w", err)
	}

	logger.Log.Info("Successfully re-logged in", map[string]interface{}{
		"token_valid":       newTokenPair.IsValid(),
		"account_token_len": len(newTokenPair.AccountToken),
	})

	return newTokenPair.AccountToken, nil
}

// extractExpiryFromJWT 从JWT令牌中提取过期时间（简单实现）
func extractExpiryFromJWT(token string) int64 {
	// 这里应该解析JWT的payload部分获取exp字段
	// 为了简化，这里返回0，表示不设置过期时间
	// 在实际项目中，可以使用jwt库来解析
	return 0
}
