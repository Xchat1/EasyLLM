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

	_ = storage.SaveSetting("auth_enabled", "true")

	fmt.Printf("[AUTH] Default password initialized and auth enabled\n")
	return nil
}

// IsAuthEnabled returns whether access password authentication is enabled.
// Preserve password protection for databases created before auth_enabled existed.
func IsAuthEnabled() bool {
	val, ok := storage.GetSetting("auth_enabled")
	if ok {
		return val == "true"
	}
	_, hasPassword := storage.GetSetting("auth_password")
	return hasPassword
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.Login)
	auth.GET("/check", h.Check)
	auth.POST("/setup", h.Setup)
	auth.POST("/enable", h.EnableAuth)
	auth.POST("/disable", h.DisableAuth)
}

func (h *AuthHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/logout", h.Logout)
	auth.POST("/change-password", h.ChangePassword)
}

func (h *AuthHandler) Check(c *gin.Context) {
	_, hasPassword := storage.GetSetting("auth_password")
	c.JSON(http.StatusOK, gin.H{
		"auth_enabled": IsAuthEnabled(),
		"password_set": hasPassword,
	})
}

func (h *AuthHandler) EnableAuth(c *gin.Context) {
	var req struct {
		Password    string `json:"password"`
		OldPassword string `json:"old_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Invalid request payload", Code: "INVALID_REQUEST"})
		return
	}

	stored, hasPassword := storage.GetSetting("auth_password")

	if IsAuthEnabled() {
		if !hasPassword || bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.OldPassword)) != nil {
			c.JSON(http.StatusUnauthorized, models.APIError{Error: "Current password required", Code: "UNAUTHORIZED"})
			return
		}
	}

	// If a password is already configured, any new password requires verifying the current password
	// (or having a valid JWT) to prevent unauthenticated account takeover.
	if hasPassword && strings.TrimSpace(req.Password) != "" {
		isJWTValid := false
		if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			cfg := config.Get()
			secret := ""
			if cfg != nil {
				secret = cfg.App.SecretKey
			}
			if token != "" && verifyJWT(token, secret) == nil {
				isJWTValid = true
			}
		}
		if !isJWTValid && bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.OldPassword)) != nil {
			c.JSON(http.StatusUnauthorized, models.APIError{Error: "Current password required", Code: "UNAUTHORIZED"})
			return
		}
	}

	if strings.TrimSpace(req.Password) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: "密码加密失败", Code: "INTERNAL_ERROR"})
			return
		}
		if err := storage.SaveSetting("auth_password", string(hash)); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: "保存密码失败", Code: "INTERNAL_ERROR"})
			return
		}
	} else if !hasPassword {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "请设置访问密码", Code: "PASSWORD_REQUIRED"})
		return
	}

	if err := storage.SaveSetting("auth_enabled", "true"); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "保存认证状态失败", Code: "INTERNAL_ERROR"})
		return
	}

	token, err := generateJWT(config.Get().App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "生成令牌失败", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"auth_enabled": true,
		"token":        token,
		"message":      "访问密码保护已开启",
	})
}

func (h *AuthHandler) DisableAuth(c *gin.Context) {
	if !IsAuthEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": true, "auth_enabled": false, "message": "访问密码未开启"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&req)

	stored, hasPassword := storage.GetSetting("auth_password")
	if hasPassword {
		if strings.TrimSpace(req.Password) == "" {
			c.JSON(http.StatusBadRequest, models.APIError{Error: "请输入当前密码以关闭密码保护", Code: "PASSWORD_REQUIRED"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, models.APIError{Error: "密码错误，无法关闭密码保护", Code: "UNAUTHORIZED"})
			return
		}
	}

	if err := storage.SaveSetting("auth_enabled", "false"); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "保存认证状态失败", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"auth_enabled": false,
		"message":      "访问密码保护已关闭（免密访问模式）",
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

	_ = storage.SaveSetting("auth_enabled", "true")

	token, err := generateJWT(config.Get().App.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to generate token", Code: "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "token": token, "auth_enabled": true})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if !IsAuthEnabled() {
		token, _ := generateJWT(config.Get().App.SecretKey)
		c.JSON(http.StatusOK, gin.H{"success": true, "token": token, "auth_enabled": false})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"success": true, "token": token, "auth_enabled": true})
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
		if !IsAuthEnabled() {
			c.Next()
			return
		}

		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		const bearerPrefix = "Bearer "
		if len(auth) <= len(bearerPrefix) || !strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIError{Error: "Authentication required", Code: "UNAUTHORIZED"})
			return
		}

		token := strings.TrimSpace(auth[len(bearerPrefix):])
		if token == "" || verifyJWT(token, config.Get().App.SecretKey) != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIError{Error: "Invalid or expired token", Code: "UNAUTHORIZED"})
			return
		}

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
func verifyJWT(tokenStr, secret string) error {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid token format")
	}
	if secret == "" {
		return fmt.Errorf("missing signing secret")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid header encoding")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Alg != "HS256" || header.Typ != "JWT" {
		return fmt.Errorf("invalid header")
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

	if payload.Iss != "easyllm" {
		return fmt.Errorf("invalid issuer")
	}
	if payload.Exp <= 0 || time.Now().Unix() >= payload.Exp {
		return fmt.Errorf("token expired")
	}

	return nil
}
