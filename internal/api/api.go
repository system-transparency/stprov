// package api provides a client and a server implementing the different
// exchanges that stprov local initiates towards stprov remote over HTTPS.
package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/foxboron/go-uefi/efi/signature"
	"system-transparency.org/stprov/internal/secrets"
)

const (
	Protocol = "stprov/v0.0.1"

	EndpointAddData       = "add-data"
	EndpointAddSecureBoot = "add-secure-boot"
	EndpointCommit        = "commit"

	BasicAuthUser = "example-user"
)

// AddDataRequest is the input of an add-data request
type AddDataRequest struct {
	// Entropy is 256 bits of entropy, used internally by stprov
	Entropy []byte `json:"entropy"`

	// Timestamp is obsolete.  The time of provisioning used to be in the
	// platform's host configuration, but has since been dropped.  See:
	// https://git.glasklar.is/system-transparency/core/stboot/-/commit/23b78d00f567925dba7b4870116ed77b6c014d50
	//
	// The timestamp is kept here until it is clear if it is not coming back.
	Timestamp int64 `json:"timestamp"`
}

// AddSecureBootRequest is a request to provision Secure Boot keys.  Implements
// JSON (un)marshalling, i.e., both MarshalJSON() and UnmarshalJSON().
type AddSecureBootRequest struct {
	PK  *signature.SignatureDatabase
	KEK *signature.SignatureDatabase
	Db  *signature.SignatureDatabase
	Dbx *signature.SignatureDatabase
}

// CommitResponse is the output of a commit request
type CommitResponse struct {
	Fingerprint    string `json:"fingerprint"`
	HostName       string `json:"hostname"`
	Authentication string `json:"authentication"`
	Identity       string `json:"identity"`
}

// NewAddData generates a new add-data request
func NewAddDataRequest() (*AddDataRequest, error) {
	entropy, err := secrets.NewEntropy()
	if err != nil {
		return nil, fmt.Errorf("api: %w", err)
	}
	return &AddDataRequest{entropy[:], time.Now().Unix()}, nil
}

// NewAddSecureBootRequest creates a new request to provision Secure Boot keys
func NewAddSecureBootRequest(pk, kek, db, dbx *signature.SignatureDatabase) (*AddSecureBootRequest, error) {
	if pk == nil || kek == nil || db == nil {
		return nil, fmt.Errorf("invalid request: PK, KEK, and db are required")
	}
	return &AddSecureBootRequest{PK: pk, KEK: kek, Db: db, Dbx: dbx}, nil
}

func NewCommitResponse(uds *secrets.UniqueDeviceSecret, hostname string) (*CommitResponse, error) {
	hk, err := uds.SSH()
	if err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	fpr, err := hk.Fingerprint()
	if err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	id, err := uds.Identity()
	if err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}
	auth, err := uds.Authentication()
	if err != nil {
		return nil, fmt.Errorf("authentication: %w", err)
	}
	return &CommitResponse{fpr, hostname, hex.EncodeToString(auth[:]), hex.EncodeToString(id[:])}, nil
}

func (r AddSecureBootRequest) MarshalJSON() ([]byte, error) {
	marshal := func(sd *signature.SignatureDatabase) []byte {
		if sd == nil {
			return nil
		}
		var buf bytes.Buffer
		sd.Marshal(&buf)
		return buf.Bytes()
	}

	return json.Marshal(addSecureBootRequest{
		PK:  marshal(r.PK),
		KEK: marshal(r.KEK),
		Db:  marshal(r.Db),
		Dbx: marshal(r.Dbx),
	})
}

func (r *AddSecureBootRequest) UnmarshalJSON(data []byte) error {
	var wire addSecureBootRequest
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	// Parse PK
	if wire.PK == nil {
		return fmt.Errorf("invalid request: PK is required")
	}
	pk, err := signature.ReadSignatureDatabase(bytes.NewBuffer(wire.PK))
	if err != nil {
		return fmt.Errorf("invalid request: PK: %w", err)
	}

	// Parse KEK
	if wire.KEK == nil {
		return fmt.Errorf("invalid request: KEK is required")
	}
	kek, err := signature.ReadSignatureDatabase(bytes.NewBuffer(wire.KEK))
	if err != nil {
		return fmt.Errorf("invalid request: KEK: %w", err)
	}

	// Parse db
	if wire.Db == nil {
		return fmt.Errorf("invalid request: db is required")
	}
	db, err := signature.ReadSignatureDatabase(bytes.NewBuffer(wire.Db))
	if err != nil {
		return fmt.Errorf("invalid request: db: %w", err)
	}

	// Parse dbx
	var dbx *signature.SignatureDatabase
	if wire.Dbx != nil {
		sd, err := signature.ReadSignatureDatabase(bytes.NewBuffer(wire.Dbx))
		if err != nil {
			return fmt.Errorf("invalid request: dbx: %w", err)
		}
		dbx = &sd
	}

	// OK
	r.PK = &pk
	r.KEK = &kek
	r.Db = &db
	r.Dbx = dbx
	return nil
}

// addSecureBootRequest is an internal struct used for JSON (un)marshalling
type addSecureBootRequest struct {
	PK  []byte `json:"pk"`
	KEK []byte `json:"kek"`
	Db  []byte `json:"db"`
	Dbx []byte `json:"dbx"`
}
