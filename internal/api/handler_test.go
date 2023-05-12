package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"system-transparency.org/stprov/internal/secrets"
)

func TestVerifyMethod(t *testing.T) {
	badMethod := http.MethodHead
	srv := Server{}
	for _, handler := range srv.handlers() {
		for _, method := range []string{
			http.MethodGet,
			http.MethodPost,
			badMethod,
		} {
			url := "http://example.com/" + Protocol + "/" + handler.Endpoint
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				t.Fatalf("create http request: %v", err)
			}

			w := httptest.NewRecorder()
			ok := handler.verifyMethod(w, req)
			if got, want := ok, handler.Method == method; got != want {
				t.Errorf("%s %s: got %v but wanted %v: %v", method, url, got, want, err)
				continue
			}
			if ok {
				continue
			}

			if method == badMethod {
				if got, want := w.Code, http.StatusBadRequest; got != want {
					t.Errorf("%s %s: got status %d, wanted %d", method, url, got, want)
				}
				if _, ok := w.Header()["Allow"]; ok {
					t.Errorf("%s %s: got Allow header, wanted none", method, url)
				}
				continue
			}

			if got, want := w.Code, http.StatusMethodNotAllowed; got != want {
				t.Errorf("%s %s: got status %d, wanted %d", method, url, got, want)
			} else if methods, ok := w.Header()["Allow"]; !ok {
				t.Errorf("%s %s: got no allow header, expected one", method, url)
			} else if got, want := len(methods), 1; got != want {
				t.Errorf("%s %s: got %d allowed method(s), wanted %d", method, url, got, want)
			} else if got, want := methods[0], handler.Method; got != want {
				t.Errorf("%s %s: got allowed method %s, wanted %s", method, url, got, want)
			}
		}
	}
}

func TestVerifyNetwork(t *testing.T) {
	srv := testServer(t)
	defer close(srv.commit)
	for _, handler := range srv.handlers() {
		url := "http://example.com/" + Protocol + "/" + handler.Endpoint
		req, err := http.NewRequest(handler.Method, url, nil)
		if err != nil {
			t.Fatalf("create http request: %v", err)
		}

		for _, table := range []struct {
			desc string
			addr string
		}{
			{"malformed (port)", "127.0.0.12"},
			{"malformed (ip)", "127.0.0:"},
			{"not allowed ip", "127.0.0.128:2009"},
			{"not allowed ip", "10.0.0.128:2009"},
			{"valid", "10.0.0.12:2009"},
			{"valid", "127.0.0.12:2009"},
		} {
			req.RemoteAddr = table.addr
			w := httptest.NewRecorder()
			ok := handler.verifyNetwork(w, req)
			if got, want := ok, table.desc == "valid"; got != want {
				t.Errorf("%s: got %v but wanted %v", table.desc, got, want)
			}
		}
	}
}

func TestAuthenticateUser(t *testing.T) {
	srv := testServer(t)
	defer close(srv.commit)
	for _, handler := range srv.handlers() {
		url := "http://example.com/" + Protocol + "/" + handler.Endpoint
		req, err := http.NewRequest(handler.Method, url, nil)
		if err != nil {
			t.Fatalf("create http request: %v", err)
		}

		for _, table := range []struct {
			desc string
			pw   string
		}{
			{"no password", ""},
			{"bad password", "hotdog"},
			{"valid", srv.basicAuthPassword},
		} {
			if table.pw != "" {
				req.SetBasicAuth(BasicAuthUser, table.pw)
			}

			w := httptest.NewRecorder()
			ok := handler.authenticateUser(w, req)
			if got, want := ok, table.desc == "valid"; got != want {
				t.Errorf("%s: got %v but wanted %v", table.desc, got, want)
			}
		}
	}
}

func TestAddData(t *testing.T) {
	srv := testServer(t)
	defer close(srv.commit)
	handler := getHandler(t, srv, EndpointAddData)
	for _, table := range []struct {
		desc string
		body io.Reader
	}{
		{"bad json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"entropy":"%s","timestamp":1`, b64Ones(t, secrets.EntropyBytes))))},
		{"no entropy", bytes.NewBuffer([]byte(fmt.Sprintf(`{"timestamp":1}`)))},
		{"bad entropy", bytes.NewBuffer([]byte(fmt.Sprintf(`{"entropy":"%s","timestamp":1}`, b64Ones(t, secrets.EntropyBytes+1))))},
		{"bad timestamp", bytes.NewBuffer([]byte(fmt.Sprintf(`{"entropy":"%s","timestamp":-1}`, b64Ones(t, secrets.EntropyBytes))))},
		{"valid", bytes.NewBuffer([]byte(fmt.Sprintf(`{"entropy":"%s","timestamp":1}`, b64Ones(t, secrets.EntropyBytes))))},
	} {
		url := "http://example.com/" + Protocol + "/" + handler.Endpoint
		req, err := http.NewRequest(handler.Method, url, table.body)
		if err != nil {
			t.Fatalf("create http request: %v", err)
		}
		req.RemoteAddr = "127.0.0.12:2009"
		req.SetBasicAuth(BasicAuthUser, srv.basicAuthPassword)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if got, want := w.Code == http.StatusOK, table.desc == "valid"; got != want {
			t.Errorf("%s: got http status code %d", table.desc, w.Code)
		}
		if w.Code != http.StatusOK {
			continue
		}

		if got, want := srv.Timestamp, int64(1); got != want {
			t.Errorf("%s: got timestamp %d but wanted %d", table.desc, got, want)
		}
		if got, want := srv.Entropy[:], bytes.Repeat([]byte{0xff}, secrets.EntropyBytes); !bytes.Equal(got, want) {
			t.Errorf("%s: got entropy\n%v\nbut wanted\n%v", table.desc, got, want)
		}
	}
}

func TestCommit(t *testing.T) {
	srv := testServer(t)
	defer close(srv.commit)
	handler := getHandler(t, srv, EndpointCommit)

	url := "http://example.com/" + Protocol + "/" + handler.Endpoint
	req, err := http.NewRequest(handler.Method, url, nil)
	if err != nil {
		t.Fatalf("create http request: %v", err)
	}
	req.RemoteAddr = "127.0.0.12:2009"
	req.SetBasicAuth(BasicAuthUser, srv.basicAuthPassword)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got, want := w.Code, http.StatusOK; got != want {
		t.Errorf("got http status code %d but wanted %d", got, want)
	}
	select {
	case <-srv.commit:
	default:
		t.Errorf("missing commit message")
	}
}

func getHandler(t *testing.T, srv *Server, endpoint string) Handler {
	t.Helper()
	for _, handler := range srv.handlers() {
		if handler.Endpoint == endpoint {
			return handler
		}
	}

	t.Fatalf("unknown endpoint %s", endpoint)
	return Handler{} // make compiler happy
}

func b64Ones(t *testing.T, numBytes int) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, numBytes))
}
