package user

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"mcrflow/internal/models"
)

var (
	ErrSetupAlreadyCompleted = errors.New("setup has already been completed")
	ErrUserNotFound          = errors.New("user not found")
	ErrUsernameTaken         = errors.New("username is already taken")
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrInvalidToken          = errors.New("invalid or expired authentication token")
	ErrCannotDeleteSelf      = errors.New("cannot delete current logged-in user")
	ErrCannotDeleteLastAdmin = errors.New("cannot delete the last administrator")
	ErrInvalidRole           = errors.New("invalid user role")
)

type SetupRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username    string          `json:"username"`
	Password    string          `json:"password"`
	DisplayName string          `json:"display_name"`
	Email       string          `json:"email"`
	Role        models.UserRole `json:"role"`
}

type UpdateUserRequest struct {
	DisplayName *string          `json:"display_name,omitempty"`
	Email       *string          `json:"email,omitempty"`
	Role        *models.UserRole `json:"role,omitempty"`
	Password    *string          `json:"password,omitempty"`
}

type Session struct {
	Token     string
	User      *models.User
	ExpiresAt time.Time
}

type Store struct {
	mu          sync.RWMutex
	users       map[string]*models.User // id -> User
	usernameIdx map[string]string       // lowercase username -> id
	sessions    map[string]*Session     // token -> Session
	storagePath string
}

