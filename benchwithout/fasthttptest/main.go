package main

import (
	"fmt"
	"log"
	"time"

	"github.com/GoBlaze/goblaze/fasthttp"
)

func main() {
	port := 8086

	server := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("Hello, world!")
		},
	}

	go func() {
		if err := server.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
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

		req.SetRequestURI(fmt.Sprintf("http://localhost:%d/", port))

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
