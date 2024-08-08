package middleware

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/GoBlaze/goblaze"
)

//don't working

// DefaultLogger is called by the Logger middleware handler to log each request.
var DefaultLogger func(next goblaze.Handler) goblaze.Handler

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return.
func Logger(next goblaze.Handler) goblaze.Handler {
	return DefaultLogger(next)
}

// RequestLogger returns a logger handler using a custom LogFormatter.
func RequestLogger(f LogFormatter) func(next goblaze.Handler) goblaze.Handler {
	return func(next goblaze.Handler) goblaze.Handler {
		return func(ctx *goblaze.Ctx) error {
			entry := f.NewLogEntry(ctx)
			start := time.Now()

			err := next(ctx)

			elapsed := time.Since(start)
			entry.Write(ctx.Response.StatusCode(), len(ctx.Response.Body()), ctx.Response.Header.String(), elapsed, nil)
			return err
		}
	}
}

// LogFormatter initiates the beginning of a new LogEntry per request.
// See DefaultLogFormatter for an example implementation.
type LogFormatter interface {
	NewLogEntry(ctx *goblaze.Ctx) LogEntry
}

// LogEntry records the final log when a request completes.
// See defaultLogEntry for an example implementation.
type LogEntry interface {
	Write(status, bytes int, header string, elapsed time.Duration, extra interface{})
	Panic(v interface{}, stack []byte)
}

// LoggerInterface accepts printing to stdlib logger or compatible logger.
type LoggerInterface interface {
	Print(v ...interface{})
}

// DefaultLogFormatter is a simple logger that implements a LogFormatter.
type DefaultLogFormatter struct {
	Logger  LoggerInterface
	NoColor bool
}

// NewLogEntry creates a new LogEntry for the request.
func (l *DefaultLogFormatter) NewLogEntry(ctx *goblaze.Ctx) LogEntry {
	useColor := !l.NoColor
	entry := &defaultLogEntry{
		DefaultLogFormatter: l,
		ctx:                 ctx,
		buf:                 &bytes.Buffer{},
		useColor:            useColor,
	}

	reqID := string(ctx.RequestCtx.Response.Header.Peek("X-Request-ID"))
	if reqID != "" {
		cW(entry.buf, useColor, nYellow, "[%s] ", reqID)
	}
	cW(entry.buf, useColor, nCyan, "\"")
	cW(entry.buf, useColor, bMagenta, "%s ", ctx.Method())

	scheme := "http"
	if ctx.IsTLS() {
		scheme = "https"
	}
	cW(entry.buf, useColor, nCyan, "%s://%s%s %s\" ", scheme, ctx.Host(), ctx.RequestURI(), ctx.Request.Header.Protocol())

	entry.buf.WriteString("from ")
	entry.buf.WriteString(ctx.RequestCtx.RemoteAddr().String())
	entry.buf.WriteString(" - ")

	return entry
}

type defaultLogEntry struct {
	*DefaultLogFormatter
	ctx      *goblaze.Ctx
	buf      *bytes.Buffer
	useColor bool
}

func (l *defaultLogEntry) Write(status, bytes int, header string, elapsed time.Duration, extra interface{}) {
	switch {
	case status < 200:
		cW(l.buf, l.useColor, bBlue, "%03d", status)
	case status < 300:
		cW(l.buf, l.useColor, bGreen, "%03d", status)
	case status < 400:
		cW(l.buf, l.useColor, bCyan, "%03d", status)
	case status < 500:
		cW(l.buf, l.useColor, bYellow, "%03d", status)
	default:
		cW(l.buf, l.useColor, bRed, "%03d", status)
	}

	cW(l.buf, l.useColor, bBlue, " %dB", bytes)

	l.buf.WriteString(" in ")
	if elapsed < 500*time.Millisecond {
		cW(l.buf, l.useColor, nGreen, "%s", elapsed)
	} else if elapsed < 5*time.Second {
		cW(l.buf, l.useColor, nYellow, "%s", elapsed)
	} else {
		cW(l.buf, l.useColor, nRed, "%s", elapsed)
	}

	l.Logger.Print(l.buf.String())
}

func (l *defaultLogEntry) Panic(v interface{}, stack []byte) {

	l.Logger.Print(fmt.Sprintf("panic: %v\n%s", v, stack))
}

// cW is a helper function to write colored output if necessary
func cW(buf *bytes.Buffer, useColor bool, colorCode int, format string, args ...interface{}) {
	if useColor {
		buf.WriteString(fmt.Sprintf("\x1b[%dm", colorCode))
	}
	buf.WriteString(fmt.Sprintf(format, args...))
	if useColor {
		buf.WriteString("\x1b[0m")
	}
}

const (
	nYellow  = 33
	nCyan    = 36
	bMagenta = 35
	bBlue    = 34
	bGreen   = 32
	bRed     = 31
	nGreen   = 32
	nRed     = 31

	bCyan   = 36
	bYellow = 33
	bWhite  = 37
	bBlack  = 30

	nWhite = 97
	nBlack = 30

	nDefault = 39
)

func init() {
	color := true
	if runtime.GOOS == "windows" {
		color = false
	}
	DefaultLogger = RequestLogger(&DefaultLogFormatter{
		Logger:  log.New(os.Stdout, "", log.LstdFlags),
		NoColor: color,
	})
}
