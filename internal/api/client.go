package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	stauth "git.glasklar.is/system-transparency/core/stauth/cmd"
	"git.glasklar.is/system-transparency/core/stauth/enrollv0"
	"system-transparency.org/stprov/internal/secrets"
)

type ClientConfig struct {
	Secret     string // shared secret between stprov local and stprov remote
	RemoteIP   net.IP // IP address used by stprov remote
	RemotePort int    // port used by stprov remote
}

type Client struct {
	ClientConfig
	http.Client

	basicAuthPassword string
	serverURL         string
}

func NewClient(cfg *ClientConfig) (*Client, error) {
	otp, err := secrets.NewOneTimePassword(cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("derive one-time password: %w", err)
	}
	crt, err := otp.X509Certificate(cfg.RemoteIP)
	if err != nil {
		return nil, fmt.Errorf("derive x509 certificate: %w", err)
	}
	basicAuthPassword, err := otp.BasicAuthPassword()
	if err != nil {
		return nil, fmt.Errorf("derive basic auth password: %w", err)
	}

	crtPool := x509.NewCertPool()
	crtPool.AddCert(crt)
	return &Client{
		ClientConfig: *cfg,
		Client: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    crtPool,
					ServerName: secrets.DummyServerName,
				},
			},
		},
		basicAuthPassword: basicAuthPassword,
		serverURL:         fmt.Sprintf("https://%s:%d/%s/", cfg.RemoteIP, cfg.RemotePort, Protocol),
	}, nil
}

func (c *Client) AddData() (*AddDataRequest, error) {
	data, err := NewAddDataRequest()
	if err != nil {
		return nil, fmt.Errorf("create data: %w", err)
	}
	if _, err := c.doPost(c.serverURL+EndpointAddData, data); err != nil {
		return nil, fmt.Errorf("post data: %w", err)
	}
	return data, nil
}

func (c *Client) Commit() (*CommitResponse, error) {
	b, err := c.doGet(c.serverURL + EndpointCommit)
	if err != nil {
		return nil, fmt.Errorf("send commit: %w", err)
	}
	var cr CommitResponse
	if err := json.Unmarshal(b, &cr); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &cr, nil
}

func (c *Client) ProvisionAIK() ([]byte, error) {
	ctx := context.Background()

	platformUrl, err := url.Parse(fmt.Sprintf("https://%s:%d", c.RemoteIP, c.RemotePort))
	if err != nil {
		return nil, err
	}

	// get AIK, EK, and SRK
	req, err := stauth.EnrollOperatorRequest(ctx, &c.Client, platformUrl)
	if err != nil {
		return nil, err
	}

	// send AIK credential encrypted with EK
	ch, nonce, err := enrollv0.NewChallenge(req)
	if err != nil {
		return nil, err
	}
	ans, err := stauth.EnrollOperatorChallenge(ctx, &c.Client, platformUrl, nonce, ch)
	if err != nil {
		return nil, err
	}

	// write out platform data
	nameHint := req.UxIdentity
	if nameHint == "" {
		nameHint = c.RemoteIP.String()
	}

	return stauth.EnrollOperatorFinish(ctx, nameHint, req, ans)
}

func (c *Client) doGet(endpointURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.SetBasicAuth(BasicAuthUser, c.basicAuthPassword)

	rsp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(rsp.StatusCode))
	}
	return io.ReadAll(rsp.Body)
}

func (c *Client) doPost(endpointURL string, i interface{}) ([]byte, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpointURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.SetBasicAuth(BasicAuthUser, c.basicAuthPassword)

	rsp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(rsp.StatusCode))
	}
	return io.ReadAll(rsp.Body)
}
