package handler

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// OpsEnvHandler handles operations environment management endpoints
type OpsEnvHandler struct{}

func NewOpsEnvHandler() *OpsEnvHandler {
	return &OpsEnvHandler{}
}

// ListOpsEnvironments handles GET /api/ops-env/list
// Supports search by customer_name or project_name, and filter by status
func (h *OpsEnvHandler) ListOpsEnvironments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	status := c.Query("status") // in_progress, done, discarded, all
	region := c.Query("region")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := repository.DB.Model(&model.OpsEnvironment{})

	// Always exclude POC environments from listing
	query = query.Where("env_type != ? AND env_type != ?", "POC", "poc")

	// Filter by status
	switch status {
	case "in_progress":
		query = query.Where("status IN (?, ?)", "正在进行", "In Progress")
	case "done":
		query = query.Where("status IN (?, ?)", "已完成", "Done")
	case "discarded":
		query = query.Where("status IN (?, ?)", "已弃用", "Discarded")
	case "pending":
		query = query.Where("status = ?", "待办")
	default:
		// all - no filter
	}

	// Region filter
	if region != "" {
		query = query.Where("ops_region = ?", region)
	}

	// Fuzzy search by customer_name or project_name
	if search != "" {
		query = query.Where("customer_name LIKE ? OR project_name LIKE ? OR cse_name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var items []model.OpsEnvironment
	offset := (page - 1) * pageSize
	err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetOpsEnvStats handles GET /api/ops-env/stats
// Returns status counts and region distribution
func (h *OpsEnvHandler) GetOpsEnvStats(c *gin.Context) {
	// Base condition: exclude POC environments
	noPOC := "env_type != 'POC' AND env_type != 'poc'"

	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	repository.DB.Model(&model.OpsEnvironment{}).
		Where(noPOC).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts)

	type RegionCount struct {
		Region string `json:"region"`
		Count  int64  `json:"count"`
	}
	var regionCounts []RegionCount
	repository.DB.Model(&model.OpsEnvironment{}).
		Where(noPOC).
		Select("ops_region as region, count(*) as count").
		Where("ops_region != ''").
		Group("ops_region").
		Order("count DESC").
		Find(&regionCounts)

	// Total node count
	type NodeSum struct {
		Total int64 `json:"total"`
	}
	var nodeSum NodeSum
	repository.DB.Model(&model.OpsEnvironment{}).
		Where(noPOC).
		Select("COALESCE(SUM(node_count), 0) as total").
		Find(&nodeSum)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"status_counts": statusCounts,
			"region_counts": regionCounts,
			"total_nodes":   nodeSum.Total,
		},
	})
}

// GetOpsEnvTopCustomers handles GET /api/ops-env/top-customers
// Returns TOP10 customers by environment count
func (h *OpsEnvHandler) GetOpsEnvTopCustomers(c *gin.Context) {
	type CustomerCount struct {
		CustomerName string `json:"customer_name"`
		Count        int64  `json:"count"`
	}
	var topCustomers []CustomerCount
	repository.DB.Model(&model.OpsEnvironment{}).
		Where("env_type != 'POC' AND env_type != 'poc'").
		Select("customer_name, count(*) as count").
		Where("customer_name != ''").
		Group("customer_name").
		Order("count DESC").
		Limit(10).
		Find(&topCustomers)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": topCustomers})
}

// GetOpsEnvTopNodes handles GET /api/ops-env/top-nodes
// Returns TOP10 environments by node count
func (h *OpsEnvHandler) GetOpsEnvTopNodes(c *gin.Context) {
	type NodeTop struct {
		CustomerName string `json:"customer_name"`
		ProjectName  string `json:"project_name"`
		CSEName      string `json:"cse_name"`
		NodeCount    int    `json:"node_count"`
	}
	var topNodes []NodeTop
	repository.DB.Model(&model.OpsEnvironment{}).
		Where("env_type != 'POC' AND env_type != 'poc'").
		Select("customer_name, project_name, cse_name, node_count").
		Where("node_count > 0").
		Order("node_count DESC").
		Limit(10).
		Find(&topNodes)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": topNodes})
}

