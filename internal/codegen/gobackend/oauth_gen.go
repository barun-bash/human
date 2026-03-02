package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// generateOAuthHandlers produces Go OAuth handler code.
func generateOAuthHandlers(moduleName string, app *ir.Application) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
`))

	// Import provider-specific endpoints
	for _, integ := range app.Integrations {
		if integ.Type == "oauth" {
			provider := strings.ToLower(integ.Service)
			if strings.Contains(provider, "google") {
				b.WriteString("\tgoogle \"golang.org/x/oauth2/google\"\n")
			} else if strings.Contains(provider, "github") {
				b.WriteString("\tgithub \"golang.org/x/oauth2/github\"\n")
			}
		}
	}

	b.WriteString(fmt.Sprintf(`
	"%s/middleware"
)

`, moduleName))

	for _, integ := range app.Integrations {
		if integ.Type == "oauth" {
			provider := strings.ToLower(integ.Service)
			providerPascal := toPascalCase(integ.Service)

			// Determine env var names
			clientIDEnv := strings.ToUpper(strings.ReplaceAll(integ.Service, " ", "_")) + "_CLIENT_ID"
			clientSecretEnv := strings.ToUpper(strings.ReplaceAll(integ.Service, " ", "_")) + "_CLIENT_SECRET"
			for key, envVar := range integ.Credentials {
				lower := strings.ToLower(key)
				if strings.Contains(lower, "secret") {
					clientSecretEnv = envVar
				} else if strings.Contains(lower, "id") || strings.Contains(lower, "client") {
					clientIDEnv = envVar
				}
			}

			// OAuth config
			fmt.Fprintf(&b, "var %sOAuthConfig = &oauth2.Config{\n", toCamelCase(integ.Service))
			fmt.Fprintf(&b, "\tClientID:     os.Getenv(\"%s\"),\n", clientIDEnv)
			fmt.Fprintf(&b, "\tClientSecret: os.Getenv(\"%s\"),\n", clientSecretEnv)

			callbackURL := "/auth/" + provider + "/callback"
			if v, ok := integ.Config["callback_url"]; ok {
				callbackURL = v
			}
			fmt.Fprintf(&b, "\tRedirectURL:  os.Getenv(\"BASE_URL\") + \"%s\",\n", callbackURL)

			if strings.Contains(provider, "google") {
				b.WriteString("\tEndpoint:     google.Endpoint,\n")
				b.WriteString("\tScopes:       []string{\"email\", \"profile\"},\n")
			} else if strings.Contains(provider, "github") {
				b.WriteString("\tEndpoint:     github.Endpoint,\n")
				b.WriteString("\tScopes:       []string{\"user:email\"},\n")
			} else {
				b.WriteString("\t// TODO: configure OAuth endpoint\n")
			}
			b.WriteString("}\n\n")

			// Login handler
			fmt.Fprintf(&b, "// %sLogin redirects to the OAuth provider.\n", providerPascal)
			fmt.Fprintf(&b, "func %sLogin() gin.HandlerFunc {\n", providerPascal)
			b.WriteString("\treturn func(c *gin.Context) {\n")
			fmt.Fprintf(&b, "\t\turl := %sOAuthConfig.AuthCodeURL(\"state\")\n", toCamelCase(integ.Service))
			b.WriteString("\t\tc.Redirect(http.StatusTemporaryRedirect, url)\n")
			b.WriteString("\t}\n")
			b.WriteString("}\n\n")

			// Callback handler
			fmt.Fprintf(&b, "// %sCallback handles the OAuth callback.\n", providerPascal)
			fmt.Fprintf(&b, "func %sCallback(cfg *middleware.Config) gin.HandlerFunc {\n", providerPascal)
			b.WriteString("\treturn func(c *gin.Context) {\n")
			b.WriteString("\t\tcode := c.Query(\"code\")\n")
			fmt.Fprintf(&b, "\t\ttoken, err := %sOAuthConfig.Exchange(c, code)\n", toCamelCase(integ.Service))
			b.WriteString("\t\tif err != nil {\n")
			b.WriteString("\t\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": \"OAuth exchange failed\"})\n")
			b.WriteString("\t\t\treturn\n")
			b.WriteString("\t\t}\n\n")
			b.WriteString("\t\t// TODO: fetch user profile and create/find user in database\n")
			b.WriteString("\t\t_ = token\n\n")
			b.WriteString("\t\tc.Redirect(http.StatusTemporaryRedirect, \"/\")\n")
			b.WriteString("\t}\n")
			b.WriteString("}\n\n")
		}
	}

	return b.String()
}

// hasOAuthIntegration returns true if any OAuth integration exists.
func hasOAuthIntegration(app *ir.Application) bool {
	for _, integ := range app.Integrations {
		if integ.Type == "oauth" {
			return true
		}
	}
	return false
}
