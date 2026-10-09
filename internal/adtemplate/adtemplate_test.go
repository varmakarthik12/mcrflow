package adtemplate

import (
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestAdTemplateStoreAndPrecedence(t *testing.T) {
	store := NewStore()

	// List templates
	tmpls := store.ListTemplates()
	if len(tmpls) < 2 {
		t.Fatalf("expected at least 2 default templates, got %d", len(tmpls))
	}

	// 1. Precedence: Content override should take priority over channel global
	effective, origin := ResolveEffectiveTemplate(store, "ad-tmpl-diwali", "ad-tmpl-generic")
	if effective == nil || effective.ID != "ad-tmpl-diwali" || origin != "CONTENT_OVERRIDE" {
		t.Errorf("expected content override to take priority, got %s / %s", effective.ID, origin)
	}

	// 2. Precedence: Fallback to channel global when content is empty
	effective, origin = ResolveEffectiveTemplate(store, "", "ad-tmpl-generic")
	if effective == nil || effective.ID != "ad-tmpl-generic" || origin != "CHANNEL_GLOBAL_DEFAULT" {
		t.Errorf("expected channel global default when content is empty, got %s / %s", effective.ID, origin)
	}

	// 3. Precedence: None when both empty
	effective, origin = ResolveEffectiveTemplate(store, "", "")
	if effective != nil || origin != "NONE" {
		t.Errorf("expected nil when no templates configured, got %v / %s", effective, origin)
	}

	// 4. Create custom template
	custom := &models.AdTemplate{
		Name:        "IPL Cricket Sponsor Bug",
		Description: "Live scorecard teaser",
		BannerOverlays: []models.BannerOverlayElement{
			{
				ID:                "elem-cricket-score",
				ElementType:       "LOWER_THIRD",
				TextContent:       "IPL 2026 LIVE: CSK 184/4 (18.2)",
				EntranceAnimation: models.TransitionBounce,
			},
		},
	}

	created, err := store.CreateTemplate(custom)
	if err != nil {
		t.Fatalf("failed to create custom template: %v", err)
	}

	if created.ID == "" {
		t.Errorf("expected generated ID")
	}

	// Delete custom template
	if err := store.DeleteTemplate(created.ID); err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}
	if _, err := store.GetTemplate(created.ID); err != ErrTemplateNotFound {
		t.Errorf("expected ErrTemplateNotFound after delete")
	}
}
