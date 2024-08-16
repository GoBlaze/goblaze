package main

import (
	"fmt"
	"log"
	"time"

	"github.com/GoBlaze/goblaze"
	"github.com/GoBlaze/goblaze/fasthttp"
)

func helloHandler(ctx *goblaze.Ctx) error {

	ctx.WriteString("Hello, world!")
	return nil
}

func main() {
	server := goblaze.New()
	server.GET("/", helloHandler)

	go func() {
		err := server.ListenAndServe("localhost", 8083)
		if err != nil {
			log.Fatalf("Error while serving: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)

	client := &fasthttp.Client{}
	start := time.Now()
	numRequests := 100000
	for i := 0; i < numRequests; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://localhost:8083/")

		if err := client.Do(req, resp); err != nil {
			log.Fatalf("Error making request: %s", err)
		}

		if string(resp.Body()) != "Hello, world!" {
			log.Printf("Unexpected response: %s", resp.Body())
		}

		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}
	duration := time.Since(start)
	server.Shutdown()

	fmt.Printf("Completed %d requests in %s\n", numRequests, duration)
	fmt.Printf("Average time per request: %s\n", duration/time.Duration(numRequests))
}
