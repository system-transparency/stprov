package run

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"system-transparency.org/stprov/internal/api"
	"system-transparency.org/stprov/internal/hexify"
	"system-transparency.org/stprov/internal/secrets"
	"system-transparency.org/stprov/internal/st"
)

func Main(args []string, optPort int, optIP string, optAllowHosts []string, optOTP string, efiUUID *uuid.UUID, efiConfigName, efiKeyName, efiHostName string) error {
	if len(args) != 0 {
		return fmt.Errorf("trailing arguments: %v", args)
	}
	if len(optOTP) == 0 {
		return fmt.Errorf("otp: one-time password is a required option")
	}
	port := optPort
	if port < 1 || port > 65535 {
		return fmt.Errorf("port: invalid: %d not in [1, 65535]", optPort)
	}
	ip := net.ParseIP(optIP)
	if ip == nil {
		return fmt.Errorf("ip: malformed ip address: %s", optIP)
	}
	allowNets, err := parseAllowedNets(optAllowHosts)
	if err != nil {
		return err
	}
	otp := optOTP

	var hostname st.HostName
	if err := hostname.ReadEFI(efiUUID, efiHostName); err != nil {
		return fmt.Errorf("ReadEFI: %s: %w", efiHostName, err)
	}
	uds, err := listen(otp, allowNets, ip, port, hostname)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := writeHostKey(uds, efiUUID, efiKeyName); err != nil {
		return fmt.Errorf("persist host key: %w", err)
	}

	return nil
}

// parseAllowedNets parses a list of addresses in CIDR format.  If an address
// omits the subnet mask, it will default to "/32" (IPv4) or "/128" (IPv6).
func parseAllowedNets(addresses []string) ([]net.IPNet, error) {
	var allowNets []net.IPNet
	for _, addr := range addresses {
		if !strings.Contains(addr, "/") {
			ip := net.ParseIP(addr)
			if ip == nil {
				return nil, fmt.Errorf("malformed address: %s", addr)
			}
			if ip.To4() != nil {
				addr += "/32"
			} else if ip.To16() != nil {
				addr += "/128"
			}
		}

		_, cidr, err := net.ParseCIDR(addr)
		if err != nil {
			return nil, fmt.Errorf("malformed address: %s", addr)
		}
		allowNets = append(allowNets, *cidr)
	}
	return allowNets, nil
}

// listen listens for incoming requests until a commit message is received.
// The admin running stprov remote must then give confirmation to proceed.
func listen(otp string, allowNets []net.IPNet, ip net.IP, port int, hostname st.HostName) (uds *secrets.UniqueDeviceSecret, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := api.NewServer(&api.ServerConfig{
		Secret:     otp,
		RemoteIP:   ip,
		RemotePort: port,
		LocalCIDR:  allowNets,
		Deadline:   15 * time.Second,
		Timeout:    60 * time.Second,
		HostName:   string(hostname),
	})
	if err != nil {
		return uds, fmt.Errorf("new server: %w", err)
	}
	log.Printf("starting server on %s:%d", srv.RemoteIP, srv.RemotePort)
	if err := srv.Run(ctx); err != nil {
		return uds, fmt.Errorf("run server: %w", err)
	}
	log.Printf("received entropy\n\n%s\n", hexify.Format(srv.Entropy[:]))
	if _, err := readLine("Press Enter to commit changes, ctrl+c to abort"); err != nil {
		return uds, fmt.Errorf("read confirmation: %w", err)
	}

	return srv.UDS, nil
}

func readLine(msg string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(msg)
	return reader.ReadString('\n')
}

// writeHostKey derives an SSH host key from a unique device secret, writing it
// to EFI-NVRAM
func writeHostKey(uds *secrets.UniqueDeviceSecret, varUUID *uuid.UUID, name string) error {
	hk, err := uds.SSH()
	if err != nil {
		return err
	}
	return hk.WriteEFI(varUUID, name)
}
