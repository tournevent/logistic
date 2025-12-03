package telemetry

import (
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new OpenTelemetry-aware zap logger.
func NewLogger(level string) (*otelzap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug", "DEBUG":
		zapLevel = zapcore.DebugLevel
	case "warn", "WARN":
		zapLevel = zapcore.WarnLevel
	case "error", "ERROR":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.Encoding = "json"
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	zapLogger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return otelzap.New(zapLogger), nil
}
