// package secrets provides utilities to derive short-term and long-term
// secrets.  These secrets are used during and after platform provisioning.
package secrets

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"time"

	"golang.org/x/crypto/hkdf"

	"system-transparency.org/stprov/internal/ssh"
)

// Entropy is a buffer storing 256 bits of entropy
type Entropy [EntropyBytes]byte

const EntropyBytes = 32 // 256 bits

// NewEntropy generates entropy using crypto/rand
func NewEntropy() (*Entropy, error) {
	var entropy Entropy
	_, err := io.ReadFull(rand.Reader, entropy[:])
	return &entropy, err
}

// Reader generates randomness using HKDF with SHA256 as the hash function
func Reader(secret []byte, label string, context uint) io.Reader {
	c := fmt.Sprintf("%d", context)
	l := fmt.Sprintf("stprov:%s", label)
	return hkdf.New(sha256.New, secret, []byte(c), []byte(l))
}

// UniqueDeviceSecret is secret used to derive other long-term secrets
type UniqueDeviceSecret Entropy

// NewUniqueDeviceSecret generates a unique device secret by mixing entropy from
// an external and an internal source
func NewUniqueDeviceSecret(ext *Entropy) (*UniqueDeviceSecret, error) {
	loc, err := NewEntropy()
	if err != nil {
		return nil, err
	}

	uds := UniqueDeviceSecret{}
	_, err = io.ReadFull(Reader(append(loc[:], ext[:]...), "uds", 1), uds[:])
	return &uds, err
}

// Identity derives a platform's identity parameter, see
// https://github.com/system-transparency/system-transparency#identity---json-string-or-null
func (uds *UniqueDeviceSecret) Identity() (*Entropy, error) {
	var id Entropy
	_, err := io.ReadFull(Reader(uds[:], "uds:identity", 1), id[:])
	return &id, err
}

// Authentication derives a platform's authentication parameter, see
// https://github.com/system-transparency/system-transparency#authentication---json-string-or-null
func (uds *UniqueDeviceSecret) Authentication() (*Entropy, error) {
	var auth Entropy
	_, err := io.ReadFull(Reader(uds[:], "uds:authentication", 1), auth[:])
	return &auth, err
}

// SSH derives a platform's Ed25519 host key (not a general ST parameter)
func (uds *UniqueDeviceSecret) SSH() (*ssh.HostKey, error) {
	return ssh.NewHostKey(Reader(uds[:], "uds:ssh", 1), "ospkg@system-transparency")
}

// OneTimePassword is a one time password used to bootstrap mutually
// authenticated HTTPS.  TLS 1.3 and a proper PSK mode should replace
// this in the future if the standard Go library adds such support.
type OneTimePassword Entropy

// NewOneTimePassword derives a one-time password from a shared secret
func NewOneTimePassword(secret string) (*OneTimePassword, error) {
	otp := OneTimePassword{}
	_, err := io.ReadFull(Reader([]byte(secret), "otp", 1), otp[:])
	return &otp, err
}

// BasicAuthPassword derives a basic auth password
func (otp *OneTimePassword) BasicAuthPassword() (string, error) {
	var pw Entropy
	_, err := io.ReadFull(Reader(otp[:], "otp:basicAuthPassword", 1), pw[:])
	return hex.EncodeToString(pw[:]), err
}

// X509 derives an X509 certificate for a given IP address
func (otp *OneTimePassword) X509Certificate(ip net.IP) (*x509.Certificate, error) {
	_, crtDER, err := otp.keyGen(ip)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(crtDER)
}

// TLSCertificate derives a TLS certificate struct containing a private key and
// the same public X.509 certificate that is derived by X509Certificate()
func (otp OneTimePassword) TLSCertificate(ip net.IP) (*tls.Certificate, error) {
	privDER, crtDER, err := otp.keyGen(ip)
	if err != nil {
		return nil, fmt.Errorf("generate key-pair: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	})
	crtPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crtDER,
	})
	crt, err := tls.X509KeyPair(crtPEM, privPEM)
	if err != nil {
		return nil, fmt.Errorf("encode key-pair: %w", err)
	}
	return &crt, nil
}

func (otp *OneTimePassword) keyGen(_ net.IP) ([]byte, []byte, error) {
	pub, priv, err := ed25519.GenerateKey(Reader(otp[:], "otp:keygen", 1))
	if err != nil {
		return nil, nil, err
	}
	tmpl := template()
	crtDER, err := x509.CreateCertificate(Reader(otp[:], "otp:keygen", 2), tmpl, tmpl, pub, priv)
	if err != nil {
		return nil, nil, err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	return privDER, crtDER, err
}

const DummyServerName = "stprov"

func template() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Version:      3,
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(math.MaxInt32, math.MaxInt32),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{DummyServerName},
	}
}
