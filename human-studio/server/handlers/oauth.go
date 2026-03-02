package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/barun-bash/human/human-studio/server/auth"
)

// OAuthStart redirects the user to the OAuth provider's authorization page.
func OAuthStart(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		if provider == "" {
			jsonError(w, "provider is required", http.StatusBadRequest)
			return
		}

		// Generate random state for CSRF protection
		stateBytes := make([]byte, 16)
		rand.Read(stateBytes)
		state := hex.EncodeToString(stateBytes)

		authURL, err := svc.GetOAuthURL(provider, state)
		if err != nil {
			jsonError(w, fmt.Sprintf("failed to get OAuth URL: %v", err), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

// OAuthCallback handles the OAuth provider callback.
// Returns an HTML page that posts auth data back to the Electron app.
func OAuthCallback(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.PathValue("provider")
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")

		if errParam != "" {
			renderOAuthResult(w, nil, fmt.Errorf("provider error: %s", errParam))
			return
		}

		if code == "" {
			renderOAuthResult(w, nil, fmt.Errorf("missing authorization code"))
			return
		}

		result, err := svc.HandleOAuthCallback(provider, code)
		if err != nil {
			renderOAuthResult(w, nil, err)
			return
		}

		renderOAuthResult(w, result, nil)
	}
}

func renderOAuthResult(w http.ResponseWriter, result interface{}, authErr error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var dataJSON string
	if authErr != nil {
		errData := map[string]string{"error": authErr.Error()}
		b, _ := json.Marshal(errData)
		dataJSON = string(b)
	} else {
		b, _ := json.Marshal(result)
		dataJSON = string(b)
	}

	// HTML page that stores the auth result in a known element
	// so Electron can read it via executeJavaScript
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication</title></head>
<body>
<div id="auth-result" style="display:none">%s</div>
<p style="font-family:system-ui;text-align:center;margin-top:40px;color:#666">
Authentication complete. You can close this window.
</p>
<script>
  document.title = "human-oauth-callback";
</script>
</body>
</html>`, dataJSON)
}
