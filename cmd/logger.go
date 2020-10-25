package cmd

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"os"

	"go.uber.org/zap"
)

// switch between a vanilla Development or Production logging format (--debug)
func startLogger(debug bool) (zapSugar *zap.SugaredLogger, zapLogger *zap.Logger) {
	// https://blog.sandipb.net/2018/05/02/using-zap-simple-use-cases/
	if debug {

		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapLogger, err := config.Build()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sugar := zapLogger.Sugar()
		sugar.Debug("debug enabled")
		return sugar, zapLogger
	} else {
		// Override the production Config (I personally don't see the point of using stderr
		// https://github.com/uber-go/zap/blob/feeb9a050b31b40eec6f2470e7599eeeadfe5bdd/config.go#L119

		logConfig := zap.NewProductionConfig()
		//logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		//logConfig.Encoding = "json" // "json" / "console"
		logConfig.OutputPaths = []string{"stdout"}
		logConfig.ErrorOutputPaths = []string{"stdout"}
		zapLogger, err := logConfig.Build()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return zapLogger.Sugar(), zapLogger
	}
}
