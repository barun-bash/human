package node

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

func TestGenerateEmailService(t *testing.T) {
	integ := &ir.Integration{
		Service:     "SendGrid",
		Type:        "email",
		Credentials: map[string]string{"api key": "SENDGRID_API_KEY"},
		Config:      map[string]string{"sender_email": "hello@example.com"},
		Templates:   []string{"welcome", "password-reset"},
		Purpose:     "sending transactional emails",
	}

	content := generateEmailService(integ)

	checks := []string{
		`@sendgrid/mail`,
		`process.env.SENDGRID_API_KEY`,
		`from: "hello@example.com"`,
		`export async function sendEmail`,
		`SendEmailOptions`,
		`EMAIL_TEMPLATES`,
		`WELCOME: "welcome"`,
		`PASSWORD_RESET: "password-reset"`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("email service missing %q", check)
		}
	}
}

func TestGenerateStorageService(t *testing.T) {
	integ := &ir.Integration{
		Service:     "AWS S3",
		Type:        "storage",
		Credentials: map[string]string{"api key": "AWS_ACCESS_KEY", "secret": "AWS_SECRET_KEY"},
		Config:      map[string]string{"region": "eu-west-1", "bucket": "my-uploads"},
	}

	content := generateStorageService(integ)

	checks := []string{
		`@aws-sdk/client-s3`,
		`@aws-sdk/s3-request-presigner`,
		`process.env.AWS_ACCESS_KEY`,
		`process.env.AWS_SECRET_KEY`,
		`"eu-west-1"`,
		`"my-uploads"`,
		`export async function uploadFile`,
		`export async function getSignedDownloadUrl`,
		`export async function deleteFile`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("storage service missing %q", check)
		}
	}
}

func TestGeneratePaymentService(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Stripe",
		Type:        "payment",
		Credentials: map[string]string{"api key": "STRIPE_SECRET_KEY"},
		Config:      map[string]string{"webhook_endpoint": "/webhooks/stripe"},
	}

	content := generatePaymentService(integ)

	checks := []string{
		`import Stripe from "stripe"`,
		`process.env.STRIPE_SECRET_KEY`,
		`createCheckoutSession`,
		`createCustomer`,
		`verifyWebhookSignature`,
		`WEBHOOK_ENDPOINT`,
		`"/webhooks/stripe"`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("payment service missing %q", check)
		}
	}
}

func TestGeneratePaymentServiceNoWebhook(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Stripe",
		Type:        "payment",
		Credentials: map[string]string{"api key": "STRIPE_SECRET_KEY"},
	}

	content := generatePaymentService(integ)

	if strings.Contains(content, "verifyWebhookSignature") {
		t.Error("should not generate webhook verification without webhook endpoint")
	}
}

func TestGenerateMessagingService(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Slack",
		Type:        "messaging",
		Credentials: map[string]string{"api key": "SLACK_WEBHOOK_URL"},
		Config:      map[string]string{"channel": "#engineering"},
	}

	content := generateMessagingService(integ)

	checks := []string{
		`@slack/webhook`,
		`process.env.SLACK_WEBHOOK_URL`,
		`export async function sendSlackMessage`,
		`export async function sendAlert`,
		`"#engineering"`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("messaging service missing %q", check)
		}
	}
}

func TestGenerateOAuthServiceGoogle(t *testing.T) {
	integ := &ir.Integration{
		Service:     "Google",
		Type:        "oauth",
		Credentials: map[string]string{"client id": "GOOGLE_CLIENT_ID", "client secret": "GOOGLE_CLIENT_SECRET"},
	}

	content := generateOAuthService(integ)

	checks := []string{
		`passport-google-oauth20`,
		`process.env.GOOGLE_CLIENT_ID`,
		`process.env.GOOGLE_CLIENT_SECRET`,
		`configureGoogleAuth`,
		`OAuthProfile`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("oauth service missing %q", check)
		}
	}
}

func TestGenerateOAuthServiceGitHub(t *testing.T) {
	integ := &ir.Integration{
		Service:     "GitHub",
		Type:        "oauth",
		Credentials: map[string]string{"client id": "GH_CLIENT_ID", "client secret": "GH_CLIENT_SECRET"},
	}

	content := generateOAuthService(integ)

	if !strings.Contains(content, `passport-github2`) {
		t.Error("github oauth should use passport-github2")
	}
	if !strings.Contains(content, `GitHubStrategy`) {
		t.Error("github oauth should use GitHubStrategy")
	}
}

func TestGenerateGenericService(t *testing.T) {
	integ := &ir.Integration{
		Service:     "CustomAPI",
		Credentials: map[string]string{"api key": "CUSTOM_KEY"},
		Purpose:     "custom integration",
	}

	content := generateGenericService(integ)

	if !strings.Contains(content, `"CustomAPI"`) {
		t.Error("generic service should include service name")
	}
	if !strings.Contains(content, `process.env.CUSTOM_KEY`) {
		t.Error("generic service should include credential env var")
	}
}

func TestGenerateIntegrationsBarrelExport(t *testing.T) {
	app := &ir.Application{
		Integrations: []*ir.Integration{
			{Service: "SendGrid", Type: "email", Credentials: map[string]string{"api key": "SG_KEY"}},
			{Service: "Slack", Type: "messaging", Credentials: map[string]string{"api key": "SLACK_URL"}},
		},
	}

	files := generateIntegrations(app)

	indexContent, ok := files["src/services/index.ts"]
	if !ok {
		t.Fatal("expected src/services/index.ts barrel export")
	}
	if !strings.Contains(indexContent, `export * from`) {
		t.Error("index.ts should contain barrel exports")
	}
}
