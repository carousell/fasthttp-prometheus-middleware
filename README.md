# fasthttp prometheus-middleware
Fasthttp [fasthttp](https://github.com/valyala/fasthttp) middleware for Prometheus

Export metrics for request duration ```request_duration_seconds```

## Example 
using fasthttp/router

    package main

    import (
	"log"

	fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
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

Example metrics for above code in /metrics endpoint

```request_duration_seconds_bucket{code="200",path="GET_/health",le="0.005"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.01"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.02"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.04"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.06"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.08"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.1"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.15"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.25"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.4"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.6"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.8"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="1"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="1.5"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="2"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="3"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="5"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="+Inf"} 25063
request_duration_seconds_sum{code="200",path="GET_/health"} 0.14781658099999923
request_duration_seconds_count{code="200",path="GET_/health"} 25063
