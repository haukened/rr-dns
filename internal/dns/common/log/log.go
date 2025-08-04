package log

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global Logger = newZapLogger(false, zapcore.InfoLevel) // default to prod/info

// SetLogger replaces the global logger instance.
// Useful for testing or overriding behavior.
func SetLogger(l Logger) {
	global = l
}

// GetLogger returns the current global logger instance.
// useful for testing or introspection.
func GetLogger() Logger {
	return global
}

// Logger defines the uDNS logging interface.
type Logger interface {
	Info(fields map[string]any, msg string)
	Error(fields map[string]any, msg string)
	Debug(fields map[string]any, msg string)
	Warn(fields map[string]any, msg string)
	Panic(fields map[string]any, msg string)
	Fatal(fields map[string]any, msg string)
}

// Configure sets up the global logger based on env and level.
func Configure(env, level string) error {
	isDev := env != "prod"

	lvl, err := zapcore.ParseLevel(strings.ToLower(level))
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	global = newZapLogger(isDev, lvl)
	return nil
}

// Info logs at info level using the global logger.
func Info(fields map[string]any, msg string) {
	global.Info(fields, msg)
}

// Error logs at error level using the global logger.
func Error(fields map[string]any, msg string) {
	global.Error(fields, msg)
}

// Debug logs at debug level using the global logger.
func Debug(fields map[string]any, msg string) {
	global.Debug(fields, msg)
}

// zapLogger implements Logger using Uber's zap.
type zapLogger struct {
	base *zap.Logger
}

// newZapLogger returns a logger configured for dev or prod mode with the given level.
func newZapLogger(dev bool, level zapcore.Level) Logger {
	var config zap.Config
	if dev {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.LevelKey = "level"

	logger, _ := config.Build()
	return &zapLogger{base: logger}
}

func (l *zapLogger) Info(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Info(msg)
}

func (l *zapLogger) Error(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Error(msg)
}

func (l *zapLogger) Debug(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Debug(msg)
}

func (l *zapLogger) Warn(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Warn(msg)
}

func (l *zapLogger) Panic(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Panic(msg)
}

func (l *zapLogger) Fatal(fields map[string]any, msg string) {
	l.base.With(zapFields(fields)...).Fatal(msg)
}

// Helper to convert map[string]any to []zap.Field
func zapFields(m map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.Any(k, v))
	}
	return fields
}

// noopLogger is a Logger implementation that discards all log messages.
type noopLogger struct{}

func (n *noopLogger) Info(map[string]any, string)  {}
func (n *noopLogger) Error(map[string]any, string) {}
func (n *noopLogger) Debug(map[string]any, string) {}
func (n *noopLogger) Warn(map[string]any, string)  {}
func (n *noopLogger) Panic(map[string]any, string) {}
func (n *noopLogger) Fatal(map[string]any, string) {}

// NewNoopLogger returns a Logger that discards all log messages.
// Useful for testing or when you want to disable logging.
func NewNoopLogger() Logger {
	return &noopLogger{}
}

// Warn logs at warn level using the global logger.
func Warn(fields map[string]any, msg string) {
	global.Warn(fields, msg)
}

// Panic logs at panic level using the global logger.
func Panic(fields map[string]any, msg string) {
	global.Panic(fields, msg)
}

// Fatal logs at fatal level using the global logger.
func Fatal(fields map[string]any, msg string) {
	global.Fatal(fields, msg)
}
