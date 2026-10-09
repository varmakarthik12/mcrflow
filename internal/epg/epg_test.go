package epg

import (
	"strings"
	"testing"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestGenerateXMLTV(t *testing.T) {
	channel := &models.Channel{
		ID:      "ch-01",
		Name:    "Star Gold HD",
		LogoURL: "https://example.com/logo.png",
	}

	start := time.Date(2026, 10, 9, 16, 15, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	items := []*models.ScheduleItem{
		{
			ID:              "sched-01",
			ChannelID:       "ch-01",
			StartTime:       start,
			EndTime:         end,
			DurationSeconds: 7200,
			TmdbMetadata: &models.TmdbMetadata{
				Title:          "Jawan",
				LocalizedTitle: "जवान",
				Overview:       "Action drama",
				ContentRating:  "U/A 16+",
				Genres:         []string{"Action", "Thriller"},
			},
		},
	}

	xmlData, err := GenerateXMLTV(channel, items)
	if err != nil {
		t.Fatalf("failed to generate XMLTV: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, "<tv generator-info-name=\"MCRFlow Master Control Playout\">") {
		t.Errorf("expected generator-info-name MCRFlow in XMLTV")
	}

	if !strings.Contains(xmlStr, "Star Gold HD") {
		t.Errorf("expected channel name in XMLTV")
	}

	if !strings.Contains(xmlStr, "जवान") || !strings.Contains(xmlStr, "Jawan") {
		t.Errorf("expected multilingual titles in XMLTV")
	}

	if !strings.Contains(xmlStr, "U/A 16+") {
		t.Errorf("expected CBFC rating in XMLTV")
	}
}
