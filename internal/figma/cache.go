package figma

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache stores classified Figma data for deterministic re-imports.
type Cache struct {
	Dir string // default: .human/cache/figma/
}

// CacheEntry represents a cached Figma file analysis.
type CacheEntry struct {
	FileKey        string           `json:"file_key"`
	LastModified   string           `json:"last_modified"`
	FetchedAt      time.Time        `json:"fetched_at"`
	ContentHash    string           `json:"content_hash"`
	FileName       string           `json:"file_name"`
	ClassifiedPages []*ClassifiedPage `json:"classified_pages"`
	InferredModels  []*InferredModel  `json:"inferred_models"`
	ExtractedTheme  *extractedTheme   `json:"extracted_theme"`
	GeneratedOutput string           `json:"generated_output"` // the .human file content
}

// maxCacheAge is the maximum age before a cache entry is auto-invalidated.
const maxCacheAge = 7 * 24 * time.Hour

// Get retrieves a cached analysis if it exists and is not expired.
func (c *Cache) Get(fileKey string) (*CacheEntry, bool) {
	path := c.entryPath(fileKey)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	// Auto-invalidate entries older than maxCacheAge
	if time.Since(entry.FetchedAt) > maxCacheAge {
		_ = os.Remove(path)
		return nil, false
	}
	return &entry, true
}

// Put stores an analysis result in the cache.
func (c *Cache) Put(entry *CacheEntry) error {
	path := c.entryPath(entry.FileKey)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// IsStale checks if the cached entry is outdated by comparing lastModified.
func (c *Cache) IsStale(entry *CacheEntry, currentLastModified string) bool {
	return entry.LastModified != currentLastModified
}

// Invalidate removes a cached entry.
func (c *Cache) Invalidate(fileKey string) error {
	return os.Remove(c.entryPath(fileKey))
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	return os.RemoveAll(c.Dir)
}

func (c *Cache) entryPath(fileKey string) string {
	return filepath.Join(c.Dir, fileKey+".json")
}

// ContentHash computes a SHA256 hash of the Figma file content for cache validation.
func ContentHash(file *FigmaFile) string {
	data, err := json.Marshal(file)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// GenerateFromCache rebuilds a .human file from cached classification data.
func GenerateFromCache(entry *CacheEntry, cfg *GenerateConfig) (string, error) {
	// If we have the previously generated output, return it directly.
	if entry.GeneratedOutput != "" {
		return entry.GeneratedOutput, nil
	}

	// Otherwise re-run the assembly from cached data.
	if cfg == nil {
		cfg = &GenerateConfig{
			AppName:  toPascalCase(entry.FileName),
			Platform: "web",
			Frontend: "React",
			Backend:  "Node",
			Database: "PostgreSQL",
		}
	}

	// Map pages from cached classified pages
	var pageBlocks []string
	for _, cp := range entry.ClassifiedPages {
		pageBlocks = append(pageBlocks, MapToHuman(cp, cfg.AppName))
	}

	// Generate CRUD APIs from cached models
	var apiBlocks []string
	for _, model := range entry.InferredModels {
		apiBlocks = append(apiBlocks, generateCRUDAPIs(model)...)
	}

	theme := entry.ExtractedTheme
	if theme == nil {
		theme = &extractedTheme{}
	}

	return assembleHumanFile(cfg, theme, pageBlocks, nil, entry.InferredModels, apiBlocks), nil
}
