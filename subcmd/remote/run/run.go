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

func Main(args []string, optPort int, optIP, optAllowHosts, optOTP, efiUUID, efiConfigName, efiKeyName, efiHostName string) error {
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
	allowNets := make([]net.IPNet, 0)
	for _, cidrStr := range strings.Split(optAllowHosts, ",") {
		if !strings.Contains(cidrStr, "/") {
			cidrStr += "/32"
		}
		_, cidr, err := net.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("allow: malformed CIDR address: %s", cidrStr)
		}
		allowNets = append(allowNets, *cidr)
	}
	otp := optOTP
	varUUID, err := uuid.Parse(efiUUID)
	if err != nil {
		return fmt.Errorf("parse efi UUID: %w", err)
	}

	var hostname st.HostName
	if err := hostname.ReadEFI(&varUUID, efiHostName); err != nil {
		return fmt.Errorf("ReadEFI: %s: %w", efiHostName, err)
	}
	uds, timestamp, err := listen(otp, allowNets, ip, port, hostname)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := writeHostKey(uds, &varUUID, efiKeyName); err != nil {
		return fmt.Errorf("persist host key: %w", err)
	}
	if err := readWriteHostConfig(uds, timestamp, &varUUID, efiConfigName); err != nil {
		return fmt.Errorf("persist host config: %w", err)
	}

	return nil
}

// listen listens for incoming requests until a commit message is received.
// The admin running stprov remote must then give confirmation to proceed.
func listen(otp string, allowNets []net.IPNet, ip net.IP, port int, hostname st.HostName) (uds *secrets.UniqueDeviceSecret, timestamp int64, err error) {
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
		return uds, timestamp, fmt.Errorf("new server: %v", err)
	}
	log.Printf("starting server on %s:%d", srv.RemoteIP, srv.RemotePort)
	if err := srv.Run(ctx); err != nil {
		return uds, timestamp, fmt.Errorf("run server: %v", err)
	}
	log.Printf("received entropy\n\n%s\n", hexify.Format(srv.Entropy[:]))
	if _, err := readLine("Press Enter to commit changes, ctrl+c to abort"); err != nil {
		return uds, timestamp, fmt.Errorf("read confirmation: %v", err)
	}

	return srv.UDS, srv.Timestamp, err
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

// readWriteHostConfig reads a partial ST host config, populating it with a
// timestamp, an identity string and an authentication string.  The resulting
// host configuration is then written back to EFI-NVRAM.
//
// Note: identity and authentication strings are hardcoded instead of deriving
// them from UDS.  It is currently out of scope to use these parameters.
func readWriteHostConfig(_ *secrets.UniqueDeviceSecret, timestamp int64, varUUID *uuid.UUID, name string) error {
	var cfg st.HostConfig
	if err := cfg.ReadEFI(varUUID, name); err != nil {
		return err
	}
	cfg.Authentication = "foo"
	cfg.Identity = "bar"
	cfg.Timestamp = timestamp
	return cfg.WriteEFI(varUUID, name)
}
