package logger

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

var (
	logger        *zap.Logger
	once          sync.Once
	loggerFields  = "logger.fields"
	RequestFields = "request.fields"
	FunctionName  = "function.name"
)

func InitLogger(mode string) error {
	var err error

	once.Do(func() {
		var config zap.Config
		if mode == "production" {
			config = zap.NewProductionConfig()
		} else {
			config = zap.NewDevelopmentConfig()
		}

		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, err = config.Build()
	})

	return err
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if logger == nil {
		panic("Logger not initialized. Call InitLogger first.")
	}
	return logger
}

// Sync flushes any buffered log entries (should be called before program exit)
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

func Info(ctx context.Context, message string, fields ...zap.Field) {
	storedFields := FromContext(ctx)
	storedFields = append(storedFields, fields...)
	logger.Info(message, storedFields...)
}

// Warn logs a warning message with optional fields
func Warn(ctx context.Context, message string, fields ...zap.Field) {
	storedFields := FromContext(ctx)
	storedFields = append(storedFields, fields...)
	logger.Info(message, storedFields...)
}

// Error logs an error message with optional fields
func Error(ctx context.Context, message string, fields ...zap.Field) {
	storedFields := FromContext(ctx)
	storedFields = append(storedFields, fields...)
	logger.Info(message, storedFields...)
}

func With(ctx context.Context, fields ...zap.Field) context.Context {
	data := ctx.Value(loggerFields)
	storedFields := make([]zap.Field, 0)
	if data != nil {
		storedFields = data.([]zap.Field)
	}

	storedFields = append(storedFields, fields...)
	return context.WithValue(ctx, loggerFields, storedFields)
}

func FromContext(ctx context.Context) []zap.Field {
	data := ctx.Value(loggerFields)
	fields := make([]zap.Field, 0)
	if data != nil {
		fields = data.([]zap.Field)
	}

	return fields
}
