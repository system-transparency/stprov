//go:build efi_nvram

package ssh

import (
	"testing"

	"github.com/google/uuid"
)

func TestWriteEFI(t *testing.T) {
	varUUID, err := uuid.Parse("f401f2c1-b005-4be0-8cee-f2e5945bcbe7")
	if err != nil {
		t.Fatal(err)
	}
	hk := newHostKey(t)
	if err := hk.WriteEFI(&varUUID, "STHostKey"); err != nil {
		t.Error(err)
	}
}
