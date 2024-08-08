package goblaze

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

type logrusLogger struct {
	*logrus.Logger
}

func (l *logrusLogger) Info(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

func (l *logrusLogger) Debug(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

func (l *logrusLogger) Fatal(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}

func (l *logrusLogger) Panic(format string, args ...interface{}) {
	l.Logger.Panicf(format, args...)
}

func (l *logrusLogger) Print(format string, args ...interface{}) {
	l.Logger.Printf(format, args...)
}

func (l *logrusLogger) Warn(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

func (l *logrusLogger) Error(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

func (l *logrusLogger) Trace(format string, args ...interface{}) {
	l.Logger.Tracef(format, args...)
}

func (l *logrusLogger) WithTime(t time.Time) {
	l.Logger.WithTime(t)
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func NewLog() *logrusLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(colorable.NewColorableStdout())
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:            isatty.IsTerminal(os.Stdout.Fd()),
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableTimestamp:       false,
		DisableLevelTruncation: false,
		PadLevelText:           true,
		QuoteEmptyFields:       false,
		FieldMap:               logrus.FieldMap{},

		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			_, file := filepath.Split(f.File)
			return "", fmt.Sprintf("%s:%d", file, f.Line)
		},
		EnvironmentOverrideColors: true,
	})

	return &logrusLogger{
		Logger: logger,
	}
}
