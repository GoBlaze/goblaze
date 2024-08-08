package goblaze

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/valyala/fasthttp"
)

var requestCtxPool = sync.Pool{
	New: func() any {
		return &Ctx{
			response:   new(fasthttp.Response),
			RequestCtx: new(fasthttp.RequestCtx),

			searchingOnAttachedCtx: 0,
		}
	},
}

func AcquireRequestCtx(ctx *fasthttp.RequestCtx) *Ctx {
	actx := requestCtxPool.Get().(*Ctx)
	actx.RequestCtx = ctx // Set the incoming RequestCtx
	return actx
}

var attachedCtxKey = fmt.Sprintf("__attachedCtx::%x__", time.Now().UnixNano())

type Ctx struct {
	app      *GoBlaze
	response *fasthttp.Response

	pnames  [32]string
	pvalues [32]string

	next                   bool
	skipView               bool
	searchingOnAttachedCtx int32
	index                  int

	handlers []func(*Ctx)

	*fasthttp.RequestCtx
}

func ReleaseRequestCtx(ctx *Ctx) {
	ctx.RequestCtx = nil
	ctx.next = false
	ctx.skipView = false

	atomic.StoreInt32(&ctx.searchingOnAttachedCtx, 0)

	requestCtxPool.Put(ctx)
}

func (ctx *Ctx) Value(key any) any {
	if atomic.LoadInt32(&ctx.searchingOnAttachedCtx) == 1 {
		return nil
	}

	if atomic.CompareAndSwapInt32(&ctx.searchingOnAttachedCtx, 0, 1) {
		defer atomic.StoreInt32(&ctx.searchingOnAttachedCtx, 0)

		if extraCtx := ctx.AttachedContext(); extraCtx != nil {
			return extraCtx.Value(key)
		}
	}

	return ctx.RequestCtx.Value(key)
}

func (ctx *Ctx) AttachedContext() context.Context {
	extraCtx, _ := ctx.UserValue(attachedCtxKey).(context.Context)
	return extraCtx
}
func (ctx *Ctx) Param(name string) string {
	for i := len(ctx.pnames) - 1; i >= 0; i-- {
		if ctx.pnames[i] == name {
			return ctx.pvalues[i]
		}
	}
	return ""
}

func (c *Ctx) Response() *fasthttp.Response {
	return c.response
}

func (c *Ctx) Method() string {
	return string(c.RequestCtx.Method())

}

// func (c *Ctx) SetContentType(contentType string) {
// 	c.response.Header.SetContentType(contentType)
// }

func (c *Ctx) SetStatusCode(statusCode int) {
	c.response.SetStatusCode(statusCode)
}

func (c *Ctx) SetBody(body []byte) {
	c.response.SetBody(body)
}

func (c *Ctx) Redirect(location string, statusCode ...int) {
	code := fasthttp.StatusMovedPermanently
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	c.response.SetStatusCode(code)
	c.response.Header.Set("Location", location)
}

func (c *Ctx) SetHeader(key, value string) {
	c.response.Header.Set(key, value)
}

func (c *Ctx) Cookie(cookie *fasthttp.Cookie) {

}

func (c *Ctx) Context() *fasthttp.RequestCtx {
	return c.RequestCtx
}

func (c *Ctx) Next() { // begin for skip all middlewares
	c.index++
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}

}

func (c *Ctx) Query(key string) string {
	r := c.RequestCtx.QueryArgs().Peek(key)

	return *(*string)(unsafe.Pointer(&r))

}

//	func (c *Ctx) JSON(body JSON) {
//		c.response.SetBody(body)
//	}
func (c *Ctx) App() *GoBlaze {
	return c.app
}
