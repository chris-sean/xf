package xf

import (
	"runtime"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.SugaredLogger

var Logger *zap.Logger

//var DefaultLoggerConfig = zap.NewProductionConfig()

//var DefaultLoggerConfig = zap.NewDevelopmentConfig()

//var LogLevel = zap.DebugLevel

func init() {
	ReloadLogger(nil)
}

func ReloadLogger(custom func(config *zap.Config /*, rotationCfg *lumberjack.Logger*/)) {
	var config = zap.NewProductionConfig()
	config.Encoding = "console"
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeCaller = nil

	//rotationCfg := &lumberjack.Logger{
	//	Filename: "./log/app.log",
	//	MaxSize:  10,              // mb
	//	MaxAge:   30,              // days
	//}

	if custom != nil {
		custom(&config)
	}

	//core := zapcore.NewCore(
	//	zapcore.NewConsoleEncoder(config.EncoderConfig),
	//	zapcore.AddSync(rotationCfg),
	//	config.Level,
	//)
	//
	//logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip())

	Logger, _ = config.Build()
	L = Logger.Sugar()
}

func LogLevelOfString(str string) zapcore.Level {
	switch str {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	}

	return zapcore.InfoLevel
}

func SetLogLevel(level zapcore.Level) {
	ReloadLogger(func(config *zap.Config) {
		config.Level = zap.NewAtomicLevelAt(level)
	})
}

func ConfigLogger(config zap.Config) {
	Logger, _ = config.Build()
	L = Logger.Sugar()
}

func Debug(args ...interface{}) {
	L.Debug(args...)
}

func Info(args ...interface{}) {
	L.Info(args...)
}

func Warn(args ...interface{}) {
	L.Warn(args...)
}

func Error(args ...interface{}) {
	L.Error(args...)
}

func Panic(args ...interface{}) {
	L.Panic(args...)
}

func Fatal(args ...interface{}) {
	L.Fatal(args...)
}

func Debugf(template string, args ...interface{}) {
	L.Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	L.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	L.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	L.Errorf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	L.Panicf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	L.Fatalf(template, args...)
}

func FileWithLineNumber(index int) string {
	_, file, line, ok := runtime.Caller(index)
	if ok {
		return file + ":" + strconv.FormatInt(int64(line), 10)
	}
	return ""
}

// FileWithLineNumberAfter return the file name and line number of the file after input theFile.
func FileWithLineNumberAfter(theFile string) string {
	// first line is current function. so i start from 1.
	found := false
	for i := 1; i < 100; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok {
			if !found && file == theFile {
				found = true
				continue
			}
			if found && file != theFile {
				return file + ":" + strconv.FormatInt(int64(line), 10)
			}
		}
	}

	return ""
}
