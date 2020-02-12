# fasthttp prometheus-middleware
Fasthttp [fasthttp](https://github.com/valyala/fasthttp) middleware for Prometheus

Export metrics for request duration ```request_duration``` and request count ```request_count```

## Example 
using fasthttp/router

    package main

    import (
	"log"

	fasthttpprom "github.com/701search/fasthttp-prometheus-middleware"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	)

    func main() {

     	r := router.New()
	p := fasthttpprom.NewPrometheus("")
	p.Use(r)

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(`{"status": "pass"}`))
		log.Println(string(ctx.Request.URI().Path()))
	})

	log.Println("main is listening on ", "8080")
	log.Fatal(fasthttp.ListenAndServe(":"+"8080", p.Handler))
	
    }
