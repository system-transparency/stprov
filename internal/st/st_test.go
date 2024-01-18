package st

import (
	"os"
	"testing"

	"github.com/google/uuid"
)

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
