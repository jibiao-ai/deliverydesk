package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/xuri/excelize/v2"
)

// TotpHandler handles TOTP application and audit endpoints
type TotpHandler struct {
	svc *service.TotpService
}

func NewTotpHandler() *TotpHandler {
	return &TotpHandler{svc: service.NewTotpService()}
}

// getCurrentUser retrieves user from context (using user_id set by AuthMiddleware)
func (h *TotpHandler) getCurrentUser(c *gin.Context) *model.User {
	userID := c.GetUint("user_id")
	if userID == 0 {
		return nil
	}
	var user model.User
	if err := repository.DB.First(&user, userID).Error; err != nil {
		return nil
	}
	return &user
}

// CreateTotpApplication handles POST /api/totp/apply
func (h *TotpHandler) CreateTotpApplication(c *gin.Context) {
	currentUser := h.getCurrentUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "用户信息获取失败"})
		return
	}

	var req struct {
		Issue        string `json:"issue"`
		IssueSummary string `json:"issue_summary"`
		Customer     string `json:"customer" binding:"required"`
		Project      string `json:"project" binding:"required"`
		Version      string `json:"version"`
		TotpType     string `json:"totp_type"`
		Reason       string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请填写必填项: " + err.Error()})
		return
	}

	if req.Version == "" {
		req.Version = "V5"
	}
	if req.TotpType == "" {
		req.TotpType = "roller"
	}

	app := &model.TotpApplication{
		UserID:       currentUser.ID,
		Username:     currentUser.DisplayName,
		Issue:        req.Issue,
		IssueSummary: req.IssueSummary,
		Customer:     req.Customer,
		Project:      req.Project,
		Version:      req.Version,
		TotpType:     req.TotpType,
		Reason:       req.Reason,
	}
	if app.Username == "" {
		app.Username = currentUser.Username
	}

	if err := h.svc.CreateApplication(app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": app})
}

// ListMyApplications handles GET /api/totp/my-applications
func (h *TotpHandler) ListMyApplications(c *gin.Context) {
	currentUser := h.getCurrentUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "用户信息获取失败"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	apps, total, err := h.svc.ListMyApplications(currentUser.ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"items": apps,
		"total": total,
		"page":  page,
	}})
}

// ListPendingReviews handles GET /api/totp/pending-reviews (admin only)
func (h *TotpHandler) ListPendingReviews(c *gin.Context) {
	currentUser := h.getCurrentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var auditorID uint
	if currentUser != nil {
		auditorID = currentUser.ID
	}

	apps, total, err := h.svc.ListPendingReviews(auditorID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"items": apps,
		"total": total,
		"page":  page,
	}})
}

// ListAllApplications handles GET /api/totp/all (admin only)
func (h *TotpHandler) ListAllApplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.DefaultQuery("status", "all")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	apps, total, err := h.svc.ListAllApplications(page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"items": apps,
		"total": total,
		"page":  page,
	}})
}

// AuditApplication handles POST /api/totp/audit
func (h *TotpHandler) AuditApplication(c *gin.Context) {
	currentUser := h.getCurrentUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "用户信息获取失败"})
		return
	}

	var req struct {
		IDs      []uint `json:"ids" binding:"required"`
		Approved bool   `json:"approved"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	auditorName := currentUser.DisplayName
	if auditorName == "" {
		auditorName = currentUser.Username
	}

	count, err := h.svc.BatchAudit(req.IDs, currentUser.ID, auditorName, req.Approved, req.Remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审核完成", "data": gin.H{"count": count}})
}

// GetSettings handles GET /api/settings
func (h *TotpHandler) GetSettings(c *gin.Context) {
	category := c.Query("category")
	if category != "" {
		settings := h.svc.GetSettings(category)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": settings})
		return
	}
	allSettings := h.svc.GetAllSettings()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": allSettings})
}

// UpdateSettings handles PUT /api/settings
func (h *TotpHandler) UpdateSettings(c *gin.Context) {
	var req struct {
		Settings []model.SystemSetting `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	if err := h.svc.UpdateSettings(req.Settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "设置已更新"})
}

