package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
)

var (
	logger *zap.Logger
	once   sync.Once
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

		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        // Format time
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Add color to levels in development mode
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

func Info(message string, fields ...zap.Field) {
	GetLogger().Info(message, fields...)
}

// Warn logs a warning message with optional fields
func Warn(message string, fields ...zap.Field) {
	GetLogger().Warn(message, fields...)
}

// Error logs an error message with optional fields
func Error(message string, fields ...zap.Field) {
	GetLogger().Error(message, fields...)
}
