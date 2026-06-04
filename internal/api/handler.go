package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"git.glasklar.is/nisse/tpm-lib/pkg/tpm"
	"system-transparency.org/stboot/stlog"
	"system-transparency.org/stprov/internal/sb"
	"system-transparency.org/stprov/internal/secrets"
)

// Handler implements the http.Handler interface
type Handler struct {
	Server      *Server
	Endpoint    string
	Method      string
	HandlerFunc func(context.Context, *Server, http.ResponseWriter, *http.Request) (int, error)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(h.Server.Deadline))
	defer cancel()

	if ok := h.verifyMethod(w, r); !ok {
		return
	}
	if ok := h.verifyNetwork(w, r); !ok {
		return
	}
	if ok := h.authenticateUser(w, r); !ok {
		return
	}

	if code, err := h.HandlerFunc(ctx, h.Server, w, r); err != nil {
		http.Error(w, http.StatusText(code), code)
	}
}

// verifyMethod checks that an appropriate HTTP method is used.  Error handling
// is based on RFC 7231, see Sections 6.5.5 (Status 405) and 6.5.1 (Status 400).
func (h Handler) verifyMethod(w http.ResponseWriter, r *http.Request) bool {
	if h.Method == r.Method {
		return true
	}

	code := http.StatusBadRequest
	if ok := h.Server.checkHTTPMethod(r.Method); ok {
		w.Header().Set("Allow", h.Method)
		code = http.StatusMethodNotAllowed
	}

	log.Printf("unexpected http method %s", r.Method)
	http.Error(w, fmt.Sprintf("%s", http.StatusText(code)), code)
	return false
}

// verifyNetwork enforces that the client connects from an expected CIDR range
func (h Handler) verifyNetwork(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("failed parsing request address %s", r.RemoteAddr)
		http.Error(w, "Malformed address:port format", http.StatusInternalServerError)
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		log.Printf("failed parsing request ip %s", r.RemoteAddr)
		http.Error(w, "Hostname must be an IP address", http.StatusInternalServerError)
		return false
	}
	for _, allowedNet := range h.Server.LocalCIDR {
		if allowedNet.Contains(ip) {
			return true
		}
	}
	log.Printf("blocked connection attempt from %s", r.RemoteAddr)
	http.Error(w, "Invalid IP address", http.StatusForbidden)
	return false
}

// authenticateUser enforces basic auth as defined in RFC 2617, Section 2.
func (h Handler) authenticateUser(w http.ResponseWriter, r *http.Request) bool {
	user, password, ok := r.BasicAuth()
	if !ok {
		log.Printf("request without basic auth header from %s", r.RemoteAddr)
		http.Error(w, "BasicAuth header is required", http.StatusForbidden)
		return false
	}
	if user != BasicAuthUser || password != h.Server.basicAuthPassword {
		log.Printf("unauthorized user %q and password %q from %s", user, password, r.RemoteAddr)
		http.Error(w, "BasicAuth credentials were insufficient", http.StatusForbidden)
		return false
	}
	return true
}

func handleTPMKeys(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	akPub, err := s.AKPub.Pack()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	keysResp := TPMKeysResponse{
		EKCert: s.EKCert,
		AKPub:  akPub,
	}
	b, err := json.Marshal(keysResp)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("marshal tpm-keys response: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("write tpm-keys response: %w", err)
	}

	return http.StatusOK, nil
}

// NOTE Below func is based st-complete-poc's server

