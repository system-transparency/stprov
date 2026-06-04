package run

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"slices"

	"git.glasklar.is/nisse/tpm-lib/pkg/tpm"
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

	ekCert, akPub, err := getKeys(cli)
	if err != nil {
		return fmt.Errorf("failed getKeys: %w", err)
	}

	quote, err := getAttestationPcr4(cli, ekCert, akPub)
	if err != nil {
		return fmt.Errorf("failed getAttestationPcr4: %w", err)
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
	fmt.Printf("publickey=%s\n", cr.PublicKey)
	fmt.Printf("fingerprint=%s\n", cr.Fingerprint)
	fmt.Printf("hostname=%s\n", cr.HostName)
	fmt.Printf("ip=%s\n", optIP)
	fmt.Printf("ekcert=%0x\n", ekCert.Raw)
	fmt.Printf("ekcert-fingerprint-sha256=%0x\n", sha256.Sum256(ekCert.Raw))
	// The quote is a hash over PCR value/values; for us it's presently sha256(PCR4-value)
	fmt.Printf("quote-pcr4=%x\n", quote)
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

func getKeys(cli *api.Client) (*x509.Certificate, *tpm.Public, error) {
	var (
		// x509 package doesn't know about Endorsement Key Certificate
		// as a key usage.
		endorsementKeyUsage = asn1.ObjectIdentifier([]int{2, 23, 133, 8, 1})
		subjectAltName      = asn1.ObjectIdentifier([]int{2, 5, 29, 17})
	)

	keysResp, err := cli.TPMKeys()
	if err != nil {
		return nil, nil, fmt.Errorf("tpmkeys: %w", err)
	}

	akPub, err := tpm.UnpackPublic(keysResp.AKPub)
	if err != nil {
		return nil, nil, err
	}

	ekCert, err := x509.ParseCertificate(keysResp.EKCert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed parse EK cert: %w", err)
	}
	if (ekCert.KeyUsage & x509.KeyUsageKeyEncipherment) == 0 {
		return nil, nil, fmt.Errorf("endorsement key cert missing proper key usage, got: %x", ekCert.KeyUsage)
	}
	if !slices.ContainsFunc(ekCert.UnknownExtKeyUsage, endorsementKeyUsage.Equal) {
		return nil, nil, fmt.Errorf("endorsement key cert missing proper extended key usage, got: %s", ekCert.UnknownExtKeyUsage)
	}
	// TODO: Do we need to care about Subject Alternative Name?
	// With swtpm, it is set and marked critical.
	ekCert.UnhandledCriticalExtensions = slices.DeleteFunc(ekCert.UnhandledCriticalExtensions, subjectAltName.Equal)

	// TODO here one could optionally validate EK cert towards TPM
	// vendor's root CA (Loïc's tools https://github.com/loicsikidi/tpm-ca-certificates)
	// Overall for stprov, an inventory should also be considered;
	// see further notes under 2026-06-08 in work.md.
	//
	// if _, err := ekCert.Verify(x509.VerifyOptions{
	// 	Roots:         roots,
	// 	Intermediates: intermediates,
	// 	KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	// }); err != nil {
	// 	return nil, fmt.Errorf("failed to validate certificate with issuer %s and subject %s: %v", ekCert.Issuer, ekCert.Subject, err)
	// }

	return ekCert, akPub, nil
}

func getAttestationPcr4(cli *api.Client, ekCert *x509.Certificate, akPub *tpm.Public) (tpm.Buffer, error) {
	// NOTE Based on GetAttestationPcr4 in st-complete-poc's client (but keys extraction factored out)

	ek, err := Certificate2Public(ekCert)
	if err != nil {
		return nil, err
	}

	verifier, err := AttestationPublic2Verifier(akPub)
	if err != nil {
		return nil, err
	}
	akName, err := akPub.Name()
	if err != nil {
		return nil, err
	}
	akQName, err := akPub.QName(tpm.TPM_RH_OWNER)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 16)
	rand.Read(nonce)

	id, encrypted, err := ek.MakeCredential(akName, nonce, rand.Reader)
	if err != nil {
		return nil, err
	}

	// NOTE Based on GetQuotePcr4 in st-complete-poc's client
	quoteResp, tlsNonce, err := cli.TPMQuote(id, encrypted)
	if err != nil {
		return nil, fmt.Errorf("tpmquote: %w", err)
	}
	sig, err := tpm.UnpackSignature(quoteResp.Signature)
	if err != nil {
		return nil, fmt.Errorf("UnpackSignature %x failed: %v", quoteResp.Signature, err)
	}
	if err := verifier.Verify(quoteResp.Attest, sig); err != nil {
		return nil, err
	}
	attest, err := tpm.UnpackAttest(quoteResp.Attest)
	if err != nil {
		return nil, err
	}

	// Check the fixed parts of the signed attestation.
	if got, want := attest.Magic, tpm.TPM_GENERATED_VALUE; got != want {
		return nil, fmt.Errorf("bad attest magic value, got %x, want %x", got, want)
	}
	if got, want := attest.Type, tpm.TPM_ST_ATTEST_QUOTE; got != want {
		return nil, fmt.Errorf("bad attest type, got %x, want %x", got, want)
	}
	if got, want := attest.Signer, akQName; !bytes.Equal(got, want) {
		return nil, fmt.Errorf("unexpected attestation signer: got %x, want %x", got, want)
	}
	if got, want := attest.Extra, bytes.Join([][]byte{tlsNonce, nonce}, nil); !bytes.Equal(got, want) {
		return nil, fmt.Errorf("signed data doesn't use the expected nonces: got %x, want %x", got, want)
	}
	if attest.Quote == nil {
		return nil, fmt.Errorf("missing quote, internal error")
	}
	quote := attest.Quote
	if got, want := len(quote.Select), 1; got != want {
		return nil, fmt.Errorf("unexpected number of selections in quote: got %d, want %d", got, want)
	}
	selected := quote.Select[0]
	if got, want := selected.Hash, tpm.TPM_ALG_SHA256; got != want {
		return nil, fmt.Errorf("unexpected hash in quote: got %d, want %d", got, want)
	}
	if got, want := len(selected.Pcr), 1; got != want {
		return nil, fmt.Errorf("unexpected number of PCRs in quote: got %d, want %d", got, want)
	}

	return quote.Digest, nil
}

