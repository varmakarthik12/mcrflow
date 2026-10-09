package adtemplate

import (
	"errors"
	"strings"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrNameRequired = errors.New("ad template name is required")
)

// Store defines persistence operations for ad templates
type Store interface {
	CreateAdTemplate(tmpl *models.AdTemplate) error
	GetAdTemplateByID(id string) (*models.AdTemplate, error)
	ListAdTemplates() ([]models.AdTemplate, error)
	UpdateAdTemplate(tmpl *models.AdTemplate) error
	DeleteAdTemplate(id string) error
}

// Service manages graphics & ad templates
type Service struct {
	store Store
}

// NewService creates a new ad template service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateTemplate validates and stores a new ad template
func (s *Service) CreateTemplate(tmpl *models.AdTemplate) error {
	tmpl.Name = strings.TrimSpace(tmpl.Name)
	if tmpl.Name == "" {
		return ErrNameRequired
	}
	if tmpl.TemplateType == "" {
		tmpl.TemplateType = "composite"
	}
	return s.store.CreateAdTemplate(tmpl)
}

// GetTemplateByID retrieves template by ID
func (s *Service) GetTemplateByID(id string) (*models.AdTemplate, error) {
	return s.store.GetAdTemplateByID(id)
}

// ListTemplates retrieves all templates
func (s *Service) ListTemplates() ([]models.AdTemplate, error) {
	return s.store.ListAdTemplates()
}

// UpdateTemplate updates a template
func (s *Service) UpdateTemplate(tmpl *models.AdTemplate) error {
	tmpl.Name = strings.TrimSpace(tmpl.Name)
	if tmpl.Name == "" {
		return ErrNameRequired
	}
	return s.store.UpdateAdTemplate(tmpl)
}

// DeleteTemplate removes a template
func (s *Service) DeleteTemplate(id string) error {
	return s.store.DeleteAdTemplate(id)
}
