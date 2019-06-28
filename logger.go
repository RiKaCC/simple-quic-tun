package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"os"
)

var log *zap.Logger
var Logger *zap.SugaredLogger

func InitLog() *zap.SugaredLogger {
	logPath := "./log/quic.log" // 需放在配置文件，从配置文件读取

	hook := lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    128,
		MaxBackups: 2,
		MaxAge:     1,
		Compress:   true,
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.FullCallerEncoder,      // 全路径编码器
	}

	//encoderConfig := zap.NewProductionEncoderConfig()

	// 设置日志级别,支持动态改变日志级别
	atomicLevel := zap.NewAtomicLevel()

	http.HandleFunc("/handle/level", atomicLevel.ServeHTTP)
	go func() {
		if err := http.ListenAndServe(":9090", nil); err != nil {
			panic(err)
		}
	}()
	//atomicLevel.SetLevel()

	logcfg := zap.NewProductionConfig()
	logcfg.Level = atomicLevel

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),                                           // 编码器配置
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook)), // 打印到控制台和文件
		atomicLevel, // 日志级别
	)

	// 开启开发模式，堆栈跟踪
	caller := zap.AddCaller()
	// 开启文件及行号
	development := zap.Development()
	// 设置初始化字段
	//filed := zap.Fields(zap.String("serviceName", "serviceName"))
	// 构造日志
	log = zap.New(core, caller, development) //filed)
	Logger = log.Sugar()

	return Logger
}
