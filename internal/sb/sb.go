package sb

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"
	"time"

	"github.com/foxboron/go-uefi/efi"
	"github.com/foxboron/go-uefi/efi/signature"
	"github.com/foxboron/go-uefi/efivar"
	"github.com/google/uuid"
	"github.com/u-root/u-root/pkg/efivarfs"
	"system-transparency.org/stprov/internal/secrets"
)

// ReadOptionalESLFile reads an EFI signature list if filename is provided
func ReadOptionalESLFile(filename string) (*signature.SignatureDatabase, error) {
	if filename == "" {
		return nil, nil
	}
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	sd, err := signature.ReadSignatureDatabase(fp)
	return &sd, err
}

// IsSetupMode outputs the status of UEFI's SetupMode variable.  Note that this
// may not be toggled back from one to zero after writing PK until next boot.
func IsSetupMode() (bool, error) {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	b, err := readEFI(&varUUID, "SetupMode")
	if err != nil {
		return false, err
	}
	if len(b) != 1 {
		return false, fmt.Errorf("unexpected length: %w", err)
	}
	if b[0] != 0 && b[0] != 1 {
		return false, fmt.Errorf("unexpected value: %d", b[0])
	}
	return b[0] == 1, nil
}

// Provision provisions PK, KEK, db, and dbx.  Setup mode is assumed.
func Provision(pk, kek, db, dbx *signature.SignatureDatabase) error {
	if err := provision(efivar.Db, db); err != nil {
		return fmt.Errorf("provision db: %w", err)
	}
	if err := provision(efivar.Dbx, dbx); err != nil {
		return fmt.Errorf("provision dbx: %w", err)
	}
	if err := provision(efivar.KEK, kek); err != nil {
		return fmt.Errorf("provision KEK: %w", err)
	}
	if err := provision(efivar.PK, pk); err != nil {
		return fmt.Errorf("provision PK: %w", err)
	}
	return nil
}

// provision writes a Secure Boot policy object with a dummy authentication
// header.  Useful to provision the initial entries of db, dbx, KEK, and PK.
// Note that provisioning with dummy authentiation only works in setup mode.
func provision(v efivar.Efivar, sd *signature.SignatureDatabase) error {
	priv, crt, err := dummyKeyPair()
	if err != nil {
		return fmt.Errorf("create dummy key-pair: %w", err)
	}
	_, descriptor, err := signature.SignEFIVariable(v, sd, priv, crt)
	if err != nil {
		return fmt.Errorf("create authentication_v2 descriptor: %w", err)
	}
	var buf bytes.Buffer
	descriptor.Marshal(&buf)
	if err := efi.WriteEFIVariable(v.Name, buf.Bytes()); err != nil {
		return fmt.Errorf("enroll: %v", err)
	}
	return nil
}

func dummyKeyPair() (crypto.Signer, *x509.Certificate, error) {
	seed := []byte("fixed seed -- not a secret")
	label := "secure boot provisioning"
	priv, err := rsa.GenerateKey(secrets.Reader(seed, label, 0), 2048)
	if err != nil {
		return priv, nil, fmt.Errorf("generate key: %w", err)
	}
	tmpl := &x509.Certificate{
		Version: 3,
		Subject: pkix.Name{
			CommonName: "Secure Boot provisioning",
		},
		PublicKeyAlgorithm: x509.RSA,
		SignatureAlgorithm: x509.SHA256WithRSA,
		NotBefore:          time.Now().Add(-24 * time.Hour),
		NotAfter:           time.Now().Add(24 * time.Hour),
	}
	crtDER, err := x509.CreateCertificate(secrets.Reader(seed, label, 1), tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return priv, nil, fmt.Errorf("create certificate: %w", err)
	}
	crt, err := x509.ParseCertificate(crtDER)
	if err != nil {
		return priv, nil, fmt.Errorf("parse certificate: %w", err)
	}
	return priv, crt, nil
}

func readEFI(varUUID *uuid.UUID, efiName string) ([]byte, error) {
	desc := efivarfs.VariableDescriptor{Name: efiName, GUID: *varUUID}
	e, err := efivarfs.New()
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	_, b, err := efivarfs.ReadVariable(e, desc)
	return b, err
}
