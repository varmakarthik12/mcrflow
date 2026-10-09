package adtemplate_test

import (
	"path/filepath"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/adtemplate"
	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestAdTemplateService(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "adt_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	svc := adtemplate.NewService(repo)

	// List seeded templates
	list, err := svc.ListTemplates()
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}
	if len(list) < 3 {
		t.Fatalf("expected at least 3 seeded templates, got %d", len(list))
	}

	// Validation check
	err = svc.CreateTemplate(&models.AdTemplate{Name: ""})
	if err != adtemplate.ErrNameRequired {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}

	// Create new template
	tmpl := &models.AdTemplate{
		Name:         "Festival Special Overlay",
		TemplateType: "overlay",
		IsActive:     true,
	}
	if err := svc.CreateTemplate(tmpl); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	fetched, err := svc.GetTemplateByID(tmpl.ID)
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}
	if fetched.Name != tmpl.Name {
		t.Errorf("name mismatch: %s vs %s", fetched.Name, tmpl.Name)
	}
}
