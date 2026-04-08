package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = func() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("deliverydesk-secret-key-2024")
}()

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	AuthType string `json:"auth_type"` // local or ldap (empty defaults to local)
}

type LoginResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

type JWTPayload struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
}

func Login(req LoginRequest) (*LoginResponse, error) {
	// Determine auth type
	authType := req.AuthType
	if authType == "" {
		authType = "local"
	}

	if authType == "ldap" {
		return loginLDAP(req)
	}

	return loginLocal(req)
}

func loginLocal(req LoginRequest) (*LoginResponse, error) {
	var user model.User
	result := repository.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		logger.Log.Warnf("Login failed: user '%s' not found in database", req.Username)
		return nil, errors.New("用户名或密码错误")
	}

	// Check if password field is empty (broken user record)
	if user.Password == "" {
		logger.Log.Warnf("Login failed: user '%s' has empty password hash", req.Username)
		return nil, errors.New("用户名或密码错误，请联系管理员重置密码")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Log.Warnf("Login failed: password mismatch for user '%s' (hash len=%d)", req.Username, len(user.Password))
		return nil, errors.New("用户名或密码错误")
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	logger.Log.Infof("User '%s' logged in successfully", req.Username)
	return &LoginResponse{Token: token, User: user}, nil
}

func loginLDAP(req LoginRequest) (*LoginResponse, error) {
	// Find the default enabled LDAP configuration
	var ldapCfg model.LDAPConfig
	if err := repository.DB.Where("is_enabled = ? AND is_default = ?", true, true).First(&ldapCfg).Error; err != nil {
		// Try any enabled LDAP config
		if err := repository.DB.Where("is_enabled = ?", true).First(&ldapCfg).Error; err != nil {
			return nil, errors.New("LDAP is not configured. Please contact administrator.")
		}
	}

	// For this implementation, we simulate LDAP authentication
	// In production, you would use an LDAP library like go-ldap
	// to bind and authenticate against the LDAP server
	// For now, we check if user exists locally with ldap auth_type
	var user model.User
	result := repository.DB.Where("username = ? AND auth_type = ?", req.Username, "ldap").First(&user)
	if result.Error != nil {
		// Create LDAP user on first login (auto-provisioning)
		user = model.User{
			Username:    req.Username,
			Password:    "", // LDAP users don't have local passwords
			Email:       req.Username + "@" + extractDomain(ldapCfg.BaseDN),
			DisplayName: req.Username,
			Role:        "user",
			AuthType:    "ldap",
		}
		if err := repository.DB.Create(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to create LDAP user: %w", err)
		}
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	return &LoginResponse{Token: token, User: user}, nil
}

func extractDomain(baseDN string) string {
	parts := strings.Split(baseDN, ",")
	var domain []string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(strings.ToLower(kv[0])) == "dc" {
			domain = append(domain, strings.TrimSpace(kv[1]))
		}
	}
	if len(domain) > 0 {
		return strings.Join(domain, ".")
	}
	return "example.com"
}

func generateToken(user model.User) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := JWTPayload{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		Exp:      time.Now().Add(24 * time.Hour).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := header + "." + payloadB64
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func ValidateToken(tokenStr string) (*JWTPayload, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errors.New("invalid token signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	var payload JWTPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, errors.New("invalid token payload")
	}

	if time.Now().Unix() > payload.Exp {
		return nil, errors.New("token expired")
	}

	return &payload, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}
