package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// TotpService handles dual-factor application and audit operations
type TotpService struct{}

func NewTotpService() *TotpService {
	return &TotpService{}
}

// CreateApplication creates a new TOTP application request
func (s *TotpService) CreateApplication(app *model.TotpApplication) error {
	app.AuditStatus = "pending"

	// Check if auto_approve is enabled
	autoApprove := s.GetSetting("totp", "auto_approve")
	if autoApprove == "true" {
		app.AuditStatus = "approved"
		now := time.Now()
		app.AuditTime = &now
		app.AuditorName = "系统自动审批"

		// Generate password immediately
		pass, ts, err := s.generateTotpPass(app.Customer, app.Project, app.Version, app.TotpType)
		if err != nil {
			logger.Log.Errorf("Auto-approve: TOTP generation failed: %v", err)
			return err
		}
		app.TotpPass = pass
		app.Timestamp = ts
	}

	if err := repository.DB.Create(app).Error; err != nil {
		return err
	}
	return nil
}

// ListMyApplications returns applications for the current user
func (s *TotpService) ListMyApplications(userID uint, page, pageSize int) ([]model.TotpApplication, int64, error) {
	var total int64
	var apps []model.TotpApplication

	query := repository.DB.Where("user_id = ?", userID)
	query.Model(&model.TotpApplication{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

// ListPendingReviews returns pending applications for admin audit
// If auditorID is specified, only returns applications assigned to that auditor (or unassigned ones)
func (s *TotpService) ListPendingReviews(auditorID uint, page, pageSize int) ([]model.TotpApplication, int64, error) {
	var total int64
	var apps []model.TotpApplication

	query := repository.DB.Where("audit_status = ?", "pending")
	// Show applications assigned to this auditor OR unassigned (assigned_auditor_id = 0)
	if auditorID > 0 {
		query = query.Where("assigned_auditor_id = ? OR assigned_auditor_id = 0", auditorID)
	}
	query.Model(&model.TotpApplication{}).Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

// ListAllApplications returns all applications (admin view)
func (s *TotpService) ListAllApplications(page, pageSize int, status string) ([]model.TotpApplication, int64, error) {
	var total int64
	var apps []model.TotpApplication

	query := repository.DB.Model(&model.TotpApplication{})
	if status != "" && status != "all" {
		query = query.Where("audit_status = ?", status)
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

// AuditApplication approves or rejects an application
func (s *TotpService) AuditApplication(appID uint, auditorID uint, auditorName string, approved bool, remark string) error {
	var app model.TotpApplication
	if err := repository.DB.First(&app, appID).Error; err != nil {
		return fmt.Errorf("application not found")
	}
	if app.AuditStatus != "pending" {
		return fmt.Errorf("application already reviewed")
	}

	now := time.Now()
	app.AuditorID = auditorID
	app.AuditorName = auditorName
	app.AuditTime = &now
	app.AuditRemark = remark

	if approved {
		app.AuditStatus = "approved"
		// Generate TOTP password upon approval
		pass, ts, err := s.generateTotpPass(app.Customer, app.Project, app.Version, app.TotpType)
		if err != nil {
			logger.Log.Errorf("TOTP generation failed for app %d: %v", appID, err)
			app.AuditStatus = "approved"
			app.TotpPass = fmt.Sprintf("生成失败: %v", err)
		} else {
			app.TotpPass = pass
			app.Timestamp = ts
		}
	} else {
		app.AuditStatus = "rejected"
	}

	return repository.DB.Save(&app).Error
}

// BatchAudit approves or rejects multiple applications
func (s *TotpService) BatchAudit(appIDs []uint, auditorID uint, auditorName string, approved bool, remark string) (int, error) {
	count := 0
	for _, id := range appIDs {
		if err := s.AuditApplication(id, auditorID, auditorName, approved, remark); err != nil {
			logger.Log.Warnf("BatchAudit: failed for app %d: %v", id, err)
			continue
		}
		count++
	}
	return count, nil
}

// GetSettings returns all settings for a given category
func (s *TotpService) GetSettings(category string) []model.SystemSetting {
	var settings []model.SystemSetting
	query := repository.DB.Order("sort_order ASC, id ASC")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	query.Find(&settings)
	return settings
}

// GetAllSettings returns all settings grouped by category
func (s *TotpService) GetAllSettings() map[string][]model.SystemSetting {
	var settings []model.SystemSetting
	repository.DB.Order("category ASC, sort_order ASC, id ASC").Find(&settings)

	result := make(map[string][]model.SystemSetting)
	for _, s := range settings {
		result[s.Category] = append(result[s.Category], s)
	}
	return result
}

// UpdateSettings batch-updates settings
func (s *TotpService) UpdateSettings(settings []model.SystemSetting) error {
	for _, setting := range settings {
		if setting.ID > 0 {
			repository.DB.Model(&model.SystemSetting{}).Where("id = ?", setting.ID).Updates(map[string]interface{}{
				"value": setting.Value,
			})
		} else if setting.Category != "" && setting.Key != "" {
			repository.DB.Model(&model.SystemSetting{}).
				Where("category = ? AND `key` = ?", setting.Category, setting.Key).
				Updates(map[string]interface{}{"value": setting.Value})
		}
	}
	return nil
}

// GetSetting returns a single setting value
func (s *TotpService) GetSetting(category, key string) string {
	var setting model.SystemSetting
	if err := repository.DB.Where("category = ? AND `key` = ?", category, key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// generateTotpPass generates a TOTP/roller password
func (s *TotpService) generateTotpPass(customer, project, version, totpType string) (string, string, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if totpType == "roller" {
		// Generate roller password using TOTP algorithm with stored secret
		secret := s.GetSetting("totp", "roller_secret")
		if secret == "" {
			secret = "6GAF6NXNYCT75FV3APTJU4R5XZJ7X6I4"
		}
		pass, err := generateOTP(secret)
		if err != nil {
			return "", "", fmt.Errorf("roller generation failed: %v", err)
		}
		return pass, timestamp, nil
	}

	// For standard TOTP (动态密码), call the external TOTP server
	pass, err := s.callTotpServer(customer, project, version)
	if err != nil {
		return "", "", fmt.Errorf("动态密码生成失败: %v", err)
	}
	return pass, timestamp, nil
}

// callTotpServer calls the external TOTP server to generate a dynamic password (动态密码)
// Three version types with different API paths:
//   - V5:   GET /totps/licenses + POST /totps
//   - V6:   GET /v6/totps/licenses + POST /v6/totps
//   - V611 (V6.1.1+): GET /topoweb/totps/topos + POST /topoweb/totps
func (s *TotpService) callTotpServer(customer, project, version string) (string, error) {
	// Read settings
	serverURL := s.GetSetting("totp", "totp_server")
	if serverURL == "" {
		serverURL = "http://lic.easystack.cn"
	}
	authUser := s.GetSetting("totp", "totp_auth_user")
	if authUser == "" {
		authUser = "totp"
	}
	authPass := s.GetSetting("totp", "totp_auth_pass")
	if authPass == "" {
		authPass = "Totp@2013"
	}

	baseURL := strings.TrimRight(serverURL, "/")

	// Determine API paths based on version
	// V611 (V6.1.1+) uses topoweb service with different endpoint paths
	// V6 uses /v6 prefix
	// V5 and V3V4 use base URL directly
	var licensesPath, totpPath string
	versionUpper := strings.ToUpper(version)

	switch versionUpper {
	case "V611":
		licensesPath = "/topoweb/totps/topos"
		totpPath = "/topoweb/totps"
	case "V6":
		licensesPath = "/v6/totps/licenses"
		totpPath = "/v6/totps"
	default: // V5, V3V4, etc.
		licensesPath = "/totps/licenses"
		totpPath = "/totps"
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: GET licenses/topos endpoint to find license keys
	licensesURL := fmt.Sprintf("%s%s?company=%s&project=%s",
		baseURL, licensesPath, url.QueryEscape(customer), url.QueryEscape(project))

	req, err := http.NewRequest("GET", licensesURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.SetBasicAuth(authUser, authPass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求TOTP服务器失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TOTP服务器返回错误状态码: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse licenses response
	var licensesResp struct {
		TotalCount int `json:"totalcount"`
		Result     []struct {
			UserRootPass string `json:"user_root_pass"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &licensesResp); err != nil {
		return "", fmt.Errorf("解析licenses响应失败: %v, body: %s", err, string(body))
	}

	if licensesResp.TotalCount == 0 {
		return "", fmt.Errorf("未查询到该项目的license信息 (company=%s, project=%s)", customer, project)
	}

	// Extract key_pass items from user_root_pass
	type keyItem struct {
		Company string `json:"company"`
		Project string `json:"project"`
		KeyPass string `json:"key_pass"`
	}
	var keyItems []keyItem
	for _, item := range licensesResp.Result {
		if item.UserRootPass != "" {
			keyItems = append(keyItems, keyItem{
				Company: customer,
				Project: project,
				KeyPass: item.UserRootPass,
			})
		}
	}

	if len(keyItems) == 0 {
		return "", fmt.Errorf("该项目没有可申请的license (无user_root_pass)")
	}

	// Step 2: POST /totps with key_items to generate TOTP
	totpReqBody := struct {
		KeyItems []keyItem `json:"key_items"`
	}{
		KeyItems: keyItems,
	}

	jsonBody, err := json.Marshal(totpReqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	totpURL := fmt.Sprintf("%s%s", baseURL, totpPath)
	req2, err := http.NewRequest("POST", totpURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("创建TOTP请求失败: %v", err)
	}
	req2.SetBasicAuth(authUser, authPass)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("请求TOTP密码生成失败: %v", err)
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("读取TOTP响应失败: %v", err)
	}

	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TOTP生成接口返回错误: %d, body: %s", resp2.StatusCode, string(body2))
	}

	// Parse TOTP response - extract password(s) from response
	// API doc response format: {"result": [{"status": "success", "totp_pass": "421177", "timestamp": "..."}]}
	var totpResp map[string]interface{}
	if err := json.Unmarshal(body2, &totpResp); err != nil {
		return "", fmt.Errorf("解析TOTP响应失败: %v, body: %s", err, string(body2))
	}

	// Try to extract password from various response formats
	var passwords []string
	var errors []string

	// Check if response has a "result" array
	if result, ok := totpResp["result"]; ok {
		if resultArr, ok := result.([]interface{}); ok {
			for _, item := range resultArr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					// Check status field — skip error items
					if status, ok := itemMap["status"]; ok {
						if statusStr, ok := status.(string); ok && statusStr == "error" {
							if tp, ok := itemMap["totp_pass"]; ok {
								if tpStr, ok := tp.(string); ok {
									errors = append(errors, tpStr)
								}
							}
							continue
						}
					}
					// Extract totp_pass from successful items
					if v, ok := itemMap["totp_pass"]; ok {
						if str, ok := v.(string); ok && str != "" {
							passwords = append(passwords, str)
						}
					}
				}
			}
		}
	}

	// Check top-level fields as fallback
	if len(passwords) == 0 {
		for _, field := range []string{"totp_pass", "password", "totp", "pass", "otp"} {
			if v, ok := totpResp[field]; ok {
				if str, ok := v.(string); ok && str != "" {
					passwords = append(passwords, str)
					break
				}
			}
		}
	}

	var password string
	if len(passwords) > 0 {
		// Deduplicate and join passwords (matching Python: " ".join(set(totp_pass)))
		seen := make(map[string]bool)
		var unique []string
		for _, p := range passwords {
			if !seen[p] {
				seen[p] = true
				unique = append(unique, p)
			}
		}
		password = strings.Join(unique, " ")
	} else if len(errors) > 0 {
		return "", fmt.Errorf("TOTP服务器生成失败: %s", strings.Join(errors, "; "))
	} else {
		// If we can't parse a known field, return the raw response (truncated for safety)
		raw := string(body2)
		if len(raw) > 2000 {
			raw = raw[:2000] + "...(truncated)"
		}
		password = raw
	}

	logger.Log.Infof("TOTP dynamic password generated for company=%s, project=%s, version=%s", customer, project, version)
	return password, nil
}

// CheckIssueFromJira delegates to JiraService for issue lookup (cache-first, then API)
func (s *TotpService) CheckIssueFromJira(issue string) (map[string]string, error) {
	return GetJiraService().CheckIssue(issue)
}

// GetAdminUsers returns all users with role=admin (for auditor assignment dropdown)
func (s *TotpService) GetAdminUsers() ([]map[string]interface{}, error) {
	var users []model.User
	if err := repository.DB.Where("role = ?", "admin").Find(&users).Error; err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		result = append(result, map[string]interface{}{
			"id":           u.ID,
			"username":     u.Username,
			"display_name": u.DisplayName,
		})
	}
	return result, nil
}

// QuickGenerateTotp is a public wrapper around generateTotpPass for use by handlers
// that need direct TOTP generation without going through the application/audit flow.
func (s *TotpService) QuickGenerateTotp(customer, project, version, totpType string) (string, string, error) {
	return s.generateTotpPass(customer, project, version, totpType)
}

// generateOTP generates a 6-digit TOTP code using the given base32 secret
func generateOTP(secret string) (string, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid secret: %v", err)
	}

	// Current time step (30 second window)
	counter := uint64(time.Now().Unix() / 30)

	// HMAC-SHA1
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0F
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	otp := code % uint32(math.Pow10(6))

	return fmt.Sprintf("%06d", otp), nil
}
