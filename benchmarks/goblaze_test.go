package benchmarks

import (
	"testing"
	"time"

	"github.com/GoBlaze/goblaze"
	"github.com/GoBlaze/goblaze/fasthttp"
)

func helloHandler(ctx *goblaze.Ctx) error {

	ctx.WriteString("Hello, world!")
	return nil
}

func BenchmarkGoBlazeServer(b *testing.B) {

	server := goblaze.New()
	server.GET("/", helloHandler)

	go func() {
		err := server.ListenAndServe("localhost", 8083)
		if err != nil {
			b.Fatalf("Error while serving: %s", err)
		}
	}()

	time.Sleep(1 * time.Second)

	b.ResetTimer()

	client := &fasthttp.Client{}

	for i := 0; i < b.N; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://localhost:8083/")

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