func handleTPMQuote(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	var quoteReq TPMQuoteRequest
	if err := unpackPost(r, &quoteReq); err != nil {
		log.Printf("invalid tpm-quote request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}

	tlsNonce, err := getTLSConnectionNonce(r.TLS)
	if err != nil {
		log.Printf("failed getTLSConnectionNonce: %v", err)
		return http.StatusMisdirectedRequest, err
	}

	id, err := tpm.UnpackIdObject(quoteReq.ID)
	if err != nil {
		return http.StatusBadRequest, err
	}
	encrypted := quoteReq.Encrypted

	// Create a TPM session
	sess, err := tpm.StartAuthSession(ctx, s.TPMDevice, tpm.TPM_ALG_SHA256, rand.Reader)
	if err != nil {
		log.Printf("StartAuthSession failed: %v", err)
		return http.StatusInternalServerError, err
	}
	defer func() { tpm.FlushContext(ctx, s.TPMDevice, sess) }()

	_, _, err = tpm.PolicySecret(ctx, s.TPMDevice, tpm.TPM_RH_ENDORSEMENT, sess, tpm.Password(""), nil, nil, nil, 0)
	if err != nil {
		log.Printf("PolicySecret failed: %v", err)
		return http.StatusInternalServerError, err
	}

	decrypted, err := tpm.ActivateCredential(ctx, s.TPMDevice, s.AKHandle, s.EKHandle, tpm.Password(""), tpm.Policy(sess), *id, tpm.Buffer(encrypted))
	if err != nil {
		log.Printf("ActivateCredential failed: %v", err)
		return http.StatusForbidden, err
	}
	if len(decrypted) != 16 {
		log.Printf("unexpected nonce size")
		return http.StatusForbidden, err
	}

	attest, sig, err := quotePCRs(ctx, s.TPMDevice, s.AKHandle, s.AKPub, []int{4}, bytes.Join([][]byte{tlsNonce, decrypted}, nil))
	if err != nil {
		log.Printf("failed quotePCRs: %v", err)
		return http.StatusInternalServerError, err
	}
	attestBuf, err := attest.Pack()
	if err != nil {
		log.Printf("failed attest.Pack: %v", err)
		return http.StatusInternalServerError, err
	}
	sigBuf, err := sig.Pack()
	if err != nil {
		log.Printf("failed sig.Pack: %v", err)
		return http.StatusInternalServerError, err
	}

	quoteResp := TPMQuoteResponse{
		Attest:    attestBuf,
		Signature: sigBuf,
	}
	b, err := json.Marshal(quoteResp)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("marshal tpm-quote response: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("write tpm-quote response: %w", err)
	}

	return http.StatusOK, nil
}

func getTLSConnectionNonce(cs *tls.ConnectionState) ([]byte, error) {
	if cs == nil {
		return nil, fmt.Errorf("no TLS connection")
	}
	return cs.ExportKeyingMaterial("EXPERIMENTAL ST RA", nil, 16)
}

func quotePCRs(ctx context.Context, dev tpm.Device, akHandle tpm.U32, akPub tpm.Public, pcr []int, nonce []byte) (tpm.Attest, tpm.Signature, error) {
	keyQName, err := akPub.QName(tpm.TPM_RH_OWNER)
	if err != nil {
		return tpm.Attest{}, tpm.Signature{}, err
	}
	attest, sig, err := tpm.Quote(ctx, dev, akHandle, tpm.Password(""),
		tpm.Scheme{Scheme: tpm.TPM_ALG_NULL}, nonce,
		tpm.PcrSelection{tpm.PcrSelect{Hash: tpm.TPM_ALG_SHA256, Pcr: pcr}})
	if err != nil {
		return tpm.Attest{}, tpm.Signature{}, err
	}
	if got, want := attest.Magic, tpm.TPM_GENERATED_VALUE; got != want {
		return tpm.Attest{}, tpm.Signature{}, fmt.Errorf("bad attest magic value, got %x, want %x", got, want)
	}
	if got, want := attest.Type, tpm.TPM_ST_ATTEST_QUOTE; got != want {
		return tpm.Attest{}, tpm.Signature{}, fmt.Errorf("bad attest type, got %x, want %x", got, want)
	}
	if got, want := attest.Signer, keyQName; !bytes.Equal(got, want) {
		return tpm.Attest{}, tpm.Signature{}, fmt.Errorf("unexpected attestation signer: got %x, want %x", got, want)
	}
	if got, want := attest.Extra, nonce; !bytes.Equal(got, want) {
		return tpm.Attest{}, tpm.Signature{}, fmt.Errorf("extra data differs from nonce, got %x, want %x", got, want)
	}
	return attest, sig, nil
}

func handleAddData(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	var data AddDataRequest
	if err := unpackPost(r, &data); err != nil {
		log.Printf("invalid add-entropy request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}
	if got, want := len(data.Entropy), secrets.EntropyBytes; got != want {
		log.Printf("invalid add-data request from %s: wrong number of entropy bytes", r.RemoteAddr)
		return http.StatusBadRequest, fmt.Errorf("invalid number of entropy bytes: %d", got)
	}
	if got := data.Timestamp; got < 0 {
		log.Printf("invalid add-data request from %s: negative timestamp value", r.RemoteAddr)
		return http.StatusBadRequest, fmt.Errorf("invalid unix timestamp %d", got)
	}

	s.Timestamp = data.Timestamp
	copy(s.Entropy[:], data.Entropy)
	return http.StatusOK, nil
}

func handleAddSecureBoot(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	var rebootIntoUEFIMenu bool
	defer func() {
		if !rebootIntoUEFIMenu {
			return
		}
		if err := sb.RequestRebootIntoUEFIMenu(); err != nil {
			log.Printf("failed to request reboot into UEFI menu: %v", err)
			return
		}
		stlog.Info("requested the firmware to reboot into the UEFI menu on next boot")
	}()

	ok, err := sb.IsSetupMode()
	if err != nil {
		log.Printf("add-secure boot request from %s: failed to read SetupMode EFI variable, trying to proceed anyway", r.RemoteAddr)
	} else if !ok {
		err = fmt.Errorf("not in setup mode")
		log.Printf("add-secure boot request from %s: %v, aborting", r.RemoteAddr, err)
		return http.StatusForbidden, err
	}

	var data AddSecureBootRequest
	if err := unpackPost(r, &data); err != nil {
		log.Printf("invalid add-secure-boot request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}
	if err := data.Check(); err != nil {
		return http.StatusBadRequest, err
	}
	rebootIntoUEFIMenu = data.RebootIntoUEFIMenu
	if err := sb.Provision(data.PK, data.KEK, data.Db, data.Dbx); err != nil {
		log.Printf("failed to provision secure boot request from %s: %v", r.RemoteAddr, err)
		return http.StatusBadRequest, err
	}

	stlog.Info("efivarfs: Secure Boot keys provisioned")
	return http.StatusOK, nil
}

func handleCommit(ctx context.Context, s *Server, w http.ResponseWriter, r *http.Request) (int, error) {
	uds, err := secrets.NewUniqueDeviceSecret(&s.Entropy)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("new unique device secret: %w", err)
	}
	cr, err := NewCommitResponse(uds, s.HostName)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("new commit response: %w", err)
	}
	b, err := json.Marshal(cr)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("marshal commit response: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("write commit response: %w", err)
	}

	s.UDS = uds
	s.commit <- struct{}{}
	return http.StatusOK, nil
}

func unpackPost(req *http.Request, any interface{}) error {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, any)
}
