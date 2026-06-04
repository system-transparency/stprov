package run

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"git.glasklar.is/nisse/tpm-lib/pkg/tpm"
	"github.com/google/uuid"

	"system-transparency.org/stboot/stlog"
	"system-transparency.org/stprov/internal/api"
	"system-transparency.org/stprov/internal/hexify"
	"system-transparency.org/stprov/internal/secrets"
	"system-transparency.org/stprov/internal/st"
)

func Main(args []string, optPort int, optIP string, optAllowHosts []string, optOTP string, efiUUID *uuid.UUID, efiConfigName, efiKeyName, efiHostName string) error {
	if len(args) != 0 {
		return fmt.Errorf("trailing arguments: %v", args)
	}
	if len(optOTP) == 0 {
		return fmt.Errorf("otp: one-time password is a required option")
	}
	port := optPort
	if port < 1 || port > 65535 {
		return fmt.Errorf("port: invalid: %d not in [1, 65535]", optPort)
	}
	ip := net.ParseIP(optIP)
	if ip == nil {
		return fmt.Errorf("ip: malformed ip address: %s", optIP)
	}
	allowNets, err := parseAllowedNets(optAllowHosts)
	if err != nil {
		return err
	}
	otp := optOTP

	ra()

	var hostname st.HostName
	if err := hostname.ReadEFI(efiUUID, efiHostName); err != nil {
		return fmt.Errorf("ReadEFI: %s: %w", efiHostName, err)
	}
	uds, err := listen(otp, allowNets, ip, port, hostname)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := writeHostKey(uds, efiUUID, efiKeyName); err != nil {
		return fmt.Errorf("persist host key: %w", err)
	}
	stlog.Info("efivar: ssh host key persisted")

	return nil
}

func ra() {
	ctx := context.Background()
	tpmDevice, err := openTPM(ctx)
	if err != nil {
		log.Printf("failed to open TPM device: %v", err)
		return
	}

	ekCertDER, err := getEkCertRsa(ctx, tpmDevice)
	if err != nil {
		log.Printf("failed getEkCertRsa: %s", err)
		return
	}

	ekCert, err := x509.ParseCertificate(ekCertDER)
	if err != nil {
		log.Printf("failed ParseCertificate: %s", err)
		return
	}

	log.Printf("ekCert fingerprint sha256: %0x", sha256.Sum256(ekCert.Raw))

	ekPub, err := x509.MarshalPKIXPublicKey(ekCert.PublicKey)
	if err != nil {
		log.Printf("failed MarshallPKIXPublicKey: %s", err)
		return
	}

	log.Printf("ekCert pubkey sha256: %0x", sha256.Sum256(ekPub))
}

// parseAllowedNets parses a list of addresses in CIDR format.  If an address
// omits the subnet mask, it will default to "/32" (IPv4) or "/128" (IPv6).
func parseAllowedNets(addresses []string) ([]net.IPNet, error) {
	var allowNets []net.IPNet
	for _, addr := range addresses {
		if !strings.Contains(addr, "/") {
			ip := net.ParseIP(addr)
			if ip == nil {
				return nil, fmt.Errorf("malformed address: %s", addr)
			}
			if ip.To4() != nil {
				addr += "/32"
			} else if ip.To16() != nil {
				addr += "/128"
			}
		}

		_, cidr, err := net.ParseCIDR(addr)
		if err != nil {
			return nil, fmt.Errorf("malformed address: %s", addr)
		}
		allowNets = append(allowNets, *cidr)
	}
	return allowNets, nil
}

// listen listens for incoming requests until a commit message is received.
// The admin running stprov remote must then give confirmation to proceed.
func listen(otp string, allowNets []net.IPNet, ip net.IP, port int, hostname st.HostName) (uds *secrets.UniqueDeviceSecret, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := api.NewServer(&api.ServerConfig{
		Secret:     otp,
		RemoteIP:   ip,
		RemotePort: port,
		LocalCIDR:  allowNets,
		Deadline:   15 * time.Second,
		Timeout:    60 * time.Second,
		HostName:   string(hostname),
	})
	if err != nil {
		return uds, fmt.Errorf("new server: %w", err)
	}
	log.Printf("starting server on %s:%d", srv.RemoteIP, srv.RemotePort)
	if err := srv.Run(ctx); err != nil {
		return uds, fmt.Errorf("run server: %w", err)
	}
	log.Printf("received entropy\n\n%s\n", hexify.Format(srv.Entropy[:]))
	if _, err := readLine("Press Enter to commit changes, ctrl+c to abort"); err != nil {
		return uds, fmt.Errorf("read confirmation: %w", err)
	}

	return srv.UDS, nil
}

