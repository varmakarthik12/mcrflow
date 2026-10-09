package auth_test

import (
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestTokenGenerationAndValidation(t *testing.T) {
	svc := auth.NewTokenService("test-secret-key-12345")

	tokenStr, err := svc.GenerateToken("usr-123", "operator1", models.RoleOperator, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if claims.UserID != "usr-123" || claims.Username != "operator1" || claims.Role != models.RoleOperator {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestExpiredToken(t *testing.T) {
	svc := auth.NewTokenService("test-secret-key-12345")

	// Generate token expired 1 hour ago
	tokenStr, err := svc.GenerateToken("usr-123", "operator1", models.RoleOperator, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error validating expired token, got nil")
	}
}

func TestPasswordHashing(t *testing.T) {
	pass := "StrongBroadcastPass!99"
	hash, err := auth.HashPassword(pass)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !auth.CheckPassword(pass, hash) {
		t.Fatal("password verification failed for matching password")
	}

	if auth.CheckPassword("WrongPass", hash) {
		t.Fatal("password verification passed for invalid password")
	}
}
