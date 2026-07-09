package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/service"
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
	rollerPass, rollerTs, rollerErr := h.svc.QuickGenerateTotp(customer, project, "V5", "roller")

	// Step 3: Generate dynamic password (calls external TOTP server)
	dynamicPass, dynamicTs, dynamicErr := h.svc.QuickGenerateTotp(customer, project, "V5", "dynamic")

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
