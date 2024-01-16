// package ssh provides utilities to manage SSH host keys in Ed25519 format.
// Some code is copied with minor edits from golang.org/x/crypto/ssh, see
// separate LICENCE which applies from the write() method and forward.
package ssh

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/u-root/u-root/pkg/efivarfs"
	"golang.org/x/crypto/ssh"
)

const (
	PEMTypePrivateKey = "OPENSSH PRIVATE KEY"
)

// HostKey represents the private output of the following command:
//
//	ssh-keygen -t ed25519 -c "some comment"
type HostKey struct {
	Private ed25519.PrivateKey
	Check   uint32
	Comment string
}

// NewHostKey generates a new host key in Ed25519 format
func NewHostKey(rand io.Reader, comment string) (*HostKey, error) {
	_, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	var check [4]byte
	if _, err := rand.Read(check[:]); err != nil {
		return nil, fmt.Errorf("read random: %w", err)
	}
	return &HostKey{
		Private: priv,
		Check:   binary.BigEndian.Uint32(check[:]),
		Comment: comment,
	}, nil
}

// Fingerprint outputs a host key's SHA256 fingerprint
func (hk *HostKey) Fingerprint() (string, error) {
	pub, err := ssh.NewPublicKey(hk.Private.Public().(ed25519.PublicKey))
	if err != nil {
		return "", fmt.Errorf("derive public key: %w", err)
	}
	return ssh.FingerprintSHA256(pub), nil
}

// WriteEFI writes a host key to EFI-NVRAM in PEM format
func (hk *HostKey) WriteEFI(varUUID *uuid.UUID, name string) error {
	buf := bytes.NewBuffer(nil)
	if err := hk.writePEM(buf); err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	desc := efivarfs.VariableDescriptor{Name: name, GUID: *varUUID}
	attrs := efivarfs.AttributeBootserviceAccess
	attrs |= efivarfs.AttributeRuntimeAccess
	attrs |= efivarfs.AttributeNonVolatile
	e, err := efivarfs.New()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := efivarfs.WriteVariable(e, desc, attrs, buf.Bytes()); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// write writes an Ed25519 host key
func (hk *HostKey) write(w io.Writer) error {
	key := ed25519KeyBody{
		Pub:     make([]byte, ed25519.PublicKeySize),
		Priv:    make([]byte, ed25519.PrivateKeySize),
		Comment: hk.Comment,
	}
	copy(key.Pub, (hk.Private.Public().(ed25519.PublicKey))[:])
	copy(key.Priv, hk.Private[:])
	key.addPad()

	pk1 := keyBody{
		Check1:  hk.Check,
		Check2:  hk.Check,
		Keytype: ssh.KeyAlgoED25519,
		Rest:    ssh.Marshal(&key),
	}

	// Where is this format defined in the specs?  Had to look at the output of
	//
	//   ssh-keygen -t ed25519
	//
	// to figure out the appropriate bytes.
	pub := struct {
		Name  string
		Bytes []byte
	}{
		Name:  "ssh-ed25519",
		Bytes: key.Pub,
	}
	l := authBody{
		CipherName:   "none",
		KdfName:      "none",
		KdfOpts:      "",
		NumKeys:      1,
		PubKey:       ssh.Marshal(pub),
		PrivKeyBlock: ssh.Marshal(&pk1),
	}

	buf := bytes.NewBuffer(nil)
	if _, err := buf.Write([]byte(authMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}
	if _, err := buf.Write(ssh.Marshal(&l)); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write buf: %w", err)
	}
	return nil
}

// writePem writes and Ed25519 host key in PEM format
func (hk *HostKey) writePEM(w io.Writer) error {
	buf := bytes.NewBuffer(nil)
	if err := hk.write(buf); err != nil {
		return err
	}
	block := &pem.Block{
		Type:  PEMTypePrivateKey,
		Bytes: buf.Bytes(),
	}
	b := pem.EncodeToMemory(block)
	if b == nil {
		return fmt.Errorf("pem-encode key: failed")
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("finalize write: %w", err)
	}
	return nil
}

// Read reads an Ed25519 SSH host key after PEM decoding.  Mostly copied from:
//
//   - https://cs.opensource.google/go/x/crypto/+/master:ssh/keys.go;l=1250-1416
func (hk *HostKey) read(r io.Reader) error {
	key, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}

	if len(key) < len(authMagic) || string(key[:len(authMagic)]) != authMagic {
		return fmt.Errorf("ssh: invalid openssh private key format")
	}
	remaining := key[len(authMagic):]

	var w authBody
	if err := ssh.Unmarshal(remaining, &w); err != nil {
		return err
	}
	if w.KdfName != "none" || w.CipherName != "none" {
		return fmt.Errorf("ssh: cannot decode encrypted private keys")
	}

	var pk1 keyBody
	if err := ssh.Unmarshal(w.PrivKeyBlock, &pk1); err != nil {
		return err
	}
	if pk1.Check1 != pk1.Check2 {
		return fmt.Errorf("ssh: checkint mismatch")
	}

	// we only handle ed25519
	switch pk1.Keytype {
	case ssh.KeyAlgoED25519:
		var key ed25519KeyBody
		if err := ssh.Unmarshal(pk1.Rest, &key); err != nil {
			return err
		}
		if len(key.Priv) != ed25519.PrivateKeySize {
			return fmt.Errorf("ssh: private key unexpected length")
		}
		for i, b := range key.Pad {
			if int(b) != i+1 {
				return fmt.Errorf("ssh: padding not as expected")
			}
		}

		hk.Private = ed25519.PrivateKey(key.Priv)
		hk.Check = pk1.Check1
		hk.Comment = key.Comment
		return nil
	default:
		return fmt.Errorf("ssh: unhandled key type")
	}
}

