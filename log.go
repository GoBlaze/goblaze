package goblaze

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

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
	setupPrettyLogrus(logger)
	return &logrusLogger{Logger: logger}
}

func setupPrettyLogrus(logger *logrus.Logger) {
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			_, file := filepath.Split(f.File)
			return "", fmt.Sprintf("%s:%d", file, f.Line)
		},
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)
}
