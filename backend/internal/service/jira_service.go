package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// JiraService handles Jira integration: issue lookup, CSE resolution, and periodic sync
type JiraService struct {
	mu      sync.Mutex
	syncing bool
}

var jiraServiceInstance *JiraService
var jiraOnce sync.Once

func GetJiraService() *JiraService {
	jiraOnce.Do(func() {
		jiraServiceInstance = &JiraService{}
	})
	return jiraServiceInstance
}

// jiraClient creates an HTTP client for Jira API calls
func (s *JiraService) jiraClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ResponseHeaderTimeout: 25 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

// getJiraConfig reads Jira settings from SystemSetting table, with environment variable fallbacks
func (s *JiraService) getJiraConfig() (server, username, password string, err error) {
	svc := &TotpService{}
	server = svc.GetSetting("jira", "jira_server")
	username = svc.GetSetting("jira", "jira_username")
	password = svc.GetSetting("jira", "jira_password")

	// Fallback to environment variables if DB settings are empty
	if server == "" {
		server = os.Getenv("JIRA_SERVER")
	}
	if username == "" {
		username = os.Getenv("JIRA_USERNAME")
	}
	if password == "" {
		password = os.Getenv("JIRA_PASSWORD")
	}

	if server == "" {
		return "", "", "", fmt.Errorf("Jira服务器未配置，请在系统设置中配置Jira信息或设置JIRA_SERVER环境变量")
	}
	if username == "" || password == "" {
		return "", "", "", fmt.Errorf("Jira认证信息未配置，请在系统设置中配置用户名和Token或设置JIRA_USERNAME/JIRA_PASSWORD环境变量")
	}
	server = strings.TrimRight(server, "/")
	return server, username, password, nil
}

// jiraAPIGet makes an authenticated GET request to Jira REST API
// Supports both API v2 and v3 (Atlassian Cloud may have migrated to v3)
func (s *JiraService) jiraAPIGet(path string) (map[string]interface{}, error) {
	server, username, password, err := s.getJiraConfig()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", server, path)
	client := s.jiraClient()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接Jira失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("Jira认证失败(HTTP %d)，请检查用户名和API Token配置", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		// If v2 returned 404 with API removal message, try v3
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if strings.Contains(bodyStr, "has been removed") && strings.Contains(path, "/rest/api/2/") {
			// Retry with API v3
			v3Path := strings.Replace(path, "/rest/api/2/", "/rest/api/3/", 1)
			return s.jiraAPIGetDirect(server, v3Path, username, password)
		}
		return nil, fmt.Errorf("工单不存在或无权限访问(HTTP 404)")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira返回错误(HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取Jira响应失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析Jira响应失败: %v", err)
	}
	return result, nil
}

// jiraAPIGetDirect makes a direct API call (used for v3 fallback)
func (s *JiraService) jiraAPIGetDirect(server, path, username, password string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s", server, path)
	client := s.jiraClient()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接Jira失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("Jira认证失败(HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("工单不存在或无权限访问(HTTP 404)")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira返回错误(HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取Jira响应失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析Jira响应失败: %v", err)
	}
	return result, nil
}

// CheckIssue looks up an issue: first from local cache, then from Jira API
// Logic follows the reference: ECSDESK issue → customfield_10007 (CSE) → CSE issue → customer/project/version
func (s *JiraService) CheckIssue(issueKey string) (map[string]string, error) {
	// Normalize issue key
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("工单号不能为空")
	}

	// 1. First try local cache (only if data is complete)
	var cached model.JiraIssueCache
	if err := repository.DB.Where("issue = ?", issueKey).First(&cached).Error; err == nil {
		// Only use cache if customer is populated (indicates successful CSE resolution)
		if cached.Customer != "" {
			result := map[string]string{
				"issue":    cached.Issue,
				"customer": cached.Customer,
				"project":  cached.Project,
				"version":  cached.TotpVersion,
				"cse":      cached.CSE,
				"summary":  cached.Summary,
			}
			return result, nil
		}
		// Cache entry exists but has empty customer - re-fetch from Jira
		logger.Log.Infof("Cache hit for %s but customer is empty, re-fetching from Jira", issueKey)
	}

	// 2. Not in cache or cache incomplete, query Jira API
	return s.fetchIssueFromJira(issueKey)
}

