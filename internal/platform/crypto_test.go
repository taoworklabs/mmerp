package platform_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func key(b byte) string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{b}, 32)) }

func TestEncryptRoundTripAndRotation(t *testing.T) {
	old, err := platform.ParseKeys("1:" + key(1))
	if err != nil {
		t.Fatal(err)
	}
	sealed := platform.Encrypt(platform.WithKeyring(t.Context(), old), "t.c", []byte("079123456789"))
	if sealed[0] != 1 || bytes.Contains(sealed, []byte("079123456789")) {
		t.Fatalf("sealed = %x", sealed)
	}

	both, err := platform.ParseKeys("1:" + key(1) + ", 2:" + key(2))
	if err != nil {
		t.Fatal(err)
	}
	ctx := platform.WithKeyring(t.Context(), both)
	if pt, err := platform.Decrypt(ctx, "t.c", sealed); err != nil || string(pt) != "079123456789" {
		t.Fatalf("old key: %q, %v", pt, err)
	}
	if v := platform.Encrypt(ctx, "t.c", []byte("x"))[0]; v != 2 {
		t.Fatalf("encrypts with version %d, want 2", v)
	}
	if _, err := platform.Decrypt(ctx, "t.other", sealed); !errors.Is(err, platform.ErrDecrypt) {
		t.Fatalf("wrong aad: %v", err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := platform.Decrypt(ctx, "t.c", sealed); !errors.Is(err, platform.ErrDecrypt) {
		t.Fatalf("tampered: %v", err)
	}
	if _, err := platform.Decrypt(platform.WithKeyring(t.Context(), mustKeys(t, "2:"+key(2))), "t.c", sealed); !errors.Is(err, platform.ErrDecrypt) {
		t.Fatalf("dropped key: %v", err)
	}
}

func mustKeys(t *testing.T, s string) *platform.Keyring {
	k, err := platform.ParseKeys(s)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestParseKeysRejects(t *testing.T) {
	for _, s := range []string{"", "x", "0:" + key(1), "1:short", "1:" + key(1) + ",1:" + key(2), "1:!!"} {
		if _, err := platform.ParseKeys(s); err == nil {
			t.Errorf("ParseKeys(%q) accepted", s)
		}
	}
}
