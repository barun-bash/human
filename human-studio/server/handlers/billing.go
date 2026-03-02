package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/barun-bash/human/human-studio/server/billing"
	"github.com/barun-bash/human/human-studio/server/middleware"
)

func GetSubscription(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		sub, err := svc.GetSubscription(userID)
		if err != nil {
			jsonError(w, "failed to get subscription", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, sub, http.StatusOK)
	}
}

func CreateCheckout(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		var body struct {
			PriceID string `json:"price_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		url, err := svc.CreateCheckoutSession(userID, body.PriceID)
		if err != nil {
			jsonError(w, "failed to create checkout session", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]string{"url": url}, http.StatusOK)
	}
}

func GetBillingHistory(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		history, err := svc.GetBillingHistory(userID)
		if err != nil {
			jsonError(w, "failed to get billing history", http.StatusInternalServerError)
			return
		}
		jsonResponse(w, history, http.StatusOK)
	}
}

func SelectPlan(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		var body struct {
			Plan string `json:"plan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if body.Plan == "" {
			jsonError(w, "plan is required", http.StatusBadRequest)
			return
		}

		sub, err := svc.SelectPlan(userID, body.Plan)
		if err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(w, sub, http.StatusOK)
	}
}

func UpdatePaymentMethod(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Update payment method via Stripe
		jsonResponse(w, map[string]string{"message": "payment method updated"}, http.StatusOK)
	}
}

func StripeWebhook(svc *billing.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(io.LimitReader(r.Body, 65536))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		signature := r.Header.Get("Stripe-Signature")
		if err := svc.HandleWebhook(payload, signature); err != nil {
			http.Error(w, "webhook processing failed", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
