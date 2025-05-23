package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"system-transparency.org/stprov/internal/sb"
	"system-transparency.org/stprov/internal/secrets"
)

// Handler implements the http.Handler interface
type Handler struct {
	Server      *Server
	Endpoint    string
	Method      string
	HandlerFunc func(context.Context, *Server, http.ResponseWriter, *http.Request) (int, error)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(h.Server.Deadline))
	defer cancel()

	if ok := h.verifyMethod(w, r); !ok {
		return
	}
	if ok := h.verifyNetwork(w, r); !ok {
		return
	}
	if ok := h.authenticateUser(w, r); !ok {
		return
	}

	if code, err := h.HandlerFunc(ctx, h.Server, w, r); err != nil {
		http.Error(w, http.StatusText(code), code)
	}
}

// verifyMethod checks that an appropriate HTTP method is used.  Error handling
// is based on RFC 7231, see Sections 6.5.5 (Status 405) and 6.5.1 (Status 400).
func (h Handler) verifyMethod(w http.ResponseWriter, r *http.Request) bool {
	if h.Method == r.Method {
		return true
	}

	code := http.StatusBadRequest
	if ok := h.Server.checkHTTPMethod(r.Method); ok {
		w.Header().Set("Allow", h.Method)
		code = http.StatusMethodNotAllowed
	}

	log.Printf("unexpected http method %s", r.Method)
	http.Error(w, fmt.Sprintf("%s", http.StatusText(code)), code)
	return false
}

// verifyNetwork enforces that the client connects from an expected CIDR range
func (h Handler) verifyNetwork(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("failed parsing request address %s", r.RemoteAddr)
		http.Error(w, "Malformed address:port format", http.StatusInternalServerError)
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		log.Printf("failed parsing request ip %s", r.RemoteAddr)
		http.Error(w, "Hostname must be an IP address", http.StatusInternalServerError)
		return false
	}
	for _, allowedNet := range h.Server.LocalCIDR {
		if allowedNet.Contains(ip) {
			return true
		}
	}
	log.Printf("blocked connection attempt from %s", r.RemoteAddr)
	http.Error(w, "Invalid IP address", http.StatusForbidden)
	return false
}

// authenticateUser enforces basic auth as defined in RFC 2617, Section 2.
func (h Handler) authenticateUser(w http.ResponseWriter, r *http.Request) bool {
	user, password, ok := r.BasicAuth()
	if !ok {
		log.Printf("request without basic auth header from %s", r.RemoteAddr)
		http.Error(w, "BasicAuth header is required", http.StatusForbidden)
		return false
	}
	if user != BasicAuthUser || password != h.Server.basicAuthPassword {
		log.Printf("unauthorized user %q and password %q from %s", user, password, r.RemoteAddr)
		http.Error(w, "BasicAuth credentials were insufficient", http.StatusForbidden)
		return false
	}
	return true
}

func handleAddData(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	var data AddDataRequest
	if err := unpackPost(r, &data); err != nil {
		log.Printf("invalid add-entropy request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}
	if got, want := len(data.Entropy), secrets.EntropyBytes; got != want {
		log.Printf("invalid add-data request from %s: wrong number of entropy bytes", r.RemoteAddr)
		return http.StatusBadRequest, fmt.Errorf("invalid number of entropy bytes: %d", got)
	}
	if got := data.Timestamp; got < 0 {
		log.Printf("invalid add-data request from %s: negative timestamp value", r.RemoteAddr)
		return http.StatusBadRequest, fmt.Errorf("invalid unix timestamp %d", got)
	}

	s.Timestamp = data.Timestamp
	copy(s.Entropy[:], data.Entropy)
	return http.StatusOK, nil
}

func handleAddSecureBoot(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	ok, err := sb.IsSetupMode()
	if err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("not in Secure Boot setup mode")
		}
		log.Printf("invalid add-secure boot request from %s: %v", r.RemoteAddr, err)
		return http.StatusForbidden, err
	}

	var data AddSecureBootRequest
	if err := unpackPost(r, &data); err != nil {
		log.Printf("invalid add-secure-boot request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}
	if err := sb.Provision(data.PK, data.KEK, data.Db, data.Dbx); err != nil {
		log.Printf("failed to provision secure boot request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

func handleCommit(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	uds, err := secrets.NewUniqueDeviceSecret(&s.Entropy)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("new unique device secret: %w", err)
	}
	cr, err := NewCommitResponse(uds, s.HostName)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("new commit response: %w", err)
	}
	b, err := json.Marshal(cr)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("marshal commit response: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("write commit response: %w", err)
	}

	s.UDS = uds
	s.commit <- struct{}{}
	return http.StatusOK, nil
}

func unpackPost(req *http.Request, any interface{}) error {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, any)
}
