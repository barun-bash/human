package billing

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/barun-bash/human/human-studio/server/models"
)

// Service manages Stripe billing integration.
// In production, this would use the Stripe Go SDK.
type Service struct {
	db        *sql.DB
	secretKey string
}

func NewService(db *sql.DB, secretKey string) *Service {
	return &Service{db: db, secretKey: secretKey}
}

// GetSubscription retrieves the user's current subscription.
func (s *Service) GetSubscription(userID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.QueryRow(
		`SELECT id, user_id, plan, status, current_period_end, trial_end, created_at
		 FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return &models.Subscription{
			UserID: userID,
			Plan:   "free",
			Status: "active",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying subscription: %w", err)
	}
	return &sub, nil
}

// SelectPlan sets the user's subscription plan. For "pro", starts a 14-day trial.
func (s *Service) SelectPlan(userID string, plan string) (*models.Subscription, error) {
	if plan != "free" && plan != "pro" {
		return nil, fmt.Errorf("invalid plan: %s", plan)
	}

	// Check if user already has a subscription
	var existingID string
	err := s.db.QueryRow("SELECT id FROM subscriptions WHERE user_id = $1", userID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("checking subscription: %w", err)
	}

	var sub models.Subscription
	if plan == "pro" {
		trialEnd := time.Now().Add(14 * 24 * time.Hour)
		if existingID != "" {
			err = s.db.QueryRow(
				`UPDATE subscriptions SET plan = 'pro', status = 'trialing', trial_end = $1
				 WHERE id = $2
				 RETURNING id, user_id, plan, status, current_period_end, trial_end, created_at`,
				trialEnd, existingID,
			).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CreatedAt)
		} else {
			err = s.db.QueryRow(
				`INSERT INTO subscriptions (user_id, plan, status, trial_end)
				 VALUES ($1, 'pro', 'trialing', $2)
				 RETURNING id, user_id, plan, status, current_period_end, trial_end, created_at`,
				userID, trialEnd,
			).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CreatedAt)
		}
	} else {
		if existingID != "" {
			err = s.db.QueryRow(
				`UPDATE subscriptions SET plan = 'free', status = 'active', trial_end = NULL
				 WHERE id = $1
				 RETURNING id, user_id, plan, status, current_period_end, trial_end, created_at`,
				existingID,
			).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CreatedAt)
		} else {
			err = s.db.QueryRow(
				`INSERT INTO subscriptions (user_id, plan, status)
				 VALUES ($1, 'free', 'active')
				 RETURNING id, user_id, plan, status, current_period_end, trial_end, created_at`,
				userID,
			).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd, &sub.CreatedAt)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("setting plan: %w", err)
	}

	return &sub, nil
}

// CreateCheckoutSession creates a Stripe Checkout session for upgrading.
func (s *Service) CreateCheckoutSession(userID, priceID string) (string, error) {
	// TODO: Use stripe-go to create a checkout session
	// Returns the checkout URL
	return "https://checkout.stripe.com/placeholder", nil
}

// GetBillingHistory returns recent invoices for the user.
func (s *Service) GetBillingHistory(userID string) ([]models.BillingRecord, error) {
	// TODO: Query from database, synced via Stripe webhooks
	return []models.BillingRecord{}, nil
}

// HandleWebhook processes incoming Stripe webhook events.
func (s *Service) HandleWebhook(payload []byte, signature string) error {
	// TODO: Verify signature with Stripe webhook secret
	// TODO: Handle events: checkout.session.completed, invoice.paid,
	//       customer.subscription.updated, customer.subscription.deleted
	return nil
}
