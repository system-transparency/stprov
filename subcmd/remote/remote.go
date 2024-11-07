package remote

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"system-transparency.org/stboot/host"
	"system-transparency.org/stprov/internal/network"
	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/internal/st"
	"system-transparency.org/stprov/subcmd/remote/dhcp"
	"system-transparency.org/stprov/subcmd/remote/run"
	"system-transparency.org/stprov/subcmd/remote/static"
)

const usage_string = `Usage:

  stprov remote run -o OTP [-i IP_ADDR] [-p PORT] [-a ALLOWED_HOST [-a ALLOWED_HOST ...]

    Starts a server on a given IP address (-i) and port (-o), waiting for
    commands from stprov local.  A one-time password (-o) is used to establish
    a mutually authenticated HTTPS connection.  Connections are only accepted
    from the allowed hosts (-a), a repeated option that uses CIDR notation.

    An SSH hostkey is written to EFI NVRAM on success.

  Options:

    -o, --otp    One-time password to establish a secure connection
    -i, --ip     Listening address (Default: 0.0.0.0)
    -p, --port   Listening port (Default: 2009)
    -a, --allow  Source IP addresses allowed to connect in CIDR notation
                 (Default: %s; can be repeated)

    If the subnet mask is omitted with the -a option, it defaults to "/32"
    (IPv4) or "/128" (IPv6).  E.g., 10.0.0.1 and 10.0.0.1/32 are equivalent.


  stprov remote dhcp -h HOSTNAME | -H FULL_HOSTNAME
                     -r OSPKG_URL [-r OSPKG_URL ...] [-u USER] [-p PASSWORD]
                     [-m MAC | -I INTERFACE | -w WAIT]
                     [-d DNS [-d DNS ...]]

    Configures the network using DHCP. If none of -m and -I are specified, the
    interface is guessed.

    A host configuration and a hostname is written to EFI NVRAM on success.


  stprov remote static -i HOST_ADDR
                       -h HOSTNAME | -H FULL_HOSTNAME
                       -r OSPKG_URL [-r OSPKG_URL ...] [-u USER] [-p PASSWORD]
                       [-A | -m MAC | -I INTERFACE | {-B | -b INTERFACE [-b INTERFACE ...]} [-M BONDING_MODE]] [-w WAIT]
                       [-g GATEWAY] [-x] [-f]
                       [-d DNS [-d DNS ...]]

    Configures a static network configuration and persist it to EFI-NVRAM.  If
    none of -m and -I are specified, the network interface is guessed.  If -A
    is specified, the interface guessing involves pinging the gateway.  If -B
    is specified, the interface guessing is instead tailored for bonding.

    A host configuration and a hostname is written to EFI NVRAM on success.

  Options:

    -i, --ip               Host address in CIDR notation (e.g., 10.0.2.10/26)
    -h, --host             Host name prefix (full host name becomes HOSTNAME.%s)
    -H, --full-host        Full host name (e.g., host.example.org)
    -r, --url              OS package URLs (see defaults below; can be repeated)
    -u, --user             User name when using a templated user:password URL (Default: %s)
    -p, --pass             Password when using a templated user:password URL (Default: %s)
    -m, --mac              MAC address of network interface to select (e.g., aa:bb:cc:dd:ee:ff)
    -I, --interface        Name of network interface to select (e.g., eth0)
    -A, --autodetect       Autodetect network interface and ping gateway
    -B, --bonding-auto     Autodetect network interfaces to bond into bond0
    -b, --bonding          Name of network interface to bond into bond0 (can be repeated)
    -M, --bonding-mode     Bonding mode (Default: %s)
    -w, --wait             Wait at most this long for link up (Default: 4s)
    -g, --gateway          Gateway IP address (Default: assuming first address in HOST_ADDR's network)
    -x, --try-last-gateway Override default gateway and instead assume last address in HOST_ADDR's network
    -f, --force            Proceed despite failing configuration sanity checks, logging ignored issues
    -d, --dns              DNS server IP addresses (Default: %s; can be repeated)

    The first occurrence of the pattern user:password in the specified OS
    package URL(s) are substituted with the values of -u and -p.  For example,
    "user:password" might get substituted to "alice:sikritpassword".

    The default OS package URL(s) are:
    %s.

    Bonding mode (-M) is one of: balance-rr, active-backup, balance-xor,
    broadcast, 802.3ad, balance-tlb, balance-alb.
`

