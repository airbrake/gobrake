package negroni

import (
	"log"
	"net/http"

	"github.com/airbrake/gobrake/v5"
	"github.com/urfave/negroni"
)

// NewMiddleware implements a middleware that can be used in Negroni
// Deprecated: This middleware will be removed in the future release.
func NewMiddleware(n *gobrake.Notifier) negroni.Handler {
	if n == nil {
		return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) { next(w, r) })
	}
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		route := r.URL.Path
		ctx := r.Context()
		ctx, routeMetric := gobrake.NewRouteMetric(ctx, r.Method, route)
		arw := newAirbrakeResponseWriter(w)
		next(arw, r)
		routeMetric.StatusCode = arw.statusCode
		err := n.Routes.Notify(ctx, routeMetric)
		if err != nil {
			log.Println("[airbrake/error]: ", err)
		}
	})
}

type airbrakeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newAirbrakeResponseWriter(w http.ResponseWriter) *airbrakeResponseWriter {
	return &airbrakeResponseWriter{w, http.StatusOK}
}

func (arw *airbrakeResponseWriter) WriteHeader(code int) {
	arw.statusCode = code
	arw.ResponseWriter.WriteHeader(code)
}
