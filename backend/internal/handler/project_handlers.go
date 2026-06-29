package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// ProjectHandler handles project management API requests
type ProjectHandler struct {
	svc *service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		svc: service.GetProjectService(),
	}
}

// GetProjectStats returns aggregated project statistics
func (h *ProjectHandler) GetProjectStats(c *gin.Context) {
	periodType := c.DefaultQuery("period_type", "year") // month, quarter, year
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.svc.GetStats(periodType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "获取项目统计失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}

// GetProjectList returns paginated project list
func (h *ProjectHandler) GetProjectList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	region := c.Query("region")
	projectType := c.Query("project_type")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	projects, total, err := h.svc.GetProjectList(page, pageSize, search, region, projectType, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "获取项目列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":      projects,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// SyncProjects triggers a manual sync from Redmine
func (h *ProjectHandler) SyncProjects(c *gin.Context) {
	go func() {
		logger.Log.Info("[ProjectSync] Manual sync triggered by user")
		if err := h.svc.SyncFromRedmine(); err != nil {
			logger.Log.Warnf("[ProjectSync] Manual sync failed: %v", err)
		} else {
			logger.Log.Info("[ProjectSync] Manual sync completed successfully")
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步任务已启动，请稍后刷新查看"})
}
