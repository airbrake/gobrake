package gobrake_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airbrake/gobrake"
)

func BenchmarkSendNotice(b *testing.B) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"123"}`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:  1,
		ProjectKey: "key",
		Host:       server.URL,
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		notice := notifier.Notice(errors.New("benchmark"), nil, 0)
		for pb.Next() {
			id, err := notifier.SendNotice(notice)
			if err != nil {
				b.Fatal(err)
			}
			if id != "123" {
				b.Fatalf("got %q, wanted 123", id)
			}
		}
	})
}

func BenchmarkNotifyRequest(b *testing.B) {
	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:  1,
		ProjectKey: "",
	})

	const n = 100
	reqs := make([]*gobrake.RouteTrace, n)
	for i := 0; i < n; i++ {
		_, trace := gobrake.NewRouteTrace(nil, "GET", fmt.Sprintf("/api/v4/groups/%d", i))
		reqs[i] = trace
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			err := notifier.Routes.Notify(nil, reqs[i%n])
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
