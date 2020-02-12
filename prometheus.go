package fasthttpprom

import (
	"strconv"
	"time"

	"github.com/fasthttp/router"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var defaultMetricPath = "/metrics"

// RequestCounterURLLabelMappingFn url label
type RequestCounterURLLabelMappingFn func(c *fasthttp.RequestCtx) string

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {
	reqCnt                  *prometheus.CounterVec
	reqDur                  *prometheus.HistogramVec
	router                  *router.Router
	listenAddress           string
	MetricsPath             string
	ReqCntURLLabelMappingFn RequestCounterURLLabelMappingFn
	URLLabelFromContext     string
}

// NewPrometheus generates a new set of metrics with a certain subsystem name
func NewPrometheus(subsystem string) *Prometheus {

	p := &Prometheus{
		MetricsPath: defaultMetricPath,
		ReqCntURLLabelMappingFn: func(c *fasthttp.RequestCtx) string {
			return c.Request.URI().String() // i.e. by default do nothing, i.e. return URI
		},
	}

	p.registerMetrics(subsystem)

	return p
}

// SetListenAddress for exposing metrics on address. If not set, it will be exposed at the
// same address of api that is being used
func (p *Prometheus) SetListenAddress(address string) {
	p.listenAddress = address
	if p.listenAddress != "" {
		p.router = router.New()
	}
}

// SetListenAddressWithRouter for using a separate router to expose metrics. (this keeps things like GET /metrics out of
// your content's access log).
func (p *Prometheus) SetListenAddressWithRouter(listenAddress string, r *router.Router) {
	p.listenAddress = listenAddress
	if len(p.listenAddress) > 0 {
		p.router = r
	}
}

// SetMetricsPath set metrics paths
func (p *Prometheus) SetMetricsPath(r *router.Router) {

	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, prometheusHandler())
		p.runServer()
	} else {
		r.GET(p.MetricsPath, prometheusHandler())
	}
}

func (p *Prometheus) runServer() {
	if p.listenAddress != "" {
		go fasthttp.ListenAndServe(p.listenAddress, p.router.Handler)
	}
}

func (p *Prometheus) registerMetrics(subsystem string) {

	p.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "request_count",
			Help:      "Number of request",
		},
		[]string{"code", "path"},
	)

	p.reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "request latencies",
			Buckets:   []float64{.005, .01, .02, 0.04, .06, 0.08, .1, 0.15, .25, 0.4, .6, .8, 1, 1.5, 2, 3, 5},
		},
		[]string{"code", "path"},
	)

	prometheus.Register(p.reqCnt)
	prometheus.Register(p.reqDur)

}

// Use adds the middleware to a fasthttp
func (p *Prometheus) Use(r *router.Router) {
	p.router = r
	p.SetMetricsPath(r)
	p.HandlerFunc()

}

// HandlerFunc defines handler function for middleware
func (p *Prometheus) HandlerFunc() fasthttp.RequestHandler {
	p.router.GET(p.MetricsPath, prometheusHandler())
	return func(ctx *fasthttp.RequestCtx) {
		if ctx.Request.URI().String() == p.MetricsPath {
			//p.router.Handler(ctx)
			return
		}

		start := time.Now()
		//c.Next()
		p.router.Handler(ctx)

		status := strconv.Itoa(ctx.Response.StatusCode())
		elapsed := float64(time.Since(start)) / float64(time.Second)

		p.reqDur.WithLabelValues(status, string(ctx.Method())+"_"+ctx.URI().String()).Observe(elapsed)
		p.reqCnt.WithLabelValues(status, string(ctx.Method())+"_"+ctx.URI().String()).Inc()
	}
}

func prometheusHandler() fasthttp.RequestHandler {
	return fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
}
