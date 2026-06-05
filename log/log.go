package log

import (
	"os"

	"github.com/kolonse/logs"
)

// const (
//
//	DBG = 0
//	INFO = 1
//	WARN = 2
//	ERR = 3
//
// )
type Logger interface {
	Debug(module string, fmt string, arg ...any)
	Info(module string, fmt string, arg ...any)
	Warn(module string, fmt string, arg ...any)
	Error(module string, fmt string, arg ...any)
}

type DefaultLogger struct {
	logger *logs.BeeLogger
}

func (l *DefaultLogger) Debug(module string, fmt string, arg ...any) {
	l.logger.Debug("["+module+"] "+fmt, arg...)
}

func (l *DefaultLogger) Info(module string, fmt string, arg ...any) {
	l.logger.Info("["+module+"] "+fmt, arg...)
}

func (l *DefaultLogger) Warn(module string, fmt string, arg ...any) {
	l.logger.Warn("["+module+"] "+fmt, arg...)
}

func (l *DefaultLogger) Error(module string, fmt string, arg ...any) {
	l.logger.Error("["+module+"] "+fmt, arg...)
}

var NormalLogger *DefaultLogger

const file_logger = `
	{
		"filename":"logs/gpbx.log",
		"maxlines":1000000,
		"maxsize":104857600,
		"daily":true,
		"maxdays":15,
		"rotate":true
	}
`

func init() {
	NormalLogger = &DefaultLogger{
		logger: logs.NewLogger(10),
	}
	NormalLogger.logger.SetLogger("console", "")
	os.MkdirAll("logs", 0755)
	e := NormalLogger.logger.SetLogger("file", file_logger)
	if e != nil {
		panic(e)
	}
	NormalLogger.logger.EnableFuncCallDepth(false)
	// NormalLogger.logger.SetLogFuncCallDepth(4)
}
