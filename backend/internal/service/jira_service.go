package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// JiraService handles Jira integration: issue lookup, CSE resolution, and periodic sync
type JiraService struct {
	mu     sync.Mutex
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

// getJiraConfig reads Jira settings from SystemSetting table
func (s *JiraService) getJiraConfig() (server, username, password string, err error) {
	svc := &TotpService{}
	server = svc.GetSetting("jira", "jira_server")
	username = svc.GetSetting("jira", "jira_username")
	password = svc.GetSetting("jira", "jira_password")

	if server == "" {
		return "", "", "", fmt.Errorf("Jira服务器未配置，请在系统设置中配置Jira信息")
	}
	if username == "" || password == "" {
		return "", "", "", fmt.Errorf("Jira认证信息未配置，请在系统设置中配置用户名和Token")
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

	// 1. First try local cache
	var cached model.JiraIssueCache
	if err := repository.DB.Where("issue = ?", issueKey).First(&cached).Error; err == nil {
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

	// 2. Not in cache, query Jira API
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

	// Get CSE key from customfield_10007 (this is the standard field in EasyStack Jira)
	cseKey := ""
	if cse, ok := fields["customfield_10007"].(string); ok && cse != "" {
		cseKey = cse
	}

	if cseKey != "" {
		result["cse"] = cseKey
		// Query the CSE issue to get customer/project/version
		cseData, err := s.jiraAPIGet(fmt.Sprintf("/rest/api/2/issue/%s", cseKey))
		if err == nil {
			if cseFields, ok := cseData["fields"].(map[string]interface{}); ok {
				// customfield_11191 = customer name (array, take first)
				if customerField, ok := cseFields["customfield_11191"].([]interface{}); ok && len(customerField) > 0 {
					if customer, ok := customerField[0].(string); ok {
						result["customer"] = customer
					}
				}
				// customfield_11190 = project name
				if project, ok := cseFields["customfield_11190"].(string); ok {
					result["project"] = project
				}
				// customfield_11196 = version (object with "name" field)
				if versionField, ok := cseFields["customfield_11196"].(map[string]interface{}); ok {
					if versionName, ok := versionField["name"].(string); ok {
						result["version"] = getTotpVersion(versionName)
					}
				}
			}
		} else {
			logger.Log.Warnf("Failed to fetch CSE issue %s: %v", cseKey, err)
		}
	}

	// If no CSE found, try alternative fields directly on the issue
	if result["customer"] == "" {
		// Try components
		if components, ok := fields["components"].([]interface{}); ok && len(components) > 0 {
			if comp, ok := components[0].(map[string]interface{}); ok {
				if name, ok := comp["name"].(string); ok {
					result["customer"] = name
				}
			}
		}
	}
	if result["project"] == "" {
		result["project"] = result["summary"]
	}
	if result["version"] == "" {
		// Try labels for version
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
// Uses JQL to search for recent ECSDESK issues and cache their CSE info
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

	// Search for recent ECSDESK issues (last 30 days) using JQL
	// Try API v3 first (Atlassian Cloud), fallback to v2
	jql := "project = ECSDESK AND created >= -30d ORDER BY created DESC"
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
				resp.Body.Close()
				continue
			}
			logger.Log.Warnf("Jira sync search returned %d", resp.StatusCode)
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

			// Get CSE from customfield_10007
			if cse, ok := fields["customfield_10007"].(string); ok && cse != "" {
				cacheData["cse"] = cse
			}

			// Try to get CSE details (customer/project/version) - but don't fail if CSE lookup fails
			if cacheData["cse"] != "" {
				cseData, err := s.jiraAPIGet(fmt.Sprintf("/rest/api/2/issue/%s", cacheData["cse"]))
				if err == nil {
					if cseFields, ok := cseData["fields"].(map[string]interface{}); ok {
						if customerField, ok := cseFields["customfield_11191"].([]interface{}); ok && len(customerField) > 0 {
							if customer, ok := customerField[0].(string); ok {
								cacheData["customer"] = customer
							}
						}
						if project, ok := cseFields["customfield_11190"].(string); ok {
							cacheData["project"] = project
						}
						if versionField, ok := cseFields["customfield_11196"].(map[string]interface{}); ok {
							if versionName, ok := versionField["name"].(string); ok {
								cacheData["version"] = getTotpVersion(versionName)
							}
						}
					}
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
