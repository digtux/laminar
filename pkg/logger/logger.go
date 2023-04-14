package logger

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLog *zap.SugaredLogger

func InitLogger(debug bool) error {
	if debug {
		zapCfg := zap.NewDevelopmentEncoderConfig()
		zapCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapLogger := zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapCfg),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		), zap.AddCaller(), zap.AddCallerSkip(1),
		)
		sugar := zapLogger.Sugar()
		sugar.Debug("Debug enabled")
		zapLog = sugar
	} else {
		// Override the default zap production Config a little
		// NewProductionConfig is json

		logConfig := zap.NewProductionConfig()
		// customise the "time" field to be ISO8601
		logConfig.EncoderConfig.TimeKey = "time"
		logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		// main message data into the key "msg"
		logConfig.EncoderConfig.MessageKey = "msg"

		// stdout+stderr into stdout
		logConfig.OutputPaths = []string{"stdout"}
		logConfig.ErrorOutputPaths = []string{"stdout"}
		zapLogger, err := logConfig.Build(zap.AddCallerSkip(1))
		if err != nil {
			return err
		}
		sugar := zapLogger.Sugar()
		zapLog = sugar
	}

	zapLog = zapLog.With(zap.Namespace("laminar"))
	return nil
}

func Infow(message string, values ...interface{}) {
	zapLog.Infow(message, values...)
}

func Info(message string) {
	zapLog.Info(message)
}

func Debugw(message string, values ...interface{}) {
	zapLog.Debugw(message, values...)
}

func Debugf(template string, values ...interface{}) {
	zapLog.Debugf(template, values...)
}

func Errorw(message string, values ...interface{}) {
	zapLog.Errorw(message, values...)
}

func Errorf(template string, values ...interface{}) {
	zapLog.Errorf(template, values...)
}

func Fatalw(message string, values ...interface{}) {
	zapLog.Fatalw(message, values...)
}

func Fatalf(template string, values ...interface{}) {
	zapLog.Fatalf(template, values...)
}

func Warnw(message string, values ...interface{}) {
	zapLog.Warnw(message, values...)
}

func Warnf(template string, values ...interface{}) {
	zapLog.Warnf(template, values...)
}

func Warn(args ...interface{}) {
	zapLog.Warn(args...)
}

func Debug(args ...interface{}) {
	zapLog.Debug(args...)
}

func Error(args ...interface{}) {
	zapLog.Error(args...)
}

func Fatal(args ...interface{}) {
	zapLog.Fatal(args...)
}

func Panic(args ...interface{}) {
	zapLog.Panic(args...)
}
