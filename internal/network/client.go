package network

import (
	"net/http"
	"time"

	"system-transparency.org/stboot/host/network"
	"system-transparency.org/stboot/opts"
)

// NewClient configures an HTTP client in the same way that stboot does. The
// root file, if it exists, needs to contain one or more valid X.509
// certificates in PEM format. These certificates are used as trust anchors
// while constructing chains.
func NewClient(rootFile string) (http.Client, error) {
	certs, err := opts.ReadOptionalCertsFile(rootFile, time.Now())
	if err != nil {
		return http.Client{}, err
	}
	c := network.NewHTTPClient(certs, false)
	return c.HTTPClient, nil
}
