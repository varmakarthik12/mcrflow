package hls

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

func TestHlsSlidingWindowAndRotation(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir, nil)
	stream := mgr.GetOrCreateStream("ch-test-01")

	// Step 1: Exactly 10 segments initially
	segs := stream.GetActiveSegments()
	if len(segs) != MaxActiveSegments {
		t.Fatalf("expected %d active segments initially, got %d", MaxActiveSegments, len(segs))
	}
	initialFirstSeq := segs[0].SequenceNum

	// Verify master.m3u8 content
	m3u8Path := filepath.Join(stream.OutputDir, "master.m3u8")
	data, err := os.ReadFile(m3u8Path)
	if err != nil {
		t.Fatalf("failed to read master.m3u8: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "#EXTM3U") {
		t.Fatalf("missing #EXTM3U header")
	}
	if !strings.Contains(content, "#EXT-X-MEDIA-SEQUENCE:") {
		t.Fatalf("missing media sequence header")
	}

	// Step 2: Append 5 new segments and ensure rotation
	for i := 0; i < 5; i++ {
		stream.AppendSegment(DefaultSegmentSec)
	}

	rotatedSegs := stream.GetActiveSegments()
	if len(rotatedSegs) != MaxActiveSegments {
		t.Fatalf("expected strictly %d active segments after rotation, got %d", MaxActiveSegments, len(rotatedSegs))
	}

	// First sequence must have advanced by 5
	expectedFirstSeq := initialFirstSeq + 5
	if rotatedSegs[0].SequenceNum != expectedFirstSeq {
		t.Fatalf("expected first sequence to advance to %d, got %d", expectedFirstSeq, rotatedSegs[0].SequenceNum)
	}

	// Verify old segment files were deleted from disk
	oldFilename := filepath.Join(stream.OutputDir, segs[0].Filename)
	if _, err := os.Stat(oldFilename); !os.IsNotExist(err) {
		t.Fatalf("expected old segment file %s to be deleted from disk", oldFilename)
	}
}

func TestHlsHttpServingAndTokenSecurity(t *testing.T) {
	tmpDir := t.TempDir()

	tokens := map[string]string{
		"ch-secure": "tok_premium_9988",
		"ch-public": "",
	}

	mgr := NewManager(tmpDir, func(channelID string) string {
		return tokens[channelID]
	})

	// Initialize streams
	mgr.GetOrCreateStream("ch-secure")
	mgr.GetOrCreateStream("ch-public")

	// Test 1: Public channel without token -> 200 OK
	reqPub := httptest.NewRequest("GET", "/hls/ch-public/master.m3u8", nil)
	recPub := httptest.NewRecorder()
	mgr.ServeHTTP(recPub, reqPub)
	if recPub.Code != http.StatusOK {
		t.Fatalf("expected public stream to return 200 OK, got %d", recPub.Code)
	}

	// Test 2: Secure channel without token -> 401 Unauthorized
	reqSecNoTok := httptest.NewRequest("GET", "/hls/ch-secure/master.m3u8", nil)
	recSecNoTok := httptest.NewRecorder()
	mgr.ServeHTTP(recSecNoTok, reqSecNoTok)
	if recSecNoTok.Code != http.StatusUnauthorized {
		t.Fatalf("expected secure stream without token to return 401, got %d", recSecNoTok.Code)
	}

	// Test 3: Secure channel with wrong token -> 401 Unauthorized
	reqSecWrongTok := httptest.NewRequest("GET", "/hls/ch-secure/master.m3u8?token=wrong_secret", nil)
	recSecWrongTok := httptest.NewRecorder()
	mgr.ServeHTTP(recSecWrongTok, reqSecWrongTok)
	if recSecWrongTok.Code != http.StatusUnauthorized {
		t.Fatalf("expected secure stream with wrong token to return 401, got %d", recSecWrongTok.Code)
	}

	// Test 4: Secure channel with valid token -> 200 OK
	reqSecValid := httptest.NewRequest("GET", "/hls/ch-secure/master.m3u8?token=tok_premium_9988", nil)
	recSecValid := httptest.NewRecorder()
	mgr.ServeHTTP(recSecValid, reqSecValid)
	if recSecValid.Code != http.StatusOK {
		t.Fatalf("expected secure stream with valid token to return 200 OK, got %d", recSecValid.Code)
	}
	if !strings.Contains(recSecValid.Body.String(), "#EXTM3U") {
		t.Fatalf("expected master.m3u8 body to contain #EXTM3U")
	}

	// Test 5: Direct URL resolution
	urlPub := ResolveHlsStreamURL("http://localhost:8080", "ch-public", "")
	if urlPub != "http://localhost:8080/hls/ch-public/master.m3u8" {
		t.Fatalf("unexpected resolved public url: %s", urlPub)
	}

	urlSec := ResolveHlsStreamURL("http://localhost:8080", "ch-secure", "tok_premium_9988")
	if urlSec != "http://localhost:8080/hls/ch-secure/master.m3u8?token=tok_premium_9988" {
		t.Fatalf("unexpected resolved secure url: %s", urlSec)
	}

	ch := &models.Channel{
		ID:          "ch-01",
		HlsWebToken: "secret123",
	}
	UpdateChannelHlsDestination(ch, "http://mcr.station.tv:8080")
	if ch.HlsStreamURL != "http://mcr.station.tv:8080/hls/ch-01/master.m3u8?token=secret123" {
		t.Fatalf("unexpected ch.HlsStreamURL: %s", ch.HlsStreamURL)
	}
	if len(ch.Destinations) != 1 || ch.Destinations[0].Protocol != models.ProtocolHLS {
		t.Fatalf("expected HLS destination added to channel")
	}
}
