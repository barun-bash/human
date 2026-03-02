package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/barun-bash/human/human-studio/server/models"
)

// OAuth provider configurations
var oauthProviders = map[string]struct {
	AuthURL  string
	TokenURL string
	UserURL  string
	Scopes   string
}{
	"google": {
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		UserURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:   "openid email profile",
	},
	"slack": {
		AuthURL:  "https://slack.com/openid/connect/authorize",
		TokenURL: "https://slack.com/api/openid.connect.token",
		UserURL:  "https://slack.com/api/openid.connect.userInfo",
		Scopes:   "openid email profile",
	},
	"outlook": {
		AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserURL:  "https://graph.microsoft.com/v1.0/me",
		Scopes:   "openid email profile User.Read",
	},
}

// GetOAuthURL returns the authorization URL for the given provider.
func (s *Service) GetOAuthURL(provider, state string) (string, error) {
	p, ok := oauthProviders[provider]
	if !ok {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	clientID, err := s.getClientID(provider)
	if err != nil {
		return "", err
	}

	redirectURI := fmt.Sprintf("%s/api/auth/oauth/%s/callback", s.OAuth.RedirectBaseURL, provider)

	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {p.Scopes},
		"state":         {state},
	}

	return p.AuthURL + "?" + params.Encode(), nil
}

// HandleOAuthCallback exchanges the auth code for tokens, fetches user info,
// and finds or creates a user.
func (s *Service) HandleOAuthCallback(provider, code string) (*models.OAuthResult, error) {
	p, ok := oauthProviders[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	clientID, err := s.getClientID(provider)
	if err != nil {
		return nil, err
	}
	clientSecret, err := s.getClientSecret(provider)
	if err != nil {
		return nil, err
	}

	redirectURI := fmt.Sprintf("%s/api/auth/oauth/%s/callback", s.OAuth.RedirectBaseURL, provider)

	// Exchange code for tokens
	tokenData, err := exchangeCode(p.TokenURL, clientID, clientSecret, code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	// Fetch user info
	accessToken, _ := tokenData["access_token"].(string)
	if accessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	email, name, providerID, err := fetchUserInfo(provider, p.UserURL, accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}

	// Find or create user
	user, isNew, err := s.findOrCreateOAuthUser(provider, providerID, email, name)
	if err != nil {
		return nil, fmt.Errorf("user lookup: %w", err)
	}

	// Generate JWTs
	jwtAccess, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}
	jwtRefresh, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.OAuthResult{
		AccessToken:  jwtAccess,
		RefreshToken: jwtRefresh,
		User:         *user,
		IsNewUser:    isNew,
	}, nil
}

func (s *Service) findOrCreateOAuthUser(provider, providerID, email, name string) (*models.User, bool, error) {
	// Try to find by provider + provider_id first
	var user models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, auth_provider, created_at, updated_at
		 FROM users WHERE auth_provider = $1 AND auth_provider_id = $2`,
		provider, providerID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AuthProvider, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return &user, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	// Try to find by email (might have signed up with email/password)
	err = s.db.QueryRow(
		`SELECT id, email, name, auth_provider, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AuthProvider, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		// Link OAuth provider to existing account
		s.db.Exec(
			`UPDATE users SET auth_provider = $1, auth_provider_id = $2, updated_at = NOW() WHERE id = $3`,
			provider, providerID, user.ID,
		)
		user.AuthProvider = provider
		return &user, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}

	// Create new user
	err = s.db.QueryRow(
		`INSERT INTO users (email, name, auth_provider, auth_provider_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, name, auth_provider, created_at, updated_at`,
		email, name, provider, providerID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AuthProvider, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, false, fmt.Errorf("creating user: %w", err)
	}

	return &user, true, nil
}

func exchangeCode(tokenURL, clientID, clientSecret, code, redirectURI string) (map[string]interface{}, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if errMsg, ok := result["error"].(string); ok {
		return nil, fmt.Errorf("oauth error: %s", errMsg)
	}

	return result, nil
}

func fetchUserInfo(provider, userURL, accessToken string) (email, name, providerID string, err error) {
	req, err := http.NewRequest("GET", userURL, nil)
	if err != nil {
		return "", "", "", err
	}

	if provider == "slack" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	} else {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", "", err
	}

	switch provider {
	case "google":
		email, _ = data["email"].(string)
		name, _ = data["name"].(string)
		providerID, _ = data["id"].(string)
	case "slack":
		email, _ = data["email"].(string)
		name, _ = data["name"].(string)
		providerID, _ = data["sub"].(string)
	case "outlook":
		email, _ = data["mail"].(string)
		if email == "" {
			email, _ = data["userPrincipalName"].(string)
		}
		name, _ = data["displayName"].(string)
		providerID, _ = data["id"].(string)
	}

	if email == "" {
		return "", "", "", fmt.Errorf("could not get email from %s", provider)
	}

	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	return email, name, providerID, nil
}

func (s *Service) getClientID(provider string) (string, error) {
	switch provider {
	case "google":
		return s.OAuth.GoogleClientID, nil
	case "slack":
		return s.OAuth.SlackClientID, nil
	case "outlook":
		return s.OAuth.OutlookClientID, nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *Service) getClientSecret(provider string) (string, error) {
	switch provider {
	case "google":
		return s.OAuth.GoogleClientSecret, nil
	case "slack":
		return s.OAuth.SlackClientSecret, nil
	case "outlook":
		return s.OAuth.OutlookClientSecret, nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}
