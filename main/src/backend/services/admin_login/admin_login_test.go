package admin_login_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ryze/backend/services/admin_login"
)

var testCredentials = []admin_login.AdminCredential{
	{ID: "ADMIN_1", Username: "ryzeADMIN1", Password: "edgar_manager123#", AccessCode: "mz7Qx2!ryze2fa_admin1"},
	{ID: "ADMIN_2", Username: "ryzeADMIN2", Password: "sandro_manager123#", AccessCode: "np9Wv4#ryze2fa_admin2"},
}

func newTestService() admin_login.AdminService {
	return admin_login.NewService(testCredentials)
}

func TestLoginAdmin1Valid(t *testing.T) {
	admin, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN1",
		Password: "edgar_manager123#",
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
		Password: "sandro_manager123#",
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
		Password: "edgar_manager123#",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginCrossAdminCredentials(t *testing.T) {
	_, err := newTestService().Login(context.Background(), admin_login.LoginInput{
		Username: "ryzeADMIN1",
		Password: "sandro_manager123#",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for mismatched credentials, got %v", err)
	}
}

func TestLoginEmptyUsername(t *testing.T) {
	for _, attempt := range []admin_login.LoginInput{
		{Username: "", Password: "edgar_manager123#"},
		{Username: "   ", Password: "edgar_manager123#"},
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
		Password: "edgar_manager123#",
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
		{Username: "ryzeADMIN1", Password: "edgar_manager123#"},
		{Username: "ryzeADMIN1", Password: "wrong_password_123#"},
		{Username: "nobody", Password: "edgar_manager123#"},
	} {
		_, err := newTestService().Login(context.Background(), attempt)
		if err == nil {
			continue
		}
		message := err.Error()
		if strings.Contains(message, "edgar_manager123#") || strings.Contains(message, "sandro_manager123#") {
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
		AccessCode: "mz7Qx2!ryze2fa_admin1",
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
		AccessCode: "np9Wv4#ryze2fa_admin2",
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
		{AdminID: "ADMIN_1", AccessCode: "np9Wv4#ryze2fa_admin2"},
		{AdminID: "ADMIN_2", AccessCode: "mz7Qx2!ryze2fa_admin1"},
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
		AccessCode: "mz7Qx2!ryze2fa_admin1",
	})
	if !errors.Is(err, admin_login.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for an unknown admin id, got %v", err)
	}
}

func TestVerifyAccessCodeEmptyInput(t *testing.T) {
	for _, attempt := range []admin_login.VerifyInput{
		{AdminID: "", AccessCode: "mz7Qx2!ryze2fa_admin1"},
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
		{AdminID: "ADMIN_1", AccessCode: "mz7Qx2!ryze2fa_admin1"},
		{AdminID: "ADMIN_1", AccessCode: "wrong_access_code_1!"},
		{AdminID: "ADMIN_3", AccessCode: "np9Wv4#ryze2fa_admin2"},
	} {
		_, err := newTestService().VerifyAccessCode(context.Background(), attempt)
		if err == nil {
			continue
		}
		message := err.Error()
		if strings.Contains(message, "mz7Qx2!ryze2fa_admin1") || strings.Contains(message, "np9Wv4#ryze2fa_admin2") {
			t.Fatalf("errors must never contain admin access codes: %q", message)
		}
		if strings.Contains(message, "ryzeADMIN") {
			t.Fatalf("errors must never reveal configured usernames: %q", message)
		}
	}
}
