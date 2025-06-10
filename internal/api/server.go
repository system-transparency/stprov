package api

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"system-transparency.org/stprov/internal/secrets"
)

type Server struct {
	ServerConfig
	http.Server

	Entropy   secrets.Entropy             // entropy received from stprov local
	Timestamp int64                       // timestamp received from stprov local
	UDS       *secrets.UniqueDeviceSecret // UDS generated in handleCommit()

	basicAuthPassword string
	commit            chan struct{}
}

type ServerConfig struct {
	Secret     string      // shared secret between stprov local and stprov remote
	RemoteIP   net.IP      // stprov-remote IP address
	RemotePort int         // stprov-remote port
	LocalCIDR  []net.IPNet // where stprov-local may connect from
	HostName   string      // host name to give back to stprov local

	Deadline time.Duration // maximum time to serve an HTTP request
	Timeout  time.Duration // maximum time to wait on a graceful shutdown
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	otp, err := secrets.NewOneTimePassword(cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("derive one-time password: %w", err)
	}
	crt, err := otp.TLSCertificate(cfg.RemoteIP)
	if err != nil {
		return nil, fmt.Errorf("derive tls certificate: %w", err)
	}
	basicAuthPassword, err := otp.BasicAuthPassword()
	if err != nil {
		return nil, fmt.Errorf("derive basic auth password: %w", err)
	}
	srv := &Server{
		ServerConfig: *cfg,
		Server: http.Server{
			Addr: fmt.Sprintf("%s:%d", cfg.RemoteIP, cfg.RemotePort),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					*crt,
				},
			},
		},
		basicAuthPassword: basicAuthPassword,
		commit:            make(chan struct{}, 1),
	}
	return srv, nil
}

func (srv *Server) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	mux := http.NewServeMux()
	http.Handle("/", mux)
	for _, handler := range srv.handlers() {
		path := "/" + Protocol + "/" + handler.Endpoint
		mux.Handle(path, handler)
	}

	wg.Add(1)
	go await(ctx, srv.commit, func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(ctx, srv.Timeout)
		defer cancel()

		srv.Shutdown(ctx)
	})

	defer close(srv.commit)
	if err := srv.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server died: %w", err)
	}
	return nil
}

func (srv *Server) checkHTTPMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodPost
}

func (srv *Server) handlers() []Handler {
	return []Handler{
		{srv, EndpointAddData, http.MethodPost, handleAddData},
		{srv, EndpointAddSecureBoot, http.MethodPost, handleAddSecureBoot},
		{srv, EndpointCommit, http.MethodGet, handleCommit},
	}
}

func await(ctx context.Context, commit chan struct{}, done func()) {
	select {
	case <-commit:
	case <-ctx.Done():
	}
	done()
}
