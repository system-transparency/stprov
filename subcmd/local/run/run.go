package run

import (
	"fmt"
	"log"
	"net"
	"os"

	"system-transparency.org/stprov/internal/api"
	"system-transparency.org/stprov/internal/hexify"
)

func Main(args []string, optPort int, optIP, optOTP, optPKFile, optKEKFile, optDBFile, optDBXFile string, optNoUEFIMenuReboot bool) error {
	// Parse options relating to secure connection
	if len(args) != 0 {
		return fmt.Errorf("trailing arguments: %v", args)
	}
	if len(optIP) == 0 {
		return fmt.Errorf("ip address is a required option")
	}
	if len(optOTP) == 0 {
		return fmt.Errorf("one-time password is a required option")
	}
	ip := net.ParseIP(optIP)
	if ip == nil {
		return fmt.Errorf("malformed ip address: %s", optIP)
	}
	port := optPort
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d not in [0, 65535]", optPort)
	}
	otp := optOTP

	// Parse options relating to Secure Boot
	pk, err := readOptionalFile(optPKFile)
	if err != nil {
		return fmt.Errorf("invalid Secure Boot PK: %w", err)
	}
	kek, err := readOptionalFile(optKEKFile)
	if err != nil {
		return fmt.Errorf("invalid Secure Boot KEK: %w", err)
	}
	db, err := readOptionalFile(optDBFile)
	if err != nil {
		return fmt.Errorf("invalid Secure Boot db: %w", err)
	}
	dbx, err := readOptionalFile(optDBXFile)
	if err != nil {
		return fmt.Errorf("invalid Secure Boot dbx: %w", err)
	}
	haveSBOpts := pk != nil || kek != nil || db != nil || dbx != nil
	okSBOpts := pk != nil && kek != nil && db != nil
	if haveSBOpts && !okSBOpts {
		return fmt.Errorf("invalid Secure Boot options: PK, KEK, and db are required")
	}

	// Perform local-remote ping pongs
	cli, err := api.NewClient(&api.ClientConfig{
		Secret:             otp,
		RemoteIP:           ip,
		RemotePort:         port,
		PK:                 pk,
		KEK:                kek,
		DB:                 db,
		DBX:                dbx,
		RebootIntoUEFIMenu: !optNoUEFIMenuReboot,
	})
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	data, err := cli.AddData()
	if err != nil {
		return fmt.Errorf("add data: %w", err)
	}
	if haveSBOpts {
		err = cli.AddSecureBootKeys()
		if err != nil {
			return fmt.Errorf("add Secure Boot keys: %w", err)
		}
	}
	cr, err := cli.Commit()
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("added entropy\n\n%s\n", hexify.Format(data.Entropy))
	fmt.Printf("fingerprint=%s\n", cr.Fingerprint)
	fmt.Printf("hostname=%s\n", cr.HostName)
	fmt.Printf("ip=%s\n", optIP)
	return nil
}

func readOptionalFile(filename string) ([]byte, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}
