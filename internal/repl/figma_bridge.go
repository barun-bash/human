package repl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/figma"
)

// parseFigmaURL extracts the fileKey and nodeId from a Figma URL.
// Supported formats:
//
//	figma.com/design/:fileKey/:fileName?node-id=:nodeId
//	figma.com/design/:fileKey/branch/:branchKey/:fileName
//	figma.com/file/:fileKey/:fileName?node-id=:nodeId
func parseFigmaURL(rawURL string) (fileKey, nodeID string, err error) {
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
				// Figma URLs use "-" as separator; API expects ":"
				nodeID = strings.ReplaceAll(nodeID, "-", ":")
			}
		}
	}

	// Parse path: figma.com/design|file/<fileKey>/...
	parts := strings.Split(u, "/")
	// parts[0] = "figma.com", parts[1] = "design"|"file"|"board", parts[2] = fileKey
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid Figma URL: expected figma.com/design/<fileKey>/...")
	}

	kind := parts[1]
	if kind != "design" && kind != "file" && kind != "board" {
		return "", "", fmt.Errorf("unsupported Figma URL type: %s (expected design, file, or board)", kind)
	}

	fileKey = parts[2]

	// Handle branch URLs: /design/:fileKey/branch/:branchKey/:fileName
	if len(parts) >= 5 && parts[3] == "branch" {
		fileKey = parts[4] // use branchKey as fileKey
	}

	if fileKey == "" {
		return "", "", fmt.Errorf("could not extract file key from Figma URL")
	}

	return fileKey, nodeID, nil
}

// figmaResponseToFile converts raw JSON from Figma MCP get_design_context or
// get_metadata into a figma.FigmaFile. The JSON structure varies by MCP tool,
// so we attempt multiple formats.
func figmaResponseToFile(name string, rawJSON string) (*figma.FigmaFile, error) {
	// Try parsing as a Figma document response (from get_metadata).
	var doc struct {
		Name     string `json:"name"`
		Document struct {
			Children []json.RawMessage `json:"children"`
		} `json:"document"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &doc); err == nil && len(doc.Document.Children) > 0 {
		file := &figma.FigmaFile{Name: doc.Name}
		if file.Name == "" {
			file.Name = name
		}
		for _, child := range doc.Document.Children {
			page, err := parsePageJSON(child)
			if err == nil {
				file.Pages = append(file.Pages, page)
			}
		}
		if len(file.Pages) > 0 {
			return file, nil
		}
	}

	// Try parsing as a flat node list (from get_design_context).
	var nodes []json.RawMessage
	if err := json.Unmarshal([]byte(rawJSON), &nodes); err == nil && len(nodes) > 0 {
		page := &figma.FigmaPage{Name: "Design"}
		for _, raw := range nodes {
			node, err := parseNodeJSON(raw)
			if err == nil {
				page.Nodes = append(page.Nodes, node)
			}
		}
		return &figma.FigmaFile{
			Name:  name,
			Pages: []*figma.FigmaPage{page},
		}, nil
	}

	// Try parsing as a single node tree.
	node, err := parseNodeJSON([]byte(rawJSON))
	if err == nil {
		return &figma.FigmaFile{
			Name: name,
			Pages: []*figma.FigmaPage{{
				Name:  "Design",
				Nodes: []*figma.FigmaNode{node},
			}},
		}, nil
	}

	return nil, fmt.Errorf("could not parse Figma MCP response as a recognizable format")
}

// parsePageJSON converts a JSON page object into a FigmaPage.
func parsePageJSON(raw json.RawMessage) (*figma.FigmaPage, error) {
	var p struct {
		Name     string            `json:"name"`
		Type     string            `json:"type"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}

	page := &figma.FigmaPage{Name: p.Name}
	for _, child := range p.Children {
		node, err := parseNodeJSON(child)
		if err == nil {
			page.Nodes = append(page.Nodes, node)
		}
	}
	return page, nil
}

// parseNodeJSON recursively converts a JSON node object into a FigmaNode.
func parseNodeJSON(raw json.RawMessage) (*figma.FigmaNode, error) {
	var n struct {
		ID            string            `json:"id"`
		Name          string            `json:"name"`
		Type          string            `json:"type"`
		Characters    string            `json:"characters"`
		Children      []json.RawMessage `json:"children"`
		LayoutMode    string            `json:"layoutMode"`
		ItemSpacing   float64           `json:"itemSpacing"`
		PaddingLeft   float64           `json:"paddingLeft"`
		PaddingRight  float64           `json:"paddingRight"`
		PaddingTop    float64           `json:"paddingTop"`
		PaddingBottom float64           `json:"paddingBottom"`
		CornerRadius  float64           `json:"cornerRadius"`
		Opacity       float64           `json:"opacity"`
		ComponentID   string            `json:"componentId"`
		AbsoluteBoundingBox struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"absoluteBoundingBox"`
		Style struct {
			FontFamily    string  `json:"fontFamily"`
			FontSize      float64 `json:"fontSize"`
			FontWeight    float64 `json:"fontWeight"`
			LineHeight    float64 `json:"lineHeightPx"`
			LetterSpacing float64 `json:"letterSpacing"`
			TextAlignHorizontal string `json:"textAlignHorizontal"`
		} `json:"style"`
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, err
	}

	node := &figma.FigmaNode{
		ID:            n.ID,
		Name:          n.Name,
		Type:          n.Type,
		Characters:    n.Characters,
		LayoutMode:    n.LayoutMode,
		ItemSpacing:   n.ItemSpacing,
		PaddingLeft:   n.PaddingLeft,
		PaddingRight:  n.PaddingRight,
		PaddingTop:    n.PaddingTop,
		PaddingBottom: n.PaddingBottom,
		CornerRadius:  n.CornerRadius,
		Opacity:       n.Opacity,
		ComponentID:   n.ComponentID,
		Width:         n.AbsoluteBoundingBox.Width,
		Height:        n.AbsoluteBoundingBox.Height,
	}

	if n.Style.FontFamily != "" {
		node.Style = &figma.TextStyle{
			FontFamily:    n.Style.FontFamily,
			FontSize:      n.Style.FontSize,
			FontWeight:    n.Style.FontWeight,
			LineHeight:    n.Style.LineHeight,
			LetterSpacing: n.Style.LetterSpacing,
			TextAlign:     n.Style.TextAlignHorizontal,
		}
	}

	for _, child := range n.Children {
		childNode, err := parseNodeJSON(child)
		if err == nil {
			node.Children = append(node.Children, childNode)
		}
	}

	return node, nil
}
