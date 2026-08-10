package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	plaintext := "correct horse battery staple"

	hash, err := HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword: expected a non-empty hash")
	}
	if hash == plaintext {
		t.Fatal("HashPassword: hash must not equal the plaintext password")
	}
	if strings.Contains(hash, plaintext) {
		t.Fatal("HashPassword: hash must not contain the plaintext password")
	}
}

func TestVerifyPasswordSuccess(t *testing.T) {
	plaintext := "s3cret-password-123"

	hash, err := HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword(plaintext, hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword: expected the correct password to match")
	}
}

func TestVerifyPasswordWrongPassword(t *testing.T) {
	hash, err := HashPassword("right-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword: expected the wrong password not to match")
	}
}

func TestHashPasswordUniqueSalts(t *testing.T) {
	first, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword (first): %v", err)
	}
	second, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword (second): %v", err)
	}

	if first == second {
		t.Fatal("HashPassword: hashing the same password twice must produce different hashes")
	}

	okFirst, err := VerifyPassword("same-password", first)
	if err != nil {
		t.Fatalf("VerifyPassword (first): %v", err)
	}
	okSecond, err := VerifyPassword("same-password", second)
	if err != nil {
		t.Fatalf("VerifyPassword (second): %v", err)
	}
	if !okFirst || !okSecond {
		t.Fatal("VerifyPassword: both hashes of the same password must verify")
	}
}

func TestVerifyPasswordMalformedHash(t *testing.T) {
	cases := []string{
		"",
		"not-a-phc-hash",
		"$argon2id$v=19$m=65536,t=3,p=4$onlysalt$onlyhash",
		"$argon2id$v=19$m=65536,t=3,p=4$!!!$!!!",
		"$argon2id$v=18$m=65536,t=3,p=4$c2FsdA$c2FsdA",
		"$bcrypt$v=19$m=65536,t=3,p=4$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=0,t=3,p=4$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=999999999,t=3,p=4$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=65536,t=99,p=4$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=65536,t=3,p=99$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=notanumber,t=3,p=4$c2FsdA$c2FsdA",
		"$argon2id$v=19$m=65536,t=3,p=4$$",
	}

	for _, hash := range cases {
		ok, err := VerifyPassword("any-password", hash)
		if !errors.Is(err, ErrInvalidHash) {
			t.Errorf("hash %q: expected ErrInvalidHash, got ok=%v err=%v", hash, ok, err)
		}
	}
}

func TestEmptyPassword(t *testing.T) {
	if _, err := HashPassword(""); !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("HashPassword(\"\"): expected ErrEmptyPassword, got %v", err)
	}

	hash, err := HashPassword("some-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if _, err := VerifyPassword("", hash); !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("VerifyPassword(\"\", hash): expected ErrEmptyPassword, got %v", err)
	}
	if _, err := VerifyPassword("some-password", ""); !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("VerifyPassword(pw, \"\"): expected ErrInvalidHash, got %v", err)
	}
}