// GetOpsEnvCalendar handles GET /api/ops-env/calendar
// Returns daily discard counts for calendar view
func (h *OpsEnvHandler) GetOpsEnvCalendar(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	viewType := c.DefaultQuery("view", "month") // month, year

	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var dayCounts []DayCount

	switch viewType {
	case "year":
		// Get monthly counts for the whole year
		repository.DB.Model(&model.OpsEnvironment{}).
			Select("DATE_FORMAT(discarded_at, '%Y-%m') as date, count(*) as count").
			Where("discarded_at IS NOT NULL AND YEAR(discarded_at) = ?", year).
			Group("DATE_FORMAT(discarded_at, '%Y-%m')").
			Order("date ASC").
			Find(&dayCounts)
	default:
		// month - Get daily counts for a specific month
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		repository.DB.Model(&model.OpsEnvironment{}).
			Select("DATE_FORMAT(discarded_at, '%Y-%m-%d') as date, count(*) as count").
			Where("discarded_at IS NOT NULL AND discarded_at >= ? AND discarded_at < ?", startDate, endDate).
			Group("DATE_FORMAT(discarded_at, '%Y-%m-%d')").
			Order("date ASC").
			Find(&dayCounts)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"counts":    dayCounts,
			"year":      year,
			"month":     month,
			"view_type": viewType,
		},
	})
}

