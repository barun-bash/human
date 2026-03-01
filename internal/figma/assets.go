package figma

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AssetManifest tracks downloaded assets and their local paths.
type AssetManifest struct {
	Assets []Asset `json:"assets"`
}

// Asset represents a single downloaded design asset.
type Asset struct {
	NodeID    string `json:"node_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`       // "image", "icon", "logo"
	RemoteURL string `json:"remote_url"`
	LocalPath string `json:"local_path"` // relative to output dir
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// maxAssetSize is the maximum size for a single asset download (10MB).
const maxAssetSize = 10 * 1024 * 1024

// maxConcurrentDownloads is the worker pool size for asset downloads.
const maxConcurrentDownloads = 5

// ExtractAssets identifies downloadable assets from a Figma file,
// renders them via Figma API, and downloads them to outputDir.
func ExtractAssets(client *Client, fileKey string, file *FigmaFile, outputDir string, frontend string) (*AssetManifest, error) {
	// Collect asset nodes
	var assetNodes []*assetCandidate
	seen := map[string]bool{} // deduplicate by imageRef or node ID

	for _, page := range file.Pages {
		walkNodes(page.Nodes, func(node *FigmaNode) {
			if !ShouldExtractAsset(node) {
				return
			}
			key := assetKey(node)
			if seen[key] {
				return
			}
			seen[key] = true

			assetNodes = append(assetNodes, &assetCandidate{
				node:     node,
				assetType: classifyAssetType(node),
			})
		})
	}

	if len(assetNodes) == 0 {
		return &AssetManifest{}, nil
	}

	// Determine output directory based on frontend framework
	assetDir := resolveAssetDir(outputDir, frontend)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("creating asset directory: %w", err)
	}

	// Batch request image URLs from Figma API
	// Icons/logos → SVG, images → PNG at 2x
	svgIDs, pngIDs := splitByFormat(assetNodes)

	imageURLs := map[string]string{}
	if len(svgIDs) > 0 {
		urls, err := client.GetImageURLs(fileKey, svgIDs, "svg", 1)
		if err != nil {
			// Fall back to PNG for SVGs that fail
			urls, _ = client.GetImageURLs(fileKey, svgIDs, "png", 2)
		}
		for k, v := range urls {
			imageURLs[k] = v
		}
	}
	if len(pngIDs) > 0 {
		urls, err := client.GetImageURLs(fileKey, pngIDs, "png", 2)
		if err != nil {
			return nil, fmt.Errorf("fetching image URLs: %w", err)
		}
		for k, v := range urls {
			imageURLs[k] = v
		}
	}

	// Download assets concurrently
	manifest := &AssetManifest{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentDownloads)

	for _, candidate := range assetNodes {
		remoteURL, ok := imageURLs[candidate.node.ID]
		if !ok || remoteURL == "" {
			continue
		}

		wg.Add(1)
		go func(c *assetCandidate, url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ext := ".png"
			if c.assetType == "icon" || c.assetType == "logo" {
				if strings.Contains(url, "svg") || strings.HasSuffix(url, ".svg") {
					ext = ".svg"
				}
			}

			filename := cleanFilename(c.node.Name) + ext
			localPath := filepath.Join(assetDir, filename)
			relPath, _ := filepath.Rel(outputDir, localPath)

			if err := downloadFile(url, localPath); err != nil {
				// Log but continue with other assets
				fmt.Printf("  warning: failed to download %s: %v\n", c.node.Name, err)
				return
			}

			asset := Asset{
				NodeID:    c.node.ID,
				Name:      filename,
				Type:      c.assetType,
				RemoteURL: url,
				LocalPath: relPath,
				Width:     int(c.node.Width),
				Height:    int(c.node.Height),
			}

			mu.Lock()
			manifest.Assets = append(manifest.Assets, asset)
			mu.Unlock()
		}(candidate, remoteURL)
	}

	wg.Wait()

	return manifest, nil
}

// ShouldExtractAsset determines if a node should be exported as an image asset.
func ShouldExtractAsset(node *FigmaNode) bool {
	// Node has an image fill
	for _, fill := range node.Fills {
		if fill.Type == "IMAGE" {
			return true
		}
	}

	// Node name indicates an asset
	lower := strings.ToLower(node.Name)
	assetKeywords := []string{"logo", "icon", "illustration", "photo", "image", "picture", "avatar", "banner", "hero-image"}
	for _, kw := range assetKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	// VECTOR nodes with no children (standalone icons)
	if node.Type == "VECTOR" && len(node.Children) == 0 {
		return true
	}

	return false
}

type assetCandidate struct {
	node      *FigmaNode
	assetType string
}

func classifyAssetType(node *FigmaNode) string {
	lower := strings.ToLower(node.Name)
	switch {
	case strings.Contains(lower, "logo"):
		return "logo"
	case strings.Contains(lower, "icon") || node.Type == "VECTOR":
		return "icon"
	default:
		return "image"
	}
}

func assetKey(node *FigmaNode) string {
	// Check for imageRef in fills for deduplication
	for _, fill := range node.Fills {
		if fill.Type == "IMAGE" {
			return "img:" + node.ID // imageRef not directly on Paint, use node ID
		}
	}
	return "node:" + node.ID
}

func splitByFormat(candidates []*assetCandidate) (svgIDs, pngIDs []string) {
	for _, c := range candidates {
		switch c.assetType {
		case "icon", "logo":
			// VECTOR nodes can be SVG; raster fills should be PNG
			hasImageFill := false
			for _, f := range c.node.Fills {
				if f.Type == "IMAGE" {
					hasImageFill = true
					break
				}
			}
			if hasImageFill {
				pngIDs = append(pngIDs, c.node.ID)
			} else {
				svgIDs = append(svgIDs, c.node.ID)
			}
		default:
			pngIDs = append(pngIDs, c.node.ID)
		}
	}
	return
}

func resolveAssetDir(outputDir, frontend string) string {
	lower := strings.ToLower(frontend)
	switch {
	case strings.Contains(lower, "angular"):
		return filepath.Join(outputDir, "node", "src", "assets")
	case strings.Contains(lower, "svelte"):
		return filepath.Join(outputDir, "node", "static", "assets")
	case strings.Contains(lower, "react"), strings.Contains(lower, "vue"):
		return filepath.Join(outputDir, "node", "src", "assets")
	default:
		return filepath.Join(outputDir, "node", "public", "assets")
	}
}

func cleanFilename(name string) string {
	// Replace spaces and special chars
	name = strings.ToLower(name)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	name = replacer.Replace(name)
	// Remove multiple consecutive hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return strings.Trim(name, "-")
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Limit read size
	limited := io.LimitReader(resp.Body, maxAssetSize)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, limited)
	return err
}

// walkNodes recursively visits all nodes in a tree.
func walkNodes(nodes []*FigmaNode, fn func(*FigmaNode)) {
	for _, node := range nodes {
		fn(node)
		walkNodes(node.Children, fn)
	}
}
