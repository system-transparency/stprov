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

    Connect to stprov remote, taking part in the provisioning of a new platform.
    A one-time password is used to establish a mutually authenticated HTTPS connection.

  Options:

    -o, --otp   One-time password to establish a secure connection
    -i, --ip    Listening address (e.g., 10.0.2.10)
    -p, --port  Listenting port (Default: 2009)
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
