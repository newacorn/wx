package log

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger
var level zap.AtomicLevel

func init() {
	level = zap.NewAtomicLevel()
	l, err := zap.Config{
		Level:       level,
		Development: true,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "T",
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			LevelKey:      "L",
			EncodeLevel:   zapcore.LowercaseLevelEncoder,
			StacktraceKey: "stackTrace",
			MessageKey:    "msg",
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
	if err != nil {
		log.Fatalln(err)
	}
	option := zap.AddStacktrace(zapcore.InfoLevel)
	l = l.WithOptions(option)
	Log = l
}

func SetLevel(l zapcore.Level) {
	level.SetLevel(l)
}
