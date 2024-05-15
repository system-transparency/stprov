package local

import (
	"flag"
	"fmt"
	"os"

	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/subcmd/local/run"
)

const usage = `Usage:

  stprov local run -o OTP -i IP_ADDR [-p PORT]

    Contributes entropy to stprov remote, which is listening on a given IP
    address (-i) and port (-p).  A one-time password (-o) is used to bootstrap
    HTTPS.  Outputs the following key-value pairs on success:

    fingerprint=<the platform's SSH hostkey fingerprint>
    hostname=<the platform's hostname>
    ip=<the platform's IP address>

  Options:

    -o, --otp   One-time password to establish a secure connection
    -i, --ip    Remote stprov address (e.g., 10.0.2.10)
    -p, --port  Remote stprov port (Default: 2009)
`

var (
	optPort       int
	optIP, optOTP string
)

func setOptions(fs *flag.FlagSet) {
	switch cmd := fs.Name(); cmd {
	case "help":
	case "run":
		options.AddInt(fs, &optPort, "p", "port", 2009)
		options.AddString(fs, &optIP, "i", "ip", "")
		options.AddString(fs, &optOTP, "o", "otp", "")
	}
}

func Main(args []string) error {
	var err error

	opt := options.New(args, func() { fmt.Fprintf(os.Stderr, usage) }, setOptions)
	switch opt.Name() {
	case "help", "":
		opt.Usage()
	case "run":
		err = run.Main(opt.Args(), optPort, optIP, optOTP)
	default:
		err = fmt.Errorf("invalid command %q, try \"help\"", opt.Name())
	}

	if err != nil {
		format := " %s: %w"
		if len(opt.Name()) == 0 {
			format = "%s: %w"
		}
		err = fmt.Errorf(format, opt.Name(), err)
	}

	return err
}
