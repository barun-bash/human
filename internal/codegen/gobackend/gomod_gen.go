package gobackend

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generateGoMod(moduleName string, app *ir.Application) string {
	var deps strings.Builder
	deps.WriteString(fmt.Sprintf(`module %s

go 1.23

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	golang.org/x/crypto v0.31.0
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
`, moduleName))

	if app != nil {
		for _, integ := range app.Integrations {
			switch integ.Type {
			case "email":
				deps.WriteString("\tgithub.com/sendgrid/sendgrid-go v3.14.0\n")
			case "storage":
				deps.WriteString("\tgithub.com/aws/aws-sdk-go-v2 v1.30.0\n")
				deps.WriteString("\tgithub.com/aws/aws-sdk-go-v2/config v1.27.0\n")
				deps.WriteString("\tgithub.com/aws/aws-sdk-go-v2/service/s3 v1.58.0\n")
			case "payment":
				deps.WriteString("\tgithub.com/stripe/stripe-go/v81 v81.0.0\n")
			case "messaging":
				deps.WriteString("\tgithub.com/slack-go/slack v0.13.0\n")
			case "oauth":
				deps.WriteString("\tgolang.org/x/oauth2 v0.21.0\n")
			}
		}
	}

	deps.WriteString(")\n")
	return deps.String()
}

func generateMain(moduleName string, app *ir.Application) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"%s/config"
	"%s/database"
	"%s/routes"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %%v", err)
	}

	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	routes.Setup(r, db)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %%s\n", err)
		}
	}()

	log.Printf("Server running on port %%s", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
`, moduleName, moduleName, moduleName)
}

func generateConfig(moduleName string) string {
	return `package config

import "os"

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "supersecretkey"
	}

	return &Config{
		DatabaseURL: dbUrl,
		JWTSecret:   jwtSecret,
		Port:        port,
	}
}
`
}
