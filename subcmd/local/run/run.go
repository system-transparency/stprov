package run

import (
	"fmt"
	"log"
	"net"

	"system-transparency.org/stprov/internal/api"
	"system-transparency.org/stprov/internal/hexify"
	"system-transparency.org/stprov/internal/sb"
)

func Main(args []string, optPort int, optIP, optOTP, optPKFile, optKEKFile, optDBFile, optDBXFile string) error {
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
	pk, err := sb.ReadOptionalESLFile(optPKFile)
	if err != nil {
		return fmt.Errorf("invalid pk: %w", err)
	}
	kek, err := sb.ReadOptionalESLFile(optKEKFile)
	if err != nil {
		return fmt.Errorf("invalid kek: %w", err)
	}
	db, err := sb.ReadOptionalESLFile(optDBFile)
	if err != nil {
		return fmt.Errorf("invalid db: %w", err)
	}
	dbx, err := sb.ReadOptionalESLFile(optDBXFile)
	if err != nil {
		return fmt.Errorf("invalid dbx: %w", err)
	}

	// Perform local-remote ping pongs
	cli, err := api.NewClient(&api.ClientConfig{
		Secret:     otp,
		RemoteIP:   ip,
		RemotePort: port,
		PK:         pk,
		KEK:        kek,
		DB:         db,
		DBX:        dbx,
	})
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	data, err := cli.AddData()
	if err != nil {
		return fmt.Errorf("add data: %w", err)
	}
	err = cli.AddSecureBootKeys()
	if err != nil {
		return fmt.Errorf("add Secure Boot keys: %w", err)
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
