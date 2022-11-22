package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"syscall"
)

func New(isProduction bool) (*zap.SugaredLogger, func(), error) {
	rawLogger, cleanup, err := newRawLogger(isProduction)
	if err != nil {
		return nil, nil, err
	}

	return rawLogger.Sugar(), cleanup, nil
}

func newRawLogger(isProduction bool) (*zap.Logger, func(), error) {
	if isProduction {
		return newProduction()
	} else {
		return newDevelopment()
	}
}

func newDevelopment() (*zap.Logger, func(), error) {
	develConfig := zap.NewDevelopmentConfig()
	develConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := develConfig.Build()
	return logger, loggerCleanupFunction(logger), err
}

func newProduction() (*zap.Logger, func(), error) {
	productionConfig := zap.NewProductionConfig()
	productionConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	logger, err := productionConfig.Build()
	return logger, loggerCleanupFunction(logger), err
}

func loggerCleanupFunction(logger *zap.Logger) func() {
	return func() {

		err := logger.Sync()

		if err == nil {
			return
		}

		// Some terminals don't support syncing. Assuming that these terminals print everything instantaneously I guess
		// this can be ignored.
		// https://github.com/uber-go/zap/issues/370
		// https://github.com/uber-go/zap/issues/772
		// https://github.com/uber-go/zap/issues/328
		if pathError, ok := err.(*os.PathError); ok {
			if pathError.Err == syscall.EINVAL && (pathError.Path == "/dev/stderr" || pathError.Path == "/dev/stdout") {
				return
			}
		}

		// could not write the log to disk (or wherever it is going).
		logger.Warn("logger sync failed", zap.Error(err))
		// just in case that the last log message isn't recorded
		fmt.Printf("logger sync failed: %v\n", err)

	}
}
