package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
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
		logger.Log.Warnf("Login failed: user '%s' not found in database (error: %v)", req.Username, result.Error)
		return nil, errors.New("用户名或密码错误")
	}

	logger.Log.Infof("Login attempt: user '%s' found (id=%d, auth_type=%s, password_hash_len=%d)",
		req.Username, user.ID, user.AuthType, len(user.Password))

	// Check if password field is empty (broken user record)
	if user.Password == "" {
		logger.Log.Warnf("Login failed: user '%s' has empty password hash in database", req.Username)
		return nil, errors.New("用户密码未设置，请联系管理员重置密码")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Log.Warnf("Login failed: bcrypt mismatch for user '%s' (hash_len=%d, input_len=%d, err=%v)",
			req.Username, len(user.Password), len(req.Password), err)
		return nil, errors.New("用户名或密码错误")
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	logger.Log.Infof("User '%s' (id=%d) logged in successfully", req.Username, user.ID)
	return &LoginResponse{Token: token, User: user}, nil
}

func loginLDAP(req LoginRequest) (*LoginResponse, error) {
	// Find the default enabled LDAP configuration
	var ldapCfg model.LDAPConfig
	if err := repository.DB.Where("is_enabled = ? AND is_default = ?", true, true).First(&ldapCfg).Error; err != nil {
		// Try any enabled LDAP config
		if err := repository.DB.Where("is_enabled = ?", true).First(&ldapCfg).Error; err != nil {
			return nil, errors.New("LDAP 未配置，请联系管理员")
		}
	}

	// SECURITY: reject empty password — many LDAP servers allow anonymous bind
	// with an empty password, which would bypass authentication entirely.
	if req.Password == "" {
		return nil, errors.New("用户名或密码错误")
	}

	// Check if user exists locally with ldap auth_type (must be synced by admin first)
	var user model.User
	result := repository.DB.Where("username = ? AND auth_type = ?", req.Username, "ldap").First(&user)
	if result.Error != nil {
		return nil, errors.New("该LDAP用户尚未同步到平台，请联系管理员在用户管理中同步LDAP用户")
	}

	// ── Real LDAP Bind authentication ────────────────────────────────
	// Step 1: Connect to the LDAP server
	addr := fmt.Sprintf("%s:%d", ldapCfg.Host, ldapCfg.Port)
	var conn *ldap.Conn
	var err error
	if ldapCfg.UseTLS {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: true, // enterprise internal LDAP often uses self-signed certs
		})
	} else {
		conn, err = ldap.Dial("tcp", addr)
	}
	if err != nil {
		logger.Log.Errorf("LDAP login: failed to connect to %s: %v", addr, err)
		return nil, errors.New("LDAP 服务器连接失败，请稍后重试")
	}
	defer conn.Close()

	// Step 2: Bind with service account to search for the user's DN
	// Retrieve bind password (it is excluded from JSON serialization)
	var fullCfg model.LDAPConfig
	repository.DB.First(&fullCfg, ldapCfg.ID)

	if fullCfg.BindDN != "" && fullCfg.BindPassword != "" {
		if err := conn.Bind(fullCfg.BindDN, fullCfg.BindPassword); err != nil {
			logger.Log.Errorf("LDAP login: service account bind failed: %v", err)
			return nil, errors.New("LDAP 服务账号认证失败，请联系管理员检查LDAP配置")
		}
	}

	// Step 3: Search for the user's DN by username attribute
	attrUsername := ldapCfg.AttrUsername
	if attrUsername == "" {
		attrUsername = "uid"
	}

	// Determine search bases: support multiple OUs separated by | character
	// (consistent with the SyncLDAPUsers logic in handlers.go)
	var searchBases []string
	if ldapCfg.UserOU != "" {
		for _, ou := range strings.Split(ldapCfg.UserOU, "|") {
			ou = strings.TrimSpace(ou)
			if ou != "" {
				searchBases = append(searchBases, ou)
			}
		}
	}
	if len(searchBases) == 0 {
		searchBases = []string{ldapCfg.BaseDN}
	}

	searchFilter := fmt.Sprintf("(%s=%s)", ldap.EscapeFilter(attrUsername), ldap.EscapeFilter(req.Username))

	var userDN string
	for _, searchBase := range searchBases {
		searchReq := ldap.NewSearchRequest(
			searchBase,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 10, false,
			searchFilter,
			[]string{"dn"},
			nil,
		)

		sr, searchErr := conn.Search(searchReq)
		if searchErr != nil {
			logger.Log.Warnf("LDAP login: search for user '%s' in base '%s' failed: %v", req.Username, searchBase, searchErr)
			continue
		}
		if len(sr.Entries) > 0 {
			userDN = sr.Entries[0].DN
			break
		}
	}

	if userDN == "" {
		logger.Log.Warnf("LDAP login: user '%s' not found in any search base (filter: %s, bases: %v)", req.Username, searchFilter, searchBases)
		return nil, errors.New("用户名或密码错误")
	}

	logger.Log.Infof("LDAP login: found user '%s' with DN: %s", req.Username, userDN)

	// Step 4: Authenticate — bind with the user's DN and their password
	if err := conn.Bind(userDN, req.Password); err != nil {
		logger.Log.Warnf("LDAP login: bind failed for user '%s' (DN: %s): %v", req.Username, userDN, err)
		return nil, errors.New("用户名或密码错误")
	}

	// ── Authentication successful ────────────────────────────────────
	logger.Log.Infof("LDAP user '%s' authenticated successfully via LDAP server %s", req.Username, addr)

	token, err := generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	return &LoginResponse{Token: token, User: user}, nil
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