// NOTE All below is copypasted from st-complete-poc

// For now, hardcoded for an EK key according to tpm.EkTemplateL1.
func Certificate2Public(cert *x509.Certificate) (tpm.Public, error) {
	template := tpm.EkTemplateL1.Public
	if cert.IsCA {
		return tpm.Public{}, fmt.Errorf("unexpected CA certificate, not allowed")
	}
	if got, want := cert.PublicKeyAlgorithm, x509.RSA; got != want {
		return tpm.Public{}, fmt.Errorf("unexpected public key algorithm %v, only RSA supproted", got)
	}
	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return tpm.Public{}, fmt.Errorf("unexpected public key type %T, only RSA supproted", cert.PublicKey)
	}
	if got, want := publicKey.N.BitLen(), int(template.RsaParameters.KeyBits); got != want {
		return tpm.Public{}, fmt.Errorf("unexpected RSA key size, got %d, want %d bits", got, want)
	}
	if got, want := publicKey.E, 65537; got != want {
		return tpm.Public{}, fmt.Errorf("unexpected RSA key exponent, got %d, want %d bits", got, want)
	}

	unique := tpm.Buffer(make([]byte, 256))
	publicKey.N.FillBytes(unique)
	return tpm.Public{
		Type:             tpm.TPM_ALG_RSA,
		NameAlg:          template.NameAlg,
		ObjectAttributes: template.ObjectAttributes,
		AuthPolicy:       template.AuthPolicy,
		RsaParameters: &tpm.RsaParms{
			Symmetric: template.RsaParameters.Symmetric,
			Scheme:    template.RsaParameters.Scheme,
			KeyBits:   template.RsaParameters.KeyBits,
			Exponent:  0,
			Unique:    unique,
		},
	}, nil
}