// authMagic is defined here:
//
//   - https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key#L8
//   - https://cs.opensource.google/go/x/crypto/+/master:ssh/keys.go;l=1251
const authMagic = "openssh-key-v1\x00"

// authBody is defined here:
//
//   - https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key#L11-L19
//   - https://cs.opensource.google/go/x/crypto/+/master:ssh/keys.go;l=1257-1264
type authBody struct {
	CipherName   string
	KdfName      string
	KdfOpts      string
	NumKeys      uint32
	PubKey       []byte
	PrivKeyBlock []byte
}

// keyBody is defined here:
//
//   - https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key#L36-L49
//   - https://cs.opensource.google/go/x/crypto/+/master:ssh/keys.go;l=1287-1292
type keyBody struct {
	Check1  uint32
	Check2  uint32
	Keytype string
	Rest    []byte `ssh:"rest"`
}

// ed25519KeyBody is encoded using the same rules as SSH agent:
//
//   - https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.key#L51-L52
//   - https://cs.opensource.google/go/x/crypto/+/master:ssh/keys.go;l=1340-1345
type ed25519KeyBody struct {
	Pub     []byte
	Priv    []byte
	Comment string
	Pad     []byte `ssh:"rest"`
}

// addPad computes the padding for a single Ed25519 key manually.  The blob to
// compute the padding for is defined and implemented by OpenSSH here:
//
//   - https://github.com/openssh/openssh-portable/blob/8241b9c0529228b4b86d88b1a6076fb9f97e4a99/PROTOCOL.key#L36-L49
//   - https://github.com/openssh/openssh-portable/blob/8241b9c0529228b4b86d88b1a6076fb9f97e4a99/sshkey.c#L2812-L2831
//
// The pad length is 8 without a cipher:
//
//   - https://github.com/openssh/openssh-portable/blob/0d96b1506b2f4757fefa5d1f884d49e96a6fd4c3/cipher.c#L108
//
// Which means that the only things that end up counting towards padding are:
//
//	byte[] privatekey1
//	string comment1
//
// Note that what the spec refers to here as privatekey1 is not the raw Ed25519
// private key.  It is the serialization of "ssh-ed25519", pubkey (32 bytes),
// and private key (64 bytes).  As all these types are serialized as strings and
// then contactenated with each other, that sums up to 119 bytes in total.  The
// comment is also a string, and thus its size will be 4 + the comment length.
//
// So, what we need to pad is 119 + 4+len(comment) so that we get zero mod 8.
// If padding is required, it is the pad bytes 1, 2, ..., k that are added.
func (b *ed25519KeyBody) addPad() {
	for i := 0; (119+4+len(b.Comment)+i)%8 != 0; i++ {
		b.Pad = append(b.Pad, byte(i+1))
	}
}