// fetchIssueFromJira queries Jira API following the reference code logic:
// ECSDESK issue → customfield_10007 (CSE key) → CSE issue → customer/project/version
func (s *JiraService) fetchIssueFromJira(issueKey string) (map[string]string, error) {
	// Get the ECSDESK/ECSL2/etc issue
	issueData, err := s.jiraAPIGet(fmt.Sprintf("/rest/api/2/issue/%s", issueKey))
	if err != nil {
		return nil, err
	}

	fields, ok := issueData["fields"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Jira响应格式异常")
	}

	result := map[string]string{
		"issue":    issueKey,
		"customer": "",
		"project":  "",
		"version":  "",
		"cse":      "",
		"summary":  "",
	}

	// Get summary
	if summary, ok := fields["summary"].(string); ok {
		result["summary"] = summary
	}

	// Get CSE key from customfield_10007 (handles various field types)
	cseKey := s.extractCSEKey(fields)

	if cseKey != "" {
		result["cse"] = cseKey
		logger.Log.Infof("Issue %s → CSE: %s, fetching CSE details...", issueKey, cseKey)

		// Query the CSE issue to get customer/project/version
		cseData, err := s.jiraAPIGet(fmt.Sprintf("/rest/api/2/issue/%s", cseKey))
		if err != nil {
			logger.Log.Warnf("Failed to fetch CSE issue %s: %v", cseKey, err)
			result["error"] = fmt.Sprintf("获取CSE工单%s失败: %v", cseKey, err)
		} else {
			if cseFields, ok := cseData["fields"].(map[string]interface{}); ok {
				// Extract customer from customfield_11191
				result["customer"] = s.extractCustomer(cseFields)
				// Extract project from customfield_11190
				result["project"] = s.extractProject(cseFields)
				// Extract version from customfield_11196
				result["version"] = s.extractVersion(cseFields)

				logger.Log.Infof("CSE %s → customer=%q, project=%q, version=%q",
					cseKey, result["customer"], result["project"], result["version"])
			} else {
				logger.Log.Warnf("CSE %s: fields is not a map", cseKey)
			}
		}
	} else {
		logger.Log.Warnf("Issue %s: no CSE key found in customfield_10007", issueKey)
	}

	// Fallback: if CSE extraction failed, try alternative approaches
	if result["customer"] == "" {
		// Try components on the original issue
		if components, ok := fields["components"].([]interface{}); ok && len(components) > 0 {
			if comp, ok := components[0].(map[string]interface{}); ok {
				if name, ok := comp["name"].(string); ok {
					result["customer"] = name
				}
			}
		}
	}
	// Smart fallback: parse customer and project from summary format
	// Format: "客户名-项目名(CSE-xxx)- 描述"
	if result["customer"] == "" && result["summary"] != "" {
		customer, project := parseSummary(result["summary"])
		if customer != "" {
			result["customer"] = customer
		}
		if result["project"] == "" && project != "" {
			result["project"] = project
		}
	}
	if result["project"] == "" {
		// Last resort: use full summary (not ideal, but better than empty)
		result["project"] = result["summary"]
	}
	if result["version"] == "" {
		// Try labels for version hint
		if labels, ok := fields["labels"].([]interface{}); ok {
			for _, label := range labels {
				if labelStr, ok := label.(string); ok {
					upper := strings.ToUpper(labelStr)
					if strings.HasPrefix(upper, "V5") || strings.HasPrefix(upper, "V6") {
						result["version"] = upper[:2]
						break
					}
				}
			}
		}
	}

	// Save to local cache for future lookups
	s.saveToCache(issueKey, result)

	return result, nil
}

