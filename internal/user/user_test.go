package user

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestUserStoreLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "users.json")

	store := NewStore(dbFile)

	// Step 1: Initial state requires setup
	if !store.IsSetupRequired() {
		t.Fatalf("expected setup to be required on empty store")
	}

	// Step 2: Create initial admin
	admin, authResp, err := store.CreateInitialAdmin(SetupRequest{
		Username:    "chief_engineer",
		Password:    "SuperSecurePass123!",
		DisplayName: "Chief Engineer",
		Email:       "chief@mcrflow.tv",
	})
	if err != nil {
		t.Fatalf("failed to create initial admin: %v", err)
	}

	if admin.Role != models.RoleAdmin {
		t.Fatalf("expected initial user to have RoleAdmin, got: %s", admin.Role)
	}
	if authResp.Token == "" {
		t.Fatalf("expected non-empty auth token")
	}

	// Step 3: Setup is no longer required
	if store.IsSetupRequired() {
		t.Fatalf("expected setup to NOT be required after admin creation")
	}

	// Duplicate setup attempt must fail
	_, _, err = store.CreateInitialAdmin(SetupRequest{
		Username: "another_admin",
		Password: "pass",
	})
	if err != ErrSetupAlreadyCompleted {
		t.Fatalf("expected ErrSetupAlreadyCompleted, got: %v", err)
	}

	// Step 4: Authentication
	validAuth, err := store.Authenticate("chief_engineer", "SuperSecurePass123!")
	if err != nil {
		t.Fatalf("failed to authenticate valid credentials: %v", err)
	}
	if validAuth.User.ID != admin.ID {
		t.Fatalf("user ID mismatch")
	}

	_, err = store.Authenticate("chief_engineer", "wrongpass")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials on wrong pass, got: %v", err)
	}

	// Step 5: Validate token
	validatedUser, err := store.ValidateToken(validAuth.Token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if validatedUser.Username != "chief_engineer" {
		t.Fatalf("username mismatch: %s", validatedUser.Username)
	}

	// Step 6: Create Content Scheduler User
	schedulerUser, err := store.CreateUser(CreateUserRequest{
		Username:    "sched_rajesh",
		Password:    "SchedulerSecret456!",
		DisplayName: "Rajesh Kumar (Scheduler)",
		Email:       "rajesh@mcrflow.tv",
		Role:        models.RoleContentScheduler,
	})
	if err != nil {
		t.Fatalf("failed to create content scheduler user: %v", err)
	}
	if schedulerUser.Role != models.RoleContentScheduler {
		t.Fatalf("expected RoleContentScheduler, got %s", schedulerUser.Role)
	}

	// Step 7: Create Operator User
	operatorUser, err := store.CreateUser(CreateUserRequest{
		Username:    "ops_priya",
		Password:    "OperatorSecret789!",
		DisplayName: "Priya Sharma (MCR Ops)",
		Email:       "priya@mcrflow.tv",
		Role:        models.RoleOperator,
	})
	if err != nil {
		t.Fatalf("failed to create operator user: %v", err)
	}

	// Step 8: Listing users
	allUsers := store.ListUsers()
	if len(allUsers) != 3 {
		t.Fatalf("expected 3 users, got %d", len(allUsers))
	}

	// Step 9: Delete rules (cannot delete self, cannot delete last admin)
	err = store.DeleteUser(admin.ID, admin.ID)
	if err != ErrCannotDeleteSelf {
		t.Fatalf("expected ErrCannotDeleteSelf, got: %v", err)
	}

	err = store.DeleteUser(admin.ID, schedulerUser.ID)
	if err != ErrCannotDeleteLastAdmin {
		t.Fatalf("expected ErrCannotDeleteLastAdmin, got: %v", err)
	}

	// Operator can be deleted
	err = store.DeleteUser(operatorUser.ID, admin.ID)
	if err != nil {
		t.Fatalf("failed to delete operator user: %v", err)
	}

	// Step 10: Disk reload persistence test
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("users db file not found: %v", err)
	}

	reloadedStore := NewStore(dbFile)
	if reloadedStore.Count() != 2 {
		t.Fatalf("expected 2 users on reload, got %d", reloadedStore.Count())
	}
	reloadedAdmin, err := reloadedStore.Authenticate("chief_engineer", "SuperSecurePass123!")
	if err != nil {
		t.Fatalf("failed to authenticate on reloaded store: %v", err)
	}
	if reloadedAdmin.User.Role != models.RoleAdmin {
		t.Fatalf("expected RoleAdmin on reloaded user")
	}
}
