package goblaze

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Middleware func(next Handler) Handler

type GoBlaze struct {
	server     *fasthttp.Server
	router     *Router
	log        *logrusLogger
	middleware []Middleware
}

func New() *GoBlaze {
	log := NewLog()

	router := NewRouter()

	server := &GoBlaze{
		router:     router,
		middleware: []Middleware{},

		server: &fasthttp.Server{
			Handler: router.ServeHTTP,

			Name: "goblaze",
		},
		log: log,
	}

	return server
}

// ListenAndServe starts the server.
func (server *GoBlaze) ListenAndServe(host string, port int, logLevel ...string) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	server.log.Printf("Listening on: http://%s/", addr)

	if len(logLevel) > 0 {
		server.log.SetLevel(logrus.Level(logrus.TraceLevel))
	}

	if err := server.server.ListenAndServe(addr); err != nil {
		server.log.Fatalf("Server error: %v", err)
	}

	return nil
}

// HTTP method shortcuts.
func (server *GoBlaze) GET(path string, handle Handler)     { server.router.GET(path, handle) }
func (server *GoBlaze) POST(path string, handle Handler)    { server.router.POST(path, handle) }
func (server *GoBlaze) PUT(path string, handle Handler)     { server.router.PUT(path, handle) }
func (server *GoBlaze) PATCH(path string, handle Handler)   { server.router.PATCH(path, handle) }
func (server *GoBlaze) DELETE(path string, handle Handler)  { server.router.DELETE(path, handle) }
func (server *GoBlaze) OPTIONS(path string, handle Handler) { server.router.OPTIONS(path, handle) }
func (server *GoBlaze) HEAD(path string, handle Handler)    { server.router.HEAD(path, handle) }

// Use adds middleware to the server.
func (g *GoBlaze) Use(handlers ...interface{}) *Router {
	for _, handler := range handlers {
		switch h := handler.(type) {
		case Middleware:
			g.middleware = append(g.middleware, h)
		}
	}

	return g.router
}

func (server *GoBlaze) HttpResponse(ctx *Ctx, response []byte, statusCode ...int) error {
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
