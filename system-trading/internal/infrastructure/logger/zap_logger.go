package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/system-trading/core/internal/infrastructure/config"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type Field = interfaces.Field

type ZapLogger struct {
	*zap.Logger
	config config.LoggingConfig
}

func NewZapLogger(cfg config.LoggingConfig) (*ZapLogger, error) {
	level := parseLogLevel(cfg.Level)
	
	var core zapcore.Core
	
	switch cfg.Output {
	case "file":
		if cfg.Filename == "" {
			return nil, fmt.Errorf("filename is required when output is file")
		}
		core = createFileCore(cfg, level)
	case "both":
		if cfg.Filename == "" {
			return nil, fmt.Errorf("filename is required when output is both")
		}
		fileCore := createFileCore(cfg, level)
		consoleCore := createConsoleCore(cfg, level)
		core = zapcore.NewTee(fileCore, consoleCore)
	default:
		core = createConsoleCore(cfg, level)
	}
	
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	
	return &ZapLogger{
		Logger: logger,
		config: cfg,
	}, nil
}

func createFileCore(cfg config.LoggingConfig, level zapcore.Level) zapcore.Core {
	writer := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
	
	encoder := createEncoder(cfg.Format)
	return zapcore.NewCore(encoder, zapcore.AddSync(writer), level)
}

func createConsoleCore(cfg config.LoggingConfig, level zapcore.Level) zapcore.Core {
	encoder := createEncoder(cfg.Format)
	return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
}

func createEncoder(format string) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.StacktraceKey = "stacktrace"
	
	switch format {
	case "console":
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		return zapcore.NewJSONEncoder(encoderConfig)
	}
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *ZapLogger) Info(msg string, fields ...interfaces.Field) {
	l.Logger.Info(msg, convertFields(fields)...)
}

func (l *ZapLogger) Error(msg string, fields ...interfaces.Field) {
	l.Logger.Error(msg, convertFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields ...interfaces.Field) {
	l.Logger.Warn(msg, convertFields(fields)...)
}

func (l *ZapLogger) Debug(msg string, fields ...interfaces.Field) {
	l.Logger.Debug(msg, convertFields(fields)...)
}

func convertFields(fields []interfaces.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}

func (l *ZapLogger) WithComponent(component string) interfaces.Logger {
	return &ZapLogger{
		Logger: l.Logger.With(zap.String("component", component)),
		config: l.config,
	}
}

func (l *ZapLogger) WithRequestID(requestID string) interfaces.Logger {
	return &ZapLogger{
		Logger: l.Logger.With(zap.String("request_id", requestID)),
		config: l.config,
	}
}

func (l *ZapLogger) Sync() error {
	return l.Logger.Sync()
}