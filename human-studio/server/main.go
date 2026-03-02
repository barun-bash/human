package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/barun-bash/human/human-studio/server/auth"
	"github.com/barun-bash/human/human-studio/server/billing"
	"github.com/barun-bash/human/human-studio/server/handlers"
	"github.com/barun-bash/human/human-studio/server/middleware"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/human_studio?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database unreachable: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Services
	oauthConfig := auth.OAuthConfig{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		SlackClientID:      os.Getenv("SLACK_CLIENT_ID"),
		SlackClientSecret:  os.Getenv("SLACK_CLIENT_SECRET"),
		OutlookClientID:    os.Getenv("OUTLOOK_CLIENT_ID"),
		OutlookClientSecret: os.Getenv("OUTLOOK_CLIENT_SECRET"),
		RedirectBaseURL:    os.Getenv("OAUTH_REDIRECT_BASE_URL"),
	}
	if oauthConfig.RedirectBaseURL == "" {
		oauthConfig.RedirectBaseURL = "http://localhost:8080"
	}

	authService := auth.NewService(db, os.Getenv("JWT_SECRET"), oauthConfig)
	billingService := billing.NewService(db, os.Getenv("STRIPE_SECRET_KEY"))

	// Router
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Auth routes (public)
	mux.HandleFunc("POST /api/auth/signup", handlers.Signup(authService))
	mux.HandleFunc("POST /api/auth/login", handlers.Login(authService))
	mux.HandleFunc("POST /api/auth/refresh", handlers.RefreshToken(authService))
	mux.HandleFunc("POST /api/auth/reset-password", handlers.ResetPassword(authService))

	// OAuth routes (public)
	mux.HandleFunc("GET /api/auth/oauth/{provider}/start", handlers.OAuthStart(authService))
	mux.HandleFunc("GET /api/auth/oauth/{provider}/callback", handlers.OAuthCallback(authService))

	// Protected routes
	protected := middleware.Auth(authService)

	mux.Handle("GET /api/user/profile", protected(http.HandlerFunc(handlers.GetProfile(authService))))
	mux.Handle("PUT /api/user/profile", protected(http.HandlerFunc(handlers.UpdateProfile(authService))))
	mux.Handle("PUT /api/user/password", protected(http.HandlerFunc(handlers.ChangePassword(authService))))
	mux.Handle("DELETE /api/user/account", protected(http.HandlerFunc(handlers.DeleteAccount(authService))))

	// Billing routes (protected)
	mux.Handle("GET /api/billing/subscription", protected(http.HandlerFunc(handlers.GetSubscription(billingService))))
	mux.Handle("POST /api/billing/checkout", protected(http.HandlerFunc(handlers.CreateCheckout(billingService))))
	mux.Handle("POST /api/billing/select-plan", protected(http.HandlerFunc(handlers.SelectPlan(billingService))))
	mux.Handle("GET /api/billing/history", protected(http.HandlerFunc(handlers.GetBillingHistory(billingService))))
	mux.Handle("PUT /api/billing/payment-method", protected(http.HandlerFunc(handlers.UpdatePaymentMethod(billingService))))

	// Stripe webhook (public, verified by signature)
	mux.HandleFunc("POST /api/webhooks/stripe", handlers.StripeWebhook(billingService))

	// MCP connections (protected)
	mux.Handle("GET /api/mcp/connections", protected(http.HandlerFunc(handlers.ListConnections(db))))
	mux.Handle("POST /api/mcp/connections", protected(http.HandlerFunc(handlers.CreateConnection(db))))
	mux.Handle("DELETE /api/mcp/connections/{service}", protected(http.HandlerFunc(handlers.DeleteConnection(db))))

	// CORS wrapper
	handler := middleware.CORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Human Studio API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			stripe_customer_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			stripe_subscription_id TEXT UNIQUE,
			plan TEXT NOT NULL DEFAULT 'free',
			status TEXT NOT NULL DEFAULT 'active',
			current_period_end TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS billing_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			stripe_invoice_id TEXT,
			amount_cents INTEGER NOT NULL,
			currency TEXT NOT NULL DEFAULT 'usd',
			status TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_connections (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			service TEXT NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			metadata JSONB DEFAULT '{}',
			connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, service)
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// OAuth support migrations
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider TEXT NOT NULL DEFAULT 'email'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS auth_provider_id TEXT`,
		`DO $$ BEGIN ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL; EXCEPTION WHEN others THEN NULL; END $$`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_end TIMESTAMPTZ`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration error: %w\nSQL: %s", err, m[:80])
		}
	}
	return nil
}
