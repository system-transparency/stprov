package network

import (
	"net/http"
	"os"

	"system-transparency.org/stboot/host/network"
)

// NewClient configures an HTTP client in the same way that stboot does.  The
// root file needs to contain one or more X.509 certificates in PEM format.
// These certificates are used as trust anchors while constructing chains.
func NewClient(rootFile string) (http.Client, error) {
	b, err := os.ReadFile(rootFile)
	if err != nil {
		return http.Client{}, err
	}
	crts, err := DecodePEM(b)
	if err != nil {
		return http.Client{}, err
	}
	c := network.NewHTTPClient(crts, false)
	return c.HTTPClient, nil
}
