package admin_login_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ryze/backend/services/admin_login"
)

var testCredentials = []admin_login.AdminCredential{
	{ID: "ADMIN_1", Username: "ryzeADMIN1", Password: "test-admin-one-password", AccessCode: "test-admin-one-access-code"},
	{ID: "ADMIN_2", Username: "ryzeADMIN2", Password: "test-admin-two-password", AccessCode: "test-admin-two-access-code"},
}

func newTestService() admin_login.AdminService {
	return admin_login.NewService(testCredentials)
}

func TestLoginAdmin1Valid(t *testing.T) {
	admin, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN1",
		Password: "test-admin-one-password",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if admin.ID != "ADMIN_1" {
		t.Fatalf("expected identity ADMIN_1, got %q", admin.ID)
	}
	if admin.Username != "ryzeADMIN1" {
		t.Fatalf("expected username ryzeADMIN1, got %q", admin.Username)
	}
}

func TestLoginAdmin2Valid(t *testing.T) {
	admin, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN2",
		Password: "test-admin-two-password",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if admin.ID != "ADMIN_2" {
		t.Fatalf("expected identity ADMIN_2, got %q", admin.ID)
	}
	if admin.Username != "ryzeADMIN2" {
		t.Fatalf("expected username ryzeADMIN2, got %q", admin.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	for _, attempt := range []admin_login.LoginInput{
		{Username: "ryzeADMIN1", Password: "wrong_password_123#"},
		{Username: "ryzeADMIN2", Password: "wrong_password_123#"},
	} {
		_, err := newTestService().Login(context.Background(), attempt)
		if !errors.Is(err, admin_login.ErrInvalidCredentials) {
			t.Fatalf("username %q: expected ErrInvalidCredentials, got %v", attempt.Username, err)
		}
	}
}

func TestLoginUnknownUsername(t *testing.T) {
	_, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "nobody",
		Password: "test-admin-one-password",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginCrossAdminCredentials(t *testing.T) {
	_, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN1",
		Password: "test-admin-two-password",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for mismatched credentials, got %v", err)
	}
}

func TestLoginEmptyUsername(t *testing.T) {
	for _, attempt := range []admin_login.LoginInput{
		{Username: "", Password: "test-admin-one-password"},
		{Username: "   ", Password: "test-admin-one-password"},
	} {
		_, err := newTestService().Login(context.Background(), attempt)
		if !errors.Is(err, admin_login.ErrInvalidInput) {
			t.Fatalf("username %q: expected ErrInvalidInput, got %v", attempt.Username, err)
		}
	}
}

func TestLoginEmptyPassword(t *testing.T) {
	_, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN1",
		Password: "",
	})
	if !errors.Is(err, admin_login.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLoginTrimsUsername(t *testing.T) {
	admin, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "  ryzeADMIN1  ",
		Password: "test-admin-one-password",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if admin.ID != "ADMIN_1" {
		t.Fatalf("expected identity ADMIN_1, got %q", admin.ID)
	}
}

func TestLoginErrorsNeverExposeCredentials(t *testing.T) {
	for _, attempt := range []admin_login.LoginInput{
		{Username: "ryzeADMIN1", Password: "test-admin-one-password"},
		{Username: "ryzeADMIN1", Password: "wrong_password_123#"},
		{Username: "nobody", Password: "test-admin-one-password"},
	} {
		_, err := newTestService().Login(context.Background(), attempt)
		if err == nil {
			continue
		}
		message := err.Error()
		if strings.Contains(message, "test-admin-one-password") || strings.Contains(message, "test-admin-two-password") {
			t.Fatalf("errors must never contain admin passwords: %q", message)
		}
		if strings.Contains(message, "ryzeADMIN") {
			t.Fatalf("errors must never reveal configured usernames: %q", message)
		}
	}
}

func TestVerifyAccessCodeAdmin1Valid(t *testing.T) {
	admin, err := newTestService().VerifyAccessCode(context.Background(), admin_login.VerifyInput{
		AdminID:    "ADMIN_1",
		AccessCode: "test-admin-one-access-code",
	})
	if err != nil {
		t.Fatalf("VerifyAccessCode: %v", err)
	}
	if admin.ID != "ADMIN_1" {
		t.Fatalf("expected identity ADMIN_1, got %q", admin.ID)
	}
	if admin.Username != "ryzeADMIN1" {
		t.Fatalf("expected username ryzeADMIN1, got %q", admin.Username)
	}
}

func TestVerifyAccessCodeAdmin2Valid(t *testing.T) {
	admin, err := newTestService().VerifyAccessCode(context.Background(), admin_login.VerifyInput{
		AdminID:    "ADMIN_2",
		AccessCode: "test-admin-two-access-code",
	})
	if err != nil {
		t.Fatalf("VerifyAccessCode: %v", err)
	}
	if admin.ID != "ADMIN_2" {
		t.Fatalf("expected identity ADMIN_2, got %q", admin.ID)
	}
	if admin.Username != "ryzeADMIN2" {
		t.Fatalf("expected username ryzeADMIN2, got %q", admin.Username)
	}
}

func TestVerifyAccessCodeWrongCode(t *testing.T) {
	for _, attempt := range []admin_login.VerifyInput{
		{AdminID: "ADMIN_1", AccessCode: "wrong_access_code_1!"},
		{AdminID: "ADMIN_2", AccessCode: "wrong_access_code_2!"},
	} {
		_, err := newTestService().VerifyAccessCode(context.Background(), attempt)
		if !errors.Is(err, admin_login.ErrInvalidCredentials) {
			t.Fatalf("admin %q: expected ErrInvalidCredentials, got %v", attempt.AdminID, err)
		}
	}
}

func TestVerifyAccessCodeCrossAdminCodes(t *testing.T) {
	for _, attempt := range []admin_login.VerifyInput{
		{AdminID: "ADMIN_1", AccessCode: "test-admin-two-access-code"},
		{AdminID: "ADMIN_2", AccessCode: "test-admin-one-access-code"},
	} {
		_, err := newTestService().VerifyAccessCode(context.Background(), attempt)
		if !errors.Is(err, admin_login.ErrInvalidCredentials) {
			t.Fatalf("admin %q: expected ErrInvalidCredentials for a foreign access code, got %v", attempt.AdminID, err)
		}
	}
}

func TestVerifyAccessCodeUnknownAdmin(t *testing.T) {
	_, err := newTestService().VerifyAccessCode(context.Background(), admin_login.VerifyInput{
		AdminID:    "ADMIN_3",
		AccessCode: "test-admin-one-access-code",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for an unknown admin id, got %v", err)
	}
}

func TestVerifyAccessCodeEmptyInput(t *testing.T) {
	for _, attempt := range []admin_login.VerifyInput{
		{AdminID: "", AccessCode: "test-admin-one-access-code"},
		{AdminID: "ADMIN_1", AccessCode: ""},
		{AdminID: "", AccessCode: ""},
	} {
		_, err := newTestService().VerifyAccessCode(context.Background(), attempt)
		if !errors.Is(err, admin_login.ErrInvalidInput) {
			t.Fatalf("input %+v: expected ErrInvalidInput, got %v", attempt, err)
		}
	}
}

func TestVerifyAccessCodeErrorsNeverExposeCredentials(t *testing.T) {
	for _, attempt := range []admin_login.VerifyInput{
		{AdminID: "ADMIN_1", AccessCode: "test-admin-one-access-code"},
		{AdminID: "ADMIN_1", AccessCode: "wrong_access_code_1!"},
		{AdminID: "ADMIN_3", AccessCode: "test-admin-two-access-code"},
	} {
		_, err := newTestService().VerifyAccessCode(context.Background(), attempt)
		if err == nil {
			continue
		}
		message := err.Error()
		if strings.Contains(message, "test-admin-one-access-code") || strings.Contains(message, "test-admin-two-access-code") {
			t.Fatalf("errors must never contain admin access codes: %q", message)
		}
		if strings.Contains(message, "ryzeADMIN") {
			t.Fatalf("errors must never reveal configured usernames: %q", message)
		}
	}
}
