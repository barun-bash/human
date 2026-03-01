package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseFigmaURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		fileKey string
		nodeID  string
		wantErr bool
	}{
		{
			name:    "design URL",
			url:     "https://www.figma.com/design/ABC123/MyDesign",
			fileKey: "ABC123",
		},
		{
			name:    "file URL",
			url:     "https://www.figma.com/file/XYZ789/Project",
			fileKey: "XYZ789",
		},
		{
			name:    "design URL with node-id",
			url:     "https://www.figma.com/design/ABC123/MyDesign?node-id=1-2",
			fileKey: "ABC123",
			nodeID:  "1:2",
		},
		{
			name:    "board URL",
			url:     "https://www.figma.com/board/BOARD1/Canvas",
			fileKey: "BOARD1",
		},
		{
			name:    "branch URL",
			url:     "https://www.figma.com/design/ABC123/branch/BRANCH456/MyDesign",
			fileKey: "BRANCH456",
		},
		{
			name:    "no www",
			url:     "https://figma.com/design/NOPREFIX/Test",
			fileKey: "NOPREFIX",
		},
		{
			name:    "invalid host",
			url:     "https://example.com/design/ABC/Test",
			wantErr: true,
		},
		{
			name:    "too short",
			url:     "https://figma.com/design",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			url:     "https://figma.com/proto/ABC/Test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fk, nid, err := ParseFigmaURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fk != tt.fileKey {
				t.Errorf("fileKey = %q, want %q", fk, tt.fileKey)
			}
			if nid != tt.nodeID {
				t.Errorf("nodeID = %q, want %q", nid, tt.nodeID)
			}
		})
	}
}

func TestIsFigmaURL(t *testing.T) {
	if !IsFigmaURL("https://www.figma.com/design/ABC/Test") {
		t.Error("expected Figma URL to be detected")
	}
	if !IsFigmaURL("https://figma.com/file/ABC/Test") {
		t.Error("expected Figma URL without www to be detected")
	}
	if IsFigmaURL("https://example.com/design/ABC/Test") {
		t.Error("expected non-Figma URL to be rejected")
	}
	if IsFigmaURL("not a url") {
		t.Error("expected non-URL to be rejected")
	}
}

func TestConvertAPIResponse(t *testing.T) {
	visible := true
	resp := &figmaAPIResponse{
		Name: "TestFile",
		Document: figmaAPINode{
			ID:   "0:0",
			Name: "Document",
			Type: "DOCUMENT",
			Children: []figmaAPINode{
				{
					ID:   "0:1",
					Name: "Page 1",
					Type: "CANVAS",
					Children: []figmaAPINode{
						{
							ID:   "1:1",
							Name: "Frame",
							Type: "FRAME",
							AbsoluteBoundingBox: &figmaAPIRect{Width: 800, Height: 600},
							Fills: []figmaAPIPaint{
								{Type: "SOLID", Color: &figmaAPIColor{R: 1, G: 0, B: 0, A: 1}, Visible: &visible},
							},
						},
					},
				},
			},
		},
	}

	file := convertAPIResponse(resp)
	if file.Name != "TestFile" {
		t.Errorf("file name = %q, want %q", file.Name, "TestFile")
	}
	if len(file.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(file.Pages))
	}
	if file.Pages[0].Name != "Page 1" {
		t.Errorf("page name = %q, want %q", file.Pages[0].Name, "Page 1")
	}
	if len(file.Pages[0].Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(file.Pages[0].Nodes))
	}
	node := file.Pages[0].Nodes[0]
	if node.Width != 800 {
		t.Errorf("width = %f, want 800", node.Width)
	}
	if len(node.Fills) != 1 || node.Fills[0].Type != "SOLID" {
		t.Error("expected SOLID fill")
	}
}

func TestConvertAPINodeHiddenSkipped(t *testing.T) {
	visible := false
	parent := figmaAPINode{
		ID:   "1:1",
		Name: "Parent",
		Type: "FRAME",
		Children: []figmaAPINode{
			{ID: "1:2", Name: "Visible", Type: "TEXT"},
			{ID: "1:3", Name: "Hidden", Type: "TEXT", Visible: &visible},
		},
	}
	node := convertAPINode(&parent)
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 visible child, got %d", len(node.Children))
	}
	if node.Children[0].Name != "Visible" {
		t.Errorf("expected Visible child, got %q", node.Children[0].Name)
	}
}

func TestClientGetFileMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-FIGMA-TOKEN") != "test-token" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"name":         "Test File",
			"lastModified": "2025-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	// Override the base URL for testing by using the server directly.
	client := &Client{Token: "test-token", HTTPClient: server.Client()}

	// We can't easily override figmaAPIBase, so test via doRequest directly.
	body, err := client.doRequest(server.URL + "/files/ABC?depth=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp struct {
		Name         string `json:"name"`
		LastModified string `json:"lastModified"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "Test File" {
		t.Errorf("name = %q, want %q", resp.Name, "Test File")
	}
}

func TestClientDoRequestRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(429)
			return
		}
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := &Client{Token: "test", HTTPClient: server.Client()}
	body, err := client.doRequest(server.URL + "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok": true}` {
		t.Errorf("body = %q", string(body))
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientDoRequestNoToken(t *testing.T) {
	client := &Client{Token: "", HTTPClient: &http.Client{}}
	_, err := client.doRequest("http://example.com")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestClientDoRequestUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	client := &Client{Token: "bad", HTTPClient: server.Client()}
	_, err := client.doRequest(server.URL + "/test")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}
