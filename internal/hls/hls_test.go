package hls_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varmakarthik12/mcrflow/internal/database"
	"github.com/varmakarthik12/mcrflow/internal/hls"
)

func TestHlsSlidingWindowAndToken(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "hls_test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	mgr := hls.NewManager(repo)

	// Channel ch-01 has seeded token: live_sec_dd1_tok_2026
	// 1. Request without token should fail with ErrUnauthorized
	_, err = mgr.GenerateSlidingPlaylist("ch-01", "")
	if err != hls.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for missing token, got: %v", err)
	}

	// 2. Request with invalid token should fail
	_, err = mgr.GenerateSlidingPlaylist("ch-01", "wrong_token")
	if err != hls.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for wrong token, got: %v", err)
	}

	// 3. Request with valid token should succeed
	validToken := "live_sec_dd1_tok_2026"
	playlist, err := mgr.GenerateSlidingPlaylist("ch-01", validToken)
	if err != nil {
		t.Fatalf("failed to generate playlist with valid token: %v", err)
	}

	// Verify sliding window contains 10 segments
	count := strings.Count(playlist, "#EXTINF:2.000,")
	if count != 10 {
		t.Fatalf("expected 10 active segments, got %d", count)
	}
	if !strings.Contains(playlist, "?token="+validToken) {
		t.Errorf("token query parameter was not preserved in segments")
	}

	// 4. Test master playlist
	master, err := mgr.GenerateMasterManifest("ch-01", validToken)
	if err != nil {
		t.Fatalf("failed to generate master playlist: %v", err)
	}
	if !strings.Contains(master, "playlist.m3u8?token="+validToken) {
		t.Errorf("master playlist missing child playlist link with token")
	}

	// 5. Test segment retrieval and MPEG-TS validity
	data, err := mgr.GetSegmentData("ch-01", 100, validToken)
	if err != nil {
		t.Fatalf("failed to get segment data: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty segment data")
	}
	if data[0] != 0x47 {
		t.Fatalf("expected MPEG-TS sync byte 0x47, got 0x%X", data[0])
	}

	// If ffprobe exists, verify it decodes the segment flawlessly
	if ffprobePath, err := exec.LookPath("ffprobe"); err == nil {
		tmpTS := filepath.Join(tempDir, "test_verify.ts")
		_ = os.WriteFile(tmpTS, data, 0644)
		cmd := exec.Command(ffprobePath, "-v", "error", "-show_format", tmpTS)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("ffprobe failed to decode segment: %v, out: %s", err, string(out))
		}
	}
}
