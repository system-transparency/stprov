package secrets

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"

	"system-transparency.org/stprov/internal/ssh"
)

func TestReader(t *testing.T) {
	for _, table := range []struct {
		desc   string
		r1, r2 io.Reader
	}{
		{"other secret ", Reader([]byte("SECRET"), "label", 1), Reader([]byte("secret"), "label", 1)},
		{"other label  ", Reader([]byte("secret"), "LABEL", 1), Reader([]byte("secret"), "label", 1)},
		{"other counter", Reader([]byte("secret"), "label", 2), Reader([]byte("secret"), "label", 1)},
		{"equal readers", Reader([]byte("secret"), "label", 1), Reader([]byte("secret"), "label", 1)},
	} {
		var b1, b2 [32]byte
		if _, err := io.ReadFull(table.r1, b1[:]); err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadFull(table.r2, b2[:]); err != nil {
			t.Fatal(err)
		}
		if got, want := bytes.Equal(b1[:], b2[:]), table.desc == "equal readers"; got != want {
			t.Errorf("%s: got %v but wanted %v\nb1: %v\nb2: %v", table.desc, got, want, b1, b2)
		}
	}
}

func TestNewUniqueDeviceSecret(t *testing.T) {
	uds, err := NewUniqueDeviceSecret(&Entropy{})
	if err != nil {
		t.Fatalf("derive uds: %v", err)
	}
	udsOther, err := NewUniqueDeviceSecret(&Entropy{})
	if err != nil {
		t.Fatalf("derive other uds: %v", err)
	}
	if bytes.Equal(uds[:], udsOther[:]) {
		t.Errorf("uds is identical to another uds")
	}
}

func TestUniqueDeviceSecretDerivations(t *testing.T) {
	derive := func(uds UniqueDeviceSecret) (id *Entropy, auth *Entropy, hk *ssh.HostKey) {
		var err error
		if id, err = uds.Identity(); err != nil {
			t.Fatalf("derive identity: %v", err)
		}
		if auth, err = uds.Authentication(); err != nil {
			t.Fatalf("derive authentication: %v", err)
		}
		if hk, err = uds.SSH(); err != nil {
			t.Fatalf("derive ssh host key: %v", err)
		}
		return
	}
	check := func(i1, i2, a1, a2 *Entropy, hk1, hk2 *ssh.HostKey) error {
		if !bytes.Equal(i1[:], i2[:]) {
			return fmt.Errorf("identity is not equal")
		}
		if !bytes.Equal(a1[:], a2[:]) {
			return fmt.Errorf("authentication is not equal")
		}
		if !reflect.DeepEqual(hk1, hk2) {
			return fmt.Errorf("ssh host key is not equal")
		}
		return nil
	}

	uds := UniqueDeviceSecret{}
	id, auth, hk := derive(uds)
	idAgain, authAgain, hkAgain := derive(uds)
	if err := check(id, idAgain, auth, authAgain, hk, hkAgain); err != nil {
		t.Error(err)
	}

	udsOther := UniqueDeviceSecret{1}
	idOther, authOther, hkOther := derive(udsOther)
	if err := check(id, idOther, auth, authOther, hk, hkOther); err == nil {
		t.Error("different uds but valid derivations")
	}
}

func TestNewOneTimePassword(t *testing.T) {
	for _, table := range []struct {
		desc   string
		s1, s2 string
	}{
		{"other secret", "cat", "dog"},
		{"equal inputs", "cat", "cat"},
	} {
		otp1, err := NewOneTimePassword(table.s1)
		if err != nil {
			t.Fatalf("%s: derive otp: %v", table.desc, err)
		}
		otp2, err := NewOneTimePassword(table.s2)
		if err != nil {
			t.Fatalf("%s: derive otp: %v", table.desc, err)
		}
		if got, want := bytes.Equal(otp1[:], otp2[:]), table.desc == "equal inputs"; got != want {
			t.Errorf("%s: got %v but wanted %v\notp1: %v\notp2: %v", table.desc, got, want, otp1, otp2)
		}
	}
}

func TestOneTimePasswordDerivation(t *testing.T) {
	derive := func(otp OneTimePassword) (ba string, xcrt *x509.Certificate, tcrt *tls.Certificate) {
		var err error
		if ba, err = otp.BasicAuthPassword(); err != nil {
			t.Fatalf("derive basic auth password: %v", err)
		}
		if xcrt, err = otp.X509Certificate(net.IPv4(192, 168, 0, 1)); err != nil {
			t.Fatalf("derive x509 certificate: %v", err)
		}
		if tcrt, err = otp.TLSCertificate(net.IPv4(192, 168, 0, 1)); err != nil {
			t.Fatalf("derive tls certificate: %v", err)
		}
		return
	}
	check := func(ba1, ba2 string, xcrt1, xcrt2 *x509.Certificate, tcrt1, tcrt2 *tls.Certificate) error {
		if ba1 != ba2 {
			return fmt.Errorf("basic auth is not equal")
		}
		if !reflect.DeepEqual(xcrt1, xcrt2) {
			return fmt.Errorf("x509 certificate is not equal")
		}
		if !reflect.DeepEqual(tcrt1, tcrt2) {
			return fmt.Errorf("tls certificate is not equal")
		}
		if got, want := len(tcrt1.Certificate), 1; got != want {
			return fmt.Errorf("invalid number of certificates: %d", got)
		}
		if !bytes.Equal(xcrt1.Raw, tcrt1.Certificate[0]) {
			return fmt.Errorf("client-server certificate mismatch")
		}
		return nil
	}

	otp := OneTimePassword{}
	otpOther := OneTimePassword{1}

	ba, xcrt, tcrt := derive(otp)
	baAgain, xcrtAgain, tcrtAgain := derive(otp)
	baOther, xcrtOther, tcrtOther := derive(otpOther)
	if err := check(ba, baAgain, xcrt, xcrtAgain, tcrt, tcrtAgain); err != nil {
		t.Error(err)
	}
	if err := check(ba, baOther, xcrt, xcrtOther, tcrt, tcrtOther); err == nil {
		t.Errorf("other otp but valid derivations")
	}
}
