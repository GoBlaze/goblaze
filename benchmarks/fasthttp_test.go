package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func BenchmarkFasthttpServer(b *testing.B) {

	port := 8086

	server := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("Hello, world!")
		},
	}

	go func() {
		if err := server.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
			b.Fatalf("Error while serving: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)

	b.ResetTimer()

	client := &fasthttp.Client{}

	for i := 0; i < b.N; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://localhost:8086/")

		if err := client.Do(req, resp); err != nil {
			b.Fatalf("Error making request: %s", err)
		}

		if string(resp.Body()) != "Hello, world!" {
			b.Errorf("Unexpected response: %s", resp.Body())
		}

		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}

	server.Shutdown()

}
