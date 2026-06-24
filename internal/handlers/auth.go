package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"easyllm/config"
	"easyllm/internal/models"
	"easyllm/internal/storage"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// InitializeDefaultPassword sets up the default password on first startup if configured
func (h *AuthHandler) InitializeDefaultPassword() error {
	defaultPwd := config.Get().App.DefaultPassword
	if defaultPwd == "" {
		return nil // No default password configured
	}

	// Check if password is already set
	_, hasPassword := storage.GetSetting("auth_password")
	if hasPassword {
		return nil // Password already set, skip
	}

	// Hash and save the default password
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	if err := storage.SaveSetting("auth_password", string(hash)); err != nil {
		return fmt.Errorf("failed to save default password: %w", err)
	}

	fmt.Printf("[AUTH] Default password initialized\n")
	return nil
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.Login)
	auth.GET("/check", h.Check)
	auth.POST("/setup", h.Setup)
}

func (h *AuthHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/logout", h.Logout)
	auth.POST("/change-password", h.ChangePassword)
}

func (h *AuthHandler) Check(c *gin.Context) {
	_, hasPassword := storage.GetSetting("auth_password")
	c.JSON(http.StatusOK, gin.H{
		"password_set": hasPassword,
	})
}

func (h *AuthHandler) Setup(c *gin.Context) {
	if _, hasPassword := storage.GetSetting("auth_password"); hasPassword {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Password already set, use change-password instead", Code: "ALREADY_SET"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Password is required", Code: "INVALID_REQUEST"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to hash password", Code: "INTERNAL_ERROR"})
		return
	}

	if err := storage.SaveSetting("auth_password", string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to save password", Code: "INTERNAL_ERROR"})
		return
	}

	token, err := generateJWT(config.Get().App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to generate token", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "token": token})
}

func (h *AuthHandler) Login(c *gin.Context) {
	stored, ok := storage.GetSetting("auth_password")
	if !ok {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "No password set, use setup first", Code: "NO_PASSWORD"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Password is required", Code: "INVALID_REQUEST"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{Error: "Invalid password", Code: "UNAUTHORIZED"})
		return
	}

	token, err := generateJWT(config.Get().App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to generate token", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "token": token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Old password and new password are required", Code: "INVALID_REQUEST"})
		return
	}

	stored, ok := storage.GetSetting("auth_password")
	if !ok {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "No password set", Code: "NO_PASSWORD"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{Error: "Old password is incorrect", Code: "UNAUTHORIZED"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to hash password", Code: "INTERNAL_ERROR"})
		return
	}

	if err := storage.SaveSetting("auth_password", string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to save password", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// Simple JWT implementation using HMAC-SHA256 (no external dependency needed)

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

func generateJWT(secret string) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	payload := jwtPayload{
		Iss: "easyllm",
		Iat: time.Now().Unix(),
		Exp: time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt header: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// verifyJWT validates a JWT token signed with HMAC-SHA256.
// Kept for future re-enabling of AuthMiddleware; currently auth is bypassed.
func verifyJWT(tokenStr, secret string) error {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return fmt.Errorf("invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid payload encoding")
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("invalid payload")
	}

	if time.Now().Unix() > payload.Exp {
		return fmt.Errorf("token expired")
	}

	return nil
}
