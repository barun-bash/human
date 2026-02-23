package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"timekeeper/config"
	"timekeeper/handlers"
)

func Setup(r *gin.Engine, db *gorm.DB) {
	cfg := config.Load()
	api := r.Group("/api")

	api.POST("/account", handlers.CreateAccount(db, cfg))
	api.POST("/login", handlers.Login(db, cfg))
	api.POST("/google-login", handlers.GoogleLogin(db, cfg))
	api.POST("/slack-login", handlers.SlackLogin(db, cfg))
	api.POST("/send-reset-code", handlers.SendResetCode(db, cfg))
	api.POST("/verify-reset-code", handlers.VerifyResetCode(db, cfg))
	api.POST("/reset-user-password", handlers.ResetUserPassword(db, cfg))
}
