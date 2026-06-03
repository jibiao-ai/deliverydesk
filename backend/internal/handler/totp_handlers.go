package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/service"
)

// TotpHandler handles TOTP application and audit endpoints
type TotpHandler struct {
	svc *service.TotpService
}

func NewTotpHandler() *TotpHandler {
	return &TotpHandler{svc: service.NewTotpService()}
}

// CreateTotpApplication handles POST /api/totp/apply
func (h *TotpHandler) CreateTotpApplication(c *gin.Context) {
	user, _ := c.Get("user")
	currentUser := user.(*model.User)

	var req struct {
		Issue    string `json:"issue"`
		Customer string `json:"customer" binding:"required"`
		Project  string `json:"project" binding:"required"`
		Version  string `json:"version"`
		TotpType string `json:"totp_type"`
		Reason   string `json:"reason" binding:"required"`
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
		UserID:   currentUser.ID,
		Username: currentUser.DisplayName,
		Issue:    req.Issue,
		Customer: req.Customer,
		Project:  req.Project,
		Version:  req.Version,
		TotpType: req.TotpType,
		Reason:   req.Reason,
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
	user, _ := c.Get("user")
	currentUser := user.(*model.User)

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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	apps, total, err := h.svc.ListPendingReviews(page, pageSize)
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
	user, _ := c.Get("user")
	currentUser := user.(*model.User)

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
