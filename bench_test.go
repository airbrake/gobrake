package gobrake_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	notice := notifier.Notice(errors.New("benchmark"), nil, 0)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
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

	tm, err := time.Parse(time.RFC3339, "2018-01-01T00:00:00Z")
	if err != nil {
		b.Fatal(err)
	}

	const n = 100
	reqs := make([]*gobrake.RouteInfo, n)
	for i := 0; i < n; i++ {
		reqs[i] = &gobrake.RouteInfo{
			Method:     "GET",
			Route:      fmt.Sprintf("/api/v4/groups/%d", i),
			StatusCode: 200,
			Start:      tm,
			End:        tm.Add(123 * time.Millisecond),
		}
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
