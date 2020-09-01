package gobrake_test

import (
	"net/http"
	"net/http/httptest"
	"sync"

	. "github.com/onsi/ginkgo"

	"github.com/airbrake/gobrake/v5"
)

var _ = Describe("Notifier", func() {
	var notifier *gobrake.Notifier

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(`{"id":"123"}`))
			if err != nil {
				panic(err)
			}
		}
		server := httptest.NewServer(http.HandlerFunc(handler))
		configServer := newConfigServer()

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
		})
	})

	AfterEach(func() {
		cleanupConfig()
	})

	It("is race free", func() {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				notifier.Notify("hello", nil)
			}()
		}
		wg.Wait()

		notifier.Flush()
	})
})
