package epg_test

import (
	"strings"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/epg"
	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestXMLTVGeneration(t *testing.T) {
	gen := epg.NewGenerator()

	ch := models.Channel{
		ID:       "ch-01",
		Name:     "DD National HD",
		LogoPath: "/logos/dd1.png",
	}

	start := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	items := []models.ScheduleItem{
		{
			ID:              "sch-01",
			ChannelID:       "ch-01",
			ProgramTitle:    "Morning News Bulletin",
			StartTime:       start,
			DurationSeconds: 1800,
			EndTime:         start.Add(1800 * time.Second),
			TmdbOverview:    "Daily morning headlines from across the nation.",
			TmdbPoster:      "https://example.com/poster.jpg",
		},
	}

	xmlData, err := gen.GenerateXMLTV(ch, items)
	if err != nil {
		t.Fatalf("failed to generate XMLTV: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, "<!DOCTYPE tv SYSTEM \"xmltv.dtd\">") {
		t.Errorf("missing XMLTV doctype")
	}
	if !strings.Contains(xmlStr, "<display-name>DD National HD</display-name>") {
		t.Errorf("missing channel display name")
	}
	if !strings.Contains(xmlStr, "Morning News Bulletin") {
		t.Errorf("missing program title")
	}
	if !strings.Contains(xmlStr, "https://example.com/poster.jpg") {
		t.Errorf("missing poster icon")
	}
}

func TestDVBEITGeneration(t *testing.T) {
	gen := epg.NewGenerator()

	start := time.Now().UTC().Add(-10 * time.Minute)
	items := []models.ScheduleItem{
		{
			ID:              "sch-01",
			ChannelID:       "ch-01",
			ProgramTitle:    "Current On-Air Feature",
			StartTime:       start,
			DurationSeconds: 3600,
			EndTime:         start.Add(3600 * time.Second),
			TmdbOverview:    "Live feature stream",
		},
	}

	eit := gen.GenerateDVBEIT(101, items)
	if eit.ServiceID != 101 {
		t.Errorf("expected service ID 101, got %d", eit.ServiceID)
	}
	if len(eit.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eit.Events))
	}
	if eit.Events[0].RunningStatus != 4 { // Currently running
		t.Errorf("expected running_status=4, got %d", eit.Events[0].RunningStatus)
	}
}
