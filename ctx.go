package goblaze

// #cgo CFLAGS: -I ./ccode
// #cgo LDFLAGS: -L ./ccode/ -lother
// #include <stdlib.h>
// #include "cJSON.h"

import (
	"C"
	"context"
	"fmt"

	"time"

	"sync/atomic"

	"github.com/GoBlaze/goblaze/pool"
	"github.com/valyala/fasthttp"
)

var requestCtxPool = pool.NewPool[*Ctx](func() *Ctx {
	return &Ctx{
		response:   &fasthttp.Response{},
		RequestCtx: &fasthttp.RequestCtx{},

		searchingOnAttachedCtx: 0,
	}
})

func AcquireRequestCtx(ctx *fasthttp.RequestCtx) *Ctx {
	actx := requestCtxPool.Get()

	actx.RequestCtx = ctx // Set the incoming RequestCtx

	return actx
}

var attachedCtxKey = fmt.Sprintf("__attachedCtx::%x__", time.Now().UnixNano())

type Ctx struct {
	_ No // nolint:structcheck,unused

	index int

	searchingOnAttachedCtx int32

	paramValues map[string]string

	app *GoBlaze
	s   *Server

	response *fasthttp.Response
	request  *fasthttp.Request

	*fasthttp.RequestCtx

	next bool

	skipView bool
}

func ReleaseRequestCtx(ctx *Ctx) {
	ctx.RequestCtx = nil
	// ctx.next = false
	// ctx.skipView = false // for future

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
func (ctx *Ctx) Param(key string) string {
	return ctx.paramValues[key]
}

func (c *Ctx) Response() *fasthttp.Response {
	return c.response
}

func (c *Ctx) Method() []byte {
	return c.RequestCtx.Method()

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

func (ctx *Ctx) Next() error {
	ctx.next = true

	return nil
}

func (c *Ctx) Query(key string) []byte {
	r := c.RequestCtx.QueryArgs().Peek(key)

	return r

}

//	func (c *Ctx) JSON(body JSON) {
//		c.response.SetBody(body)
//	}
func (c *Ctx) App() *GoBlaze {
	return c.app
}

// func (c *Ctx) ParseJSON(jsonStr string) (map[string]interface{}, error) {
// 	cstr := C.CString(jsonStr)

// 	C.cJSON_Parse(cstr)

//		return result, nil
//	}
func (ctx *Ctx) HttpResponse(response []byte, statusCode ...int) error {
	ctx.SetContentType("text/html; charset=utf-8")

	if len(statusCode) > 0 {
		ctx.SetStatusCode(statusCode[0])
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}

	ctx.RequestCtx.ResetBody()

	_, err := ctx.RequestCtx.Write(response)
	return err
}
