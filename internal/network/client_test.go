package network

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	//
	// Setup a local HTTPS server where the served root certificate is included
	// in a temporary PEM file that contains multiple trust anchors.
	//
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rootFile, err := os.CreateTemp("", "stprov-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(rootFile.Name())
	if _, err := rootFile.Write(bytes.Join([][]byte{
		testPEMCertificate(t),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.TLS.Certificates[0].Certificate[0]}),
		testPEMCertificate(t),
	}, nil)); err != nil {
		t.Fatal(err)
	}

	//
	// Run tests
	//
	t.Run("invalid: file does not exist", func(t *testing.T) {
		if _, err := NewClient("stprov.client_test.go.pem"); err == nil {
			t.Errorf("non-existing file accepted")
		}
	})
	t.Run("invalid: no certificate in file", func(t *testing.T) {
		rootFile, err := os.CreateTemp("", "stprov-*.pem")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(rootFile.Name())

		if _, err = NewClient(rootFile.Name()); err == nil {
			t.Errorf("existing file without any certificate accepted")
		}
	})
	t.Run("invalid: HEAD request local HTTPS server with unconfigured trust root", func(t *testing.T) {
		rootFile, err := os.CreateTemp("", "stprov-*.pem")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(rootFile.Name())
		if _, err := rootFile.Write(testPEMCertificate(t)); err != nil {
			t.Fatal(err)
		}

		cli, err := NewClient(rootFile.Name())
		if err != nil {
			t.Errorf("create new client: %v", err)
			return
		}
		rsp, err := cli.Head(srv.URL)
		if err == nil && rsp.StatusCode == http.StatusOK {
			t.Errorf("HEAD request succeeded with invalid trust root")
		}
	})
	t.Run("valid: HEAD request local HTTPS server", func(t *testing.T) {
		cli, err := NewClient(rootFile.Name())
		if err != nil {
			t.Errorf("create new client: %v", err)
			return
		}
		rsp, err := cli.Head(srv.URL)
		if err != nil {
			t.Errorf("HTTP HEAD: %v", err)
		} else if got, want := rsp.StatusCode, http.StatusOK; got != want {
			t.Errorf("got status code %d, wanted %d", got, want)
		}
	})
}

func testPEMCertificate(t *testing.T) []byte {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-123 * time.Minute),
		NotAfter:              time.Now().Add(123 * time.Minute),
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	b, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: b})
}
