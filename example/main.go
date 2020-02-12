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
