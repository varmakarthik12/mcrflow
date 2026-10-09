package tmdb

import (
	"strings"
	"testing"
)

func TestCleanTitleAndExtractYear(t *testing.T) {
	cases := []struct {
		input         string
		expectedTitle string
	}{
		{"Jawan.2023.1080p.Hindi.Atmos.mkv", "Jawan"},
		{"Dunki.2023.1080p.mkv", "Dunki"},
		{"Avengers_Endgame_2019_4K_HDR.mp4", "Avengers Endgame"},
	}

	for _, c := range cases {
		title, _ := CleanTitleAndExtractYear(c.input)
		if !strings.EqualFold(title, c.expectedTitle) {
			t.Errorf("for input %s expected title %s, got %s", c.input, c.expectedTitle, title)
		}
	}
}

func TestSearchMovie(t *testing.T) {
	client := NewClient("test-key")

	results, err := client.SearchMovie("Jawan.2023.mkv", "hi")
	if err != nil {
		t.Fatalf("unexpected error searching movie: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected results for Jawan")
	}

	top := results[0]
	if top.Title != "Jawan" {
		t.Errorf("expected title Jawan, got %s", top.Title)
	}
	if top.LocalizedTitle != "जवान" {
		t.Errorf("expected Hindi localized title जवान, got %s", top.LocalizedTitle)
	}
	if top.ContentRating != "U/A 16+" {
		t.Errorf("expected CBFC rating U/A 16+, got %s", top.ContentRating)
	}
}
