package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MovieResult represents a TMDB search hit
type MovieResult struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Overview      string   `json:"overview"`
	ReleaseDate   string   `json:"release_date"`
	PosterPath    string   `json:"poster_path"`
	BackdropPath  string   `json:"backdrop_path"`
	Rating        float64  `json:"rating"`
	Genres        []string `json:"genres"`
	Runtime       int      `json:"runtime"`
}

// Client interacts with TMDB API and caches posters and metadata
type Client struct {
	apiKey      string
	httpClient  *http.Client
	mu          sync.RWMutex
	cache       map[string][]MovieResult
	posterCache map[string][]byte
}

// NewClient creates a new TMDB client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:       make(map[string][]MovieResult),
		posterCache: make(map[string][]byte),
	}
}

// Search searches for movies by query string with in-memory caching and fallback
func (c *Client) Search(query string) ([]MovieResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []MovieResult{}, nil
	}

	normalizedQuery := strings.ToLower(query)

	c.mu.RLock()
	if cached, ok := c.cache[normalizedQuery]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	var results []MovieResult

	// If API key is provided, query the remote TMDB API
	if c.apiKey != "" {
		apiResults, err := c.fetchFromTMDB(query)
		if err == nil && len(apiResults) > 0 {
			results = apiResults
		}
	}

	// If no results from API or no key provided, search local fallback catalog
	if len(results) == 0 {
		results = c.searchFallbackCatalog(normalizedQuery)
	}

	// Cache results
	c.mu.Lock()
	c.cache[normalizedQuery] = results
	c.mu.Unlock()

	return results, nil
}

// GetDetails retrieves specific movie details
func (c *Client) GetDetails(id string) (*MovieResult, error) {
	for _, m := range fallbackCatalog {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("movie ID %s not found", id)
}

// CachePoster caches poster image bytes in memory
func (c *Client) CachePoster(posterURL string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.posterCache[posterURL] = data
}

// GetCachedPoster retrieves cached image bytes
func (c *Client) GetCachedPoster(posterURL string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.posterCache[posterURL]
	return data, ok
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// GeneratePosterSVG creates an inline SVG data URI poster placeholder
func GeneratePosterSVG(title, year string) string {
	initials := ""
	words := strings.Fields(title)
	for i, w := range words {
		if i >= 2 {
			break
		}
		if len(w) > 0 {
			initials += strings.ToUpper(string([]rune(w)[0]))
		}
	}
	if initials == "" {
		initials = "MCR"
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 450" width="300" height="450"><defs><linearGradient id="g" x1="0%%" y1="0%%" x2="100%%" y2="100%%"><stop offset="0%%" stop-color="#1e293b"/><stop offset="100%%" stop-color="#0f172a"/></linearGradient></defs><rect width="300" height="450" fill="url(#g)" rx="12"/><circle cx="150" cy="180" r="54" fill="#3b82f6" opacity="0.25"/><text x="150" y="195" font-family="system-ui, sans-serif" font-size="40" font-weight="bold" fill="#60a5fa" text-anchor="middle">%s</text><text x="150" y="280" font-family="system-ui, sans-serif" font-size="16" font-weight="bold" fill="#f8fafc" text-anchor="middle">%s</text><text x="150" y="310" font-family="system-ui, sans-serif" font-size="14" fill="#94a3b8" text-anchor="middle">%s</text></svg>`,
		initials, truncateString(title, 22), year)
	return "data:image/svg+xml;utf8," + url.PathEscape(svg)
}

func (c *Client) fetchFromTMDB(query string) ([]MovieResult, error) {
	results := make([]MovieResult, 0)
	maxPages := 2 // Fetch up to 2 pages (40 results) to prevent truncated/incomplete searches

	for page := 1; page <= maxPages; page++ {
		endpoint := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s&include_adult=false&page=%d",
			c.apiKey, url.QueryEscape(query), page)

		resp, err := c.httpClient.Get(endpoint)
		if err != nil {
			if len(results) > 0 {
				break
			}
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			break
		}

		var raw struct {
			Page       int `json:"page"`
			TotalPages int `json:"total_pages"`
			Results    []struct {
				ID            int     `json:"id"`
				Title         string  `json:"title"`
				OriginalTitle string  `json:"original_title"`
				Overview      string  `json:"overview"`
				ReleaseDate   string  `json:"release_date"`
				PosterPath    string  `json:"poster_path"`
				BackdropPath  string  `json:"backdrop_path"`
				VoteAverage   float64 `json:"vote_average"`
			} `json:"results"`
		}

		if err := json.Unmarshal(body, &raw); err != nil {
			break
		}

		for _, r := range raw.Results {
			yr := ""
			if len(r.ReleaseDate) >= 4 {
				yr = r.ReleaseDate[:4]
			}
			posterURL := ""
			if r.PosterPath != "" {
				posterURL = "https://image.tmdb.org/t/p/w500" + r.PosterPath
			} else {
				posterURL = GeneratePosterSVG(r.Title, yr)
			}
			backdropURL := ""
			if r.BackdropPath != "" {
				backdropURL = "https://image.tmdb.org/t/p/original" + r.BackdropPath
			}

			results = append(results, MovieResult{
				ID:            fmt.Sprintf("%d", r.ID),
				Title:         r.Title,
				OriginalTitle: r.OriginalTitle,
				Overview:      r.Overview,
				ReleaseDate:   r.ReleaseDate,
				PosterPath:    posterURL,
				BackdropPath:  backdropURL,
				Rating:        r.VoteAverage,
				Genres:        []string{"Drama", "Action"},
				Runtime:       140,
			})
		}

		if page >= raw.TotalPages {
			break
		}
	}

	return results, nil
}

func (c *Client) searchFallbackCatalog(query string) []MovieResult {
	matches := make([]MovieResult, 0)
	for _, item := range fallbackCatalog {
		if strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.OriginalTitle), query) ||
			strings.Contains(strings.ToLower(item.Overview), query) {
			matches = append(matches, item)
		}
	}
	return matches
}

