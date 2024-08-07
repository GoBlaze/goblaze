package main

// #cgo CFLAGS: -I../ccode
// #cgo LDFLAGS: -L../ccode/ -lother
// #include <stdlib.h>
// #include <other.h>
import (
	"C"
)
import (
	"errors"
	"fmt"

	"github.com/GoBlaze/goblaze"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func IsGet(ctx *goblaze.Ctx) (int, error) {
	logrus.Info("This middleware launch an error...")
	return fasthttp.StatusBadRequest, errors.New("Fake error")
}

func helloHandler(ctx *goblaze.Ctx, params goblaze.Params) error {
	ctx.GoBlaze.HttpResponse(ctx, []byte("<h1>goblaze Framework yeeeeeeeeee</h1>"))

	return nil
}

func main() {
	result := C.add(1, 2)
	fmt.Printf("Result: %d\n", result)

	server := goblaze.New()

	server.GET("/", helloHandler)
	server.Use(IsGet)

	server.ListenAndServe("localhost", 8080)
}
