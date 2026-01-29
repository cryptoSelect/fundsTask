package utils

import (
	"time"

	"github.com/cryptoSelect/fundsTask/config"
	"github.com/cryptoSelect/fundsTask/utils/logger"
)

// ShouldDelay 判断是否需要延时
func ShouldDelay() bool {
	// 如果是生产环境，需要延时
	return config.Cfg.Mode == "prod"
}

// WaitForNext5MinuteMark 等待下一个5分钟时间点
func WaitForNext5MinuteMark() {
	now := time.Now()

	// 计算下一个5分钟时间点
	nextMinute := ((now.Minute() / 5) + 1) * 5
	if nextMinute >= 60 {
		nextMinute = 0
	}

	// 计算到下一个5分钟时间点的时长
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, now.Location())
	if nextTime.Before(now) {
		nextTime = nextTime.Add(1 * time.Hour)
	}

	delay := nextTime.Sub(now)

	logger.Log.Info("Waiting for next execution time", map[string]interface{}{
		"next_time":     nextTime.Format("15:04:05"),
		"delay_seconds": int(delay.Seconds()),
		"mode":          config.Cfg.Mode,
	})

	time.Sleep(delay)
}

// WaitForNext4HourMark 等待下一个4小时时间点
func WaitForNext4HourMark() {
	now := time.Now()

	// 计算下一个4小时时间点（00:00, 04:00, 08:00, 12:00, 16:00, 20:00）
	nextHour := ((now.Hour() / 4) + 1) * 4
	if nextHour >= 24 {
		nextHour = 0
	}

	// 计算到下一个4小时时间点的时长
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), nextHour, 0, 0, 0, now.Location())
	if nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}

	delay := nextTime.Sub(now)

	logger.Log.Info("Waiting for next coin info execution time", map[string]interface{}{
		"next_time":   nextTime.Format("15:04:05"),
		"delay_hours": delay.Hours(),
		"mode":        config.Cfg.Mode,
	})

	time.Sleep(delay)
}
