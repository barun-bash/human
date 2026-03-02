package python

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func TestGenerateIntegrationsEmpty(t *testing.T) {
	app := &ir.Application{}
	files := generateIntegrations(app)
	if files != nil {
		t.Errorf("expected nil for empty integrations, got %d files", len(files))
	}
}

func TestGenerateEmailServicePython(t *testing.T) {
	integ := &ir.Integration{
		Service:     "SendGrid",
		Type:        "email",
		Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
		Config:      map[string]string{"sender_email": "hello@example.com"},
	}

	content := generateEmailService(integ)

	checks := []string{"sendgrid", "send_email", "SENDGRID_API_KEY"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Python email service missing %q", check)
		}
	}
}

func TestGenerateStorageServicePython(t *testing.T) {
	integ := &ir.Integration{
		Service: "AWS S3",
		Type:    "storage",
	}

	content := generateStorageService(integ)

	checks := []string{"boto3", "upload_file", "s3_client"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Python storage service missing %q", check)
		}
	}
}

func TestGeneratePaymentServicePython(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Stripe",
		Type:        "payment",
		Credentials: map[string]string{"api key": "STRIPE_SECRET_KEY"},
	}

	content := generatePaymentService(integ)

	checks := []string{"stripe", "create_checkout_session", "STRIPE_SECRET_KEY"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Python payment service missing %q", check)
		}
	}
}

func TestGenerateMessagingServicePython(t *testing.T) {
	integ := &ir.Integration{
		Service: "Slack",
		Type:    "messaging",
	}

	content := generateMessagingService(integ)

	checks := []string{"slack", "send_slack_message", "webhook"}
	for _, check := range checks {
		if !strings.Contains(strings.ToLower(content), check) {
			t.Errorf("Python messaging service missing %q", check)
		}
	}
}

func TestRequirementsWithIntegrations(t *testing.T) {
	app := &ir.Application{
		Integrations: []*ir.Integration{
			{Service: "SendGrid", Type: "email"},
			{Service: "Stripe", Type: "payment"},
			{Service: "Slack", Type: "messaging"},
		},
	}

	output := generateRequirements(app)

	checks := []string{"sendgrid", "stripe", "slack-sdk"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("requirements.txt should contain %q", check)
		}
	}
}
