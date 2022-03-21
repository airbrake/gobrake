package gorilla

import (
	"net/http"

	"github.com/airbrake/gobrake/v5"
	"github.com/gorilla/mux"
)

type airbrakeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (arw *airbrakeResponseWriter) WriteHeader(code int) {
	arw.statusCode = code
	arw.ResponseWriter.WriteHeader(code)
}

// New returns a function that satisfies mux.MiddlewareFunc interface
// It can be used with Use() methods.
func New(notifier *gobrake.Notifier, next http.Handler) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			matchedRoute, _ := mux.CurrentRoute(r).GetPathTemplate()
			ctx, routeMetric := gobrake.NewRouteMetric(ctx, r.Method, matchedRoute)
			arw := newAirbrakeResponseWriter(w)

			next.ServeHTTP(arw, r)
			routeMetric.StatusCode = arw.statusCode
			_ = notifier.Routes.Notify(ctx, routeMetric)
		})
	}
}

func newAirbrakeResponseWriter(w http.ResponseWriter) *airbrakeResponseWriter {
	// Returns 200 OK if WriteHeader isn't called
	return &airbrakeResponseWriter{w, http.StatusOK}
}
