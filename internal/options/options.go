package options

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"strings"
	"time"

	"system-transparency.org/stprov/internal/network"

	"github.com/vishvananda/netlink"
)

// Defaults optionally set in Makefile, through -ldflags which makes
// it necessary to keep these as variables and not put them in a
// struct.
var DefHostname = "localhost.local"
var DefUser = "stboot"
var DefPassword = "stboot"
var DefDNS = "9.9.9.9"
var DefAllowedNetworks = "127.0.0.1/32"
var DefBondingMode = "balance-rr"
var DefTemplateURL = "https://user:password@stpackage.example.org/os-stable.json"

func MaxHost(network *net.IPNet) string {
	prefixLen, bits := network.Mask.Size()
	if prefixLen == bits {
		return network.IP.String()
	}
	firstIPInt := &big.Int{}
	firstIPInt.SetBytes(network.IP)
	hostLen := uint(bits) - uint(prefixLen)
	lastIPInt := big.NewInt(1)              // Starting with 1.
	lastIPInt.Lsh(lastIPInt, hostLen)       // Shifting it up one bit into the network part.
	lastIPInt.Sub(lastIPInt, big.NewInt(1)) // Subtracting 1 to get all 1's in the host part.
	lastIPInt.Or(lastIPInt, firstIPInt)     // Adding in the network part.
	// We want the last usable IP so we subtract one from the broadcast address.
	// The exception is IPv4 /31 which is used for Point-to-Point networks where there is no broadcast address.
	if !(prefixLen == 31 && len(network.IP) == 4) {
		lastIPInt.Sub(lastIPInt, big.NewInt(1))
	}
	return net.IP(lastIPInt.Bytes()).String()
}

// New initializes a flag set using the provided arguments.
//
//   - args should start with the (sub)command's name
//   - usage is a function that prints a usage message
//   - set is a function that sets the command's flag arguments
func New(args []string, usage func(), set func(*flag.FlagSet)) *flag.FlagSet {
	if len(args) == 0 {
		args = append(args, "")
	}

	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Usage = func() {
		usage()
	}
	set(fs)
	fs.Parse(args[1:])
	return fs
}

// AddBool adds a bool option to a flag set
func AddBool(fs *flag.FlagSet, opt *bool, short, long string, value bool) {
	fs.BoolVar(opt, short, value, "")
	fs.BoolVar(opt, long, value, "")
}

// AddString adds a string option to a flag set
func AddString(fs *flag.FlagSet, opt *string, short, long, value string) {
	fs.StringVar(opt, short, value, "")
	fs.StringVar(opt, long, value, "")
}

type SliceFlag []string

func (i *SliceFlag) String() string {
	return "[]string"
}

func (i *SliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// AddStringS adds a string slice option to a flag set
func AddStringS(fs *flag.FlagSet, opt *SliceFlag, short, long, value string) {
	fs.Var(opt, short, value)
	fs.Var(opt, long, value)
}

// AddInt adds an int option to a flag set
func AddInt(fs *flag.FlagSet, opt *int, short, long string, value int) {
	fs.IntVar(opt, short, value, "")
	fs.IntVar(opt, long, value, "")
}

// ParseProvisioningURL constructs a provisioning URL from an absolute value or
// a template where the pattern "user:password" should be replaced with values.
func ParseProvisioningURL(absoluteURL, templateURL, user, password string) (string, error) {
	if absoluteURL == "" {
		if user == "" {
			user = DefUser
		}
		if password == "" {
			password = DefPassword
		}
		return verifyWebPrefix(strings.Replace(templateURL, "user:password", user+":"+password, 1))
	}
	if user != "" {
		return "", fmt.Errorf("cannot have both user and absolute provisiniong url")
	}
	if password != "" {
		return "", fmt.Errorf("cannot have both password and absolute provisiniong url")
	}
	return verifyWebPrefix(absoluteURL)
}

// verifyWebPrefix checks that url starts with "http://" or "https://"
func verifyWebPrefix(url string) (string, error) {
	if strings.HasPrefix(url, "http://") {
		return url, nil
	}
	if strings.HasPrefix(url, "https://") {
		return url, nil
	}
	return "", fmt.Errorf("provisioning URL must start with http:// or https://")
}

// flagRunning is defined in include/uapi/linux/if.h
const flagRunning = 1 << 6

// DefaultInterfaces outputs a list with one or more MAC addresses.  The
// associated interfaces have state UP and can thus be used as sane defaults.
// This corresponds to the network interface flags IFF_UP and IFF_RUNNING.
//
// Interfaces are put into state UP on a best-effort level.  If the appropriate
// permissions are lacking, an interface is simply skipped without any error.
func DefaultInterfaces(waitForInterface time.Duration) ([]net.HardwareAddr, error) {
	network.ForEachInterface(func(link netlink.Link) error {
		netlink.LinkSetUp(link)
		ctx, cancel := context.WithTimeout(context.Background(), waitForInterface)
		defer cancel()
		network.WaitForDeviceEvent(ctx, link.Attrs().Name, netlink.OperUp)
		return nil
	})

	var candidates []net.HardwareAddr
	network.ForEachInterface(func(link netlink.Link) error {
		// Skip bonding interfaces
		if strings.HasPrefix(link.Attrs().Name, "bond") {
			return nil
		}
		if link.Attrs().Flags&net.FlagUp != 0 && link.Attrs().RawFlags&flagRunning != 0 {
			candidates = append(candidates, link.Attrs().HardwareAddr)
		}
		return nil
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no default interface available")
	}

	return candidates, nil
}

func ValidateHostAndGateway(optHostIP, optGateway string, allowMinorQuirks, optTryLastIPForGateway bool) (string, error) {
	if len(optHostIP) == 0 {
		return "", fmt.Errorf("host ip is a required option")
	}

	hostIPAddr, hostIPPrefix, err := net.ParseCIDR(optHostIP)
	if err != nil {
		return "", fmt.Errorf("parsing host address: %v", err)
	}

	if len(optGateway) != 0 {
		gwIPAddr, _, err := net.ParseCIDR(appendPrefixLength(optGateway))
		if err != nil {
			return "", fmt.Errorf("%s: parsing gateway address: %v", optGateway, err)
		}
		if !allowMinorQuirks && !hostIPPrefix.Contains(gwIPAddr) {
			return "", fmt.Errorf("%s: gateway not within host IP network (%s)", gwIPAddr.String(), hostIPPrefix.String())
		}
	} else {
		if optTryLastIPForGateway {
			optGateway = MaxHost(hostIPPrefix)
		} else {
			addr, err := netip.ParseAddr(hostIPPrefix.IP.String())
			if err != nil {
				return "", fmt.Errorf("parsing host prefix: %v", err)
			}
			optGateway = addr.Next().String()
		}
	}

	gwIPAddr, _, err := net.ParseCIDR(appendPrefixLength(optGateway))
	if err != nil {
		return "", fmt.Errorf("%s: parsing gateway address: %v", optGateway, err)
	}
	if !allowMinorQuirks && hostIPAddr.Equal(gwIPAddr) {
		return "", fmt.Errorf("%v: host address must be distinct from gateway address", hostIPAddr)
	}

	return optGateway, nil
}

func appendPrefixLength(addr string) string {
	if strings.Contains(addr, ":") {
		if strings.HasSuffix(addr, "/128") {
			return addr
		} else {
			return addr + "/128"
		}
	} else {
		if strings.HasSuffix(addr, "/32") {
			return addr
		} else {
			return addr + "/32"
		}
	}
}
