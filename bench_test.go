package gobrake_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airbrake/gobrake/v4"
)

func BenchmarkSendNotice(b *testing.B) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"id":"123"}`))
		if err != nil {
			panic(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	configServer := newConfigServer()

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:        1,
		ProjectKey:       "key",
		Host:             server.URL,
		RemoteConfigHost: configServer.URL,
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

func BenchmarkRoutesNotify(b *testing.B) {
	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:  1,
		ProjectKey: "",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			_, metric := gobrake.NewRouteMetric(context.TODO(), "GET", fmt.Sprintf("/api/v4/groups/%d", i))
			err := notifier.Routes.Notify(context.TODO(), metric)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}
