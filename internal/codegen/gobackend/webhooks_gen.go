package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// generateWebhookHandlers produces Go webhook handler code.
func generateWebhookHandlers(moduleName string, app *ir.Application) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

`))

	// Stripe webhook handler
	for _, integ := range app.Integrations {
		if integ.Type == "payment" {
			if _, ok := integ.Config["webhook_endpoint"]; ok {
				b.WriteString(`// StripeWebhook handles Stripe webhook events.
func StripeWebhook() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
			return
		}

		signature := c.GetHeader("Stripe-Signature")
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		_ = signature
		_ = secret

		var event map[string]interface{}
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
			return
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "checkout.session.completed":
			// TODO: handle successful payment
		case "payment_intent.payment_failed":
			// TODO: handle failed payment
		default:
			// Unhandled event type
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	}
}
`)
			}
		}
	}

	return b.String()
}

// hasWebhookIntegration returns true if any integration has a webhook endpoint configured.
func hasWebhookIntegration(app *ir.Application) bool {
	for _, integ := range app.Integrations {
		if integ.Type == "payment" {
			if _, ok := integ.Config["webhook_endpoint"]; ok {
				return true
			}
		}
	}
	return false
}
