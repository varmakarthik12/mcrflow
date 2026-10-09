package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	if !strings.HasPrefix(token, TokenPrefix) {
		t.Errorf("expected token to start with %s, got: %s", TokenPrefix, token)
	}

	// 8 chars prefix + 64 hex chars = 72 chars total
	if len(token) != len(TokenPrefix)+64 {
		t.Errorf("expected token length %d, got %d", len(TokenPrefix)+64, len(token))
	}

	token2, _ := GenerateSecureToken()
	if token == token2 {
		t.Errorf("two consecutive tokens should not be identical")
	}
}

func TestValidateToken(t *testing.T) {
	validToken, _ := GenerateSecureToken()

	if !ValidateToken(validToken, validToken) {
		t.Errorf("identical tokens should validate")
	}

	if ValidateToken(validToken, validToken+"x") {
		t.Errorf("mismatched tokens should fail validation")
	}

	if ValidateToken("", validToken) || ValidateToken(validToken, "") {
		t.Errorf("empty tokens should fail validation")
	}
}

func TestTokenPersistenceAndRestart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcrflow-auth-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	authFilePath := filepath.Join(tempDir, "agent_auth.json")

	// Phase 1: First boot (Fresh generation)
	manager1 := NewManager(authFilePath)
	creds1, isFresh1, err := manager1.InitializeOrLoadAgentToken("test-agent-delhi")
	if err != nil {
		t.Fatalf("first initialization failed: %v", err)
	}
	if !isFresh1 {
		t.Errorf("expected first boot to report isFresh=true")
	}
	if creds1.AgentID != "test-agent-delhi" {
		t.Errorf("expected agent ID test-agent-delhi, got %s", creds1.AgentID)
	}
	originalToken := creds1.PairingToken

	// Phase 2: Restart simulation (Load from disk)
	manager2 := NewManager(authFilePath)
	creds2, isFresh2, err := manager2.InitializeOrLoadAgentToken("another-name-ignored")
	if err != nil {
		t.Fatalf("second initialization failed: %v", err)
	}
	if isFresh2 {
		t.Errorf("expected restart boot to report isFresh=false")
	}
	if creds2.PairingToken != originalToken {
		t.Errorf("token drift detected! original %s, restored %s", originalToken, creds2.PairingToken)
	}
	if creds2.AgentID != "test-agent-delhi" {
		t.Errorf("expected persisted agent ID to be preserved, got %s", creds2.AgentID)
	}

	// Phase 3: Explicit Reset
	creds3, err := manager2.ResetToken()
	if err != nil {
		t.Fatalf("token reset failed: %v", err)
	}
	if creds3.PairingToken == originalToken {
		t.Errorf("reset token should differ from original token")
	}

	// Phase 4: Verify reset token persisted
	manager3 := NewManager(authFilePath)
	creds4, _, _ := manager3.InitializeOrLoadAgentToken("")
	if creds4.PairingToken != creds3.PairingToken {
		t.Errorf("new reset token should persist across next boot")
	}
}