// extractCSEKey extracts the CSE key from customfield_10007
// Handles both string type and object type (some Jira configs return {"key": "CSE-xxx"})
func (s *JiraService) extractCSEKey(fields map[string]interface{}) string {
	cf10007 := fields["customfield_10007"]
	if cf10007 == nil {
		return ""
	}

	// Direct string: "CSE-4525"
	if cse, ok := cf10007.(string); ok && cse != "" {
		return strings.TrimSpace(cse)
	}

	// Object with key: {"key": "CSE-4525", ...}
	if cseMap, ok := cf10007.(map[string]interface{}); ok {
		if key, ok := cseMap["key"].(string); ok {
			return strings.TrimSpace(key)
		}
		if val, ok := cseMap["value"].(string); ok {
			return strings.TrimSpace(val)
		}
	}

	// Could be a float64 (number) - unlikely but handle
	if num, ok := cf10007.(float64); ok {
		return fmt.Sprintf("CSE-%.0f", num)
	}

	logger.Log.Warnf("customfield_10007 unexpected type: %T, value: %v", cf10007, cf10007)
	return ""
}

// extractCustomer extracts customer name from CSE issue's customfield_11191
// Handles: array of strings ["客户名"], array of objects [{"value": "客户名"}], or single string
func (s *JiraService) extractCustomer(cseFields map[string]interface{}) string {
	cf11191 := cseFields["customfield_11191"]
	if cf11191 == nil {
		logger.Log.Warnf("customfield_11191 is nil in CSE issue")
		return ""
	}

	// Array type (most common): ["中国邮政储蓄银行"]
	if arr, ok := cf11191.([]interface{}); ok && len(arr) > 0 {
		first := arr[0]
		// Element is a string
		if str, ok := first.(string); ok {
			return str
		}
		// Element is an object: {"value": "客户名"} or {"name": "客户名"}
		if obj, ok := first.(map[string]interface{}); ok {
			if val, ok := obj["value"].(string); ok {
				return val
			}
			if name, ok := obj["name"].(string); ok {
				return name
			}
		}
		logger.Log.Warnf("customfield_11191[0] unexpected type: %T", first)
		return ""
	}

	// Direct string
	if str, ok := cf11191.(string); ok {
		return str
	}

	logger.Log.Warnf("customfield_11191 unexpected type: %T", cf11191)
	return ""
}

// extractProject extracts project name from CSE issue's customfield_11190
func (s *JiraService) extractProject(cseFields map[string]interface{}) string {
	cf11190 := cseFields["customfield_11190"]
	if cf11190 == nil {
		return ""
	}

	// String type (most common)
	if str, ok := cf11190.(string); ok {
		return str
	}

	// Object with value/name
	if obj, ok := cf11190.(map[string]interface{}); ok {
		if val, ok := obj["value"].(string); ok {
			return val
		}
		if name, ok := obj["name"].(string); ok {
			return name
		}
	}

	logger.Log.Warnf("customfield_11190 unexpected type: %T", cf11190)
	return ""
}

// extractVersion extracts version from CSE issue's customfield_11196
func (s *JiraService) extractVersion(cseFields map[string]interface{}) string {
	cf11196 := cseFields["customfield_11196"]
	if cf11196 == nil {
		return ""
	}

	// Object with "name" field: {"name": "6.1.1", ...}
	if obj, ok := cf11196.(map[string]interface{}); ok {
		if name, ok := obj["name"].(string); ok {
			return getTotpVersion(name)
		}
	}

	// Direct string
	if str, ok := cf11196.(string); ok {
		return getTotpVersion(str)
	}

	logger.Log.Warnf("customfield_11196 unexpected type: %T", cf11196)
	return ""
}

