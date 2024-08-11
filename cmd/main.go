package main

// #cgo CFLAGS: -I../ccode
// #cgo LDFLAGS: -L../ccode/ -lother
// #include <stdlib.h>
// #include <other.h>
import (
	"C"
)
import (
	"log"
	"os"
	"runtime"

	"github.com/GoBlaze/goblaze"
	"github.com/GoBlaze/goblaze/fasthttp"

	"github.com/sirupsen/logrus"
)

func helloHandler(ctx *goblaze.Ctx) error {
	if err := ctx.HttpResponse([]byte(` 
	<!DOCTYPE html>
	<html>
	<head>
		<title>Title</title>
	</head>
	<body>
		<h1>Hello, goblaze!</h1>
	</body>
	</html>
	`)); err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

// func ParseJson(ctx *goblaze.Ctx) error {
//	ctx.ParseJSON("fjafjajf")

//	return nil
// }

func helloHandler2(ctx *goblaze.Ctx) error {
	if _, err := ctx.Write([]byte("<h1>lol</h1>")); err != nil {
		if err != nil {
			logrus.Error(err)
			return err
		}
	}

	SetResponse(ctx)

	return nil
}

func SetResponse(ctx *goblaze.Ctx) error {
	ctx.Redirect("/youGay")
	return nil
}

func YouGay(ctx *goblaze.Ctx) error {
	if err := ctx.HttpResponse([]byte("<h1>lol</h1>")); err != nil {
		if err != nil {
			logrus.Error(err)
			return err
		}
	}

	return nil
}

func UsingParams(ctx *goblaze.Ctx) error {
	paramName := ctx.Param("name")
	if paramName == "" {
		log.Printf("Param 'name' not found in request")
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte("Missing 'name' parameter"))
		return nil
	}

	log.Printf("Param 'name' = %s", paramName)
	ctx.HttpResponse([]byte(paramName))

	return nil
}
func main() {
	if runtime.GOMAXPROCS(runtime.NumCPU()) == 1 {
		logrus.Fatal("Please use a system with more than 1 CPU, You're fucking poor, bro....", nil)
		logrus.Error("GOMAXPROCS must be greater than 1")
		os.Exit(1)
	}
	result := C.add(1, 2)
	logrus.Errorf("Result: %d\n", result)

	// ctx := goblaze.NewJSONContext()

	// ctx.AddString("first", "Yuki")
	// ctx.AddString("second", "Nagato")

	// jsonOutput := ctx.Print()
	// logrus.Errorf("Result: %s\n", jsonOutput)

	// jsonString := `{"toilet": "lol", "nineplusten": 21}`
	// parsedCtx := goblaze.ParseJSON(jsonString)

	// defer parsedCtx.Delete()

	// name := parsedCtx.GetString("toilet")
	// logrus.Errorf("%s\n", name)

	// age := parsedCtx.GetInt("nineplusten")
	// logrus.Errorf("%d\n", age)

	server := goblaze.New()

	server.GET("/", helloHandler)
	// server.GET("/json", ParseJson)
	server.GET("/hello", helloHandler2)
	server.GET("/youGay", YouGay)
	server.GET("/usingParams/:name", UsingParams)

	// server.Use(middleware.DefaultLogger)
	log.Println("Listening on: http://localhost:8080")

	server.ListenAndServe("localhost", 8080, "info")

	defer server.Shutdown()

}
