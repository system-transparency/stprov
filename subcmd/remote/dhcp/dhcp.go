package dhcp

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"system-transparency.org/stboot/host"
	"system-transparency.org/stboot/host/network"

	mptnetwork "system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/internal/st"
)

func Main(args []string, optDNS, optInterface, optHostName, optUser, optPassword, optURL, efiUUID, efiName, efiHost, provURL string, interfaceWait time.Duration, optAutodetect bool) error {
	if len(args) != 0 {
		return fmt.Errorf("trailing arguments: %v", args)
	}
	if len(optHostName) == 0 {
		return fmt.Errorf("host name is a required option")
	}
	url, err := options.ParseProvisioningURL(optURL, provURL, optUser, optPassword)
	if err != nil {
		return err // either invalid option combination or values
	}
	if strings.Contains(url, options.DefUser+":"+options.DefPassword) {
		log.Println("WARNING: using default username and password")
	}
	if ip := net.ParseIP(optDNS); ip == nil {
		return fmt.Errorf("malformed dns address: %s", optDNS)
	}
	if optInterface == "" {
		defaultMACs, err := options.DefaultInterfaces(interfaceWait)
		if err != nil {
			return fmt.Errorf("no suitable network interface available")
		}
		optInterface = defaultMACs[0].String()
	}
	mac, err := net.ParseMAC(optInterface)
	if err != nil {
		return fmt.Errorf("malformed mac address: %s", optInterface)
	}
	hostName := st.HostName(optHostName)
	varUUID, err := uuid.Parse(efiUUID)
	if err != nil {
		return fmt.Errorf("parse efi UUID: %w", err)
	}

	if err := mptnetwork.ResetInterfaces(); err != nil {
		return fmt.Errorf("failed to reset network interfaces: %v", err)
	}
	ifname := mptnetwork.GetInterfaceName(&mac)
	mode := host.IPDynamic
	cfg := &host.Config{
		IPAddrMode: &mode,
		NetworkInterfaces: &[]*host.NetworkInterface{
			{InterfaceName: &ifname, MACAddress: &mac},
		},
	}
	if err := network.SetupNetworkInterface(cfg); err != nil {
		return fmt.Errorf("setup network: %w", err)
	}
	config := st.NewDHCPHostConfig(&url, optDNS, *cfg.NetworkInterfaces)
	if err := config.WriteEFI(&varUUID, efiName); err != nil {
		return fmt.Errorf("persist host config: %w", err)
	}
	if err := hostName.WriteEFI(&varUUID, efiHost); err != nil {
		return fmt.Errorf("persist host name: %w", err)
	}

	return nil
}