// SyncOpsEnvironments handles POST /api/ops-env/sync
// Triggers a sync from Jira CSE project
func (h *OpsEnvHandler) SyncOpsEnvironments(c *gin.Context) {
	go func() {
		if err := syncOpsEnvFromJira(); err != nil {
			logger.Log.Errorf("OpsEnv sync failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "同步任务已启动，请稍后刷新查看",
	})
}

// RunOpsEnvSync is a package-level exported function that triggers OpsEnvironment sync.
// This is used by main.go to run periodic background sync.
func RunOpsEnvSync() error {
	return syncOpsEnvFromJira()
}

// StartPeriodicOpsEnvSync starts background periodic sync for ops environments (every 6 hours)
func StartPeriodicOpsEnvSync(interval time.Duration) {
	go func() {
		// Initial delay to let the server and Jira cache warm up first
		time.Sleep(2 * time.Minute)

		// Do initial sync
		logger.Log.Info("Starting initial OpsEnvironment sync...")
		if err := syncOpsEnvFromJira(); err != nil {
			logger.Log.Warnf("Initial OpsEnvironment sync failed (will retry next cycle): %v", err)
		} else {
			logger.Log.Info("Initial OpsEnvironment sync completed")
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			logger.Log.Info("Running periodic OpsEnvironment sync...")
			if err := syncOpsEnvFromJira(); err != nil {
				logger.Log.Warnf("Periodic OpsEnvironment sync failed: %v", err)
			} else {
				logger.Log.Info("Periodic OpsEnvironment sync completed")
			}
		}
	}()
}

// GetRegions handles GET /api/ops-env/regions
func (h *OpsEnvHandler) GetRegions(c *gin.Context) {
	var regions []string
	repository.DB.Model(&model.OpsEnvironment{}).
		Distinct("ops_region").
		Where("ops_region != ''").
		Order("ops_region ASC").
		Pluck("ops_region", &regions)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": regions})
}

// DiagnoseOpsEnv handles GET /api/ops-env/diagnose
// Returns diagnostic information to help troubleshoot sync issues
func (h *OpsEnvHandler) DiagnoseOpsEnv(c *gin.Context) {
	diag := gin.H{}

	// 1. Check database table record count
	var totalRecords int64
	repository.DB.Model(&model.OpsEnvironment{}).Count(&totalRecords)
	diag["db_total_records"] = totalRecords

	// 2. Check last sync time
	var lastSynced model.OpsEnvironment
	err := repository.DB.Order("synced_at DESC").First(&lastSynced).Error
	if err != nil {
		diag["last_sync_time"] = nil
		diag["last_sync_error"] = "no records found or query error"
	} else {
		diag["last_sync_time"] = lastSynced.SyncedAt
		diag["last_sync_env_key"] = lastSynced.EnvDetailKey
	}

	// 3. Check Jira configuration status
	jiraServer, jiraUser, jiraToken, cfgErr := getJiraConfigForOpsEnv()
	if cfgErr != nil {
		diag["jira_config_status"] = "MISSING"
		diag["jira_config_error"] = cfgErr.Error()
	} else {
		diag["jira_config_status"] = "OK"
		diag["jira_server"] = jiraServer
		diag["jira_user"] = jiraUser
		diag["jira_token_len"] = len(jiraToken)
	}

	// 4. Test Jira connectivity (quick test)
	if cfgErr == nil {
		testJQL := `project = CSE and issuetype = 环境详细信息`
		_, _, _, testErr := jiraSearch(jiraServer, jiraUser, jiraToken, testJQL, "summary", "", 1)
		if testErr != nil {
			diag["jira_connectivity"] = "FAILED"
			diag["jira_test_error"] = testErr.Error()
		} else {
			diag["jira_connectivity"] = "OK"
		}
	}

	// 5. Status distribution
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	repository.DB.Model(&model.OpsEnvironment{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts)
	diag["status_distribution"] = statusCounts

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": diag})
}

// ─── Jira REST API integration ────────────────────────────────────────────────

// jiraSearchResponse represents Jira REST v3 search API response (cursor-based)
type jiraSearchResponse struct {
	Issues        []jiraIssue `json:"issues"`
	NextPageToken string      `json:"nextPageToken"`
	IsLast        bool        `json:"isLast"`
	Total         int         `json:"total"`
}

type jiraIssue struct {
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields"`
}

// jiraFieldValue for option fields like {value: "xxx"}
type jiraFieldValue struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

// getJiraConfigForOpsEnv reads Jira settings from SystemSetting table first, then falls back to environment variables.
// Uses the same TotpService.GetSetting() approach as jira_service.go to ensure consistency.
func getJiraConfigForOpsEnv() (server, username, token string, err error) {
	// Use TotpService.GetSetting for consistent DB access (same as jira_service.go)
	svc := &service.TotpService{}
	server = svc.GetSetting("jira", "jira_server")
	username = svc.GetSetting("jira", "jira_username")
	token = svc.GetSetting("jira", "jira_password")

	logger.Log.Debugf("OpsEnv Jira config from DB: server=%q (len=%d), user=%q (len=%d), token_len=%d",
		server, len(server), username, len(username), len(token))

	// Fallback to environment variables
	cfg := config.Load()
	if server == "" {
		server = cfg.Jira.Server
	}
	if username == "" {
		username = cfg.Jira.Username
	}
	if token == "" {
		token = cfg.Jira.APIToken
	}

	if server == "" || username == "" || token == "" {
		return "", "", "", fmt.Errorf("Jira configuration missing (check SystemSettings or env vars): server=%q, user=%q, token_len=%d",
			server, username, len(token))
	}
	server = strings.TrimRight(server, "/")
	return server, username, token, nil
}

// syncOpsEnvFromJira fetches environments from Jira and updates the database
// Uses concurrent workers to speed up CSE issue fetching
func syncOpsEnvFromJira() error {
	// Prevent concurrent sync operations
	opsEnvSyncingMu.Lock()
	if opsEnvSyncing {
		opsEnvSyncingMu.Unlock()
		logger.Log.Info("OpsEnvironment sync already in progress, skipping")
		return nil
	}
	opsEnvSyncing = true
	opsEnvSyncingMu.Unlock()
	defer func() {
		opsEnvSyncingMu.Lock()
		opsEnvSyncing = false
		opsEnvSyncingMu.Unlock()
	}()

	jiraServer, jiraUser, jiraToken, err := getJiraConfigForOpsEnv()
	if err != nil {
		return err
	}

	logger.Log.Infof("Starting OpsEnvironment sync from Jira %s ...", jiraServer)

	// Step 1: Fetch all 环境详细信息 issues
	jql := `project = CSE and issuetype = 环境详细信息`
	envIssues, err := jiraSearchAll(jiraServer, jiraUser, jiraToken, jql,
		"customfield_11191,customfield_11190,customfield_11307,customfield_11271,"+
			"customfield_11338,customfield_11354,customfield_11289,customfield_11303,"+
			"customfield_11357,customfield_11359,customfield_11360,customfield_11288,"+
			"customfield_11273,customfield_10007,status")
	if err != nil {
		return fmt.Errorf("fetch env detail issues: %w", err)
	}

	logger.Log.Infof("Fetched %d environment detail issues from Jira, processing with concurrency...", len(envIssues))

	now := time.Now()
	var syncCount, errCount int64
	var mu sync.Mutex

	// Use worker pool for concurrent CSE issue fetching (10 concurrent workers)
	concurrency := 10
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, envIssue := range envIssues {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(issue jiraIssue, idx int) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			if err := processEnvIssue(issue, jiraServer, jiraUser, jiraToken, &now); err != nil {
				if idx < 5 || idx%100 == 0 { // only log first few and periodic ones
					logger.Log.Debugf("Skip issue %s: %v", issue.Key, err)
				}
				mu.Lock()
				errCount++
				mu.Unlock()
				return
			}
			mu.Lock()
			syncCount++
			if syncCount%50 == 0 {
				logger.Log.Infof("OpsEnv sync progress: %d/%d processed", syncCount+errCount, len(envIssues))
			}
			mu.Unlock()
		}(envIssue, i)
	}

	wg.Wait()

	logger.Log.Infof("OpsEnvironment sync complete: %d synced, %d errors, %d total",
		syncCount, errCount, len(envIssues))
	return nil
}

func processEnvIssue(envIssue jiraIssue, server, user, token string, now *time.Time) error {
	// Parse env detail fields
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envIssue.Fields, &fields); err != nil {
		return fmt.Errorf("unmarshal fields: %w", err)
	}

	customerName := extractStringArray(fields["customfield_11191"])
	projectName := extractString(fields["customfield_11190"])
	contractNum := extractString(fields["customfield_11307"])
	customerLevel := extractOptionValue(fields["customfield_11271"])
	city := extractString(fields["customfield_11338"])
	sales := extractString(fields["customfield_11354"])
	sla := extractOptionValue(fields["customfield_11289"])
	opsRegion := extractOptionValue(fields["customfield_11303"])
	isRenewal := extractString(fields["customfield_11357"])
	renewalStart := extractString(fields["customfield_11359"])
	renewalEnd := extractString(fields["customfield_11360"])
	maintainStart := extractString(fields["customfield_11288"])
	maintainEnd := extractString(fields["customfield_11273"])

	// Get CSE key from customfield_10007
	cseKey := extractString(fields["customfield_10007"])
	if cseKey == "" {
		return fmt.Errorf("no CSE key (customfield_10007)")
	}

	// Get env issue status
	envStatus := extractStatusName(fields["status"])

	// Fetch CSE issue for additional details
	cseIssue, err := jiraGetIssue(server, user, token, cseKey,
		"summary,status,fixVersions,customfield_11196,customfield_11220,customfield_11299,"+
			"customfield_11287,created,"+
			"customfield_11197,customfield_11199,customfield_11201,"+
			"customfield_11203,customfield_11205,customfield_11207")
	if err != nil {
		return fmt.Errorf("fetch CSE %s: %w", cseKey, err)
	}

	var cseFields map[string]json.RawMessage
	if err := json.Unmarshal(cseIssue.Fields, &cseFields); err != nil {
		return fmt.Errorf("unmarshal CSE fields: %w", err)
	}

	cseName := extractString(cseFields["summary"])
	cseStatus := extractStatusName(cseFields["status"])
	// Try fixVersions first, then fall back to customfield_11196
	version := extractVersionName(cseFields["fixVersions"])
	if version == "" {
		version = extractVersionName(cseFields["customfield_11196"])
	}
	envType := extractString(cseFields["customfield_11220"])

	// Filter out POC environments
	if strings.EqualFold(envType, "POC") || strings.EqualFold(envType, "poc") {
		return fmt.Errorf("skip POC environment (env_type=%s)", envType)
	}

	cpuArch := extractStringArrayFirst(cseFields["customfield_11299"])
	projectNum := extractString(cseFields["customfield_11287"])
	deployTime := extractString(cseFields["created"])

	// Calculate node count
	nodeCount := sumNodeFields(cseFields)

	// Use CSE status as the main status (In Progress / Done / Discarded)
	status := cseStatus
	if status == "" {
		status = envStatus
	}

	// Determine discarded_at
	var discardedAt *time.Time
	if strings.EqualFold(status, "Discarded") || strings.EqualFold(status, "discarded") {
		discardedAt = now
	}

	// Upsert by env_detail_key
	var existing model.OpsEnvironment
	result := repository.DB.Where("env_detail_key = ?", envIssue.Key).First(&existing)

	env := model.OpsEnvironment{
		CSEKey:        cseKey,
		EnvDetailKey:  envIssue.Key,
		CustomerName:  customerName,
		ProjectName:   projectName,
		CSEName:       cseName,
		Status:        status,
		ProjectNum:    projectNum,
		ContractNum:   contractNum,
		OpsRegion:     opsRegion,
		EnvType:       envType,
		CPUArch:       cpuArch,
		Version:       version,
		CustomerLevel: customerLevel,
		NodeCount:     nodeCount,
		City:          city,
		Sales:         sales,
		SLA:           sla,
		IsRenewal:     isRenewal,
		DeployTime:    deployTime,
		RenewalStart:  renewalStart,
		RenewalEnd:    renewalEnd,
		MaintainStart: maintainStart,
		MaintainEnd:   maintainEnd,
		SyncedAt:      now,
	}

	// Keep existing discarded_at if already set
	if result.Error == nil {
		if existing.DiscardedAt != nil {
			discardedAt = existing.DiscardedAt
		}
		env.ID = existing.ID
		env.CreatedAt = existing.CreatedAt
		env.DiscardedAt = discardedAt
		repository.DB.Save(&env)
	} else {
		env.DiscardedAt = discardedAt
		repository.DB.Create(&env)
	}

	return nil
}