func NewStore(storagePath string) *Store {
	s := &Store{
		users:       make(map[string]*models.User),
		usernameIdx: make(map[string]string),
		sessions:    make(map[string]*Session),
		storagePath: storagePath,
	}

	if storagePath != "" {
		_ = s.loadFromDisk()
	}

	return s
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

func (s *Store) IsSetupRequired() bool {
	return s.Count() == 0
}

func (s *Store) CreateInitialAdmin(req SetupRequest) (*models.User, *models.UserAuthResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.users) > 0 {
		return nil, nil, ErrSetupAlreadyCompleted
	}

	if req.Username == "" || req.Password == "" {
		return nil, nil, errors.New("username and password are required")
	}

	salt := generateSalt()
	hash := hashPassword(req.Password, salt)

	admin := &models.User{
		ID:           "usr-" + uuid.New().String()[:8],
		Username:     req.Username,
		PasswordHash: salt + ":" + hash,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         models.RoleAdmin,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if admin.DisplayName == "" {
		admin.DisplayName = admin.Username
	}

	s.users[admin.ID] = admin
	s.usernameIdx[admin.Username] = admin.ID

	token := s.createSessionLocked(admin)
	_ = s.saveToDiskLocked()

	return admin, &models.UserAuthResponse{
		Token:     token,
		User:      admin,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func (s *Store) Authenticate(username, password string) (*models.UserAuthResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, exists := s.usernameIdx[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	user, ok := s.users[id]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	if !verifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token := s.createSessionLocked(user)

	return &models.UserAuthResponse{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}, nil
}

func (s *Store) ValidateToken(token string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[token]
	if !exists {
		return nil, ErrInvalidToken
	}

	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	return sess.User, nil
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Store) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		// Return copy without password hash
		uCopy := *u
		uCopy.PasswordHash = ""
		out = append(out, &uCopy)
	}
	return out
}

func (s *Store) GetUser(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	uCopy := *u
	uCopy.PasswordHash = ""
	return &uCopy, nil
}

func (s *Store) CreateUser(req CreateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Username == "" || req.Password == "" {
		return nil, errors.New("username and password are required")
	}

	if _, exists := s.usernameIdx[req.Username]; exists {
		return nil, ErrUsernameTaken
	}

	if !isValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	salt := generateSalt()
	hash := hashPassword(req.Password, salt)

	u := &models.User{
		ID:           "usr-" + uuid.New().String()[:8],
		Username:     req.Username,
		PasswordHash: salt + ":" + hash,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         req.Role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if u.DisplayName == "" {
		u.DisplayName = u.Username
	}

	s.users[u.ID] = u
	s.usernameIdx[u.Username] = u.ID
	_ = s.saveToDiskLocked()

	uCopy := *u
	uCopy.PasswordHash = ""
	return &uCopy, nil
}

func (s *Store) UpdateUser(id string, req UpdateUserRequest) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	if req.Role != nil {
		if !isValidRole(*req.Role) {
			return nil, ErrInvalidRole
		}
		// Check if changing role away from admin leaves zero admins
		if u.Role == models.RoleAdmin && *req.Role != models.RoleAdmin {
			if s.countAdminsLocked() <= 1 {
				return nil, ErrCannotDeleteLastAdmin
			}
		}
		u.Role = *req.Role
	}

	if req.DisplayName != nil {
		u.DisplayName = *req.DisplayName
	}

	if req.Email != nil {
		u.Email = *req.Email
	}

	if req.Password != nil && *req.Password != "" {
		salt := generateSalt()
		u.PasswordHash = salt + ":" + hashPassword(*req.Password, salt)
	}

	u.UpdatedAt = time.Now().UTC()
	_ = s.saveToDiskLocked()

	uCopy := *u
	uCopy.PasswordHash = ""
	return &uCopy, nil
}

func (s *Store) DeleteUser(id string, requestingUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == requestingUserID {
		return ErrCannotDeleteSelf
	}

	u, exists := s.users[id]
	if !exists {
		return ErrUserNotFound
	}

	if u.Role == models.RoleAdmin && s.countAdminsLocked() <= 1 {
		return ErrCannotDeleteLastAdmin
	}

	delete(s.usernameIdx, u.Username)
	delete(s.users, id)

	// Clean up any active sessions for this user
	for tok, sess := range s.sessions {
		if sess.User.ID == id {
			delete(s.sessions, tok)
		}
	}

	_ = s.saveToDiskLocked()
	return nil
}

func (s *Store) createSessionLocked(user *models.User) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := "usr_tok_" + hex.EncodeToString(b)

	s.sessions[token] = &Session{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	return token
}

func (s *Store) countAdminsLocked() int {
	count := 0
	for _, u := range s.users {
		if u.Role == models.RoleAdmin {
			count++
		}
	}
	return count
}

type storedUser struct {
	ID           string          `json:"id"`
	Username     string          `json:"username"`
	PasswordHash string          `json:"password_hash"`
	DisplayName  string          `json:"display_name"`
	Email        string          `json:"email"`
	Role         models.UserRole `json:"role"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (s *Store) saveToDiskLocked() error {
	if s.storagePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0755); err != nil {
		return err
	}
	stored := make(map[string]storedUser, len(s.users))
	for id, u := range s.users {
		stored[id] = storedUser{
			ID:           u.ID,
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			DisplayName:  u.DisplayName,
			Email:        u.Email,
			Role:         u.Role,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		}
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, data, 0600)
}

func (s *Store) loadFromDisk() error {
	if s.storagePath == "" {
		return nil
	}
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		return err
	}
	var loaded map[string]storedUser
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]*models.User, len(loaded))
	s.usernameIdx = make(map[string]string, len(loaded))
	for id, su := range loaded {
		u := &models.User{
			ID:           su.ID,
			Username:     su.Username,
			PasswordHash: su.PasswordHash,
			DisplayName:  su.DisplayName,
			Email:        su.Email,
			Role:         su.Role,
			CreatedAt:    su.CreatedAt,
			UpdatedAt:    su.UpdatedAt,
		}
		s.users[id] = u
		s.usernameIdx[u.Username] = id
	}
	return nil
}

func isValidRole(r models.UserRole) bool {
	return r == models.RoleAdmin || r == models.RoleOperator || r == models.RoleContentScheduler
}

func generateSalt() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + password))
	return hex.EncodeToString(h.Sum(nil))
}

func verifyPassword(password, storedHash string) bool {
	var salt, expectedHash string
	n, err := fmt.Sscanf(storedHash, "%32s:%s", &salt, &expectedHash)
	if err != nil || n != 2 {
		return false
	}
	computed := hashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) == 1
}
