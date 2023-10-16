//go:build efi_nvram

package st

import (
	"net"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"system-transparency.org/stboot/host"
)

func TestReadWriteEFI(t *testing.T) {
	testIfname := "eth0"
	ifaceConfig := host.NetworkInterface{InterfaceName: &testIfname, MACAddress: testHardwareAddr(t)}
	for i, cfg := range []*HostConfig{
		NewStaticHostConfig("192.168.0.2/32", "192.168.0.1", "1.1.1.1", []*host.NetworkInterface{&ifaceConfig}),
		NewDHCPHostConfig("2.2.2.2", nil),
	} {
		if err := cfg.WriteEFI(testUUID(t), "STHostConfig"); err != nil {
			t.Errorf("%d: %v", i, err)
			return
		}
		var cfgAgain HostConfig
		if err := cfgAgain.ReadEFI(testUUID(t), "STHostConfig"); err != nil {
			t.Errorf("%d: %v", i, err)
		}
		if got, want := &cfgAgain, cfg; !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got config\n%v\nbut wanted\n%v", i, got, want)
		}
	}
}

func TestReadWriteHostName(t *testing.T) {
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
