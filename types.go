package goblaze

import (
	"github.com/valyala/fasthttp"
)

type No struct{}

func (*No) Lock() {}

func (*No) Unlock() {}

// Handler defines a function to serve HTTP requests.
type Handler func(ctx *Ctx) error
type RequestHandler func(*fasthttp.RequestHandler)

type JSON map[string]any

type RouteMessage struct {
	name     string
	method   string
	path     string
	handlers string
}
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