const (
	efiKeyName  = "STHostKey"
	efiHostName = "STHostName"
	httpTimeout = 20 * time.Second

	trustPolicyRootFile = "/etc/trust_policy/tls_roots.pem"
)

var (
	optMAC, optHostName, optUser, optPassword                  string
	optHostIP, optGateway, optOTP, optFullHostName             string
	optInterfaceWait, optInterface                             string
	optPort                                                    int
	optAutodetect, optBondingAuto, optTryLastGateway, optForce bool
	optBondingInterfaces, optDNS, optURL, optAllowedCIDRs      options.SliceFlag
	optBondingMode                                             string
)

func usage() {
	fmt.Fprintf(os.Stderr, usage_string,
		strings.ReplaceAll(options.DefAllowedNetworks, ",", ", "),
		options.DefHostname,
		options.DefUser,
		options.DefPassword,
		options.DefBondingMode,
		strings.ReplaceAll(options.DefDNS, ",", ", "),
		strings.ReplaceAll(options.DefTemplateURL, ",", ",\n    "),
	)
}

func setOptions(fs *flag.FlagSet) {
	// common options for static and dhcp configuration
	common := func() {
		options.AddStringS(fs, &optDNS, "d", "dns", options.DefDNS)
		options.AddString(fs, &optMAC, "m", "mac", "")
		options.AddString(fs, &optInterface, "I", "interface", "")
		options.AddString(fs, &optHostName, "h", "host", "")
		options.AddString(fs, &optFullHostName, "H", "full-host", "")
		options.AddString(fs, &optUser, "u", "user", options.DefUser)
		options.AddString(fs, &optPassword, "p", "pass", options.DefPassword)
		options.AddStringS(fs, &optURL, "r", "url", options.DefTemplateURL)
		options.AddString(fs, &optInterfaceWait, "w", "wait", "4s")
		options.AddBool(fs, &optForce, "f", "force", false)
	}

	switch cmd := fs.Name(); cmd {
	case "help":
	case "static":
		common()
		options.AddBool(fs, &optAutodetect, "A", "autodetect", false)
		options.AddString(fs, &optHostIP, "i", "ip", "")
		options.AddString(fs, &optGateway, "g", "gateway", "")
		options.AddBool(fs, &optTryLastGateway, "x", "try-last-gateway", false)
		//TODO: Include with DHCP
		options.AddStringS(fs, &optBondingInterfaces, "b", "bonding", "")
		options.AddBool(fs, &optBondingAuto, "B", "bonding-auto", false)
		options.AddString(fs, &optBondingMode, "M", "bonding-mode", options.DefBondingMode)
	case "dhcp":
		common()
	case "run":
		options.AddInt(fs, &optPort, "p", "port", 2009)
		options.AddString(fs, &optHostIP, "i", "ip", "0.0.0.0")
		options.AddStringS(fs, &optAllowedCIDRs, "a", "allow", options.DefAllowedNetworks)
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

func countTrue(b ...bool) int {
	n := 0
	for _, v := range b {
		if v {
			n++
		}
	}
	return n
}

func Main(args []string) error {
	var err error
	var interfaceWait time.Duration

	opt := options.New(args, usage, setOptions)
	if optHostName != "" && optFullHostName != "" {
		return fmtErr(fmt.Errorf("-h and -H options are mutually exclusive"), opt.Name())
	}
	if countTrue(optAutodetect, optMAC != "", optInterface != "", optBondingAuto, len(optBondingInterfaces.Values) > 0) > 1 {
		return fmtErr(fmt.Errorf("-A, -m, -I, -B, and -b options are mutually exclusive"), opt.Name())
	}
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

	dnsServers, err := parseIPs(optDNS.Values)
	if err != nil {
		return fmtErr(fmt.Errorf("dns: %w", err), opt.Name())
	}

	efiConfigName, efiUUID, err := st.HostConfigEFIVariableName()
	if err != nil {
		return fmtErr(err, opt.Name())
	}

	switch opt.Name() {
	case "help", "":
		opt.Usage()
		return nil
	case "static":
		config, err := static.Config(opt.Args(), dnsServers, optMAC, optHostIP, optGateway, interfaceWait, optAutodetect, optBondingAuto, optBondingInterfaces.Values, optBondingMode, optForce, optTryLastGateway)
		if err != nil {
			return fmtErr(err, opt.Name())
		}
		return fmtErr(commitConfig(optHostName, config, optURL.Values, optUser, optPassword, optForce), opt.Name())
	case "dhcp":
		config, err := dhcp.Config(opt.Args(), dnsServers, optMAC, interfaceWait, optAutodetect)
		if err != nil {
			return fmtErr(err, opt.Name())
		}
		return fmtErr(commitConfig(optHostName, config, optURL.Values, optUser, optPassword, optForce), opt.Name())
	case "run":
		return fmtErr(run.Main(opt.Args(), optPort, optHostIP, optAllowedCIDRs.Values, optOTP, efiUUID, efiConfigName, efiKeyName, efiHostName), opt.Name())
	default:
		return fmt.Errorf("invalid command %q, try \"help\"", opt.Name())
	}
}

// parseIPs parses a list of zero or more IP addresses
func parseIPs(ips []string) ([]*net.IP, error) {
	var ret []*net.IP
	for _, ip := range ips {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return nil, fmt.Errorf("failed to parse %q as an IP address", ip)
		}
		ret = append(ret, &parsedIP)
	}
	return ret, nil
}

// checkURL checks url for validity, logging success or returning
// error
func checkURL(client http.Client, url string) error {
	if strings.Contains(url, options.DefUser+":"+options.DefPassword) {
		log.Println("WARNING: using default username and password")
	}
	resp, err := client.Head(url)
	if err != nil {
		return fmt.Errorf("HEAD request on %q failed: %w", url, err)
	}
	// Ignore any response body
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEAD request on %q failed, returned status: %q", url, resp.Status)
	}
	log.Printf("HEAD request on provisioning url gave content-length: %d, content-type: %q",
		resp.ContentLength, resp.Header.Get("content-type"))
	return nil
}

