package iam

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP's argon2id baseline. Stored in each hash, so raising them needs no migration.
const (
	argonMemory  = 19 * 1024 // KiB
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
)

var b64 = base64.RawStdEncoding

// hashPassword returns a PHC string: $argon2id$v=19$m=…,t=…,p=…$salt$hash.
func hashPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}
	salt, err1 := b64.DecodeString(parts[4])
	key, err2 := b64.DecodeString(parts[5])
	if err1 != nil || err2 != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1
}
