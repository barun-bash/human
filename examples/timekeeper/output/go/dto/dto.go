package dto

type CreateAccountRequest struct {
	Name string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

type LoginRequest struct {
	Email string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type GoogleLoginRequest struct {
	GoogleToken string `json:"googleToken" binding:"required"`
}

type SlackLoginRequest struct {
	SlackToken string `json:"slackToken" binding:"required"`
}

type SendResetCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type VerifyResetCodeRequest struct {
	Email string `json:"email" binding:"required"`
	Code string `json:"code" binding:"required"`
}

type ResetUserPasswordRequest struct {
	ResetToken string `json:"resetToken" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