type Verifier interface {
	Verify([]byte, *tpm.Signature) error
}

type ecdsaVerifier struct {
	pub ecdsa.PublicKey
}

func (v *ecdsaVerifier) Verify(data []byte, sig *tpm.Signature) error {
	if sig.Algorithm != tpm.TPM_ALG_ECDSA {
		return fmt.Errorf("unexpected signature algorithm 0x%x", sig.Algorithm)
	}
	if sig.Ecc == nil {
		return fmt.Errorf("ecc signature parameters missing")
	}

	if sig.Ecc.Hash != tpm.TPM_ALG_SHA256 {
		return fmt.Errorf("unsupported ecdsa hash algorithm 0x%x", sig.Ecc.Hash)
	}
	if lr, ls := len(sig.Ecc.R), len(sig.Ecc.S); lr != 32 || ls != 32 {
		return fmt.Errorf("unexpected size of signature blobs: %d, %d (expected == 32)", lr, ls)
	}
	r := big.NewInt(0)
	s := big.NewInt(0)
	r.SetBytes(sig.Ecc.R)
	s.SetBytes(sig.Ecc.S)
	hash := sha256.Sum256(data)
	if !ecdsa.Verify(&v.pub, hash[:], r, s) {
		return fmt.Errorf("invalid ecdsa signature")
	}
	return nil
}

// Checks that attestation key has expected properties (attributes
// tpm.TPMA_OBJECT_RESTRICTED_SIGN being crucial), and return a
// verifier.
func AttestationPublic2Verifier(ak *tpm.Public) (Verifier, error) {
	if ak.Type != tpm.TPM_ALG_ECC {
		return nil, fmt.Errorf("unsupported attestation signature algorithm: 0x%x", ak.Type)
	}
	if ak.NameAlg != tpm.TPM_ALG_SHA256 {
		return nil, fmt.Errorf("unsupported attestation name algorithm: 0x%x", ak.NameAlg)
	}
	if got, want := ak.ObjectAttributes, tpm.TPMA_OBJECT_RESTRICTED_SIGN; got != want {
		return nil, fmt.Errorf("unexpected attestation key properties: got %x, want %x (RESTRICTED_SIGN)", got, want)
	}
	if !bytes.Equal(ak.AuthPolicy, make([]byte, 32)) {
		return nil, fmt.Errorf("unexpected attestation auth policy: 0x%x", ak.AuthPolicy)

	}

	ecc := ak.EccParameters
	if ecc == nil {
		return nil, fmt.Errorf("missing ECC parameters")
	}
	if got := ecc.CurveID; got != tpm.TPM_ECC_NIST_P256 {
		return nil, fmt.Errorf("unsupported curve: 0x%x", got)
	}
	if got := ecc.Symmetric.Algorithm; got != tpm.TPM_ALG_NULL {
		return nil, fmt.Errorf("unsupported symmetric algorithm: 0x%x", got)
	}
	if got := ecc.Scheme.Scheme; got != tpm.TPM_ALG_ECDSA {
		return nil, fmt.Errorf("unsupported ecc scheme: 0x%x", got)
	}
	if got := ecc.Scheme.HashAlg; got != tpm.TPM_ALG_SHA256 {
		return nil, fmt.Errorf("unsupported ecc hash algorithm: 0x%x", got)
	}
	if got := ecc.KDF.Scheme; got != tpm.TPM_ALG_NULL {
		return nil, fmt.Errorf("unsupported ecc kdf: 0x%x", got)
	}

	point := ak.EccParameters.Unique
	if lx, ly := len(point.X), len(point.Y); lx != 32 || ly != 32 {
		return nil, fmt.Errorf("unexpected size of ecc points: %d, %d (expected == 32)", lx, ly)
	}

	verifier := ecdsaVerifier{pub: ecdsa.PublicKey{Curve: elliptic.P256(), X: big.NewInt(0), Y: big.NewInt(0)}}
	verifier.pub.X.SetBytes(point.X)
	verifier.pub.Y.SetBytes(point.Y)
	return &verifier, nil
}
