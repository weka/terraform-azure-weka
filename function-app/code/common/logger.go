package common

import (
	"context"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

func init() {
	setCallerDirDisplayLevel()
}

// Set the amount of nested dirs displayed before `<file_name>:<line_number>` for `caller` field in logger.
// `LOG_CALLER_DIR_LVL` is used for this.
// If unset - does nothing (default `caller` formatting is used)
// If `LOG_CALLER_DIR_LVL=0`, only the filename and line number are displayed (e.g. `message_processor.go:89`)
// see https://github.com/rs/zerolog/blob/master/README.md#add-file-and-line-number-to-log
func setCallerDirDisplayLevel() {
	callerDirLvl, ok := os.LookupEnv("LOG_CALLER_DIR_LVL")
	if !ok {
		return
	}
	// get "caller" dir level value
	var lvl int // 0 by default (only file name will be displayed - with no dirs)
	if val, err := strconv.Atoi(callerDirLvl); err == nil {
		lvl = val
	}
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		dirsNum := lvl
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				if dirsNum < 1 {
					break
				}
				dirsNum--
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
}

type Logger struct {
	*zerolog.Logger
}

func getLogLevel() zerolog.Level {
	lvlStr := os.Getenv("LOG_LEVEL")
	lvl := 1 // info level
	if val, err := strconv.Atoi(lvlStr); err == nil {
		lvl = val
	}
	return zerolog.Level(lvl)
}

func NewLogger() *Logger {
	log := zerolog.New(os.Stdout).
		Level(getLogLevel()).
		With().
		Caller().
		Logger()
	return &Logger{&log}
}

func LoggerFromCtx(ctx context.Context) *Logger {
	return &Logger{zerolog.Ctx(ctx)}
}
