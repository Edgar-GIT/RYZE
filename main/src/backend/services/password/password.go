package password

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

var (
	ErrEmptyPassword = errors.New("password must not be empty")
	ErrInvalidHash   = errors.New("invalid password hash")
)

// Argon2id parameters. Explicit so hashing behaviour is reproducible and
// never scattered as magic numbers.
const (
	argon2Version = 19
	saltLength    = 16
	keyLength     = 32
	timeCost      = 3
	memoryKiB     = 64 * 1024
	threads       = 4
)

// Verification safety caps: hashes declaring excessive parameters are
// rejected so a malformed or hostile hash cannot trigger an unbounded
// memory/CPU allocation during verification.
const (
	maxVerifyMemoryKiB = 512 * 1024
	maxVerifyTimeCost  = 32
	maxVerifyThreads   = 32
)

// HashPassword returns an Argon2id hash in PHC string format:
// $argon2id$v=19$m=...,t=...,p=...$<salt>$<key>. The hash contains every
// parameter required for verification and a unique random salt, so hashing
// the same password twice produces different results. The plaintext password
// is never stored or logged.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, timeCost, memoryKiB, threads, keyLength)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		memoryKiB,
		timeCost,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether the plaintext password matches the given
// hash. Malformed or empty hashes return ErrInvalidHash; empty passwords
// return ErrEmptyPassword. Verification never panics on malformed input.
func VerifyPassword(password, hash string) (bool, error) {
	if password == "" {
		return false, ErrEmptyPassword
	}

	params, err := parseHash(hash)
	if err != nil {
		return false, err
	}

	expected, err := base64.RawStdEncoding.DecodeString(params.key)
	if err != nil || len(expected) != keyLength {
		return false, ErrInvalidHash
	}

	key := argon2.IDKey([]byte(password), params.salt, params.time, params.memory, params.threads, keyLength)

	if subtle.ConstantTimeCompare(key, expected) != 1 {
		return false, nil
	}
	return true, nil
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	key     string
}

func parseHash(hash string) (*argon2Params, error) {
	if hash == "" {
		return nil, ErrInvalidHash
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return nil, ErrInvalidHash
	}
	if parts[2] != "v="+strconv.Itoa(argon2Version) {
		return nil, ErrInvalidHash
	}

	var params argon2Params
	for _, param := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(param, "=", 2)
		if len(pair) != 2 {
			return nil, ErrInvalidHash
		}

		var err error
		switch pair[0] {
		case "m":
			var value uint64
			value, err = strconv.ParseUint(pair[1], 10, 32)
			params.memory = uint32(value)
		case "t":
			var value uint64
			value, err = strconv.ParseUint(pair[1], 10, 32)
			params.time = uint32(value)
		case "p":
			var value uint64
			value, err = strconv.ParseUint(pair[1], 10, 8)
			params.threads = uint8(value)
		default:
			return nil, ErrInvalidHash
		}
		if err != nil {
			return nil, ErrInvalidHash
		}
	}

	if params.memory == 0 || params.time == 0 || params.threads == 0 {
		return nil, ErrInvalidHash
	}
	if params.memory > maxVerifyMemoryKiB || params.time > maxVerifyTimeCost || params.threads > maxVerifyThreads {
		return nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return nil, ErrInvalidHash
	}
	params.salt = salt
	params.key = parts[5]

	return &params, nil
}
