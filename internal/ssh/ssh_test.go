package ssh

import (
	"bytes"
	"crypto/rand"
	"encoding/pem"
	"os/user"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

func TestReadWriteFingerprint(t *testing.T) {
	for i, table := range []struct {
		Raw         []byte
		Fingerprint string
	}{
		{mustDecodePEM(t, HostKey1), Fingerprint1},
		{mustDecodePEM(t, HostKey2), Fingerprint2},
		{mustDecodePEM(t, HostKey3), Fingerprint3},
		{mustDecodePEM(t, HostKey4), Fingerprint4},
		{mustDecodePEM(t, HostKey5), Fingerprint5},
		{mustDecodePEM(t, HostKey6), Fingerprint6},
		{mustDecodePEM(t, HostKey7), Fingerprint7},
		{mustDecodePEM(t, HostKey8), Fingerprint8},
	} {
		var hk HostKey
		if err := hk.read(bytes.NewBuffer(table.Raw)); err != nil {
			t.Errorf("%d: failed reading valid host key: %v", i+1, err)
			continue
		}

		buf := bytes.NewBuffer(nil)
		if err := hk.write(buf); err != nil {
			t.Errorf("%d: failed writing valid host key: %v", i+1, err)
			continue
		}
		if got, want := buf.Bytes(), table.Raw; !bytes.Equal(got, want) {
			t.Errorf("%d: got host key\n%x\nbut wanted\n%x", i+1, got, want)
			continue
		}

		fpr, err := hk.Fingerprint()
		if err != nil {
			t.Errorf("%d: failed deriving fingerprint: %v", i+1, err)
			continue
		}
		if got, want := fpr, table.Fingerprint; got != want {
			t.Errorf("%d: got fingerprint\n%s\nbut wanted\n%s", i+1, got, want)
		}
	}
}

func TestWriteAndFingerprint(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	hk := newHostKey(t)
	if err := hk.writePEM(buf); err != nil {
		t.Errorf("writePEM host key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(buf.Bytes())
	if err != nil {
		t.Errorf("parse host key: %v", err)
	}

	fprThere := ssh.FingerprintSHA256(signer.PublicKey())
	fprHere, err := hk.Fingerprint()
	if err != nil {
		t.Errorf("failed deriving fingerprint: %v", err)
	}
	if got, want := fprHere, fprThere; got != want {
		t.Errorf("got fingerprint\n%s\nbut wanted\n%s", got, want)
	}
}

func TestWriteEFI(t *testing.T) {
	user, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "root" {
		t.Skip("need sudo to clutter efi-nvram, skipping TestWriteEFI()")
	}

	varUUID, err := uuid.Parse("f401f2c1-b005-4be0-8cee-f2e5945bcbe7")
	if err != nil {
		t.Fatal(err)
	}
	hk := newHostKey(t)
	if err := hk.WriteEFI(&varUUID, "STHostKey"); err != nil {
		t.Error(err)
	}
}

func TestWrite(t *testing.T) {
	hk := newHostKey(t)
	buf := bytes.NewBuffer(nil)
	if err := hk.writePEM(buf); err != nil {
		t.Errorf("write host key: %v", err)
	}
	if _, err := ssh.ParsePrivateKey(buf.Bytes()); err != nil {
		t.Errorf("parse resulting host key: %v", err)
	}
}

func newHostKey(t *testing.T) HostKey {
	hk, err := NewHostKey(rand.Reader, "testkey")
	if err != nil {
		t.Fatal(err)
	}
	return *hk
}

func mustDecodePEM(t *testing.T, str string) []byte {
	block, rest := pem.Decode([]byte(str))
	if block == nil {
		t.Fatal("no pem block")
	}
	if len(rest) != 0 {
		t.Fatal("too many pem blocks")
	}
	if block.Type != PEMTypePrivateKey {
		t.Fatalf("wrong pem type %s", block.Type)
	}
	return block.Bytes
}

//  unset c; for n in $(seq 8); do c=$c$n; [ -f hostkey$n ] && rm hostkey$n; ssh-keygen -t ed25519 -f hostkey$n -C $c -N "" | egrep ^SHA256; done

const (
	Fingerprint1 = "SHA256:BFPEgN8YSMXIlemQrxLdl08OfRc6v1HqTsuXeRgpZb4"
	Fingerprint2 = "SHA256:lNlinAAih0CIbPJ9gAMX7TfFjh4U+2ZGnNkD8Aez//c"
	Fingerprint3 = "SHA256:Z6VgsN2M+VHC92wbJ/8oJjgCT+vAKtnJxpgDOmBL594"
	Fingerprint4 = "SHA256:pz8NANAzK65jUmKGo7CDXtzhn56mww+zWAP7/6fEKxA"
	Fingerprint5 = "SHA256:wQH+kYKBWBQmSibrvp+6XeRN8kz6cHKrRnvsFQHwg6I"
	Fingerprint6 = "SHA256:vHDboZeN8+DoD5u3siUTSmxvpfrDaXPa7Fe01885tU8"
	Fingerprint7 = "SHA256:WJ57YAICbGjG6lDLBGw9wmlLN3biGOMN3u/b6z8i3EE"
	Fingerprint8 = "SHA256:+JMQx6mofEmNTLzKrrMFV6PI9CDMJ8qLWGE8shjFVsM"
)

const HostKey1 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACACkpiBmGa83PgQi36MvbdNmBEO5WNopb7qJFw97XGMlAAAAIj+g+Pw/oPj
8AAAAAtzc2gtZWQyNTUxOQAAACACkpiBmGa83PgQi36MvbdNmBEO5WNopb7qJFw97XGMlA
AAAEDNy07C4jNtZxAbsKVKxtTSOwrOANxTcsq2QOC7DUUWzAKSmIGYZrzc+BCLfoy9t02Y
EQ7lY2ilvuokXD3tcYyUAAAAATEBAgME
-----END OPENSSH PRIVATE KEY-----`

const HostKey2 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDU8pFoKTK6icWARBHCXNI8Rc+b9B56AbGtMyoP39Rw9gAAAIiK/2lmiv9p
ZgAAAAtzc2gtZWQyNTUxOQAAACDU8pFoKTK6icWARBHCXNI8Rc+b9B56AbGtMyoP39Rw9g
AAAEAntCmcMUNYppXw6mr2M9uFfpx/QP76J4cus7F698JavdTykWgpMrqJxYBEEcJc0jxF
z5v0HnoBsa0zKg/f1HD2AAAAAjEyAQID
-----END OPENSSH PRIVATE KEY-----
`

const HostKey3 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDHHF2BEfw0jKi51bCTUp8qZ6svg6THKSEr/OatFiQdvQAAAIiUYhwblGIc
GwAAAAtzc2gtZWQyNTUxOQAAACDHHF2BEfw0jKi51bCTUp8qZ6svg6THKSEr/OatFiQdvQ
AAAEAN8NN90e06FWRmUhvK46dNLR9L+vWvVm+L2EvANi5CVMccXYER/DSMqLnVsJNSnypn
qy+DpMcpISv85q0WJB29AAAAAzEyMwEC
-----END OPENSSH PRIVATE KEY-----
`

const HostKey4 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCt6vUp93l30aCUsmY3Uf4tzf43j9XSVM6CIeOoB05v/AAAAIg0D2+jNA9v
owAAAAtzc2gtZWQyNTUxOQAAACCt6vUp93l30aCUsmY3Uf4tzf43j9XSVM6CIeOoB05v/A
AAAEAlFE6RTcbokZul8hEVeA23aADd1qFb8QfyMs9QZwaoba3q9Sn3eXfRoJSyZjdR/i3N
/jeP1dJUzoIh46gHTm/8AAAABDEyMzQB
-----END OPENSSH PRIVATE KEY-----
`

const HostKey5 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB2G6knqrJ2a4HPqVFEI08AcBFp2IwLQU/Cl4WctX7YOAAAAIiAKjS/gCo0
vwAAAAtzc2gtZWQyNTUxOQAAACB2G6knqrJ2a4HPqVFEI08AcBFp2IwLQU/Cl4WctX7YOA
AAAEBqNAZHvgR8WjDygWYuLPGR+Ujm7bxxj+RfhtiFJqlRpXYbqSeqsnZrgc+pUUQjTwBw
EWnYjAtBT8KXhZy1ftg4AAAABTEyMzQ1
-----END OPENSSH PRIVATE KEY-----
`

const HostKey6 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACCAN6EEbkCy0KQPP1rAmvv1FIUFEg4bNcgHpHw30fLoRQAAAJAAJt2NACbd
jQAAAAtzc2gtZWQyNTUxOQAAACCAN6EEbkCy0KQPP1rAmvv1FIUFEg4bNcgHpHw30fLoRQ
AAAEDZiiI9WBHxgEefzIljW+n32LK0gbYLphsaW1cPAl5JroA3oQRuQLLQpA8/WsCa+/UU
hQUSDhs1yAekfDfR8uhFAAAABjEyMzQ1NgECAwQFBgc=
-----END OPENSSH PRIVATE KEY-----
`

const HostKey7 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBa7s4Um3CWj4vwoPr4EU3sRxnWIHED0PcW2p0qGbKUMwAAAJC2bLdNtmy3
TQAAAAtzc2gtZWQyNTUxOQAAACBa7s4Um3CWj4vwoPr4EU3sRxnWIHED0PcW2p0qGbKUMw
AAAEB1WeWgM92Qcrsu0euQyiBW8ElRJ5mifKLiyqufZvgIM1ruzhSbcJaPi/Cg+vgRTexH
GdYgcQPQ9xbanSoZspQzAAAABzEyMzQ1NjcBAgMEBQY=
-----END OPENSSH PRIVATE KEY-----
`

const HostKey8 = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB1kDobwXf+CUXK8LantxT55pAUgc77hYbmtDK3aRL7NAAAAJDnUZUo51GV
KAAAAAtzc2gtZWQyNTUxOQAAACB1kDobwXf+CUXK8LantxT55pAUgc77hYbmtDK3aRL7NA
AAAEB8kWlD3p8om7kuIEKjfpm67cCK8l0u6w4eLYMBmj0m6XWQOhvBd/4JRcrwtqe3FPnm
kBSBzvuFhua0MrdpEvs0AAAACDEyMzQ1Njc4AQIDBAU=
-----END OPENSSH PRIVATE KEY-----
`
