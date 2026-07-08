package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
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

	// Filter by status
	switch status {
	case "in_progress":
		query = query.Where("status = ?", "In Progress")
	case "done":
		query = query.Where("status = ?", "Done")
	case "discarded":
		query = query.Where("status = ?", "Discarded")
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
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	repository.DB.Model(&model.OpsEnvironment{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts)

	type RegionCount struct {
		Region string `json:"region"`
		Count  int64  `json:"count"`
	}
	var regionCounts []RegionCount
	repository.DB.Model(&model.OpsEnvironment{}).
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

// GetOpsEnvCalendar handles GET /api/ops-env/calendar
// Returns daily discard counts for calendar view
func (h *OpsEnvHandler) GetOpsEnvCalendar(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	viewType := c.DefaultQuery("view", "month") // day, month, year

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
	case "month":
		// Get daily counts for a specific month
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		repository.DB.Model(&model.OpsEnvironment{}).
			Select("DATE_FORMAT(discarded_at, '%Y-%m-%d') as date, count(*) as count").
			Where("discarded_at IS NOT NULL AND discarded_at >= ? AND discarded_at < ?", startDate, endDate).
			Group("DATE_FORMAT(discarded_at, '%Y-%m-%d')").
			Order("date ASC").
			Find(&dayCounts)
	default:
		// day - get a single day's discarded items (use date query param)
		dateStr := c.Query("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		repository.DB.Model(&model.OpsEnvironment{}).
			Select("DATE_FORMAT(discarded_at, '%Y-%m-%d') as date, count(*) as count").
			Where("discarded_at IS NOT NULL AND DATE(discarded_at) = ?", dateStr).
			Group("DATE_FORMAT(discarded_at, '%Y-%m-%d')").
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

// syncOpsEnvFromJira fetches environments from Jira and updates the database
func syncOpsEnvFromJira() error {
	logger.Log.Info("Starting OpsEnvironment sync from Jira...")
	// This will be called by the handler, actual Jira sync logic uses the same
	// credentials from system_settings (jira_server, jira_username, jira_password)
	// For now, we'll let the admin import data via the frontend or use the settings

	// The sync logic is similar to get_case.py but adapted for Go
	// It queries Jira for CustomerEnvironment issues and populates the database
	// This is a placeholder - the actual implementation would use the Jira API
	logger.Log.Info("OpsEnvironment sync: placeholder - configure Jira settings for full sync")
	return nil
}
