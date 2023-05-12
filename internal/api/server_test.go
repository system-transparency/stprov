package api

import (
	"context"
	"net/http"
	"testing"
)

func TestHTTPMethod(t *testing.T) {
	var server Server
	for _, table := range []struct {
		wantOK bool
		method string
	}{
		{false, http.MethodHead},
		{true, http.MethodPost},
		{true, http.MethodGet},
	} {
		ok := server.checkHTTPMethod(table.method)
		if got, want := ok, table.wantOK; got != want {
			t.Errorf("%s: got %v but wanted %v", table.method, got, want)
		}
	}
}

func TestHandlers(t *testing.T) {
	endpoints := map[string]bool{
		EndpointAddData: false,
		EndpointCommit:  false,
	}
	srv := Server{}
	for _, handler := range srv.handlers() {
		if _, ok := endpoints[handler.Endpoint]; !ok {
			t.Errorf("got unexpected endpoint: %s", handler.Endpoint)
		}
		endpoints[handler.Endpoint] = true
	}
	for endpoint, ok := range endpoints {
		if !ok {
			t.Errorf("endpoint %s is not configured", endpoint)
		}
	}
}

func TestAwait(t *testing.T) {
	ch := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var i int
	done := func() { i += 1 }

	ch <- struct{}{}
	await(ctx, ch, done)

	cancel()
	await(ctx, ch, done)

	if got, want := i, 2; got != want {
		t.Errorf("got %d done() invocations but wanted %d", got, want)
	}
}
