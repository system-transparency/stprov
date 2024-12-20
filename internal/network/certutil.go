package network

// Copied from stboot, see:
// https://git.glasklar.is/system-transparency/core/stboot/-/blob/main/internal/certutil/certutil.go?ref_type=heads
//
// Likely to be unified in a future "stlib" after refactoring, see:
// https://git.glasklar.is/system-transparency/core/stboot/-/issues/179

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func DecodePEM(pemBytes []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	for len(pemBytes) > 0 {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			pemBytes = rest

			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		certs = append(certs, cert)
		pemBytes = rest
	}

	if len(certs) == 0 {
		return nil, errors.New("no PEM block of type CERTIFICATE found")
	}

	return certs, nil
}