func readLine(msg string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(msg)
	return reader.ReadString('\n')
}

// writeHostKey derives an SSH host key from a unique device secret, writing it
// to EFI-NVRAM
func writeHostKey(uds *secrets.UniqueDeviceSecret, varUUID *uuid.UUID, name string) error {
	hk, err := uds.SSH()
	if err != nil {
		return err
	}
	return hk.WriteEFI(varUUID, name)
}

// // Returns ek and ak keys.
// func getKeys(ctx context.Context, dev tpm.Device) (*TpmKey, *TpmKey, func(), error) {
// 	ek, f1, err := NewEndorsementKey(ctx, dev)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}
// 	ak, f2, err := NewAttestationKey(ctx, dev)
// 	if err != nil {
// 		f1()
// 		return nil, nil, nil, err
// 	}
// 	return ek, ak, func() { f1(); f2() }, nil
// }

func openTPM(ctx context.Context) (tpm.Device, error) {
	tpmName := os.Getenv("TEST_TPM")
	if tpmName == "" {
		// Requires root privs.
		tpmName = "/dev/tpm0"
	}

	dev, err := tpm.OpenTPM(tpmName)
	if err != nil {
		return nil, err
	}
	// Needed in case of swtpm.
	if err := tpm.Startup(ctx, dev, tpm.TPM_SU_CLEAR); err != nil && err != tpm.ErrRcInitialize {
		return nil, err
	}
	return dev, nil
}

func getEkCertRsa(ctx context.Context, dev tpm.Device) ([]byte, error) {
	idx := uint32(tpm.EkTemplateIndexL1)
	template, _, err := tpm.NvReadPublic(ctx, dev, idx)
	if err == nil {
		return nil, fmt.Errorf("endorsement key with non-default template: %x", template)
	} else if !errors.Is(err, tpm.ErrRcHandle) {
		return nil, fmt.Errorf("failed NvReadPublic idx==0x%x: %w", idx, err)
	}
	_, der, err := readNvIndex(ctx, dev, tpm.EkCertificateIndexL1, "")
	return der, err
}

type TpmKey struct {
	handle tpm.U32
	pub    tpm.Public
}

func readNvIndex(ctx context.Context, dev tpm.Device, idx uint32, pw string) (tpm.NvPublic, []byte, error) {
	nvpub, _, err := tpm.NvReadPublic(ctx, dev, idx)
	if err != nil {
		return tpm.NvPublic{}, nil, fmt.Errorf("failed NvReadPublic idx==0x%x: %w", idx, err)
	}

	// Get hold of TPM's nvBufMax property, which is the largest size
	// that the TPM can handle for NvRead, NvWrite, etc
	capData, _, err := tpm.GetCapability(ctx, dev, uint32(tpm.TPM_CAP_TPM_PROPERTIES), uint32(tpm.TPM_PT_NV_BUFFER_MAX), 1)
	if err != nil {
		return tpm.NvPublic{}, nil, fmt.Errorf("failed GetCapability: %w", err)
	}
	nvBufMax, ok := capData.TpmProperties[tpm.TPM_PT_NV_BUFFER_MAX]
	if !ok {
		return tpm.NvPublic{}, nil, fmt.Errorf("missing nvBufMax property")
	}

	var data bytes.Buffer
	offset := uint16(0)
	for data.Len() < int(nvpub.DataSize) {
		remaining := int(nvpub.DataSize) - data.Len()
		readSize := uint16(min(int(nvBufMax), remaining))
		block, err := tpm.NvRead(ctx, dev, idx, tpm.Password(pw), idx, readSize, offset)
		if err != nil {
			return tpm.NvPublic{}, nil, fmt.Errorf("failed NvRead idx==0x%x readSize==%d offset==%d DataSize==%d data.Len()==%d: %w", idx, readSize, offset, nvpub.DataSize, data.Len(), err)
		}
		data.Write(block)
		offset += readSize
	}

	return nvpub, data.Bytes(), nil
}
