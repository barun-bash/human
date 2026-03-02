package gobackend

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateIntegrationsEmpty(t *testing.T) {
	app := &ir.Application{}
	files := generateIntegrations("testapp", app)
	if files != nil {
		t.Errorf("expected nil for empty integrations, got %d files", len(files))
	}
}

func TestGenerateEmailServiceGo(t *testing.T) {
	integ := &ir.Integration{
		Service:     "SendGrid",
		Type:        "email",
		Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
	}

	content := generateEmailService("testapp", integ)

	checks := []string{"package services", "SendEmail", "sendgrid", "SENDGRID_API_KEY"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Go email service missing %q", check)
		}
	}
}

func TestGeneratePaymentServiceGo(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Stripe",
		Type:        "payment",
		Credentials: map[string]string{"api key": "STRIPE_SECRET_KEY"},
	}

	content := generatePaymentService("testapp", integ)

	checks := []string{"package services", "CreateCheckoutSession", "stripe", "STRIPE_SECRET_KEY"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Go payment service missing %q", check)
		}
	}
}

func TestGenerateMessagingServiceGo(t *testing.T) {
	integ := &ir.Integration{
		Service: "Slack",
		Type:    "messaging",
	}

	content := generateMessagingService("testapp", integ)

	checks := []string{"package services", "SendSlackMessage", "slack"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Go messaging service missing %q", check)
		}
	}
}

func TestGoModWithIntegrations(t *testing.T) {
	app := &ir.Application{
		Integrations: []*ir.Integration{
			{Service: "SendGrid", Type: "email"},
			{Service: "Stripe", Type: "payment"},
			{Service: "Slack", Type: "messaging"},
		},
	}

	output := generateGoMod("testapp", app)

	checks := []string{"sendgrid", "stripe", "slack"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("go.mod should contain %q", check)
		}
	}
}

func TestWebhookHandlerGenerated(t *testing.T) {
	app := &ir.Application{
		Integrations: []*ir.Integration{
			{Service: "Stripe", Type: "payment",
				Config: map[string]string{"webhook_endpoint": "/webhooks/stripe"},
			},
		},
	}

	if !hasWebhookIntegration(app) {
		t.Error("should detect webhook integration")
	}

	output := generateWebhookHandlers("testapp", app)
	if !strings.Contains(output, "StripeWebhook") {
		t.Error("should generate StripeWebhook handler")
	}
}

func TestOAuthHandlerGenerated(t *testing.T) {
	app := &ir.Application{
		Integrations: []*ir.Integration{
			{Service: "Google", Type: "oauth",
				Credentials: map[string]string{"client id": "GOOGLE_CLIENT_ID", "secret": "GOOGLE_CLIENT_SECRET"},
			},
		},
	}

	if !hasOAuthIntegration(app) {
		t.Error("should detect OAuth integration")
	}

	output := generateOAuthHandlers("testapp", app)
	if !strings.Contains(output, "GoogleLogin") {
		t.Error("should generate GoogleLogin handler")
	}
	if !strings.Contains(output, "GoogleCallback") {
		t.Error("should generate GoogleCallback handler")
	}
}
