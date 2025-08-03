package main

import (
	"encoding/json"
	"os"

	"github.com/stephensulimani/internly-bot/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	args := os.Args

	zapConfig := zap.NewProductionConfig()

	zapConfig.Encoding = "console"
	zapConfig.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}

	config_file := "config.json"

	for i, arg := range args {
		if arg == "--config" {
			config_file = args[i+1]
		}
	}

	log, err := zapConfig.Build()

	if err != nil {
		panic(err)
	}
	defer log.Sync()

	logger := log.Sugar()

	config_f, err := os.Open(config_file)

	if err != nil {
		logger.Error(err)
		return
	}

	defer config_f.Close()

	config := &pkg.Config{}
	err = json.NewDecoder(config_f).Decode(config)

	if err != nil {
		logger.Error(err)
		return
	}

	err = config.Validate()

	if err != nil {
		logger.Error(err)
		return
	}

}
