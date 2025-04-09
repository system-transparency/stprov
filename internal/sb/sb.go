package sb

import (
	"bytes"
	"crypto"
	"crypto/rand"
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
)

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

func IsDeployedMode() (bool, error) {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	b, err := readEFI(&varUUID, "DeployedMode")
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

func IsAuditMode() (bool, error) {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	b, err := readEFI(&varUUID, "AuditMode")
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

func IsSecureBoot() (bool, error) {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	b, err := readEFI(&varUUID, "SecureBoot")
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

func SetDeployedMode() error {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	return writeEFI([]byte{1}, &varUUID, "DeployedMode")
}

func SetSecureBoot() error {
	varUUID, _ := uuid.Parse("8be4df61-93ca-11d2-aa0d-00e098032b8c")
	return writeEFI([]byte{1}, &varUUID, "SecureBoot")
}

func writeEFI(b []byte, varUUID *uuid.UUID, efiName string) error {
	desc := efivarfs.VariableDescriptor{Name: efiName, GUID: *varUUID}
	attrs := efivarfs.AttributeBootserviceAccess
	attrs |= efivarfs.AttributeRuntimeAccess
	attrs |= efivarfs.AttributeNonVolatile
	e, err := efivarfs.New()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return efivarfs.WriteVariable(e, desc, attrs, b)
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

func ProvisionKeys(pkESL, kekESL, dbESL, dbxESL []byte) error {
	if len(dbESL) == 0 {
		return fmt.Errorf("required argument: db")
	}
	if len(kekESL) == 0 {
		return fmt.Errorf("required argument: KEK")
	}
	if len(pkESL) == 0 {
		return fmt.Errorf("required argument: PK")
	}

	dbx, err := parseESL(dbxESL)
	if err != nil {
		return fmt.Errorf("parse dbx: %w", err)
	}
	db, err := parseESL(dbESL)
	if err != nil {
		return fmt.Errorf("parse KEK: %w", err)
	}
	kek, err := parseESL(kekESL)
	if err != nil {
		return fmt.Errorf("parse KEK: %w", err)
	}
	pk, err := parseESL(pkESL)
	if err != nil {
		return fmt.Errorf("parse PK: %w", err)
	}

	if err := provision("db", efivar.Db, &db); err != nil {
		return fmt.Errorf("provision db: %w", err)
	}
	if err := provision("dbx", efivar.Dbx, &dbx); err != nil {
		return fmt.Errorf("provision dbx: %w", err)
	}
	if err := provision("KEK", efivar.KEK, &kek); err != nil {
		return fmt.Errorf("provision KEK: %w", err)
	}
	if err := provision("PK", efivar.PK, &pk); err != nil {
		return fmt.Errorf("provision PK: %w", err)
	}

	return nil
}

func parseESL(b []byte) (signature.SignatureDatabase, error) {
	if len(b) == 0 {
		return signature.SignatureDatabase{}, nil
	}

	buf := bytes.NewBuffer(b)
	return signature.ReadSignatureDatabase(buf)
}

func provision(name string, v efivar.Efivar, sd *signature.SignatureDatabase) error {
	priv, crt, err := fixedKeyPair()
	if err != nil {
		return fmt.Errorf("create fixed key-pair: %w", err)
	}
	_, descriptor, err := signature.SignEFIVariable(v, sd, priv, crt)
	if err != nil {
		return fmt.Errorf("create authentication_v2 descriptor: %w", err)
	}
	var buf bytes.Buffer
	descriptor.Marshal(&buf)
	if err := efi.WriteEFIVariable(name, buf.Bytes()); err != nil {
		return fmt.Errorf("enroll: %v", err)
	}
	return nil
}

func fixedKeyPair() (crypto.Signer, *x509.Certificate, error) {
	//seed := []byte("fixed seed -- not a secret")
	//label := "secure boot provisioning"
	//priv, err := rsa.GenerateKey(secrets.Reader(seed, label, 0), 2048)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
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
	//crtDER, err := x509.CreateCertificate(secrets.Reader(seed, label, 1), tmpl, tmpl, &priv.PublicKey, priv)
	crtDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return priv, nil, fmt.Errorf("create certificate: %w", err)
	}
	crt, err := x509.ParseCertificate(crtDER)
	if err != nil {
		return priv, nil, fmt.Errorf("parse certificate: %w", err)
	}
	return priv, crt, nil
}