// CheckIssue handles GET /api/totp/check-issue?issue=ECSDESK-xxx
// Queries local cache first, then Jira API to auto-fill customer and project name
func (h *TotpHandler) CheckIssue(c *gin.Context) {
	issue := c.Query("issue")
	if issue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "issue参数不能为空"})
		return
	}

	result, err := h.svc.CheckIssueFromJira(issue)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// SyncJiraIssues handles POST /api/totp/sync-jira (admin only)
// Triggers a manual Jira data sync
func (h *TotpHandler) SyncJiraIssues(c *gin.Context) {
	jiraSvc := service.GetJiraService()
	count, err := jiraSvc.SyncJiraIssues()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步完成", "data": gin.H{"count": count}})
}

// ListJiraCache handles GET /api/totp/jira-cache (for auto-complete)
func (h *TotpHandler) ListJiraCache(c *gin.Context) {
	keyword := c.DefaultQuery("keyword", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	jiraSvc := service.GetJiraService()
	items, total, err := jiraSvc.GetCachedIssues(keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"items": items,
		"total": total,
		"page":  page,
	}})
}

// GetAdminList handles GET /api/totp/admins
// Returns list of admin users for the "assign auditor" dropdown
func (h *TotpHandler) GetAdminList(c *gin.Context) {
	admins, err := h.svc.GetAdminUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": admins})
}

// ExportAllApplications handles GET /api/admin/totp/export
// Exports all TOTP application records to an Excel file
func (h *TotpHandler) ExportAllApplications(c *gin.Context) {
	status := c.DefaultQuery("status", "all")

	// Fetch all records without pagination
	var apps []model.TotpApplication
	query := repository.DB.Model(&model.TotpApplication{})
	if status != "" && status != "all" {
		query = query.Where("audit_status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询数据失败: " + err.Error()})
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	sheet := "双因子申请记录"
	f.SetSheetName("Sheet1", sheet)

	// Define styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#B4C6E7", Style: 1},
			{Type: "top", Color: "#B4C6E7", Style: 1},
			{Type: "right", Color: "#B4C6E7", Style: 1},
			{Type: "bottom", Color: "#B4C6E7", Style: 1},
		},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9E2F3", Style: 1},
			{Type: "top", Color: "#D9E2F3", Style: 1},
			{Type: "right", Color: "#D9E2F3", Style: 1},
			{Type: "bottom", Color: "#D9E2F3", Style: 1},
		},
	})

	dataStyleAlt, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#F2F7FC"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9E2F3", Style: 1},
			{Type: "top", Color: "#D9E2F3", Style: 1},
			{Type: "right", Color: "#D9E2F3", Style: 1},
			{Type: "bottom", Color: "#D9E2F3", Style: 1},
		},
	})

	approvedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#2E7D32"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9E2F3", Style: 1},
			{Type: "top", Color: "#D9E2F3", Style: 1},
			{Type: "right", Color: "#D9E2F3", Style: 1},
			{Type: "bottom", Color: "#D9E2F3", Style: 1},
		},
	})

	rejectedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#C62828"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9E2F3", Style: 1},
			{Type: "top", Color: "#D9E2F3", Style: 1},
			{Type: "right", Color: "#D9E2F3", Style: 1},
			{Type: "bottom", Color: "#D9E2F3", Style: 1},
		},
	})

	pendingStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "#F57F17"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#D9E2F3", Style: 1},
			{Type: "top", Color: "#D9E2F3", Style: 1},
			{Type: "right", Color: "#D9E2F3", Style: 1},
			{Type: "bottom", Color: "#D9E2F3", Style: 1},
		},
	})

	// Set column widths
	f.SetColWidth(sheet, "A", "A", 6)   // 序号
	f.SetColWidth(sheet, "B", "B", 18)  // 申请时间
	f.SetColWidth(sheet, "C", "C", 12)  // 申请人
	f.SetColWidth(sheet, "D", "D", 16)  // 工单号
	f.SetColWidth(sheet, "E", "E", 30)  // 工单标题
	f.SetColWidth(sheet, "F", "F", 18)  // 客户
	f.SetColWidth(sheet, "G", "G", 20)  // 项目
	f.SetColWidth(sheet, "H", "H", 8)   // 版本
	f.SetColWidth(sheet, "I", "I", 10)  // 类型
	f.SetColWidth(sheet, "J", "J", 12)  // 审批人
	f.SetColWidth(sheet, "K", "K", 10)  // 状态
	f.SetColWidth(sheet, "L", "L", 18)  // 审批时间
	f.SetColWidth(sheet, "M", "M", 25)  // 申请原因

	// Write header
	headers := []string{"序号", "申请时间", "申请人", "工单号", "工单标题", "客户", "项目", "版本", "类型", "审批人", "状态", "审批时间", "申请原因"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetRowHeight(sheet, 1, 24)
	f.SetCellStyle(sheet, "A1", "M1", headerStyle)

	// Write data rows
	statusText := map[string]string{"pending": "待审核", "approved": "已通过", "rejected": "已拒绝"}
	typeText := map[string]string{"roller": "Roller", "totp": "动态密码"}

	for i, app := range apps {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), app.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), app.Username)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), app.Issue)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), app.IssueSummary)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), app.Customer)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), app.Project)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), app.Version)
		typeLabel := typeText[app.TotpType]
		if typeLabel == "" {
			typeLabel = app.TotpType
		}
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), typeLabel)

		// Auditor: show assigned or actual
		auditor := app.AuditorName
		if auditor == "" {
			auditor = app.AssignedAuditorName
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), auditor)

		// Status
		stLabel := statusText[app.AuditStatus]
		if stLabel == "" {
			stLabel = app.AuditStatus
		}
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), stLabel)

		// Audit time
		auditTime := ""
		if app.AuditTime != nil {
			auditTime = app.AuditTime.Format("2006-01-02 15:04:05")
		}
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), auditTime)

		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), app.Reason)

		// Apply row style (alternating)
		rowStyle := dataStyle
		if i%2 == 1 {
			rowStyle = dataStyleAlt
		}
		for col := 1; col <= 13; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, rowStyle)
		}

		// Apply status-specific style to column K
		statusCell := fmt.Sprintf("K%d", row)
		switch app.AuditStatus {
		case "approved":
			f.SetCellStyle(sheet, statusCell, statusCell, approvedStyle)
		case "rejected":
			f.SetCellStyle(sheet, statusCell, statusCell, rejectedStyle)
		case "pending":
			f.SetCellStyle(sheet, statusCell, statusCell, pendingStyle)
		}

		f.SetRowHeight(sheet, row, 20)
	}

	// Freeze first row (header)
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Set response headers for download
	filename := fmt.Sprintf("双因子申请记录_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "生成Excel失败: " + err.Error()})
		return
	}
}