// ─── Jira REST helpers ────────────────────────────────────────────────────────

func jiraSearchAll(server, user, token, jql, fields string) ([]jiraIssue, error) {
	var allIssues []jiraIssue
	nextPageToken := ""
	maxResults := 100

	for {
		issues, nextToken, isLast, err := jiraSearch(server, user, token, jql, fields, nextPageToken, maxResults)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if isLast || len(issues) == 0 || nextToken == "" {
			break
		}
		nextPageToken = nextToken
	}
	return allIssues, nil
}

// opsEnvHTTPClient is a shared HTTP client with extended timeout and TLS skip for Jira API calls.
// Reusing a single client enables TCP connection pooling across requests.
var opsEnvHTTPClient = &http.Client{
	Timeout: 300 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 240 * time.Second,
	},
}

// opsEnvSyncing prevents concurrent sync operations
var (
	opsEnvSyncing   bool
	opsEnvSyncingMu sync.Mutex
)

func jiraSearch(server, user, token, jql, fields, nextPageToken string, maxResults int) ([]jiraIssue, string, bool, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&maxResults=%d&fields=%s",
		server, url.QueryEscape(jql), maxResults, fields)
	if nextPageToken != "" {
		apiURL += "&nextPageToken=" + url.QueryEscape(nextPageToken)
	}

	// Retry up to 3 times on timeout/transient errors
	maxRetries := 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*10) * time.Second
			logger.Log.Warnf("OpsEnv jiraSearch retry %d/%d after %v (previous error: %v)", attempt+1, maxRetries, backoff, lastErr)
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, "", true, err
		}
		req.SetBasicAuth(user, token)
		req.Header.Set("Accept", "application/json")

		resp, err := opsEnvHTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			// Retry on timeout or connection errors
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") ||
				strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "reset") {
				continue
			}
			return nil, "", true, lastErr
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body[:int(math.Min(float64(len(body)), 500))]))
			continue // retry on rate limit or server errors
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, "", true, fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body[:int(math.Min(float64(len(body)), 500))]))
		}

		var result jiraSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, "", true, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()
		return result.Issues, result.NextPageToken, result.IsLast, nil
	}

	return nil, "", true, lastErr
}

