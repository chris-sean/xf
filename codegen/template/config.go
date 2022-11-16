package template

const CONFIG_GO = `package config

import (
	"github.com/chris-sean/xf"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap/zapcore"
)

func init() {
	godotenv.Load()

	// 本机开发用的配置文件
	godotenv.Overload("./config/dev.env")

	xf.SetLogLevel(LogLevel())
}

func LogLevel() zapcore.Level {
	l := os.Getenv("#MODULE_NAME#_LOG_LEVEL")
	return xf.LogLevelOfString(l)
}
`
