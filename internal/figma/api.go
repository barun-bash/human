package figma

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/config"
)

const figmaAPIBase = "https://api.figma.com/v1"

// Client interacts with the Figma REST API.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a Figma API client.
// Token resolution: direct token > FIGMA_TOKEN env > FIGMA_ACCESS_TOKEN env > global config MCP.
func NewClient(token string) *Client {
	if token == "" {
		token = resolveFigmaToken()
	}
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// resolveFigmaToken resolves the Figma token from environment and global config.
func resolveFigmaToken() string {
	if t := os.Getenv("FIGMA_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("FIGMA_ACCESS_TOKEN"); t != "" {
		return t
	}
	// Check global MCP config for Figma server env vars.
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		return ""
	}
	for _, mcp := range gc.MCP {
		if strings.EqualFold(mcp.Name, "figma") {
			if t, ok := mcp.Env["FIGMA_ACCESS_TOKEN"]; ok && t != "" {
				return t
			}
		}
	}
	return ""
}

// FileMetadata holds lightweight file info without the full document tree.
type FileMetadata struct {
	Name         string
	LastModified string
}

// GetFileMetadata fetches only the file's name and lastModified timestamp.
// Much cheaper than GetFile (uses depth=1 to skip the full document tree).
func (c *Client) GetFileMetadata(fileKey string) (*FileMetadata, error) {
	url := fmt.Sprintf("%s/files/%s?depth=1", figmaAPIBase, fileKey)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Name         string `json:"name"`
		LastModified string `json:"lastModified"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing file metadata: %w", err)
	}

	return &FileMetadata{
		Name:         resp.Name,
		LastModified: resp.LastModified,
	}, nil
}

// GetFile fetches a complete Figma file and converts it to FigmaFile.
func (c *Client) GetFile(fileKey string) (*FigmaFile, error) {
	url := fmt.Sprintf("%s/files/%s", figmaAPIBase, fileKey)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp figmaAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing Figma file: %w", err)
	}

	return convertAPIResponse(&resp), nil
}

// GetFileNodes fetches specific nodes from a Figma file.
func (c *Client) GetFileNodes(fileKey string, nodeIDs []string) (*FigmaFile, error) {
	ids := strings.Join(nodeIDs, ",")
	url := fmt.Sprintf("%s/files/%s/nodes?ids=%s", figmaAPIBase, fileKey, ids)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Name  string                       `json:"name"`
		Nodes map[string]figmaAPINodeEntry `json:"nodes"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing Figma nodes: %w", err)
	}

	file := &FigmaFile{Name: resp.Name}
	page := &FigmaPage{Name: "Selected"}
	for _, entry := range resp.Nodes {
		if entry.Document.ID != "" {
			page.Nodes = append(page.Nodes, convertAPINode(&entry.Document))
		}
	}
	if len(page.Nodes) > 0 {
		file.Pages = append(file.Pages, page)
	}
	return file, nil
}

