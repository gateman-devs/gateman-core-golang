package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerOptions struct {
	Key  string
	Data interface{}
}

var RequestMetricMonitor = (&APIToolKitMonitor{})

// This logs info level messages.
func Info(msg string, payload ...LoggerOptions) {
	zapFields := []zapcore.Field{}
	for _, data := range payload {
		zapFields = append(zapFields, zap.Any(data.Key, data.Data))
	}
	// MetricMonitor.Log(msg, payload, InfoLevel)
	Logger.Info(msg, zapFields...)
}

// This logs error messages.
// describe the incident in msg and pass the error through logger options
// with key error
func Error(msg string, payload ...LoggerOptions) {
	zapFields := []zapcore.Field{}
	for _, data := range payload {
		zapFields = append(zapFields, zap.Any(data.Key, data.Data))
	}
	// MetricMonitor.ReportError(err, payload)
	Logger.Error(msg, zapFields...)
}

// This logs warning messages.
func Warning(msg string, payload ...LoggerOptions) {
	zapFields := []zapcore.Field{}
	for _, data := range payload {
		zapFields = append(zapFields, zap.Any(data.Key, data.Data))
	}
	// MetricMonitor.Log(msg, payload, InfoLevel)
	Logger.Warn(msg, zapFields...)
}
