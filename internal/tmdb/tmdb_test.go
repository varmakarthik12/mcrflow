package tmdb_test

import (
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/tmdb"
)

func TestTMDBSearchAndCache(t *testing.T) {
	client := tmdb.NewClient("")

	// Search for RRR in fallback catalog
	results, err := client.Search("RRR")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected results for 'RRR', got 0")
	}
	if results[0].Title != "RRR" {
		t.Errorf("expected RRR, got %s", results[0].Title)
	}

	// Verify caching
	cachedResults, err := client.Search("RRR")
	if err != nil || len(cachedResults) == 0 {
		t.Fatalf("cached search failed")
	}

	// Verify poster caching
	fakePoster := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // JPEG magic bytes
	client.CachePoster("poster-url-1", fakePoster)
	got, ok := client.GetCachedPoster("poster-url-1")
	if !ok || len(got) != len(fakePoster) {
		t.Fatalf("poster cache failed")
	}
}
