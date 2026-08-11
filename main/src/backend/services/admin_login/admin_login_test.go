package admin_login_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ryze/backend/services/admin_login"
)

var testCredentials = []admin_login.AdminCredential{
	{ID: "ADMIN_1", Username: "ryzeADMIN1", Password: "edgar_manager123#"},
	{ID: "ADMIN_2", Username: "ryzeADMIN2", Password: "sandro_manager123#"},
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
