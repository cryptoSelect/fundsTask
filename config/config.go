package config

import (
	"encoding/json"
	"os"
)

// LoginConfig 登录配置
type LoginConfig struct {
	PhoneOrEmail string `json:"phoneOrEmail"`
	Code         string `json:"code"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
	SSLMode  string `json:"sslMode"`
}

// TimerConfig 定时器配置
type TimerConfig struct {
	SkipFirstDelay     bool `json:"skipFirstDelay"`
	ImmediateExecution bool `json:"immediateExecution"`
}

// Config 应用配置
type Config struct {
	Mode     string         `json:"mode"`
	Login    LoginConfig    `json:"login"`
	Database DatabaseConfig `json:"database"`
	Timer    TimerConfig    `json:"timer"`
}

var Cfg *Config

// Init 初始化配置
func Init() {
	configFile := "config/config.json"
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic("Failed to read config file: " + err.Error())
	}

	err = json.Unmarshal(data, &Cfg)
	if err != nil {
		panic("Failed to parse config file: " + err.Error())
	}
}
