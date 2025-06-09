// package api provides a client and a server implementing the different
// exchanges that stprov local initiates towards stprov remote over HTTPS.
package api

import (
	"encoding/hex"
	"fmt"
	"time"

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

// AddSecureBootRequest is a request to provision Secure Boot keys.  The
// serialized blobs are expected to be valid authentication_v2 descriptors such
// that PK is self signed, KEK is signed by PK, and db/dbx are signed by KEK.
type AddSecureBootRequest struct {
	PK                 []byte `json:"pk"`
	KEK                []byte `json:"kek"`
	Db                 []byte `json:"db"`
	Dbx                []byte `json:"dbx"`
	RebootIntoUEFIMenu bool   `json:"reboot_into_uefi_menu"`
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
func NewAddSecureBootRequest(pk, kek, db, dbx []byte, rebootIntoUEFIMenu bool) (*AddSecureBootRequest, error) {
	req := AddSecureBootRequest{PK: pk, KEK: kek, Db: db, Dbx: dbx, RebootIntoUEFIMenu: rebootIntoUEFIMenu}
	return &req, req.Check()
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

// Check checks that the request has a PK, KEK, and db (dbx is optional)
func (r AddSecureBootRequest) Check() error {
	if len(r.PK) == 0 {
		return fmt.Errorf("invalid request: PK is required")
	}
	if len(r.KEK) == 0 {
		return fmt.Errorf("invalid request: KEK is required")
	}
	if len(r.Db) == 0 {
		return fmt.Errorf("invalid request: db is required")
	}
	return nil
}
