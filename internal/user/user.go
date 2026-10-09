package user

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/auth"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrUsernameRequired   = errors.New("username is required")
)

// Store defines persistence operations for users
type Store interface {
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	ListUsers() ([]models.User, error)
	CreateUser(u *models.User, plainPassword string) error
	UpdateUser(u *models.User, newPassword *string) error
	DeleteUser(id string) error
}

// Service manages user operations and authentication
type Service struct {
	store        Store
	tokenService *auth.TokenService
}

// NewService creates a new user service
func NewService(store Store, tokenService *auth.TokenService) *Service {
	return &Service{
		store:        store,
		tokenService: tokenService,
	}
}

// Authenticate verifies credentials and returns the user with a signed JWT
func (s *Service) Authenticate(username, password string) (*models.User, string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, "", ErrInvalidCredentials
	}

	u, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if !auth.CheckPassword(password, u.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.tokenService.GenerateToken(u.ID, u.Username, u.Role, 24*time.Hour)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return u, token, nil
}

// CreateUser validates and stores a new user
func (s *Service) CreateUser(u *models.User, password string) error {
	u.Username = strings.TrimSpace(u.Username)
	if u.Username == "" {
		return ErrUsernameRequired
	}
	if u.Role != models.RoleAdmin && u.Role != models.RoleOperator && u.Role != models.RoleContentScheduler {
		return ErrInvalidRole
	}
	return s.store.CreateUser(u, password)
}

// GetUserByID returns user by ID
func (s *Service) GetUserByID(id string) (*models.User, error) {
	return s.store.GetUserByID(id)
}

// ListUsers returns all users
func (s *Service) ListUsers() ([]models.User, error) {
	return s.store.ListUsers()
}

// UpdateUser updates user details or password
func (s *Service) UpdateUser(u *models.User, newPassword *string) error {
	if u.Role != "" && u.Role != models.RoleAdmin && u.Role != models.RoleOperator && u.Role != models.RoleContentScheduler {
		return ErrInvalidRole
	}
	return s.store.UpdateUser(u, newPassword)
}

// DeleteUser deletes user
func (s *Service) DeleteUser(id string) error {
	return s.store.DeleteUser(id)
}
