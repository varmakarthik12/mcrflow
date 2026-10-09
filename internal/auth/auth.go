package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	TokenPrefix = "agt_sec_"
	TokenEntropyBytes = 32
)

// AgentCredentials represents the persistent token stored on the edge agent.
type AgentCredentials struct {
	AgentID      string    `json:"agent_id"`
	PairingToken string    `json:"pairing_token"`
	CreatedAt    time.Time `json:"created_at"`
	Version      string    `json:"version"`
}

// Manager handles token creation, persistence, and validation.
type Manager struct {
	mu          sync.RWMutex
	storagePath string
	credentials *AgentCredentials
}

// NewManager creates a new auth manager targeting the specified credentials file path.
func NewManager(storagePath string) *Manager {
	return &Manager{
		storagePath: storagePath,
	}
}

// GenerateSecureToken generates a cryptographically random 256-bit token with "agt_sec_" prefix.
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, TokenEntropyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read secure random bytes: %w", err)
	}
	return TokenPrefix + hex.EncodeToString(bytes), nil
}

// InitializeOrLoadAgentToken ensures a persistent token exists on disk.
// If it exists, it is loaded without modification.
// If it does not exist, a new token is generated and persisted.
func (m *Manager) InitializeOrLoadAgentToken(agentID string) (*AgentCredentials, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded in memory
	if m.credentials != nil {
		return m.credentials, false, nil
	}

	// Check if storage file exists on disk
	if _, err := os.Stat(m.storagePath); err == nil {
		data, err := os.ReadFile(m.storagePath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read existing agent auth file: %w", err)
		}
		var creds AgentCredentials
		if err := json.Unmarshal(data, &creds); err != nil {
			return nil, false, fmt.Errorf("failed to parse agent auth file: %w", err)
		}
		m.credentials = &creds
		return &creds, false, nil // false = not newly generated, loaded from disk
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, false, fmt.Errorf("error checking agent auth file: %w", err)
	}

	// Generate new token
	token, err := GenerateSecureToken()
	if err != nil {
		return nil, false, err
	}

	if agentID == "" {
		agentID = "mcrflow-agent-" + hex.EncodeToString([]byte(time.Now().Format("150405")))
	}

	creds := &AgentCredentials{
		AgentID:      agentID,
		PairingToken: token,
		CreatedAt:    time.Now().UTC(),
		Version:      "1.0.0",
	}

	// Ensure parent directory exists
	if dir := filepath.Dir(m.storagePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, false, fmt.Errorf("failed to create auth directory %s: %w", dir, err)
		}
	}

	// Write file with strict 0600 permissions
	bytes, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return nil, false, err
	}

	if err := os.WriteFile(m.storagePath, bytes, 0600); err != nil {
		return nil, false, fmt.Errorf("failed to write agent auth file: %w", err)
	}

	m.credentials = creds
	return creds, true, nil // true = fresh token generated
}

// ResetToken regenerates and overwrites the existing token on disk.
func (m *Manager) ResetToken() (*AgentCredentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentID := "mcrflow-agent"
	if m.credentials != nil && m.credentials.AgentID != "" {
		agentID = m.credentials.AgentID
	}

	token, err := GenerateSecureToken()
	if err != nil {
		return nil, err
	}

	creds := &AgentCredentials{
		AgentID:      agentID,
		PairingToken: token,
		CreatedAt:    time.Now().UTC(),
		Version:      "1.0.0",
	}

	bytes, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(m.storagePath, bytes, 0600); err != nil {
		return nil, fmt.Errorf("failed to overwrite agent auth file: %w", err)
	}

	m.credentials = creds
	return creds, nil
}

// ValidateToken performs constant-time comparison against expected token.
func ValidateToken(providedToken, expectedToken string) bool {
	if len(providedToken) == 0 || len(expectedToken) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
}

// GetCredentials returns current in-memory credentials.
func (m *Manager) GetCredentials() *AgentCredentials {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.credentials
}
