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
type Handler func(ctx *Ctx) error
type RequestHandler func(*fasthttp.RequestHandler)

type Router struct {
	noCopy             No // nolint:structcheck,unused
	customMethodsIndex map[string]int
	registeredPaths    map[string][]string

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
	noCopy   No // nolint:structcheck,unused
	App      *GoBlaze
	response *fasthttp.Response

	route *Router
	*fasthttp.RequestCtx
}

type JSON map[string]any