func jiraGetIssue(server, user, token, issueKey, fields string) (*jiraIssue, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=%s",
		server, issueKey, fields)

	// Retry up to 3 times on timeout/transient errors
	maxRetries := 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*5) * time.Second)
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(user, token)
		req.Header.Set("Accept", "application/json")

		resp, err := opsEnvHTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection") ||
				strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "reset") {
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body[:int(math.Min(float64(len(body)), 500))]))
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Jira API returned %d: %s", resp.StatusCode, string(body[:int(math.Min(float64(len(body)), 500))]))
		}

		var issue jiraIssue
		if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode issue: %w", err)
		}
		resp.Body.Close()
		return &issue, nil
	}

	return nil, lastErr
}

// ─── Field extraction helpers ─────────────────────────────────────────────────

func extractString(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try as number
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return fmt.Sprintf("%.0f", n)
	}
	return ""
}

func extractStringArray(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	// Try as single string
	return extractString(raw)
}

func extractStringArrayFirst(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	// Try as array of strings
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	// Try as array of objects with value
	var objArr []jiraFieldValue
	if err := json.Unmarshal(raw, &objArr); err == nil && len(objArr) > 0 {
		if objArr[0].Value != "" {
			return objArr[0].Value
		}
		return objArr[0].Name
	}
	return extractString(raw)
}

