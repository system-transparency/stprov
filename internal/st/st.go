// package st provides utilities to manage host configurations in EFI-NVRAM
package st

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/u-root/u-root/pkg/efivarfs"
	"github.com/vishvananda/netlink"
)

type NetworkInterface struct {
	InterfaceName string `json:"interface_name"`
	MACAddress    string `json:"mac_address"`
}

// HostConfig is an ST host configuration, see:
// https://github.com/system-transparency/system-transparency#host_configurationjson
// TODO: Replace with stboot hostcfg
type HostConfig struct {
	Version           int                 `json:"version"`
	IPAddrMode        int                 `json:"network_mode"`
	HostIP            string              `json:"host_ip"`
	Gateway           string              `json:"gateway"`
	DNS               string              `json:"dns"`
	NetworkInterfaces []*NetworkInterface `json:"network_interfaces"`
	ProvisioningURLs  []string            `json:"provisioning_urls"`
	Identity          string              `json:"identity"`
	Authentication    string              `json:"authentication"`
	BondingMode       string              `json:"bonding_mode"`
	BondName          string              `json:"bond_name"`
}

// NewStaticHostConfig outputs a static host configuration without setting
// any identity string, authentication string, and timestamp.  You may
// leave dnsAddr and interfaceAddr as empty strings, see ST documentation.
func NewStaticHostConfig(hostIP, gateway string, provisioningURLs []string, dnsAddr string, interfaceAddr *string, ifname string) *HostConfig {
	return &HostConfig{
		Version:    1,
		IPAddrMode: 1,
		HostIP:     hostIP,
		Gateway:    gateway,
		DNS:        dnsAddr,
		NetworkInterfaces: []*NetworkInterface{
			{InterfaceName: ifname, MACAddress: *interfaceAddr},
		},
		ProvisioningURLs: provisioningURLs,
	}
}

// NewDHCPHostConfig outputs a dhcp host configuration without setting any
// identity string, authentication string, and timestamp.  You may leave dnsAddr
// and interfaceAddr as empty strings, see ST documentation.
func NewDHCPHostConfig(provisioningURLs []string, dnsAddr string, interfaceAddr *string, ifname string) *HostConfig {
	return &HostConfig{
		Version:    1,
		IPAddrMode: 2,
		DNS:        dnsAddr,
		NetworkInterfaces: []*NetworkInterface{
			{InterfaceName: ifname, MACAddress: *interfaceAddr},
		},
		ProvisioningURLs: provisioningURLs,
	}
}

// ReadEFI reads a host configuration from EFI-NVRAM in JSON format
func (cfg *HostConfig) ReadEFI(varUUID *uuid.UUID, efiName string) error {
	b, err := readEFI(varUUID, efiName)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}

// WriteEFI writes a host configuration to EFI-NVRAM in JSON format
func (cfg *HostConfig) WriteEFI(varUUID *uuid.UUID, efiName string) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return writeEFI(b, varUUID, efiName)
}

// SetBonding sets up the bonding mode on the host configuration
func (cfg *HostConfig) SetBonding(name, mode string, interfaces []string) error {
	if netlink.StringToBondMode(mode) == netlink.BOND_MODE_UNKNOWN {
		return fmt.Errorf("unknown bonding mode: %s", mode)
	}
	cfg.Bonding = true
	cfg.BondingMode = mode
	cfg.NetworkInterfaces = interfaces
	cfg.BondName = name
	return nil
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
