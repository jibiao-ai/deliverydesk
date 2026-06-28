package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// WorktimeHandler handles worktime management endpoints
type WorktimeHandler struct {
	svc *service.WorktimeService
}

// NewWorktimeHandler creates a new worktime handler
func NewWorktimeHandler() *WorktimeHandler {
	return &WorktimeHandler{
		svc: service.GetWorktimeService(),
	}
}

// GetWorktimeStats returns worktime statistics for the given period
func (h *WorktimeHandler) GetWorktimeStats(c *gin.Context) {
	periodType := c.DefaultQuery("period", "month")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// If no explicit dates provided, calculate from period type
	if startDate == "" || endDate == "" {
		now := time.Now()
		startDate, endDate = service.GetDateRange(periodType, now)
	}

	summary, err := h.svc.GetWorktimeStats(periodType, startDate, endDate)
	if err != nil {
		logger.Log.Errorf("GetWorktimeStats error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "查询工时数据失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": summary,
	})
}

// ListWorktimeUsers returns the list of tracked users
func (h *WorktimeHandler) ListWorktimeUsers(c *gin.Context) {
	users, err := h.svc.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取人员列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": users,
	})
}

// AddWorktimeUser adds a new tracked user
func (h *WorktimeHandler) AddWorktimeUser(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "请输入人员姓名",
		})
		return
	}

	userID := c.GetUint("user_id")
	user, err := h.svc.AddUser(req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "添加人员失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

// RemoveWorktimeUser removes a tracked user
func (h *WorktimeHandler) RemoveWorktimeUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "无效的用户ID",
		})
		return
	}

	if err := h.svc.RemoveUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "删除人员失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已删除",
	})
}

// BatchAddWorktimeUsers adds multiple users at once
func (h *WorktimeHandler) BatchAddWorktimeUsers(c *gin.Context) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "请输入人员姓名列表",
		})
		return
	}

	userID := c.GetUint("user_id")
	users, err := h.svc.BatchAddUsers(req.Names, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "批量添加失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": users,
	})
}
