package admin_login

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"
)

var (
	// ErrInvalidInput indicates the admin login input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid admin login input")
	// ErrInvalidCredentials is the single authentication error returned for
	// unknown usernames, wrong passwords and wrong access codes. Callers
	// cannot distinguish which admin failed, whether a username exists or
	// which stage failed.
	ErrInvalidCredentials = errors.New("invalid admin credentials")
)

// Admin is the authenticated administrator identity.
type Admin struct {
	ID       string
	Username string
}

// AdminCredential is one configured administrator used to verify logins.
type AdminCredential struct {
	ID         string
	Username   string
	Password   string
	AccessCode string
}

// AdminService authenticates administrators against the configured
// credentials. Login completes the username/password stage; VerifyAccessCode
// completes the second authentication factor.
type AdminService interface {
	Login(ctx context.Context, input LoginInput) (Admin, error)
	VerifyAccessCode(ctx context.Context, input VerifyInput) (Admin, error)
}

// LoginInput carries the supplied credentials. The plaintext password is never
// logged, stored, returned or embedded in an error.
type LoginInput struct {
	Username string
	Password string
}

// VerifyInput carries the access code and the admin identity resolved from the
// temporary authentication state. The plaintext access code is never logged,
// stored, returned or embedded in an error.
type VerifyInput struct {
	AdminID    string
	AccessCode string
}

type adminLoginService struct {
	credentials []AdminCredential
}

func NewService(credentials []AdminCredential) AdminService {
	return &adminLoginService{credentials: credentials}
}

func (s *adminLoginService) Login(_ context.Context, input LoginInput) (Admin, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return Admin{}, ErrInvalidInput
	}

	for _, credential := range s.credentials {
		if credential.Username == username && constantTimeEqual(input.Password, credential.Password) {
			return Admin{ID: credential.ID, Username: credential.Username}, nil
		}
	}
	return Admin{}, ErrInvalidCredentials
}

func (s *adminLoginService) VerifyAccessCode(_ context.Context, input VerifyInput) (Admin, error) {
	if input.AdminID == "" || input.AccessCode == "" {
		return Admin{}, ErrInvalidInput
	}

	for _, credential := range s.credentials {
		if credential.ID == input.AdminID {
			if !constantTimeEqual(input.AccessCode, credential.AccessCode) {
				return Admin{}, ErrInvalidCredentials
			}
			return Admin{ID: credential.ID, Username: credential.Username}, nil
		}
	}
	return Admin{}, ErrInvalidCredentials
}

// constantTimeEqual compares two strings without leaking how much of the
// values match. Both sides are hashed first so the comparison always runs over
// fixed-length digests.
func constantTimeEqual(a, b string) bool {
	digestA := sha256.Sum256([]byte(a))
	digestB := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(digestA[:], digestB[:]) == 1
}
