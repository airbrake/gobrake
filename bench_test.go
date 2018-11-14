package gobrake_test

import (
	"errors"
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

	info := &gobrake.RequestInfo{
		Method:     "GET",
		Route:      "/api/v4/groups",
		StatusCode: 200,
		Start:      tm,
		End:        tm.Add(123 * time.Millisecond),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := notifier.NotifyRequest(info)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