// QuickQueryTotp handles GET /api/totp/quick-query?issue=ECSL2-50579
// This is the core handler that makes the "双因子申请" skill functional:
// It takes a case number, looks up the customer/project from Jira, then generates
// both Roller OTP and dynamic password, returning everything in one response.
func (h *TotpHandler) QuickQueryTotp(c *gin.Context) {
	issue := c.Query("issue")
	if issue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "issue参数不能为空，请提供Case号（如 ECSL2-50579）"})
		return
	}

	// Step 1: Look up issue from Jira to get customer/project info
	issueInfo, err := h.svc.CheckIssueFromJira(issue)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "Case号查询失败: " + err.Error(),
			"issue":   issue,
		})
		return
	}

	customer := issueInfo["customer"]
	project := issueInfo["project"]
	summary := issueInfo["summary"]

	if customer == "" || project == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "未能从Case号中解析到客户名和项目名",
			"issue":   issue,
			"data":    issueInfo,
		})
		return
	}

	// Step 2: Generate Roller OTP (always available, uses local HMAC-SHA1)
	rollerPass, rollerTs, rollerErr := h.svc.QuickGenerateTotp(customer, project, "", "roller")

	// Step 3: Generate dynamic password (tries V5, V6, V611 automatically)
	dynamicPass, dynamicTs, dynamicErr := h.svc.QuickGenerateTotp(customer, project, "", "dynamic")

	// Build response
	result := gin.H{
		"issue":    issue,
		"summary":  summary,
		"customer": customer,
		"project":  project,
	}

	if rollerErr == nil {
		result["roller_otp"] = rollerPass
		result["roller_timestamp"] = rollerTs
	} else {
		result["roller_otp_error"] = rollerErr.Error()
	}

	if dynamicErr == nil {
		result["dynamic_password"] = dynamicPass
		result["dynamic_timestamp"] = dynamicTs
	} else {
		result["dynamic_password_error"] = dynamicErr.Error()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
