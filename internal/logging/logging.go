package logging

import (
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Initialize() {
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	zapLogger, err := config.Build()
	if err != nil {
		panic(err)
	}

	defer zapLogger.Sync()

	Log = zapLogger.Sugar()
	Log.Info("Logging initialized")
}
