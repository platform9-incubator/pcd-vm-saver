package log

import (
	"fmt"
	"os"
	"time"

	"github.com/platform9/pcd-vm-saver/pkg/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createDirectoryIfNotExists() error {
	var err error
	// Create PcdVMSaverLogDir
	if _, err = os.Stat(util.PcdVMSaverLogDir); os.IsNotExist(err) {
		errlogdir := os.Mkdir(util.PcdVMSaverLogDir, os.ModePerm)
		if errlogdir != nil {
			return errlogdir
		}
		return nil
	}
	return err
}

func fileConfig() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.9999Z"))
	})
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(config)
}

func Logger() error {
	// Create the directory structure if it does not exist.
	err := createDirectoryIfNotExists()
	if err != nil {
		return fmt.Errorf("failed to create Director. \nError is: %s", err)
	}

	// Create the log file if it does not exist, or open it for appending.
	file, err := os.OpenFile(util.VMSaverLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("couldn't open the log file: %s. \nError is: %s", util.VMSaverLog, err)
	}

	core := zapcore.NewCore(fileConfig(), zapcore.AddSync(file), zapcore.DebugLevel)

	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	return nil
}
