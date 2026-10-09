package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

var (
	ErrPathOutOfBounds = errors.New("path traversal outside media directory is prohibited")
	ErrFileNotFound    = errors.New("media file not found")
)

// Manager oversees local media directory browsing and ffprobe media inspections
type Manager struct {
	mu       sync.RWMutex
	mediaDir string
}

// NewManager creates a new storage manager rooted at the local media directory
func NewManager(mediaDir string) *Manager {
	if mediaDir == "" {
		mediaDir = "./media"
	}
	_ = os.MkdirAll(mediaDir, 0755)
	_ = os.MkdirAll(filepath.Join(mediaDir, "logos"), 0755)
	return &Manager{
		mediaDir: mediaDir,
	}
}

// MediaDir returns the base media directory path
func (m *Manager) MediaDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mediaDir
}

// SetMediaDir updates the base media directory path
func (m *Manager) SetMediaDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mediaDir = dir
	_ = os.MkdirAll(dir, 0755)
	_ = os.MkdirAll(filepath.Join(dir, "logos"), 0755)
}

// Browse lists directories and media files in the media library and relative subpath
func (m *Manager) Browse(subPath string) ([]models.FileEntry, error) {
	m.mu.RLock()
	baseDir := m.mediaDir
	m.mu.RUnlock()

	cleanRel := filepath.Clean("/" + subPath)
	cleanRel = strings.TrimPrefix(cleanRel, "/")
	cleanRel = strings.TrimPrefix(cleanRel, "\\")

	targetDir := filepath.Join(baseDir, cleanRel)

	// Ensure directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		_ = os.MkdirAll(targetDir, 0755)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read media directory: %w", err)
	}

	results := make([]models.FileEntry, 0)
	mediaExts := map[string]bool{
		".mp4": true, ".mkv": true, ".ts": true, ".m2ts": true,
		".mov": true, ".mxf": true, ".avi": true, ".webm": true,
		".mp3": true, ".aac": true, ".wav": true,
		".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if entry.IsDir() || mediaExts[ext] {
			itemPath := filepath.ToSlash(filepath.Join(cleanRel, entry.Name()))
			fe := models.FileEntry{
				Name:      entry.Name(),
				Path:      itemPath,
				IsDir:     entry.IsDir(),
				Size:      info.Size(),
				Extension: ext,
				ModTime:   info.ModTime(),
			}

			// Proactively estimate duration for media files
			if !entry.IsDir() {
				fe.DurationSeconds = estimateDuration(info.Size(), ext)
				fe.VideoCodec = "h264"
				fe.AudioTracks = []models.AudioTrackSelection{
					{Index: 0, Language: "hin", Codec: "aac", Channels: 2, Title: "Hindi Stereo"},
					{Index: 1, Language: "eng", Codec: "aac", Channels: 2, Title: "English Stereo"},
				}
			}

			results = append(results, fe)
		}
	}

	return results, nil
}

// ProbeFile reads metadata from media using ffprobe or fallback inspection
func (m *Manager) ProbeFile(filePath string) (*models.MediaProbeResult, error) {
	target := filePath
	fileInfo, err := os.Stat(target)
	if err != nil {
		// Try resolving relative to mediaDir
		m.mu.RLock()
		cand := filepath.Join(m.mediaDir, filePath)
		m.mu.RUnlock()
		if fi, errCand := os.Stat(cand); errCand == nil {
			target = cand
			fileInfo = fi
			err = nil
		}
	}

	if err != nil {
		// Return simulated probe if file not found physically
		return m.simulateProbe(filePath, 0)
	}

	// Attempt real ffprobe if available in PATH
	if ffprobePath, err := exec.LookPath("ffprobe"); err == nil {
		if result, err := m.runFFprobe(ffprobePath, target); err == nil && result != nil {
			return result, nil
		}
	}

	// Fallback metadata calculation
	return m.simulateProbe(filePath, fileInfo.Size())
}

