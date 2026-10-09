package user_test

import (
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
	"github.com/varmakarthik12/mcrflow/internal/user"
)

func setupUserService(t *testing.T) (*database.DB, *user.Service) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "user_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	repo := database.NewRepository(db)
	tokenSvc := auth.NewTokenService("test-user-secret")
	svc := user.NewService(repo, tokenSvc)
	return db, svc
}

func TestUserAuthenticationSuccess(t *testing.T) {
	db, svc := setupUserService(t)
	defer db.Close()

	u, token, err := svc.Authenticate("admin", "admin123")
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}
	if u.Username != "admin" || token == "" {
		t.Fatalf("invalid authentication response: u=%+v, token=%s", u, token)
	}
}

func TestUserAuthenticationFailure(t *testing.T) {
	db, svc := setupUserService(t)
	defer db.Close()

	_, _, err := svc.Authenticate("admin", "wrongpass")
	if err != user.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}

	_, _, err = svc.Authenticate("nonexistent", "whatever")
	if err != user.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestUserRoleValidation(t *testing.T) {
	db, svc := setupUserService(t)
	defer db.Close()

	err := svc.CreateUser(&models.User{
		Username: "badrole",
		Role:     "superuser", // not valid
	}, "secret123")

	if err != user.ErrInvalidRole {
		t.Fatalf("expected ErrInvalidRole, got: %v", err)
	}

	err = svc.CreateUser(&models.User{
		Username: "scheduler1",
		Role:     models.RoleContentScheduler,
	}, "secret123")
	if err != nil {
		t.Fatalf("failed to create valid scheduler: %v", err)
	}
}
