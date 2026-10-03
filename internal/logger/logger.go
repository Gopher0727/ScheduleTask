package logger

import "go.uber.org/zap"

var L = zap.NewNop().Sugar()

func Init(level string) {
	var cfg zap.Config
	if level == "debug" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	L = l.Sugar()
}

func Sync() {
	if L != nil {
		L.Sync()
	}
}
