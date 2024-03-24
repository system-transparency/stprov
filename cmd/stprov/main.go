package main

import (
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
		os.Exit(1)
	}
}
