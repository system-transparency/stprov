// Package sb provision Secure Boot keys in setup mode
package sb

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
	"github.com/u-root/u-root/pkg/efivarfs"
)

var (
	efiGlobalVariableGUID      = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
	efiGlobalVariableSetupMode = "SetupMode"
	efiGlobalVariablePK        = "PK"
	efiGlobalVariableKEK       = "KEK"

	efiGlobalVariableOSIndications   = "OsIndications"
	efiOsInditationsBootToFirmwareUI = uint64(1)

	efiImageSecurityDatabaseGUID = "d719b2cb-3d3a-4596-a3bc-dad00e67656f"
	efiImageSecurityDatabaseDb   = "db"
	efiImageSecurityDatabaseDbx  = "dbx"
)

// IsSetupMode outputs true if the system is in SecureBoot setup mode
func IsSetupMode() (bool, error) {
	b, err := efiRead(efiGlobalVariableSetupMode, efiGlobalVariableGUID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", efiGlobalVariableSetupMode, err)
	}
	if len(b) != 1 {
		return false, fmt.Errorf("unexpected length: %w", err)
	}
	if b[0] != 0 && b[0] != 1 {
		return false, fmt.Errorf("unexpected value: %d", b[0])
	}
	return b[0] == 1, nil
}

// Provision writes PK, KEK, db, and dbx (optional) to EFI NVRAM.  The input
// must be valid authentication_v2 descriptors (PK is self signed, KEK is signed
// by PK, and db and dbx are signed by KEK). Setup Mode is also required.
//
// PK is provisioned *first* so the user can be sure that signing with PK and
// KEK works.  In other words, there should not be any surprises in the future.
func Provision(pk, kek, db, dbx []byte) error {
	if err := efiAuthenticatedWrite(efiGlobalVariablePK, efiGlobalVariableGUID, pk); err != nil {
		return fmt.Errorf("%s: %w", efiGlobalVariablePK, err)
	}
	if err := efiAuthenticatedWrite(efiGlobalVariableKEK, efiGlobalVariableGUID, kek); err != nil {
		return fmt.Errorf("%s: %w", efiGlobalVariableKEK, err)
	}
	if err := efiAuthenticatedWrite(efiImageSecurityDatabaseDb, efiImageSecurityDatabaseGUID, db); err != nil {
		return fmt.Errorf("%s: %w", efiImageSecurityDatabaseDb, err)
	}
	if len(dbx) != 0 {
		if err := efiAuthenticatedWrite(efiImageSecurityDatabaseDbx, efiImageSecurityDatabaseGUID, dbx); err != nil {
			return fmt.Errorf("%s: %w", efiImageSecurityDatabaseDbx, err)
		}
	}
	return nil
}

// RequestRebootIntoUEFIMenu asks the firmware to go straight into the UEFI menu
// on next boot
func RequestRebootIntoUEFIMenu() error {
	b, err := efiRead(efiGlobalVariableOSIndications, efiGlobalVariableGUID)
	if err != nil {
		b = make([]byte, 8)
	}
	if len(b) != 8 {
		return fmt.Errorf("%s: unexpected data length %d", efiGlobalVariableOSIndications, len(b))
	}
	osIndications := binary.LittleEndian.Uint64(b)
	if osIndications&efiOsInditationsBootToFirmwareUI != 0 {
		return nil // already requested
	}

	osIndications |= efiOsInditationsBootToFirmwareUI
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, osIndications)
	if err := efiWrite(efiGlobalVariableOSIndications, efiGlobalVariableGUID, data); err != nil {
		return fmt.Errorf("%s: %w", efiGlobalVariableOSIndications, err)
	}
	return nil
}

func efiRead(name, guid string) ([]byte, error) {
	id, err := uuid.Parse(guid)
	if err != nil {
		return nil, fmt.Errorf("parse guid: %w", err)
	}
	desc := efivarfs.VariableDescriptor{Name: name, GUID: id}
	fs, err := efivarfs.New()
	if err != nil {
		return nil, fmt.Errorf("new efivarfs: %w", err)
	}
	_, b, err := efivarfs.ReadVariable(fs, desc)
	if err != nil {
		return nil, fmt.Errorf("read efivarfs: %w", err)
	}
	return b, err
}

func efiWrite(name, guid string, data []byte) error {
	fs, err := efivarfs.New()
	if err != nil {
		return fmt.Errorf("new efivarfs: %w", err)
	}
	id, err := uuid.Parse(guid)
	if err != nil {
		return fmt.Errorf("parse guid %s: %w", guid, err)
	}
	desc := efivarfs.VariableDescriptor{Name: name, GUID: id}
	attrs := efivarfs.AttributeNonVolatile
	attrs |= efivarfs.AttributeBootserviceAccess
	attrs |= efivarfs.AttributeRuntimeAccess
	return efivarfs.WriteVariable(fs, desc, attrs, data)
}

func efiAuthenticatedWrite(name, guid string, authData []byte) error {
	fs, err := efivarfs.New()
	if err != nil {
		return fmt.Errorf("new efivarfs: %w", err)
	}
	id, err := uuid.Parse(guid)
	if err != nil {
		return fmt.Errorf("parse guid %s: %w", guid, err)
	}
	desc := efivarfs.VariableDescriptor{Name: name, GUID: id}
	attrs := efivarfs.AttributeNonVolatile
	attrs |= efivarfs.AttributeBootserviceAccess
	attrs |= efivarfs.AttributeRuntimeAccess
	attrs |= efivarfs.AttributeTimeBasedAuthenticatedWriteAccess
	return efivarfs.WriteVariable(fs, desc, attrs, authData)
}