// parseSummary attempts to parse customer and project from summary format:
// "客户名-项目名(CSE-xxx)- 描述" or "客户名-项目名-描述"
func parseSummary(summary string) (customer, project string) {
	clean := summary

	// Remove (CSE-xxxx) or （CSE-xxxx）part
	if idx := strings.Index(clean, "(CSE-"); idx > 0 {
		clean = clean[:idx]
	} else if idx := strings.Index(clean, "（CSE-"); idx > 0 {
		clean = clean[:idx]
	}

	// Trim trailing spaces and dashes
	clean = strings.TrimRight(clean, "- ")

	// Split by first "-" to get customer and project
	parts := strings.SplitN(clean, "-", 2)
	if len(parts) == 2 {
		customer = strings.TrimSpace(parts[0])
		project = strings.TrimSpace(parts[1])
		// If project still has trailing description after "-", try to clean it
		// But be conservative - only trim if the trailing part is clearly a description
		if idx := strings.LastIndex(project, "-"); idx > 0 {
			after := strings.TrimSpace(project[idx+1:])
			before := strings.TrimSpace(project[:idx])
			// If the part after last "-" is very long (description), remove it
			if len([]rune(after)) > 15 && len([]rune(before)) > 2 {
				project = before
			}
		}
	}
	return
}

// saveToCache stores issue data in local JiraIssueCache table
func (s *JiraService) saveToCache(issueKey string, data map[string]string) {
	cache := model.JiraIssueCache{
		Issue:       issueKey,
		CSE:         data["cse"],
		Customer:    data["customer"],
		Project:     data["project"],
		Summary:     data["summary"],
		TotpVersion: data["version"],
	}

	var existing model.JiraIssueCache
	if err := repository.DB.Where("issue = ?", issueKey).First(&existing).Error; err == nil {
		// Update existing
		repository.DB.Model(&existing).Updates(map[string]interface{}{
			"cse":          cache.CSE,
			"customer":     cache.Customer,
			"project":      cache.Project,
			"summary":      cache.Summary,
			"totp_version": cache.TotpVersion,
		})
	} else {
		// Create new
		repository.DB.Create(&cache)
	}
}

// getTotpVersion converts version string to V5/V6/V3V4 format
func getTotpVersion(version string) string {
	if strings.HasPrefix(version, "6") || strings.HasPrefix(version, "V6") {
		return "V6"
	} else if strings.HasPrefix(version, "5") || strings.HasPrefix(version, "V5") {
		return "V5"
	}
	return "V5" // default
}

