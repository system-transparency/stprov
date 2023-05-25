package remote

import (
	"flag"
	"fmt"
	"os"
	"time"

	"system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/subcmd/remote/dhcp"
	"system-transparency.org/stprov/subcmd/remote/run"
	"system-transparency.org/stprov/subcmd/remote/static"
)

const usage_string = `Usage:

  stprov remote dhcp   [-d DNS] [-m MAC] -h HOST_NAME -u USER -p PASSWORD
  stprov remote dhcp   [-d DNS] [-m MAC] -h HOST_NAME -r URL
  stprov remote static [-w WAIT] [-d DNS] [-m MAC] [-A] [-g GATEWAY] [-b INTERFACE] [-B] [-M BONDING_MODE] [-f] -h HOST_NAME -i HOST_IP [-u USER -p PASSWORD]
  stprov remote static [-w WAIT] [-d DNS] [-m MAC] [-A] [-g GATEWAY] [-b INTERFACE ] [-B] [-M BONDING_MODE] [-f] -h HOST_NAME -i HOST_IP -r URL

    Configures and persists a network configuration to EFI-NVRAM.

    Options:
    -d, --dns          DNS server (Default: %s)
    -m, --mac          MAC address of network interface (Default: guess)
    -A                 Attempt to auto-detect network interface
    -b, --bonding      Bonding interface into bond0, can be repeated
    -B, --bonding-auto Bonding auto-detected interfaces into bond0
    -M, --bonding-mode Bonding mode (Default: %s)
    -h, --host         Host name (amended with: %s)
    -H, --full-host    Full host name (e.g., localhost.local)
    -f                 Don't protect against minor configuration anomalies, like gw outside of subnet
    -i, --ip           Host IP in CIDR notation (e.g., 185.195.233.75/26)
    -I, --interface    Interface name of the network interface
    -g, --gateway      Default gateway (e.g., 185.195.233.65) (Default: Assumes first IP in the subnet)
    -u, --user         User name at provisioning server (Default: %s)
    -p, --pass         Password at provisioning server (e.g., mjaoouww)
    -r, --url          Absolute provisioning URL
    -w, --wait         Wait at most this long for link up (Default: 4s)

    The values of -u and -p will be incorporated into a hard-coded provisioning
    URL: "https://user:password@stpackage.example.org/os-stable.json".

    Bonding mode (-M) is one of: balance-rr, active-backup,
    balance-xor, broadcast, 802.3ad, balance-tlb, balance-alb.

  stprov remote run [-p PORT] [-i IP] [-a ALLOWED_HOSTS] -o OTP

    Awaits further configuration that is driven by stprov local.  A one-time
    password OTP is used to establish mutually authenticated HTTPS.

    Options:
    -p, --port   stprov remote listenting port (Default: 2009)
    -i, --ip     stprov remote listening ip (Default: 0.0.0.0)
    -a, --allow  stprov remote allowed bastion hosts (Default: %s)
    -o, --otp    one-time password (e.g., mjaoouuuuw)
`

const (
	efiUUID       = "f401f2c1-b005-4be0-8cee-f2e5945bcbe7"
	efiConfigName = "STHostConfig"
	efiKeyName    = "STHostKey"
	efiHostName   = "STHostName"
	provURL       = "https://user:password@stpackage.example.org/os-stable.json"
)

var (
	optDNS, optMAC, optHostName, optUser, optPassword, optURL          string
	optHostIP, optGateway, optAllowedCIDRs, optOTP, optFullHostName    string
	optInterfaceWait, optInterface                                     string
	optPort                                                            int
	optAutodetect, optBonding, optAllowConfigQuirks, optTryLastGateway bool
	optBondingInterfaces                                               options.SliceFlag
	optBondingMode                                                     string
)

func usage() {
	fmt.Fprintf(os.Stderr, usage_string,
		options.DefDNS,
		options.DefBondingMode,
		options.DefHostname,
		options.DefUser,
		options.DefAllowedNetworks)
}

