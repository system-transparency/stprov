package static

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vishvananda/netlink"
	"system-transparency.org/stboot/host"
	"system-transparency.org/stboot/host/network"

	mptnetwork "system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
)

func Config(args []string, dnsServers []*net.IP, optInterface, optHostIP, optGateway string, interfaceWait time.Duration, optAutodetect bool, optBondingAuto bool, optBondingInterfaces []string, optBondingMode string, optForce, optTryLastIPForGateway bool) (*host.Config, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("trailing arguments: %v", args)
	}
	optGateway, err := options.ValidateHostAndGateway(optHostIP, optGateway, optForce, optTryLastIPForGateway)
	if err != nil {
		return nil, err
	}
	bondingName := "bond0"
	var bondedInterfaces = make([]string, 0, 10)
	if len(optBondingInterfaces) > 0 {
		bondedInterfaces = optBondingInterfaces
		firstIf := bondedInterfaces[0]
		link, err := netlink.LinkByName(firstIf)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid first bonded interface: %w", firstIf, err)
		}
		optInterface = link.Attrs().HardwareAddr.String()
	}

	if optInterface == "" && (optAutodetect || optBondingAuto) {
		devices, err := mptnetwork.TestInterfaces(optGateway, optHostIP, interfaceWait)
		if err != nil {
			log.Printf("failed autodetection: %v\n", err)
		}
		if len(devices) != 0 {
			if optBondingAuto {
				for _, link := range devices {
					name := link.Attrs().Name
					log.Printf("selecting interface %s for bonding into %s\n", name, bondingName)
					bondedInterfaces = append(bondedInterfaces, name)
				}
			}

			// Using the first interface found. In the
			// case of bonding, bond0 will be using the
			// MAC address of first physical interface so
			// this is good.
			optInterface = devices[0].Attrs().HardwareAddr.String()
			log.Printf("selecting MAC %s for the network device\n", optInterface)
		} else {
			log.Printf("found no good devices. Defaulting to next best effort...")
		}
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
	hostIP, err := netlink.ParseAddr(optHostIP)
	if err != nil {
		return nil, fmt.Errorf("malformed host address: %w", err)
	}
	gateway := net.ParseIP(optGateway)
	if gateway == nil {
		return nil, fmt.Errorf("malformed gateway: %s", optGateway)
	}

	if err := mptnetwork.ResetInterfaces(); err != nil {
		return nil, fmt.Errorf("failed to reset network interfaces: %w", err)
	}

	ifname := mptnetwork.GetInterfaceName(&mac)
	mode := host.IPStatic
	cfg := host.Config{
		HostIP:         hostIP,
		DefaultGateway: &gateway,
		IPAddrMode:     &mode,
		DNSServer:      &dnsServers,
		NetworkInterfaces: &[]*host.NetworkInterface{
			{InterfaceName: &ifname, MACAddress: &mac},
		},
	}

	if len(bondedInterfaces) > 0 {
		bondmode := host.StringToBondingMode(optBondingMode)
		if bondmode == host.BondingUnknown {
			return nil, fmt.Errorf("bonding mode unknown: %s", optBondingMode)
		}
		cfg.BondingMode = bondmode
		var ifaces []*host.NetworkInterface
		for _, iface := range bondedInterfaces {
			// This is a bug in the go for-loop semantics. See
			// https://github.com/golang/go/discussions/56010
			iface := iface
			mac := mptnetwork.GetHardwareAddr(iface)
			ifaces = append(ifaces, &host.NetworkInterface{
				InterfaceName: &iface,
				MACAddress:    mac,
			})
		}
		cfg.NetworkInterfaces = &ifaces
		cfg.BondName = &bondingName
	}

	if err := network.SetupNetworkInterface(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("setup network: %w", err)
	}

	return &cfg, nil
}