// SyncJiraIssues performs a bulk sync of Jira issues into local cache
// Uses JQL to search for recent ECSDESK/ECSL2 issues and cache their CSE info
func (s *JiraService) SyncJiraIssues() (int, error) {
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		return 0, fmt.Errorf("同步任务正在执行中，请稍后再试")
	}
	s.syncing = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.syncing = false
		s.mu.Unlock()
	}()

	server, username, password, err := s.getJiraConfig()
	if err != nil {
		return 0, err
	}

	// Search for recent ECSDESK/ECSL2 issues (last 30 days) using JQL
	// Use API v3 (Atlassian Cloud has deprecated v2 search)
	jql := "project in (ECSDESK, ECSL2) AND created >= -30d ORDER BY created DESC"
	startAt := 0
	maxResults := 50
	totalSynced := 0

	client := s.jiraClient()
	apiVersion := "3" // Start with v3 for Atlassian Cloud

	for {
		var searchURL string
		if apiVersion == "3" {
			searchURL = fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&startAt=%d&maxResults=%d&fields=key,summary,customfield_10007,components,labels,status,assignee",
				server, strings.ReplaceAll(jql, " ", "+"), startAt, maxResults)
		} else {
			searchURL = fmt.Sprintf("%s/rest/api/2/search?jql=%s&startAt=%d&maxResults=%d&fields=key,summary,customfield_10007,components,labels,status,assignee",
				server, strings.ReplaceAll(jql, " ", "+"), startAt, maxResults)
		}

		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			break
		}
		req.SetBasicAuth(username, password)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			logger.Log.Warnf("Jira sync search failed at startAt=%d: %v", startAt, err)
			break
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			// If v3 failed, try v2
			if apiVersion == "3" {
				apiVersion = "2"
				continue
			}
			logger.Log.Warnf("Jira sync search returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
			break
		}

		var searchResult struct {
			Total  int `json:"total"`
			Issues []struct {
				Key    string                 `json:"key"`
				Fields map[string]interface{} `json:"fields"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(body, &searchResult); err != nil {
			logger.Log.Warnf("Failed to parse Jira search result: %v", err)
			break
		}

		if len(searchResult.Issues) == 0 {
			break
		}

		for _, issue := range searchResult.Issues {
			issueKey := issue.Key
			fields := issue.Fields

			cacheData := map[string]string{
				"issue":    issueKey,
				"customer": "",
				"project":  "",
				"version":  "",
				"cse":      "",
				"summary":  "",
			}

			if summary, ok := fields["summary"].(string); ok {
				cacheData["summary"] = summary
			}

			// Get CSE from customfield_10007 (handles string and object types)
			cseKey := s.extractCSEKey(fields)
			if cseKey != "" {
				cacheData["cse"] = cseKey
			}

			// Try to get CSE details (customer/project/version)
			if cacheData["cse"] != "" {
				cseData, err := s.jiraAPIGet(fmt.Sprintf("/rest/api/2/issue/%s", cacheData["cse"]))
				if err == nil {
					if cseFields, ok := cseData["fields"].(map[string]interface{}); ok {
						cacheData["customer"] = s.extractCustomer(cseFields)
						cacheData["project"] = s.extractProject(cseFields)
						cacheData["version"] = s.extractVersion(cseFields)
					}
				} else {
					logger.Log.Warnf("Sync: failed to fetch CSE %s for %s: %v", cacheData["cse"], issueKey, err)
				}
			}

			// Fallback: parse from summary if CSE extraction failed
			if cacheData["customer"] == "" && cacheData["summary"] != "" {
				customer, project := parseSummary(cacheData["summary"])
				if customer != "" {
					cacheData["customer"] = customer
				}
				if cacheData["project"] == "" && project != "" {
					cacheData["project"] = project
				}
			}

			s.saveToCache(issueKey, cacheData)
			totalSynced++
		}

		startAt += len(searchResult.Issues)
		if startAt >= searchResult.Total {
			break
		}

		// Rate limit: don't hammer Jira too fast
		time.Sleep(500 * time.Millisecond)
	}

	logger.Log.Infof("Jira sync completed: %d issues synced", totalSynced)
	return totalSynced, nil
}

// StartPeriodicSync starts a background goroutine that syncs Jira data every interval
func (s *JiraService) StartPeriodicSync(interval time.Duration) {
	go func() {
		// Initial delay to let the server start up
		time.Sleep(30 * time.Second)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Do initial sync
		logger.Log.Info("Starting initial Jira sync...")
		count, err := s.SyncJiraIssues()
		if err != nil {
			logger.Log.Warnf("Initial Jira sync failed (will retry): %v", err)
		} else {
			logger.Log.Infof("Initial Jira sync: %d issues", count)
		}

		for range ticker.C {
			logger.Log.Info("Running periodic Jira sync...")
			count, err := s.SyncJiraIssues()
			if err != nil {
				logger.Log.Warnf("Periodic Jira sync failed: %v", err)
			} else {
				logger.Log.Infof("Periodic Jira sync completed: %d issues", count)
			}
		}
	}()
}

// GetCachedIssues returns cached issues for display/search
func (s *JiraService) GetCachedIssues(keyword string, page, pageSize int) ([]model.JiraIssueCache, int64, error) {
	var total int64
	var items []model.JiraIssueCache

	query := repository.DB.Model(&model.JiraIssueCache{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("issue LIKE ? OR customer LIKE ? OR project LIKE ? OR summary LIKE ?", like, like, like, like)
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	return items, total, err
}
