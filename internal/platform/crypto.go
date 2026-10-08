package platform

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Keyring holds the column-encryption keys. The highest version encrypts;
// every version still listed decrypts, so keys rotate gradually.
type Keyring struct {
	current byte
	aeads   map[byte]cipher.AEAD
}

// ParseKeys reads "1:<base64 32 bytes>[,2:<base64 32 bytes>…]".
func ParseKeys(s string) (*Keyring, error) {
	keys := map[byte][]byte{}
	for part := range strings.SplitSeq(s, ",") {
		ver, b64, ok := strings.Cut(strings.TrimSpace(part), ":")
		v, err := strconv.ParseUint(ver, 10, 8)
		if !ok || err != nil || v == 0 {
			return nil, errors.New("want <version 1-255>:<base64 key>[,…]")
		}
		key, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("key %d: %w", v, err)
		}
		if _, dup := keys[byte(v)]; dup {
			return nil, fmt.Errorf("key %d listed twice", v)
		}
		keys[byte(v)] = key
	}
	return NewKeyring(keys)
}

// NewKeyring builds a keyring from 32-byte AES-256 keys by version.
func NewKeyring(keys map[byte][]byte) (*Keyring, error) {
	k := &Keyring{aeads: map[byte]cipher.AEAD{}}
	for v, key := range keys {
		if len(key) != 32 {
			return nil, fmt.Errorf("key %d: want 32 bytes, got %d", v, len(key))
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		if k.aeads[v], err = cipher.NewGCMWithRandomNonce(block); err != nil {
			return nil, err
		}
		k.current = max(k.current, v)
	}
	if len(k.aeads) == 0 {
		return nil, errors.New("no key")
	}
	return k, nil
}

type keyringKey struct{}

// WithKeyring puts the column-encryption keys into ctx; only cmd/server and internal/app call it.
func WithKeyring(ctx context.Context, k *Keyring) context.Context {
	return context.WithValue(ctx, keyringKey{}, k)
}

func keyringFrom(ctx context.Context) *Keyring {
	k, ok := ctx.Value(keyringKey{}).(*Keyring)
	if !ok {
		// Wiring bug, not a runtime condition.
		panic("platform: no keyring in context")
	}
	return k
}

// Encrypt seals plaintext as [key version][nonce][ciphertext]. aad names the
// column, so a value cannot be copied into another column.
func Encrypt(ctx context.Context, aad string, plaintext []byte) []byte {
	k := keyringFrom(ctx)
	return k.aeads[k.current].Seal([]byte{k.current}, nil, plaintext, []byte(aad))
}

var ErrDecrypt = errors.New("platform: cannot decrypt")

// Decrypt opens a value sealed by Encrypt under the same aad.
func Decrypt(ctx context.Context, aad string, sealed []byte) ([]byte, error) {
	if len(sealed) < 1 {
		return nil, ErrDecrypt
	}
	aead, ok := keyringFrom(ctx).aeads[sealed[0]]
	if !ok {
		return nil, fmt.Errorf("%w: unknown key version %d", ErrDecrypt, sealed[0])
	}
	pt, err := aead.Open(nil, nil, sealed[1:], []byte(aad))
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}