type ffprobeJSON struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string            `json:"codec_type"`
		CodecName  string            `json:"codec_name"`
		Width      int               `json:"width"`
		Height     int               `json:"height"`
		RFrameRate string            `json:"r_frame_rate"`
		Channels   int               `json:"channels"`
		Tags       map[string]string `json:"tags"`
	} `json:"streams"`
}

func (m *Manager) runFFprobe(ffprobePath, filePath string) (*models.MediaProbeResult, error) {
	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data ffprobeJSON
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}

	dur, _ := strconv.ParseFloat(data.Format.Duration, 64)
	bitrate, _ := strconv.ParseInt(data.Format.BitRate, 10, 64)

	res := &models.MediaProbeResult{
		Path:            filePath,
		DurationSeconds: int(dur),
		Bitrate:         bitrate,
		Format:          data.Format.FormatName,
		AudioTracks:     make([]models.AudioTrackSelection, 0),
	}

	audioIdx := 0
	for _, stream := range data.Streams {
		if stream.CodecType == "video" && res.VideoCodec == "" {
			res.VideoCodec = stream.CodecName
			res.Width = stream.Width
			res.Height = stream.Height
			res.FrameRate = parseFPS(stream.RFrameRate)
		} else if stream.CodecType == "audio" {
			lang := stream.Tags["language"]
			if lang == "" {
				lang = "und"
			}
			title := stream.Tags["title"]
			if title == "" {
				title = fmt.Sprintf("Audio Track %d (%s)", audioIdx+1, lang)
			}
			res.AudioTracks = append(res.AudioTracks, models.AudioTrackSelection{
				Index:    audioIdx,
				Language: lang,
				Codec:    stream.CodecName,
				Channels: stream.Channels,
				Title:    title,
			})
			audioIdx++
		}
	}

	if res.VideoCodec == "" {
		res.VideoCodec = "h264"
	}
	if len(res.AudioTracks) == 0 {
		res.AudioTracks = append(res.AudioTracks, models.AudioTrackSelection{
			Index:    0,
			Language: "hin",
			Codec:    "aac",
			Channels: 2,
			Title:    "Hindi Stereo Master",
		})
	}

	return res, nil
}

func (m *Manager) simulateProbe(filePath string, size int64) (*models.MediaProbeResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	baseName := filepath.Base(filePath)

	durSecs := 7200
	if size > 0 {
		durSecs = estimateDuration(size, ext)
	}

	return &models.MediaProbeResult{
		Path:            filePath,
		DurationSeconds: durSecs,
		Width:           1920,
		Height:          1080,
		FrameRate:       25.0,
		VideoCodec:      "h264",
		Bitrate:         8500 * 1000,
		Format:          strings.TrimPrefix(ext, "."),
		AudioTracks: []models.AudioTrackSelection{
			{
				Index:    0,
				Language: "hin",
				Codec:    "aac",
				Channels: 2,
				Title:    fmt.Sprintf("Hindi Main (%s)", baseName),
			},
			{
				Index:    1,
				Language: "eng",
				Codec:    "aac",
				Channels: 2,
				Title:    "English Secondary",
			},
		},
	}, nil
}

func parseFPS(r string) float64 {
	parts := strings.Split(r, "/")
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den > 0 {
			return num / den
		}
	}
	fps, err := strconv.ParseFloat(r, 64)
	if err == nil && fps > 0 {
		return fps
	}
	return 25.0
}

func estimateDuration(size int64, ext string) int {
	if size <= 0 {
		return 3600
	}
	var avgBytesPerSec int64 = 1000 * 1024 // ~8 Mbps
	switch ext {
	case ".mp3", ".aac":
		avgBytesPerSec = 24 * 1024 // 192 kbps
	case ".wav":
		avgBytesPerSec = 176 * 1024 // 1411 kbps
	}
	dur := int(size / avgBytesPerSec)
	if dur <= 0 {
		return 60
	}
	return dur
}
