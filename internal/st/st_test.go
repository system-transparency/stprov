package st

import (
	"os/user"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestReadWriteEFI(t *testing.T) {
	maybeSkip(t)
	testAddr := "aa:aa:aa:aa:aa"
	urls := []string{"http://localhost:80", "https://localhost:443"}
	for i, cfg := range []*HostConfig{
		NewStaticHostConfig("192.168.0.2/32", "192.168.0.1", urls, "1.1.1.1", &testAddr),
		NewDHCPHostConfig(urls, "2.2.2.2", nil),
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
	maybeSkip(t)
	hn := HostName("mullis")
	if err := hn.WriteEFI(testUUID(t), "STHostName"); err != nil {
		t.Errorf("%v", err)
	}
	var hnAgain HostName
	if err := hnAgain.ReadEFI(testUUID(t), "STHostName"); err != nil {
		t.Errorf("%v", err)
	}
	if hn != hnAgain {
		t.Errorf("got host name %q, want %q", hn, hnAgain)
	}
}

func maybeSkip(t *testing.T) {
	t.Helper()
	user, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "root" {
		t.Skip("need sudo to clutter efi-nvram, skipping a test")
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
