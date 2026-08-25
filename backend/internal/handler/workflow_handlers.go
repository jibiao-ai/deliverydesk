package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/response"
)

// WorkflowHandler handles workflow CRUD endpoints
type WorkflowHandler struct{}

func NewWorkflowHandler() *WorkflowHandler {
	return &WorkflowHandler{}
}

// ListWorkflows handles GET /api/admin/workflows
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	var workflows []model.Workflow
	if err := repository.DB.Order("updated_at DESC").Find(&workflows).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, workflows)
}

// GetWorkflow handles GET /api/admin/workflows/:id
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	var workflow model.Workflow
	if err := repository.DB.First(&workflow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "工作流不存在"})
		return
	}
	response.Success(c, workflow)
}

// CreateWorkflow handles POST /api/admin/workflows
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		FlowData    string `json:"flow_data"`
		IsActive    bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请输入工作流名称"})
		return
	}

	workflow := model.Workflow{
		Name:        req.Name,
		Description: req.Description,
		FlowData:    req.FlowData,
		IsActive:    req.IsActive,
		CreatedBy:   c.GetUint("user_id"),
	}

	if err := repository.DB.Create(&workflow).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, workflow)
}

// UpdateWorkflow handles PUT /api/admin/workflows/:id
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	var workflow model.Workflow
	if err := repository.DB.First(&workflow, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "工作流不存在"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		FlowData    string `json:"flow_data"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.FlowData != "" {
		updates["flow_data"] = req.FlowData
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := repository.DB.Model(&workflow).Updates(updates).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}

	repository.DB.First(&workflow, id)
	response.Success(c, workflow)
}

// DeleteWorkflow handles DELETE /api/admin/workflows/:id
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的ID"})
		return
	}

	if err := repository.DB.Delete(&model.Workflow{}, id).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "删除成功"})
}
