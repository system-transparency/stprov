package main

import (
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"

	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/internal/version"
	"system-transparency.org/stprov/subcmd/local"
	"system-transparency.org/stprov/subcmd/remote"
)

const usage = `Usage:

  stprov help
  stprov version
  stprov local <SUBCOMMAND> [Options]
  stprov remote <SUBCOMMAND> [Options]

The local and remote commands accept the subcommand "help".
`

func main() {
	var err error

	opt := options.New(os.Args[1:], func() { fmt.Fprintf(os.Stderr, usage) }, func(_ *flag.FlagSet) {})
	switch opt.Name() {
	case "help", "":
		opt.Usage()
	case "local":
		err = local.Main(opt.Args())
	case "remote":
		err = remote.Main(opt.Args())
	case "version":
		fmt.Println(version.Version)
	default:
		err = fmt.Errorf(": invalid command %q, try \"help\"", opt.Name())
	}

	if err != nil {
		format := "stprov %s%s\n"
		if len(opt.Name()) == 0 {
			format = "stprov%s%s\n"
		}

		fmt.Fprintf(os.Stderr, format, opt.Name(), err.Error())

		if opt.Name() == "local" {
			// Detect the err we get when user runs:
			// stprov local run -o incorrect-password
			testErr := x509.UnknownAuthorityError{}
			if ok := errors.As(err, &testErr); ok {
				fmt.Fprintf(os.Stderr, "The one-time password may be incorrect.\n")
			}
		}

		os.Exit(1)
	}
}
