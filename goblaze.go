package goblaze

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var colors = &DefaultColors

type Middleware Handler

type GoBlaze struct {
	noCopy No // nolint:structcheck,unused
	// stack      [][]*Router
	middleware []Middleware
	server     *fasthttp.Server
	router     *Router
	log        *logrusLogger
}

func New() *GoBlaze {
	log := NewLog()

	router := NewRouter()

	server := &GoBlaze{
		router:     router,
		middleware: []Middleware{},

		server: &fasthttp.Server{
			Handler: router.ServeHTTP,
			Name:    "goblaze",
		},
		log: log,
	}

	return server
}

// ListenAndServe starts the server.
func (server *GoBlaze) ListenAndServe(host string, port int, logLevel ...string) error {
	// log.Printf("%s%s%s", colors.Green, goblazeW, colors.Reset)

	addr := fmt.Sprintf("%s:%d", host, port)

	// server.log.Printf("Listening on: http://%s/", addr)

	if len(logLevel) > 0 {
		server.log.SetLevel(logrus.Level(logrus.DebugLevel))
	}

	if err := server.server.ListenAndServe(addr); err != nil {
		server.log.Fatalf("Server error: %v", err)
	}

	// go server.printRoutesMessage()

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

func (server *GoBlaze) Shutdown() error {
	return server.server.Shutdown()
}

// func (app *GoBlaze) printRoutesMessage() {

// 	var routes []RouteMessage
// 	for _, routeStack := range app.stack {
// 		for _, route := range routeStack {
// 			var newRoute RouteMessage
// 			newRoute.name = route.Name
// 			newRoute.method = route.Method
// 			newRoute.path = route.Path
// 			for _, handler := range route.Handlers {
// 				newRoute.handlers += runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name() + " "
// 			}
// 			routes = append(routes, newRoute)
// 		}
// 	}

// 	out := colorable.NewColorableStdout()
// 	if os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") == "1" || (!isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd())) {
// 		out = colorable.NewNonColorable(os.Stdout)
// 	}

// 	w := tabwriter.NewWriter(out, 1, 1, 1, ' ', 0)
// 	// Sort routes by path
// 	sort.Slice(routes, func(i, j int) bool {
// 		return routes[i].path < routes[j].path
// 	})

// 	fmt.Fprintf(w, "%smethod\t%s| %spath\t%s| %sname\t%s| %shandlers\t%s\n", colors.Blue, colors.White, colors.Green, colors.White, colors.Cyan, colors.White, colors.Yellow, colors.Reset)
// 	fmt.Fprintf(w, "%s------\t%s| %s----\t%s| %s----\t%s| %s--------\t%s\n", colors.Blue, colors.White, colors.Green, colors.White, colors.Cyan, colors.White, colors.Yellow, colors.Reset)

// 	for _, route := range routes {
// 		fmt.Fprintf(w, "%s%s\t%s| %s%s\t%s| %s%s\t%s| %s%s%s\n", colors.Blue, route.method, colors.White, colors.Green, route.path, colors.White, colors.Cyan, route.name, colors.White, colors.Yellow, route.handlers, colors.Reset)
// 	}

// 	_ = w.Flush()
// }
