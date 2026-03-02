package models

import "time"

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Name             string    `json:"name"`
	PasswordHash     string    `json:"-"`
	AuthProvider     string    `json:"auth_provider"`
	AuthProviderID   *string   `json:"-"`
	StripeCustomerID *string   `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Subscription struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	StripeSubscriptionID *string    `json:"-"`
	Plan                 string     `json:"plan"`
	Status               string     `json:"status"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	TrialEnd             *time.Time `json:"trial_end,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type BillingRecord struct {
	ID              string    `json:"id"`
	UserID          string    `json:"-"`
	StripeInvoiceID *string   `json:"-"`
	AmountCents     int       `json:"amount_cents"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	Description     *string   `json:"description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type MCPConnection struct {
	ID           string    `json:"id"`
	UserID       string    `json:"-"`
	Service      string    `json:"service"`
	ConnectedAt  time.Time `json:"connected_at"`
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type ProfileUpdate struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type PasswordChange struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type OAuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
	IsNewUser    bool   `json:"is_new_user"`
}

type SelectPlanRequest struct {
	Plan string `json:"plan"`
}
