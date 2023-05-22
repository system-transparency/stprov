package main

import (
	"flag"
	"fmt"
	"os"

	"system-transparency.org/stprov/internal/options"
	"system-transparency.org/stprov/subcmd/local"
	"system-transparency.org/stprov/subcmd/remote"
)

var Version = "undefined"

const usage = `Usage:

  stprov local   Outputs detailed usage of stprov-local
  stprov remote  Outputs detailed usage of stprov-remote
  stprov version Outputs the version of this program

Cheat sheet:

  ### REMOTE
  stprov remote static -h example.org -i 192.168.1.75/26 -g 192.168.1.1 -b eth0 -b eth1 -u stboot -p ospkg-password
  stprov remote run -o "operations one-time password"
  shutdown -r +0

  ### LOCAL
  go install system-transparency.org/stprov/cmd/stprov@latest
  stprov local run -i 185.195.233.75 -o "operations one-time password" 
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
		fmt.Println(Version)
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
