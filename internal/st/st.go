// package st provides utilities to manage host configurations in EFI-NVRAM
package st

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/u-root/u-root/pkg/efivarfs"
	"system-transparency.org/stboot/host"
)

func WriteHostConfigEFI(cfg *host.Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	efiName, efiGuid, err := HostConfigEFIVariableName()
	if err != nil {
		return fmt.Errorf("invalid host config EFI var name: %s", host.HostConfigEFIVarName)
	}

	return writeEFI(b, efiGuid, efiName)
}

func HostConfigEFI() (*host.Config, error) {
	efiName, efiGuid, err := HostConfigEFIVariableName()
	if err != nil {
		return nil, fmt.Errorf("invalid host config EFI var name: %s", host.HostConfigEFIVarName)
	}

	b, err := readEFI(efiGuid, efiName)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var cfg host.Config
	err = json.Unmarshal(b, &cfg)
	return &cfg, err
}

func HostConfigEFIVariableName() (string, *uuid.UUID, error) {
	efiVarFrag := strings.SplitN(host.HostConfigEFIVarName, "-", 2)
	if len(efiVarFrag) != 2 {
		return "", nil, fmt.Errorf("invalid host config EFI var name: %s", host.HostConfigEFIVarName)
	}

	efiUuid, err := uuid.Parse(efiVarFrag[1])
	if err != nil {
		return "", nil, fmt.Errorf("invalid host config EFI var name: %s", host.HostConfigEFIVarName)
	}

	return efiVarFrag[0], &efiUuid, nil
}

// HostName is a host name
type HostName string

// WriteEFI writes a host name to EFI-NVRAM
func (hn *HostName) WriteEFI(varUUID *uuid.UUID, efiName string) error {
	return writeEFI([]byte(*hn), varUUID, efiName)
}

func (hn *HostName) ReadEFI(varUUID *uuid.UUID, efiName string) error {
	b, err := readEFI(varUUID, efiName)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	*hn = HostName(b)
	return nil
}

func writeEFI(b []byte, varUUID *uuid.UUID, efiName string) error {
	desc := efivarfs.VariableDescriptor{Name: efiName, GUID: *varUUID}
	attrs := efivarfs.AttributeBootserviceAccess
	attrs |= efivarfs.AttributeRuntimeAccess
	attrs |= efivarfs.AttributeNonVolatile
	e, err := efivarfs.New()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return efivarfs.WriteVariable(e, desc, attrs, b)
}

func readEFI(varUUID *uuid.UUID, efiName string) ([]byte, error) {
	desc := efivarfs.VariableDescriptor{Name: efiName, GUID: *varUUID}
	e, err := efivarfs.New()
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	_, b, err := efivarfs.ReadVariable(e, desc)
	return b, err
}
