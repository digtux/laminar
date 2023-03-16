package common

import (
	"fmt"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap/zapcore"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Misc helper functions

// GetFileAbsPath will expand on something such as ~/.ssh/my_id_rsa and return a string like /home/user/.ssh/my_id_rsa
func GetFileAbsPath(fileName string, log *zap.SugaredLogger) (result string) {
	if strings.HasPrefix(fileName, "~/") {
		usr, _ := user.Current()
		dir := usr.HomeDir
		fileName = filepath.Join(dir, fileName[2:])
	}

	result, err := filepath.Abs(fileName)
	if err != nil {
		log.Fatalw("unable to determine path to a operations",
			"fileName", fileName,
			"error", err,
		)
	}

	return result
}

// IsDir will return true if the path is a directory
func IsDir(path string, log *zap.SugaredLogger) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Debugw("couldn't reading path",
			"path", path,
			"error", err,
		)
		return false
	}
	return fileInfo.IsDir()
}

// UniqueStrings takes an array of strings in, returns only the unique ones
func UniqueStrings(input []string) []string {
	// credit : https://kylewbanks.com/blog/creating-unique-slices-in-go
	u := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u
}

// GetLogger provides us with sugared logger
// switch between a vanilla Development or Production logging format (--debug)
// The only change from vanilla zap is the ProductionConfig outputs to stdout instead of stderr
func GetLogger(debug bool) (zapLog *zap.SugaredLogger) {
	// https://blog.sandipb.net/2018/05/02/using-zap-simple-use-cases/
	if debug {
		zapCfg := zap.NewDevelopmentEncoderConfig()
		zapCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapLogger := zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zapCfg),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		), zap.AddCaller(),
		)
		sugar := zapLogger.Sugar()
		sugar.Debug("debug enabled")
		return sugar
	}
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
	zapLogger, err := logConfig.Build()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return zapLogger.Sugar()
}