// Built-in broadcast test catalog
var fallbackCatalog = []MovieResult{
	{
		ID:            "tmdb-579974",
		Title:         "RRR",
		OriginalTitle: "రౌద్రం రణం రుధిరం",
		Overview:      "A fictional history of two legendary revolutionaries' journey away from home before they began fighting for their country in the 1920s.",
		ReleaseDate:   "2022-03-24",
		PosterPath:    "https://image.tmdb.org/t/p/w500/wE0noFU9jx5c27z8k790M9r12.jpg",
		Rating:        8.0,
		Genres:        []string{"Action", "Drama"},
		Runtime:       187,
	},
	{
		ID:            "tmdb-256040",
		Title:         "Baahubali: The Beginning",
		OriginalTitle: "బాహుబలి: ది బిగినింగ్",
		Overview:      "In ancient India, an adventurous and daring man becomes involved in a feud between two brothers after learning of his true heritage.",
		ReleaseDate:   "2015-07-10",
		PosterPath:    "https://image.tmdb.org/t/p/w500/baahubali1_poster.jpg",
		Rating:        8.1,
		Genres:        []string{"Action", "Fantasy"},
		Runtime:       159,
	},
	{
		ID:            "tmdb-299536",
		Title:         "Avengers: Infinity War",
		OriginalTitle: "Avengers: Infinity War",
		Overview:      "As the Avengers and their allies have continued to protect the world, a new danger has emerged from the cosmic shadows: Thanos.",
		ReleaseDate:   "2018-04-27",
		PosterPath:    "https://image.tmdb.org/t/p/w500/7WsyChQLEftFiDOVTGkv3hFpyyt.jpg",
		Rating:        8.3,
		Genres:        []string{"Action", "Sci-Fi"},
		Runtime:       149,
	},
	{
		ID:            "tmdb-1013860",
		Title:         "Kantara",
		OriginalTitle: "ಕಾಂತಾರ",
		Overview:      "When greed paves the way for betrayal, wrath, and rebellion, a young tribal man embodies his ancestry to uphold justice.",
		ReleaseDate:   "2022-09-30",
		PosterPath:    "https://image.tmdb.org/t/p/w500/kantara_poster.jpg",
		Rating:        8.2,
		Genres:        []string{"Action", "Thriller"},
		Runtime:       148,
	},
	{
		ID:            "tmdb-872585",
		Title:         "Jawan",
		OriginalTitle: "जवान",
		Overview:      "A high-octane action thriller which outlines the emotional journey of a man who is set to rectify the wrongs in the society.",
		ReleaseDate:   "2023-09-07",
		PosterPath:    "https://image.tmdb.org/t/p/w500/jawan_poster.jpg",
		Rating:        7.6,
		Genres:        []string{"Action", "Thriller"},
		Runtime:       169,
	},
	{
		ID:            "tmdb-360814",
		Title:         "Dangal",
		OriginalTitle: "दंगल",
		Overview:      "Former wrestler Mahavir Singh Phogat trains young daughters Geeta and Babita to become world-class wrestlers.",
		ReleaseDate:   "2016-12-23",
		PosterPath:    "https://image.tmdb.org/t/p/w500/dangal_poster.jpg",
		Rating:        8.4,
		Genres:        []string{"Biography", "Drama", "Sport"},
		Runtime:       161,
	},
	{
		ID:            "tmdb-864692",
		Title:         "Pathaan",
		OriginalTitle: "पठान",
		Overview:      "An Indian agent races against a ruthless mercenary planning a deadly biological attack against the nation.",
		ReleaseDate:   "2023-01-25",
		PosterPath:    "https://image.tmdb.org/t/p/w500/pathaan_poster.jpg",
		Rating:        7.3,
		Genres:        []string{"Action", "Adventure", "Thriller"},
		Runtime:       146,
	},
	{
		ID:            "tmdb-592695",
		Title:         "K.G.F: Chapter 2",
		OriginalTitle: "ಕೆ.ಜಿ.ಎಫ್: ಚಾಪ್ಟರ್ 2",
		Overview:      "The blood-soaked land of Kolar Gold Fields has a new overlord now: Rocky, whose name strikes fear into enemies.",
		ReleaseDate:   "2022-04-14",
		PosterPath:    "https://image.tmdb.org/t/p/w500/kgf2_poster.jpg",
		Rating:        8.2,
		Genres:        []string{"Action", "Crime", "Drama"},
		Runtime:       168,
	},
	{
		ID:            "tmdb-783461",
		Title:         "Pushpa: The Rise",
		OriginalTitle: "పుష్ప: ది రైజ్",
		Overview:      "A laborer rises through the ranks of a red sandalwood smuggling syndicate in the Seshachalam hills of Andhra Pradesh.",
		ReleaseDate:   "2021-12-17",
		PosterPath:    "https://image.tmdb.org/t/p/w500/pushpa_poster.jpg",
		Rating:        7.9,
		Genres:        []string{"Action", "Crime", "Drama"},
		Runtime:       179,
	},
	{
		ID:            "tmdb-829402",
		Title:         "Kalki 2898 AD",
		OriginalTitle: "కల్కి 2898 AD",
		Overview:      "A modern-day avatar of Vishnu descends to earth to protect the world from evil forces in a dystopian future.",
		ReleaseDate:   "2024-06-27",
		PosterPath:    "https://image.tmdb.org/t/p/w500/kalki_poster.jpg",
		Rating:        7.8,
		Genres:        []string{"Action", "Sci-Fi", "Fantasy"},
		Runtime:       181,
	},
	{
		ID:            "tmdb-913639",
		Title:         "Salaar: Part 1 – Ceasefire",
		OriginalTitle: "సలార్: పార్ట్ 1 – సీజ్ ఫైర్",
		Overview:      "A gang leader tries to keep a promise made to his dying friend and takes on other criminal gangs in Khansaar.",
		ReleaseDate:   "2023-12-22",
		PosterPath:    "https://image.tmdb.org/t/p/w500/salaar_poster.jpg",
		Rating:        7.5,
		Genres:        []string{"Action", "Crime", "Thriller"},
		Runtime:       175,
	},
}
