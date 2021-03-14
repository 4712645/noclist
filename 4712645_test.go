package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

//TODO: Add backoff tests

func TestRetryingClient(t *testing.T) {
	requestCount := 0

	// Our bad server only responds properly on the third try
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {

			// Simulate a dropped connection.
			// Abort such that the client sees an interrupted response but the server doesn't log an error
			// https://golang.org/pkg/net/http/#Handler
			panic(http.ErrAbortHandler)
		}
		if requestCount == 2 {

			// Any non-200 response should be considered failure, including e.g. 202
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer badServer.Close()

	client := &RetryingClient{
		client: &http.Client{
			Timeout: time.Millisecond * 500,
		},
		MaxRetries: 2,
		Backoff:    0,
	}

	badServerReq, err := http.NewRequest("GET", badServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(badServerReq)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("client did not retry the configured number of times")
	}
	if requestCount != 3 {
		t.Fatalf("unexpected number of requests made, want 3 got %d", requestCount)
	}
}
