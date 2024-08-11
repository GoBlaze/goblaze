package goblaze

import (
	"net"
	"time"

	"github.com/GoBlaze/goblaze/fasthttp"
)

const DefaultMaxRequestBodySize = 4 * 1024 * 1024

var zeroTime time.Time

type No struct{}

func (*No) Lock() {}

func (*No) Unlock() {}

// Handler defines a function to serve HTTP requests.
type PanicHandler func(*Ctx, interface{})
type Handler func(ctx *Ctx) error
type RequestHandler func(*fasthttp.RequestCtx)
type ErrorView func(*Ctx, error, int)
type FormValueFunc func(*Ctx, string) []byte
type ServeHandler func(c net.Conn) error
type HijackHandler func(c net.Conn)

type ErrNothingRead struct {
	error
}

// ErrNothingRead is returned when a keep-alive connection is closed, either because the remote closed it or because of a read timeout.

type JSON map[string]any

//	type RouteMessage struct {
//		name     string
//		method   string
//		path     string
//		handlers string
//	}
type Colors struct {
	noCopy No //nolint:unused,structcheck
	// Black color.
	//
	// Optional. Default: "\u001b[90m"
	Black string

	// Red color.
	//
	// Optional. Default: "\u001b[91m"
	Red string

	// Green color.
	//
	// Optional. Default: "\u001b[92m"
	Green string

	// Yellow color.
	//
	// Optional. Default: "\u001b[93m"
	Yellow string

	// Blue color.
	//
	// Optional. Default: "\u001b[94m"
	Blue string

	// Magenta color.
	//
	// Optional. Default: "\u001b[95m"
	Magenta string

	// Cyan color.
	//
	// Optional. Default: "\u001b[96m"
	Cyan string

	// White color.
	//
	// Optional. Default: "\u001b[97m"
	White string

	// Reset color.
	//
	// Optional. Default: "\u001b[0m"
	Reset string
}
