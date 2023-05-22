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

	EndpointAddData = "add-data"
	EndpointCommit  = "commit"

	BasicAuthUser = "example-user"
)

// AddDataRequest is the input of an add-data request
type AddDataRequest struct {
	// Entropy is 256 bits of entropy, used internally by stprov
	Entropy []byte `json:"entropy"`

	// Timestamp is the current UNIX time, used by stboot:
	// https://github.com/system-transparency/system-transparency#timestamp---json-number-or-null
	Timestamp int64 `json:"timestamp"`
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
		return nil, fmt.Errorf("api: %v", err)
	}
	return &AddDataRequest{entropy[:], time.Now().Unix()}, nil
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
