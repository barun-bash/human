package figma

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCachePutAndGet(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: dir}

	entry := &CacheEntry{
		FileKey:      "ABC123",
		LastModified: "2025-01-01T00:00:00Z",
		FetchedAt:    time.Now(),
		ContentHash:  "deadbeef",
		FileName:     "TestFile",
	}

	if err := cache.Put(entry); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, ok := cache.Get("ABC123")
	if !ok {
		t.Fatal("Get returned not found")
	}
	if got.FileKey != "ABC123" {
		t.Errorf("FileKey = %q, want %q", got.FileKey, "ABC123")
	}
	if got.LastModified != "2025-01-01T00:00:00Z" {
		t.Errorf("LastModified = %q, want %q", got.LastModified, "2025-01-01T00:00:00Z")
	}
}

func TestCacheGetMissing(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}
	_, ok := cache.Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestCacheGetExpired(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: dir}

	entry := &CacheEntry{
		FileKey:   "OLD",
		FetchedAt: time.Now().Add(-(maxCacheAge + time.Hour)),
	}
	if err := cache.Put(entry); err != nil {
		t.Fatal(err)
	}

	_, ok := cache.Get("OLD")
	if ok {
		t.Fatal("expected expired entry to not be returned")
	}

	// Verify file was cleaned up.
	if _, err := os.Stat(filepath.Join(dir, "OLD.json")); !os.IsNotExist(err) {
		t.Error("expected expired cache file to be deleted")
	}
}

func TestCacheIsStale(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}
	entry := &CacheEntry{LastModified: "v1"}

	if cache.IsStale(entry, "v1") {
		t.Error("same version should not be stale")
	}
	if !cache.IsStale(entry, "v2") {
		t.Error("different version should be stale")
	}
}

func TestCacheInvalidate(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: dir}

	entry := &CacheEntry{
		FileKey:   "DEL",
		FetchedAt: time.Now(),
	}
	if err := cache.Put(entry); err != nil {
		t.Fatal(err)
	}

	if _, ok := cache.Get("DEL"); !ok {
		t.Fatal("expected entry to exist before invalidation")
	}

	if err := cache.Invalidate("DEL"); err != nil {
		t.Fatal(err)
	}

	if _, ok := cache.Get("DEL"); ok {
		t.Fatal("expected entry to be gone after invalidation")
	}
}

func TestCacheClear(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: filepath.Join(dir, "cache")}

	entry := &CacheEntry{FileKey: "A", FetchedAt: time.Now()}
	cache.Put(entry)

	if err := cache.Clear(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cache.Dir); !os.IsNotExist(err) {
		t.Error("expected cache dir to be removed")
	}
}

func TestContentHash(t *testing.T) {
	file := &FigmaFile{Name: "Test", Pages: []*FigmaPage{{Name: "P1"}}}
	hash := ContentHash(file)
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if len(hash) != 64 { // SHA256 hex
		t.Errorf("expected 64-char hash, got %d", len(hash))
	}

	// Same input = same hash.
	hash2 := ContentHash(file)
	if hash != hash2 {
		t.Error("expected same hash for same input")
	}

	// Different input = different hash.
	file2 := &FigmaFile{Name: "Other", Pages: []*FigmaPage{{Name: "P2"}}}
	hash3 := ContentHash(file2)
	if hash == hash3 {
		t.Error("expected different hash for different input")
	}
}

func TestGenerateFromCacheWithOutput(t *testing.T) {
	entry := &CacheEntry{
		GeneratedOutput: "app MyApp is a web application\n",
	}
	code, err := GenerateFromCache(entry, nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != entry.GeneratedOutput {
		t.Errorf("expected cached output, got %q", code)
	}
}