func setOptions(fs *flag.FlagSet) {
	switch cmd := fs.Name(); cmd {
	case "help":
	case "static":
		options.AddString(fs, &optDNS, "d", "dns", options.DefDNS)
		options.AddString(fs, &optMAC, "m", "mac", "")
		options.AddString(fs, &optInterface, "I", "interface", "")
		options.AddBool(fs, &optAutodetect, "A", "autodetect", false)
		options.AddString(fs, &optHostName, "h", "host", "")
		options.AddString(fs, &optFullHostName, "H", "full-host", "")
		options.AddString(fs, &optHostIP, "i", "ip", "")
		options.AddString(fs, &optGateway, "g", "gateway", "")
		options.AddString(fs, &optUser, "u", "user", "")
		options.AddString(fs, &optPassword, "p", "pass", "")
		options.AddString(fs, &optURL, "r", "url", "")
		options.AddString(fs, &optInterfaceWait, "w", "wait", "4s")
		options.AddBool(fs, &optAllowConfigQuirks, "f", "force", false)
		//TODO: Include with DHCP
		options.AddStringS(fs, &optBondingInterfaces, "b", "bonding", "")
		options.AddBool(fs, &optBonding, "B", "bonding-auto", false)
		options.AddString(fs, &optBondingMode, "M", "bonding-mode", options.DefBondingMode)
		options.AddBool(fs, &optTryLastGateway, "x", "try-last-gateway", false)
	case "dhcp":
		options.AddString(fs, &optDNS, "d", "dns", options.DefDNS)
		options.AddString(fs, &optMAC, "m", "mac", "")
		options.AddString(fs, &optInterface, "I", "interface", "")
		options.AddString(fs, &optHostName, "h", "host", "")
		options.AddString(fs, &optFullHostName, "H", "full-host", "")
		options.AddString(fs, &optUser, "u", "user", "")
		options.AddString(fs, &optPassword, "p", "pass", "")
		options.AddString(fs, &optURL, "r", "url", "")
		options.AddString(fs, &optInterfaceWait, "w", "wait", "4s")
	case "run":
		options.AddInt(fs, &optPort, "p", "port", 2009)
		options.AddString(fs, &optHostIP, "i", "ip", "0.0.0.0")
		options.AddString(fs, &optAllowedCIDRs, "a", "allow", options.DefAllowedNetworks)
		options.AddString(fs, &optOTP, "o", "otp", "")
	}
}

func fmtErr(err error, name string) error {
	if err != nil {
		format := " %s: %w"
		if len(name) == 0 {
			format = "%s: %w"
		}
		err = fmt.Errorf(format, name, err)
	}

	return err
}

func Main(args []string) error {
	var err error
	var interfaceWait time.Duration

	opt := options.New(args, usage, setOptions)
	optHostName = fmt.Sprintf("%s.%s", optHostName, options.DefHostname)
	if optFullHostName != "" {
		optHostName = optFullHostName
	}
	if opt.Name() == "static" || opt.Name() == "dhcp" {
		if interfaceWait, err = time.ParseDuration(optInterfaceWait); err != nil {
			return fmtErr(err, opt.Name())
		}
	}

	if optInterface != "" {
		addr := network.GetHardwareAddr(optInterface)
		if addr == nil {
			return fmtErr(fmt.Errorf("invalid interface name %s", optInterface), opt.Name())
		}
		optMAC = addr.String()
	}

	switch opt.Name() {
	case "help", "":
		opt.Usage()
	case "static":
		err = static.Main(opt.Args(), optDNS, optMAC, optHostName, optHostIP, optGateway, optUser, optPassword, optURL, efiUUID, efiConfigName, efiHostName, provURL, interfaceWait, optAutodetect, optBonding, optBondingInterfaces, optBondingMode, optAllowConfigQuirks, optTryLastGateway)
	case "dhcp":
		err = dhcp.Main(opt.Args(), optDNS, optMAC, optHostName, optUser, optPassword, optURL, efiUUID, efiConfigName, efiHostName, provURL, interfaceWait, optAutodetect)
	case "run":
		err = run.Main(opt.Args(), optPort, optHostIP, optAllowedCIDRs, optOTP, efiUUID, efiConfigName, efiKeyName, efiHostName)
	default:
		err = fmt.Errorf("invalid command %q, try \"help\"", opt.Name())
	}

	return fmtErr(err, opt.Name())
}