func commitConfig(optHostName string, config *host.Config, optURL []string, optUser, optPassword string, optForce bool) error {
	if len(optHostName) == 0 {
		return fmt.Errorf("host name is a required option")
	}
	hostName := st.HostName(optHostName)

	client, err := network.NewClient(trustPolicyRootFile)
	if err != nil {
		return fmt.Errorf("configure tls client: %w", err)
	}
	var urls []string
	for _, url := range optURL {
		u, err := options.ConstructURL(url, optUser, optPassword)
		if err != nil {
			return err // invalid url
		}
		if err := checkURL(client, u); err != nil {
			if !optForce {
				return fmt.Errorf("HEAD request failed: %w", err)
			}
			log.Printf("force flag: ignoring: %v", err)
		}
		urls = append(urls, u)
	}
	ospkgPointer := strings.Join(urls, ",")
	config.OSPkgPointer = &ospkgPointer

	_, efiGuid, err := st.HostConfigEFIVariableName()
	if err != nil {
		return fmt.Errorf("parse efi UUID: %w", err)
	}

	if err := hostName.WriteEFI(efiGuid, efiHostName); err != nil {
		return fmt.Errorf("persist host name: %w", err)
	}
	if err := st.WriteHostConfigEFI(config); err != nil {
		return fmt.Errorf("persist host config: %w", err)
	}
	return nil
}
