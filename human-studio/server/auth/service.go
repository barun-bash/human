package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/barun-bash/human/human-studio/server/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type OAuthConfig struct {
	GoogleClientID      string
	GoogleClientSecret  string
	SlackClientID       string
	SlackClientSecret   string
	OutlookClientID     string
	OutlookClientSecret string
	RedirectBaseURL     string
}

type Service struct {
	db        *sql.DB
	jwtSecret []byte
	OAuth     OAuthConfig
}

func NewService(db *sql.DB, jwtSecret string, oauth OAuthConfig) *Service {
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	return &Service{db: db, jwtSecret: []byte(jwtSecret), OAuth: oauth}
}

func (s *Service) Signup(req models.SignupRequest) (*models.LoginResponse, error) {
	// Check email uniqueness
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking email: %w", err)
	}
	if exists {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	// Insert user
	var user models.User
	err = s.db.QueryRow(
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, name, created_at, updated_at`,
		req.Email, req.Name, string(hash),
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Create free subscription
	_, err = s.db.Exec(
		`INSERT INTO subscriptions (user_id, plan, status) VALUES ($1, 'free', 'active')`,
		user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("creating subscription: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *Service) Login(req models.LoginRequest) (*models.LoginResponse, error) {
	var user models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *Service) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", ErrInvalidToken
	}

	return userID, nil
}

func (s *Service) RefreshTokens(refreshTokenStr string) (*models.LoginResponse, error) {
	// Hash the token to look it up
	hash := hashToken(refreshTokenStr)

	var userID string
	var expiresAt time.Time
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`,
		hash,
	).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("looking up refresh token: %w", err)
	}

	if time.Now().After(expiresAt) {
		// Clean up expired token
		s.db.Exec("DELETE FROM refresh_tokens WHERE token_hash = $1", hash)
		return nil, ErrInvalidToken
	}

	// Delete old refresh token (rotation)
	s.db.Exec("DELETE FROM refresh_tokens WHERE token_hash = $1", hash)

	// Get user
	var user models.User
	err = s.db.QueryRow(
		`SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}
	newRefreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *Service) GetUser(userID string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		`SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	return &user, nil
}

func (s *Service) UpdateProfile(userID string, update models.ProfileUpdate) (*models.User, error) {
	if update.Name != nil {
		_, err := s.db.Exec("UPDATE users SET name = $1, updated_at = NOW() WHERE id = $2", *update.Name, userID)
		if err != nil {
			return nil, fmt.Errorf("updating name: %w", err)
		}
	}
	if update.Email != nil {
		_, err := s.db.Exec("UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2", *update.Email, userID)
		if err != nil {
			return nil, fmt.Errorf("updating email: %w", err)
		}
	}
	return s.GetUser(userID)
}

func (s *Service) ChangePassword(userID string, change models.PasswordChange) error {
	var currentHash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(change.CurrentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(change.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	_, err = s.db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(newHash), userID)
	return err
}

func (s *Service) DeleteUser(userID string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = $1", userID)
	return err
}

func (s *Service) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) generateRefreshToken(userID string) (string, error) {
	// Generate random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	tokenStr := hex.EncodeToString(b)

	// Store hashed version
	hash := hashToken(tokenStr)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	_, err := s.db.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("storing refresh token: %w", err)
	}

	return tokenStr, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