func extractOptionValue(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	// Try as array of objects with value field
	var arr []jiraFieldValue
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		if arr[0].Value != "" {
			return arr[0].Value
		}
		return arr[0].Name
	}
	// Try as single object
	var obj jiraFieldValue
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Value != "" {
			return obj.Value
		}
		return obj.Name
	}
	return extractString(raw)
}

func extractStatusName(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	var status struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &status); err == nil {
		return status.Name
	}
	return extractString(raw)
}

func extractVersionName(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	// Try as array of version objects (fixVersions style)
	var verArr []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &verArr); err == nil && len(verArr) > 0 {
		return verArr[0].Name
	}
	// Try as single version object
	var ver struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &ver); err == nil && ver.Name != "" {
		return ver.Name
	}
	return extractString(raw)
}

func sumNodeFields(fields map[string]json.RawMessage) int {
	nodeFields := []string{
		"customfield_11197", // 控制节点
		"customfield_11199", // 控制存储节点
		"customfield_11201", // 计算节点
		"customfield_11203", // 存储节点
		"customfield_11205", // 计算存储节点
		"customfield_11207", // 融合节点
	}
	total := 0
	for _, f := range nodeFields {
		raw := fields[f]
		if raw == nil || string(raw) == "null" {
			continue
		}
		var n float64
		if err := json.Unmarshal(raw, &n); err == nil {
			total += int(n)
		}
	}
	return total
}

// QuickQueryOpsEnv handles GET /api/ops-env/quick-query?search=XXX
// This is the API endpoint for the "过保维保查询" skill to enable natural-language
// warranty queries via chat. Supports search by customer name, project name, or CSE key.
func (h *OpsEnvHandler) QuickQueryOpsEnv(c *gin.Context) {
	search := c.Query("search")
	if search == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "search参数不能为空，请提供客户名称、项目名称或CSE编号"})
		return
	}

	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	query := repository.DB.Model(&model.OpsEnvironment{}).
		Where("env_type != ? AND env_type != ?", "POC", "poc").
		Where("customer_name LIKE ? OR project_name LIKE ? OR cse_name LIKE ? OR cse_key LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")

	// Optional status filter
	switch status {
	case "in_progress":
		query = query.Where("status IN (?, ?)", "正在进行", "In Progress")
	case "done":
		query = query.Where("status IN (?, ?)", "已完成", "Done")
	case "discarded":
		query = query.Where("status IN (?, ?)", "已弃用", "Discarded")
	}

	var total int64
	query.Count(&total)

	var items []model.OpsEnvironment
	err := query.Order("maintain_end ASC").Limit(limit).Find(&items).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":  items,
			"total":  total,
			"search": search,
		},
	})
}
