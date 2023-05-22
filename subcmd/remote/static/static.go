package static

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
	"system-transparency.org/stboot/host"
	"system-transparency.org/stboot/host/network"

	mptnetwork "system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/internal/st"
)

func Main(args []string, optDNS, optInterface, optHostName, optHostIP, optGateway, optUser, optPassword, optURL, efiUUID, efiName, efiHost, provURL string, interfaceWait time.Duration, optAutodetect bool, optBondingAuto bool, optBondingInterfaces []string, optBondingMode string, allowConfigQuirks, optTryLastIPForGateway bool) error {
	if len(args) != 0 {
		return fmt.Errorf("trailing arguments: %v", args)
	}
	if len(optHostName) == 0 {
		return fmt.Errorf("host name is a required option")
	}
	optGateway, err := options.ValidateHostAndGateway(optHostIP, optGateway, allowConfigQuirks, optTryLastIPForGateway)
	if err != nil {
		return err
	}
	url, err := options.ParseProvisioningURL(optURL, provURL, optUser, optPassword)
	if err != nil {
		return err // either invalid option combination or values
	}
	if strings.Contains(url, options.DefUser+":"+options.DefPassword) {
		log.Println("WARNING: using default username and password")
	}
	if ip := net.ParseIP(optDNS); optDNS != "" && ip == nil {
		return fmt.Errorf("malformed dns address: %s", optDNS)
	}
	if optBondingAuto && len(optBondingInterfaces) > 0 {
		return fmt.Errorf("use -b or -B, not both")
	}

	bondingName := "bond0"
	var bondedInterfaces = make([]string, 0, 10)
	if len(optBondingInterfaces) > 0 {
		bondedInterfaces = optBondingInterfaces
		firstIf := bondedInterfaces[0]
		link, err := netlink.LinkByName(firstIf)
		if err != nil {
			return fmt.Errorf("%s: invalid first bonded interface: %v", firstIf, err)
		}
		optInterface = link.Attrs().HardwareAddr.String()
	}

	if optInterface == "" && optAutodetect || optBondingAuto {
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
			return fmt.Errorf("no suitable network interface available")
		}
		optInterface = defaultMACs[0].String()
	}
	mac, err := net.ParseMAC(optInterface)
	if err != nil {
		return fmt.Errorf("malformed mac address: %s", optInterface)
	}
	varUUID, err := uuid.Parse(efiUUID)
	if err != nil {
		return fmt.Errorf("parse efi UUID: %w", err)
	}
	hostName := st.HostName(optHostName)
	hostIP, err := netlink.ParseAddr(optHostIP)
	if err != nil {
		return fmt.Errorf("malformed host address: %s", err)
	}
	gateway := net.ParseIP(optGateway)
	if gateway == nil {
		return fmt.Errorf("malformed gateway: %s", optGateway)
	}

	if err := mptnetwork.ResetInterfaces(); err != nil {
		return fmt.Errorf("failed to reset network interfaces: %v", err)
	}

	ifname := mptnetwork.GetInterfaceName(&mac)
	mode := host.IPStatic
	cfg := host.Config{
		HostIP:         hostIP,
		DefaultGateway: &gateway,
		IPAddrMode:     &mode,
		NetworkInterfaces: &[]*host.NetworkInterface{
			{InterfaceName: &ifname, MACAddress: &mac},
		},
	}

	if len(bondedInterfaces) > 0 {
		bondmode := host.StringToBondingMode(optBondingMode)
		if bondmode == host.BondingUnknown {
			return fmt.Errorf("bonding mode unknown: %s", optBondingMode)
		}
		// cfg.BondingMode = bondmode
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

	if err := network.SetupNetworkInterface(&cfg); err != nil {
		return fmt.Errorf("setup network: %w", err)
	}

	var config *st.HostConfig
	if len(bondedInterfaces) > 0 {
		config = st.NewBondingHostConfig(&url, optDNS, optBondingMode, *cfg.NetworkInterfaces)
	} else {
		config = st.NewStaticHostConfig(optHostIP, optGateway, &url, optDNS, *cfg.NetworkInterfaces)
	}

	if err := config.WriteEFI(&varUUID, efiName); err != nil {
		return fmt.Errorf("persist host config: %w", err)
	}
	if err := hostName.WriteEFI(&varUUID, efiHost); err != nil {
		return fmt.Errorf("persist host name: %w", err)
	}

	return nil
}
