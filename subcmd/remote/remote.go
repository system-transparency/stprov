package remote

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/internal/st"
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
    -i, --ip           Host IP in CIDR notation (e.g., 10.0.2.10/26)
    -I, --interface    Interface name of the network interface
    -g, --gateway      Default gateway (e.g., 10.0.2.2) (Default: Assumes first IP in the subnet)
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
	httpTimeout   = 20 * time.Second
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
		return nil
	case "static":
		config, err := static.Config(opt.Args(), optDNS, optMAC, optHostIP, optGateway, interfaceWait, optAutodetect, optBonding, optBondingInterfaces, optBondingMode, optAllowConfigQuirks, optTryLastGateway)
		if err != nil {
			return fmtErr(err, opt.Name())
		}
		return fmtErr(commitConfig(optHostIP, config, optURL, provURL, optUser, optPassword), opt.Name())
	case "dhcp":
		config, err := dhcp.Config(opt.Args(), optDNS, optMAC, interfaceWait, optAutodetect)
		if err != nil {
			return fmtErr(err, opt.Name())
		}
		return fmtErr(commitConfig(optHostIP, config, optURL, provURL, optUser, optPassword), opt.Name())
	case "run":
		return fmtErr(run.Main(opt.Args(), optPort, optHostIP, optAllowedCIDRs, optOTP, efiUUID, efiConfigName, efiKeyName, efiHostName), opt.Name())
	default:
		return fmt.Errorf("invalid command %q, try \"help\"", opt.Name())
	}
}

// Checks url for validity, and logs any errors.
func checkURL(url string) {
	if strings.Contains(url, options.DefUser+":"+options.DefPassword) {
		log.Println("WARNING: using default username and password")
	}
	client := http.Client{Timeout: httpTimeout}
	resp, err := client.Head(url)
	if err != nil {
		log.Printf("WARNING: HEAD request on %q failed: %v", url, err)
		return
	}
	// Ignore any response body
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("WARNING: HEAD request on %q returned status: %q", url, resp.Status)
		return
	}
	log.Printf("HEAD request on provisioning url gave content-length: %d, content-type: %q",
		resp.ContentLength, resp.Header.Get("content-type"))
}

func commitConfig(optHostName string, config *st.HostConfig, optURL, provURL, optUser, optPassword string) error {
	if len(optHostName) == 0 {
		return fmt.Errorf("host name is a required option")
	}
	hostName := st.HostName(optHostName)

	url, err := options.ParseProvisioningURL(optURL, provURL, optUser, optPassword)
	if err != nil {
		return err // either invalid option combination or values
	}
	checkURL(url)
	config.OSPkgPointer = &url

	UUID, err := uuid.Parse(efiUUID)
	if err != nil {
		return fmt.Errorf("parse efi UUID: %w", err)
	}

	if err := hostName.WriteEFI(&UUID, efiHostName); err != nil {
		return fmt.Errorf("persist host name: %w", err)
	}
	if err := config.WriteEFI(&UUID, efiConfigName); err != nil {
		return fmt.Errorf("persist host config: %w", err)
	}
	return nil
}
