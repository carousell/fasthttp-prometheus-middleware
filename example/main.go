package main

import (
	"log"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	fasthttpprom "github.com/701search/fasthttp-prometheus-middleware"
)

func main() {

	r := router.New()
	p := fasthttpprom.NewPrometheus("")
	p.Use(r)
	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(`{"status": "pass"}`))
	})

	log.Println("main is listening on ", "8080")
	log.Fatal(fasthttp.ListenAndServe(":"+"8080", r.Handler))
}