// GetImageURLs gets rendered image URLs for specific nodes.
func (c *Client) GetImageURLs(fileKey string, nodeIDs []string, format string, scale float64) (map[string]string, error) {
	if format == "" {
		format = "png"
	}
	if scale == 0 {
		scale = 2
	}

	ids := strings.Join(nodeIDs, ",")
	url := fmt.Sprintf("%s/images/%s?ids=%s&format=%s&scale=%g", figmaAPIBase, fileKey, ids, format, scale)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Images map[string]string `json:"images"`
		Err    string            `json:"err"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing image URLs: %w", err)
	}
	if resp.Err != "" {
		return nil, fmt.Errorf("Figma image API error: %s", resp.Err)
	}

	return resp.Images, nil
}

// doRequest executes an authenticated GET request with retry on 429.
func (c *Client) doRequest(reqURL string) ([]byte, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("no Figma token configured. Set FIGMA_TOKEN env var or add Figma MCP server")
	}

	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("X-FIGMA-TOKEN", c.Token)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Figma API request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return body, nil
		case http.StatusTooManyRequests:
			if attempt == maxRetries {
				return nil, fmt.Errorf("Figma API rate limit exceeded after %d retries", maxRetries)
			}
			backoff := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
			time.Sleep(backoff)
			continue
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("Figma API: unauthorized (401) — check your FIGMA_TOKEN")
		case http.StatusForbidden:
			return nil, fmt.Errorf("Figma API: forbidden (403) — you may not have access to this file")
		case http.StatusNotFound:
			return nil, fmt.Errorf("Figma API: file not found (404) — check the file key")
		default:
			return nil, fmt.Errorf("Figma API error %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil, fmt.Errorf("Figma API: unexpected retry exhaustion")
}

// ── Figma API Response Types ──

type figmaAPIResponse struct {
	Name         string       `json:"name"`
	LastModified string       `json:"lastModified"`
	Document     figmaAPINode `json:"document"`
}

type figmaAPINodeEntry struct {
	Document figmaAPINode `json:"document"`
}

type figmaAPINode struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Children   []figmaAPINode `json:"children"`
	Characters string         `json:"characters"`
	Visible    *bool          `json:"visible"` // nil = visible

	// Layout
	LayoutMode  string  `json:"layoutMode"`
	PrimaryAxisAlignItems string `json:"primaryAxisAlignItems"`
	CounterAxisAlignItems string `json:"counterAxisAlignItems"`
	ItemSpacing   float64 `json:"itemSpacing"`
	PaddingLeft   float64 `json:"paddingLeft"`
	PaddingRight  float64 `json:"paddingRight"`
	PaddingTop    float64 `json:"paddingTop"`
	PaddingBottom float64 `json:"paddingBottom"`

	// Visual
	Fills        []figmaAPIPaint  `json:"fills"`
	Strokes      []figmaAPIPaint  `json:"strokes"`
	Effects      []figmaAPIEffect `json:"effects"`
	CornerRadius float64          `json:"cornerRadius"`
	Opacity      float64          `json:"opacity"`

	// Text
	Style *figmaAPITextStyle `json:"style"`

	// Size
	AbsoluteBoundingBox *figmaAPIRect `json:"absoluteBoundingBox"`

	// Component
	ComponentID string `json:"componentId"`

	// Export
	ExportSettings []figmaAPIExportSetting `json:"exportSettings"`
}

type figmaAPIPaint struct {
	Type    string        `json:"type"`
	Color   *figmaAPIColor `json:"color"`
	Visible *bool         `json:"visible"`
	Opacity float64       `json:"opacity"`
	ImageRef string       `json:"imageRef"`
}

type figmaAPIColor struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}

type figmaAPIEffect struct {
	Type    string         `json:"type"`
	Visible bool           `json:"visible"`
	Radius  float64        `json:"radius"`
	Color   *figmaAPIColor `json:"color"`
	Offset  *figmaAPIVector `json:"offset"`
}

type figmaAPIVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type figmaAPITextStyle struct {
	FontFamily          string  `json:"fontFamily"`
	FontSize            float64 `json:"fontSize"`
	FontWeight          float64 `json:"fontWeight"`
	LineHeightPx        float64 `json:"lineHeightPx"`
	LetterSpacing       float64 `json:"letterSpacing"`
	TextAlignHorizontal string  `json:"textAlignHorizontal"`
}

type figmaAPIRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type figmaAPIExportSetting struct {
	Format     string  `json:"format"` // PNG, SVG, JPG, PDF
	Constraint struct {
		Type  string  `json:"type"`
		Value float64 `json:"value"`
	} `json:"constraint"`
}

// ── Conversion ──

func convertAPIResponse(resp *figmaAPIResponse) *FigmaFile {
	file := &FigmaFile{Name: resp.Name}

	for _, child := range resp.Document.Children {
		if child.Type == "CANVAS" {
			page := &FigmaPage{Name: child.Name}
			for _, node := range child.Children {
				if isNodeVisible(&node) {
					page.Nodes = append(page.Nodes, convertAPINode(&node))
				}
			}
			file.Pages = append(file.Pages, page)
		}
	}

	return file
}

func convertAPINode(n *figmaAPINode) *FigmaNode {
	node := &FigmaNode{
		ID:            n.ID,
		Name:          n.Name,
		Type:          n.Type,
		Characters:    n.Characters,
		LayoutMode:    n.LayoutMode,
		PrimaryAxis:   n.PrimaryAxisAlignItems,
		CounterAxis:   n.CounterAxisAlignItems,
		ItemSpacing:   n.ItemSpacing,
		PaddingLeft:   n.PaddingLeft,
		PaddingRight:  n.PaddingRight,
		PaddingTop:    n.PaddingTop,
		PaddingBottom: n.PaddingBottom,
		CornerRadius:  n.CornerRadius,
		Opacity:       n.Opacity,
		ComponentID:   n.ComponentID,
	}

	if n.AbsoluteBoundingBox != nil {
		node.Width = n.AbsoluteBoundingBox.Width
		node.Height = n.AbsoluteBoundingBox.Height
	}

	if n.Style != nil {
		node.Style = &TextStyle{
			FontFamily:    n.Style.FontFamily,
			FontSize:      n.Style.FontSize,
			FontWeight:    n.Style.FontWeight,
			LineHeight:    n.Style.LineHeightPx,
			LetterSpacing: n.Style.LetterSpacing,
			TextAlign:     n.Style.TextAlignHorizontal,
		}
	}

	// Convert fills
	for _, f := range n.Fills {
		p := Paint{
			Type:    f.Type,
			Visible: f.Visible == nil || *f.Visible,
			Opacity: f.Opacity,
		}
		if f.Color != nil {
			p.Color = Color{R: f.Color.R, G: f.Color.G, B: f.Color.B, A: f.Color.A}
		}
		node.Fills = append(node.Fills, p)
	}

	// Convert strokes
	for _, s := range n.Strokes {
		p := Paint{
			Type:    s.Type,
			Visible: s.Visible == nil || *s.Visible,
			Opacity: s.Opacity,
		}
		if s.Color != nil {
			p.Color = Color{R: s.Color.R, G: s.Color.G, B: s.Color.B, A: s.Color.A}
		}
		node.Strokes = append(node.Strokes, p)
	}

	// Convert effects
	for _, e := range n.Effects {
		eff := Effect{
			Type:    e.Type,
			Visible: e.Visible,
			Radius:  e.Radius,
		}
		if e.Color != nil {
			eff.Color = Color{R: e.Color.R, G: e.Color.G, B: e.Color.B, A: e.Color.A}
		}
		if e.Offset != nil {
			eff.OffsetX = e.Offset.X
			eff.OffsetY = e.Offset.Y
		}
		node.Effects = append(node.Effects, eff)
	}

	// Recursively convert children, skipping hidden nodes
	for i := range n.Children {
		child := &n.Children[i]
		if isNodeVisible(child) {
			node.Children = append(node.Children, convertAPINode(child))
		}
	}

	return node
}

func isNodeVisible(n *figmaAPINode) bool {
	return n.Visible == nil || *n.Visible
}

// ── URL Parsing ──

// ParseFigmaURL extracts file key and optional node ID from a Figma URL.
// Supports:
//
//	https://www.figma.com/file/XXXXX/Name
//	https://www.figma.com/design/XXXXX/Name
//	https://www.figma.com/design/XXXXX/Name?node-id=1-2
//	https://www.figma.com/design/XXXXX/branch/BBBBB/Name
func ParseFigmaURL(rawURL string) (fileKey string, nodeID string, err error) {
	// Strip protocol and www
	u := rawURL
	for _, prefix := range []string{"https://", "http://", "www."} {
		u = strings.TrimPrefix(u, prefix)
	}

	// Extract query params for node-id
	if idx := strings.Index(u, "?"); idx >= 0 {
		query := u[idx+1:]
		u = u[:idx]
		for _, param := range strings.Split(query, "&") {
			if strings.HasPrefix(param, "node-id=") {
				nodeID = strings.TrimPrefix(param, "node-id=")
				nodeID = strings.ReplaceAll(nodeID, "-", ":")
			}
		}
	}

	// Parse path: figma.com/design|file/<fileKey>/...
	parts := strings.Split(u, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid Figma URL: expected figma.com/design/<fileKey>/...")
	}

	host := parts[0]
	if !strings.Contains(host, "figma.com") {
		return "", "", fmt.Errorf("not a Figma URL: host is %s", host)
	}

	kind := parts[1]
	if kind != "design" && kind != "file" && kind != "board" {
		return "", "", fmt.Errorf("unsupported Figma URL type: %s (expected design, file, or board)", kind)
	}

	fileKey = parts[2]

	// Handle branch URLs: /design/:fileKey/branch/:branchKey/:fileName
	if len(parts) >= 5 && parts[3] == "branch" {
		fileKey = parts[4]
	}

	if fileKey == "" {
		return "", "", fmt.Errorf("could not extract file key from Figma URL")
	}

	return fileKey, nodeID, nil
}

// IsFigmaURL checks if a string looks like a Figma URL.
func IsFigmaURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	return strings.Contains(host, "figma.com") || strings.HasPrefix(strings.ToLower(s), "figma.com")
}
