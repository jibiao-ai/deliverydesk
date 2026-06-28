package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
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

// ExportWorktime exports worktime data as CSV file
// type=delivery: 交付工时表 (project-user-month-task detail with dates)
// type=personnel: 人员工时表 (user×project matrix)
func (h *WorktimeHandler) ExportWorktime(c *gin.Context) {
	periodType := c.DefaultQuery("period", "month")
	exportType := c.DefaultQuery("type", "delivery")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		now := time.Now()
		startDate, endDate = service.GetDateRange(periodType, now)
	}

	summary, err := h.svc.GetWorktimeStats(periodType, startDate, endDate)
	if err != nil {
		logger.Log.Errorf("ExportWorktime error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "导出失败: " + err.Error(),
		})
		return
	}

	// Generate filename
	var filename string
	if exportType == "personnel" {
		filename = fmt.Sprintf("%s-%s-人员工时.csv", startDate, endDate)
	} else {
		filename = fmt.Sprintf("%s-%s-交付工时.csv", startDate, endDate)
	}

	// Set CSV response headers with UTF-8 BOM for Excel compatibility
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(filename)))

	writer := csv.NewWriter(c.Writer)

	// Write UTF-8 BOM for Excel to recognize Chinese characters
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	if exportType == "personnel" {
		h.exportPersonnelCSV(writer, summary)
	} else {
		h.exportDeliveryCSV(writer, summary)
	}

	writer.Flush()
}

// exportDeliveryCSV generates the delivery worktime CSV
// Format: 项目名称,项目编号,销售,售前,合同号,合同甲方,最终用户,合同签署时间,项目负责人,区域,省份,交付类型,是否重点,客户类型,项目类型,项目状态,执行人,月份,任务名称,实际工时（人天）,成本（人天）,[dates...]
func (h *WorktimeHandler) exportDeliveryCSV(writer *csv.Writer, summary *service.WorktimeSummary) {
	// Header
	header := []string{
		"项目名称", "项目编号", "销售", "售前", "合同号", "合同甲方", "最终用户",
		"合同签署时间", "项目负责人", "区域", "省份", "交付类型", "是否重点",
		"客户类型", "项目类型", "项目状态", "执行人", "月份", "任务名称",
		"实际工时（人天）", "成本（人天）",
	}
	writer.Write(header)

	for _, user := range summary.Users {
		for _, proj := range user.ProjectDetails {
			for _, month := range proj.MonthDetails {
				for _, task := range month.Tasks {
					row := []string{
						proj.ProjectName,                                    // 项目名称
						proj.ProjectNo,                                      // 项目编号
						"",                                                  // 销售
						"",                                                  // 售前
						"",                                                  // 合同号
						proj.ContractParty,                                  // 合同甲方
						proj.EndUser,                                        // 最终用户
						"",                                                  // 合同签署时间
						"",                                                  // 项目负责人
						"",                                                  // 区域
						"",                                                  // 省份
						"",                                                  // 交付类型
						"",                                                  // 是否重点
						"",                                                  // 客户类型
						"",                                                  // 项目类型
						"",                                                  // 项目状态
						user.Name,                                           // 执行人
						month.Month,                                         // 月份
						task.TaskName,                                       // 任务名称
						fmt.Sprintf("%g", task.ManDays),                     // 实际工时（人天）
						fmt.Sprintf("%g", task.CostDays),                    // 成本（人天）
					}
					// Append individual dates as extra columns
					row = append(row, task.Dates...)
					writer.Write(row)
				}
			}
		}
	}
}

// exportPersonnelCSV generates the user×project matrix CSV
// Format: 人名/项目or合同, ProjectNo1, ProjectNo2, ...
//         UserName,       hours1,     hours2,     ...
func (h *WorktimeHandler) exportPersonnelCSV(writer *csv.Writer, summary *service.WorktimeSummary) {
	// Collect all unique project numbers
	projectSet := make(map[string]bool)
	var projectList []string

	for _, user := range summary.Users {
		for _, proj := range user.ProjectDetails {
			if !projectSet[proj.ProjectNo] {
				projectSet[proj.ProjectNo] = true
				projectList = append(projectList, proj.ProjectNo)
			}
		}
	}

	// Header row
	headerRow := []string{"人名/项目or合同"}
	headerRow = append(headerRow, projectList...)
	writer.Write(headerRow)

	// Data rows - one per user
	for _, user := range summary.Users {
		// Build project→hours map for this user
		projHours := make(map[string]float64)
		for _, proj := range user.ProjectDetails {
			projHours[proj.ProjectNo] = proj.TotalHours
		}

		row := []string{user.Name}
		for _, projNo := range projectList {
			h := projHours[projNo]
			if h == 0 {
				row = append(row, "0")
			} else {
				row = append(row, fmt.Sprintf("%g", h))
			}
		}
		writer.Write(row)
	}
}
