package goblaze

import (
	"github.com/valyala/fasthttp"
)

type No struct{}

func (*No) Lock() {}

func (*No) Unlock() {}

type Params []struct {
	Key, Value string
}

// Handler defines a function to serve HTTP requests.
type Handler func(ctx *Ctx, params Params) error
type RequestHandler func(*fasthttp.RequestHandler)

type Router struct {
	noCopy No // nolint:structcheck,unused

	Handler Handler

	trees map[string]*node

	RedirectTrailingSlash bool

	RedirectFixedPath bool

	HandleMethodNotAllowed bool

	HandleOPTIONS bool

	NotFound fasthttp.RequestHandler

	MethodNotAllowed fasthttp.RequestHandler

	PanicHandler func(*fasthttp.RequestCtx, interface{})
}

type Ctx struct {
	noCopy No // nolint:structcheck,unused
	*GoBlaze
	*fasthttp.RequestCtx

	route *Router
}

type JSON map[string]any
