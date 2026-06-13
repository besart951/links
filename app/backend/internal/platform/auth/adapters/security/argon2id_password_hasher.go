package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

type Argon2idPasswordHasher struct{}

func NewArgon2idPasswordHasher() Argon2idPasswordHasher {
	return Argon2idPasswordHasher{}
}

func (Argon2idPasswordHasher) Hash(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: random salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		enc.EncodeToString(salt),
		enc.EncodeToString(hash),
	), nil
}

func (Argon2idPasswordHasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, errors.New("password: invalid hash format")
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, errors.New("password: invalid params")
	}
	memory, err := parseParam(params[0], "m")
	if err != nil {
		return false, err
	}
	iterations, err := parseParam(params[1], "t")
	if err != nil {
		return false, err
	}
	parallelism, err := parseParam(params[2], "p")
	if err != nil {
		return false, err
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("password: decode salt: %w", err)
	}
	expected, err := enc.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("password: decode hash: %w", err)
	}
	actual := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parseParam(value, key string) (int, error) {
	prefix := key + "="
	if !strings.HasPrefix(value, prefix) {
		return 0, fmt.Errorf("password: missing %s param", key)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
	if err != nil {
		return 0, fmt.Errorf("password: parse %s: %w", key, err)
	}
	return n, nil
}
