package st

import (
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"system-transparency.org/stboot/host"
	"system-transparency.org/stprov/subcmd/remote/dhcp"
	"system-transparency.org/stprov/subcmd/remote/static"
)

func TestReadWriteEFI(t *testing.T) {
	if os.Getenv("TEST_CLOBBER_EFI_NVRAM") == "" {
		t.Skip("Skipping tests associated with TEST_CLOBBER_EFI_NVRAM")
	}

	cfg1, _ := static.Config(nil, "1.1.1.1", "eth0", "192.168.0.2/32", "192.168.0.1", 10*time.Second, true, false, nil, "", true, false)
	cfg2, _ := dhcp.Config(nil, "2.2.2.2", "eth0", 10*time.Second, true)
	for i, cfg := range []*host.Config{cfg1, cfg2} {
		if err := WriteHostConfigEFI(cfg); err != nil {
			t.Errorf("%d: %v", i, err)
			return
		}
		cfgAgain, err := HostConfigEFI()
		if err != nil {
			t.Errorf("%d: %v", i, err)
		}
		if got, want := &cfgAgain, cfg; !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got config\n%v\nbut wanted\n%v", i, got, want)
		}
	}
}

func TestReadWriteHostName(t *testing.T) {
	if os.Getenv("TEST_CLOBBER_EFI_NVRAM") == "" {
		t.Skip("Skipping tests associated with TEST_CLOBBER_EFI_NVRAM")
	}

	hn := HostName("mullis")
	if err := hn.WriteEFI(testUUID(t), "STHostName"); err != nil {
		t.Errorf("%v", err)
		return
	}
	var hnAgain HostName
	if err := hnAgain.ReadEFI(testUUID(t), "STHostName"); err != nil {
		t.Errorf("%v", err)
		return
	}
	if hn != hnAgain {
		t.Errorf("got host name %q, want %q", hn, hnAgain)
	}
}

func testUUID(t *testing.T) *uuid.UUID {
	t.Helper()
	varUUID, err := uuid.Parse("f401f2c1-b005-4be0-8cee-f2e5945bcbe7")
	if err != nil {
		t.Fatal(err)
	}
	return &varUUID
}

func testHardwareAddr(t *testing.T) *net.HardwareAddr {
	a, err := net.ParseMAC("aa:aa:aa:bb:bb:bb")
	if err != nil {
		t.Fatal(err)
	}
	return &a
}
