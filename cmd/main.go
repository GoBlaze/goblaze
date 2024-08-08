package main

// #cgo CFLAGS: -I../ccode
// #cgo LDFLAGS: -L../ccode/ -lother
// #include <stdlib.h>
// #include <other.h>
import (
	"C"
)
import (
	"fmt"

	"github.com/GoBlaze/goblaze"
	"github.com/GoBlaze/goblaze/middleware"
	"github.com/sirupsen/logrus"
)

func helloHandler(ctx *goblaze.Ctx) error {
	if err := ctx.App.HttpResponse(ctx, []byte("<h1>goblaze Framework yeeeeeeeeee</h1>")); err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

func helloHandler2(ctx *goblaze.Ctx) error {
	if err := ctx.App.HttpResponse(ctx, []byte("<h1>lol</h1>")); err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

func main() {
	result := C.add(1, 2)
	fmt.Printf("Result: %d\n", result)

	server := goblaze.New()

	server.GET("/", helloHandler)
	server.GET("/hello", helloHandler2)

	server.Use(middleware.DefaultLogger)

	server.ListenAndServe("localhost", 8080, "info")
}
