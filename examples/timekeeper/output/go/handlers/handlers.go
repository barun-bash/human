package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"timekeeper/config"
	"timekeeper/dto"
	"timekeeper/middleware"
	"timekeeper/models"
)

func CreateAccount(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		// create a User with the given fields
		newItem := models.User{
			Name: req.Name,
			Email: req.Email,
			Password: req.Password,
		}
		if err := db.Create(&newItem).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create"})
			return
		}
		// send welcome email to the user
		// respond with the created user and auth token
		c.JSON(http.StatusCreated, gin.H{"data": newItem})
	}
}

func Login(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}
		if req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
			return
		}
		// fetch the user by email
		var item models.User
		if err := db.Where("email = ?", req.Email).First(&item).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		// if user does not exist, respond with invalid credentials
		// if password does not match, respond with invalid credentials
		if !middleware.CheckPasswordHash(req.Password, item.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		// respond with the user and auth token
		token, err := middleware.GenerateToken(item.ID, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": item, "token": token})
	}
}

func GoogleLogin(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.GoogleLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate the google_token with Google OAuth
		// TODO: extract email from validated Google token
		email := "" // placeholder — replace with actual Google token validation
		_ = req
		// fetch the user by email from the token
		var item models.User
		if err := db.Where("email = ?", email).First(&item).Error; err != nil {
			// if user does not exist, create a User from the token
			newUser := models.User{Email: email}
			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
			item = newUser
		}
		// respond with the user and auth token
		token, err := middleware.GenerateToken(item.ID, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": item, "token": token})
	}
}

func SlackLogin(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.SlackLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate the slack_token with Slack OAuth
		// TODO: extract email from validated Slack token
		email := "" // placeholder — replace with actual Slack token validation
		_ = req
		// fetch the user by email from the token
		var item models.User
		if err := db.Where("email = ?", email).First(&item).Error; err != nil {
			// if user does not exist, create a User from the token
			newUser := models.User{Email: email}
			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
				return
			}
			item = newUser
		}
		// respond with the user and auth token
		token, err := middleware.GenerateToken(item.ID, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": item, "token": token})
	}
}

func SendResetCode(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.SendResetCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}
		// fetch the user by email
		var item models.User
		if err := db.Where("email = ?", req.Email).First(&item).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		// if user does not exist, respond with if an account exists, a code has been sent
		// generate a 6 digit reset code
		// set reset_code and reset_code_expires on the user
		// send the reset code to the user's email
		// respond with if an account exists, a code has been sent
		c.JSON(http.StatusOK, gin.H{"data": item})
	}
}

func VerifyResetCode(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.VerifyResetCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}
		// fetch the user by email
		var item models.User
		if err := db.Where("email = ?", req.Email).First(&item).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		// if user does not exist, respond with invalid code
		// check that reset_code_expires is not in the past
		// if code does not match, respond with invalid code
		// respond with a password reset token
		c.JSON(http.StatusOK, gin.H{"data": item})
	}
}

func ResetUserPassword(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResetUserPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate the reset_token
		// fetch the user from the reset_token
		var item models.User
		if err := db.Where("id = ?", req.ResetToken).First(&item).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		// update the user's password
		// clear reset_code and reset_code_expires
		// respond with password reset successfully
		c.JSON(http.StatusOK, gin.H{"data": item})
	}
}

