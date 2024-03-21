package dhcp

import (
	"context"
	"fmt"
	"net"
	"time"

	"system-transparency.org/stboot/host"
	"system-transparency.org/stboot/host/network"

	mptnetwork "system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
)

func Config(args []string, dnsServers []*net.IP, optInterface string, interfaceWait time.Duration, optAutodetect bool) (*host.Config, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("trailing arguments: %v", args)
	}
	if optInterface == "" {
		defaultMACs, err := options.DefaultInterfaces(interfaceWait)
		if err != nil {
			return nil, fmt.Errorf("no suitable network interface available")
		}
		optInterface = defaultMACs[0].String()
	}
	mac, err := net.ParseMAC(optInterface)
	if err != nil {
		return nil, fmt.Errorf("malformed mac address: %s", optInterface)
	}
	if err := mptnetwork.ResetInterfaces(); err != nil {
		return nil, fmt.Errorf("failed to reset network interfaces: %v", err)
	}
	ifname := mptnetwork.GetInterfaceName(&mac)
	mode := host.IPDynamic
	cfg := host.Config{
		IPAddrMode: &mode,
		DNSServer:  &dnsServers,
		NetworkInterfaces: &[]*host.NetworkInterface{
			{InterfaceName: &ifname, MACAddress: &mac},
		},
	}
	if err := network.SetupNetworkInterface(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("setup network: %w", err)
	}

	return &cfg, err
}
