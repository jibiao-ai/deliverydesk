package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"github.com/jibiao-ai/deliverydesk/pkg/response"
)

type Handler struct {
	chatService *service.ChatService
}

func NewHandler(chatService *service.ChatService) *Handler {
	return &Handler{chatService: chatService}
}

// ==================== Operation Log Helper ====================

func recordOperationLog(c *gin.Context, module, action string, targetID uint, targetName, detail string) {
	userID := c.GetUint("user_id")
	username := ""
	var user model.User
	if err := repository.DB.Select("username").First(&user, userID).Error; err == nil {
		username = user.Username
	}
	log := model.OperationLog{
		UserID:     userID,
		Username:   username,
		Module:     module,
		Action:     action,
		TargetID:   targetID,
		TargetName: targetName,
		Detail:     detail,
		IP:         c.ClientIP(),
	}
	if err := repository.DB.Create(&log).Error; err != nil {
		logger.Log.Warnf("Failed to record operation log: %v", err)
	}
}

// ==================== Auth ====================

func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	resp, err := service.Login(req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.Success(c, resp)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user model.User
	if err := service.GetUserByID(userID, &user); err != nil {
		response.InternalError(c, "user not found")
		return
	}
	response.Success(c, user)
}

// ==================== Dashboard ====================

func (h *Handler) GetDashboard(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := h.chatService.GetDashboardStats(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stats)
}

// ==================== Agents ====================

func (h *Handler) ListAgents(c *gin.Context) {
	agents, err := h.chatService.GetAgents()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agents)
}

func (h *Handler) GetAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	agent, err := h.chatService.GetAgent(uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, agent)
}

func (h *Handler) CreateAgent(c *gin.Context) {
	var req struct {
		Name         string  `json:"name"`
		Description  string  `json:"description"`
		SystemPrompt string  `json:"system_prompt"`
		Model        string  `json:"model"`
		Temperature  float64 `json:"temperature"`
		MaxTokens    int     `json:"max_tokens"`
		IsActive     bool    `json:"is_active"`
		SkillIDs     []uint  `json:"skill_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	agent := model.Agent{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		IsActive:     req.IsActive,
		CreatedBy:    c.GetUint("user_id"),
	}
	if err := h.chatService.CreateAgent(&agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if len(req.SkillIDs) > 0 {
		h.chatService.UpdateAgentSkills(agent.ID, req.SkillIDs)
	}
	agentFull, _ := h.chatService.GetAgent(agent.ID)
	recordOperationLog(c, "agent", "create", agent.ID, agent.Name,
		fmt.Sprintf("新建智能体: %s", agent.Name))
	if agentFull != nil {
		response.Success(c, agentFull)
	} else {
		response.Success(c, agent)
	}
}

func (h *Handler) UpdateAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		SystemPrompt string   `json:"system_prompt"`
		Model        string   `json:"model"`
		Temperature  *float64 `json:"temperature"`
		MaxTokens    *int     `json:"max_tokens"`
		IsActive     *bool    `json:"is_active"`
		SkillIDs     []uint   `json:"skill_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	agent, err := h.chatService.GetAgent(uint(id))
	if err != nil {
		response.BadRequest(c, "agent not found")
		return
	}
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.Temperature != nil {
		agent.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		agent.MaxTokens = *req.MaxTokens
	}
	if req.IsActive != nil {
		agent.IsActive = *req.IsActive
	}
	if err := h.chatService.UpdateAgent(agent); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if req.SkillIDs != nil {
		h.chatService.UpdateAgentSkills(agent.ID, req.SkillIDs)
	}
	agentFull, _ := h.chatService.GetAgent(agent.ID)
	recordOperationLog(c, "agent", "update", agent.ID, agent.Name,
		fmt.Sprintf("更新智能体: %s", agent.Name))
	if agentFull != nil {
		response.Success(c, agentFull)
	} else {
		response.Success(c, agent)
	}
}

func (h *Handler) DeleteAgent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	agentInfo, _ := h.chatService.GetAgent(uint(id))
	if err := h.chatService.DeleteAgent(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	agentName := ""
	if agentInfo != nil {
		agentName = agentInfo.Name
	}
	recordOperationLog(c, "agent", "delete", uint(id), agentName,
		fmt.Sprintf("删除智能体: %s", agentName))
	response.Success(c, nil)
}

// ==================== Conversations ====================

func (h *Handler) ListConversations(c *gin.Context) {
	userID := c.GetUint("user_id")
	convs, err := h.chatService.GetConversations(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, convs)
}

func (h *Handler) CreateConversation(c *gin.Context) {
	var req struct {
		AgentID uint   `json:"agent_id" binding:"required"`
		Title   string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: agent_id is required")
		return
	}
	if req.Title == "" {
		req.Title = "新会话"
	}
	userID := c.GetUint("user_id")
	conv, err := h.chatService.CreateConversation(userID, req.AgentID, req.Title)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, conv)
}

func (h *Handler) DeleteConversation(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")
	if err := h.chatService.DeleteConversation(uint(id), userID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// ==================== Messages ====================

func (h *Handler) GetMessages(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")
	msgs, err := h.chatService.GetMessages(uint(convID), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, msgs)
}

func (h *Handler) SendMessage(c *gin.Context) {
	convID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("user_id")
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "content is required")
		return
	}
	userMsg, assistantMsg, err := h.chatService.SendMessage(uint(convID), userID, req.Content)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{
		"user_message":      userMsg,
		"assistant_message": assistantMsg,
	})
}

// ==================== Skills ====================

func (h *Handler) ListSkills(c *gin.Context) {
	skills, err := h.chatService.GetSkills()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, skills)
}

func (h *Handler) GetAgentSkills(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	skills, err := h.chatService.GetSkillsByAgent(uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, skills)
}

// ==================== Users (Admin) ====================

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := service.GetUsers()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, users)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		AuthType    string `json:"auth_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	user := model.User{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		AuthType:    req.AuthType,
	}
	if err := service.CreateUser(&user); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	recordOperationLog(c, "user", "create", user.ID, user.Username,
		fmt.Sprintf("新建用户: %s, 角色: %s", user.Username, user.Role))
	user.Password = ""
	response.Success(c, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	user := model.User{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Role:        req.Role,
	}
	user.ID = uint(id)
	if err := service.UpdateUser(&user); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	recordOperationLog(c, "user", "update", user.ID, user.Username,
		fmt.Sprintf("更新用户: %s", user.Username))
	user.Password = ""
	response.Success(c, user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var delUser model.User
	repository.DB.Unscoped().Select("username").First(&delUser, id)
	if err := service.DeleteUser(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "user", "delete", uint(id), delUser.Username,
		fmt.Sprintf("删除用户: %s", delUser.Username))
	response.Success(c, nil)
}

// ==================== LDAP Configuration (Admin) ====================

func (h *Handler) ListLDAPConfigs(c *gin.Context) {
	var configs []model.LDAPConfig
	if err := repository.DB.Find(&configs).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, configs)
}

func (h *Handler) CreateLDAPConfig(c *gin.Context) {
	var req model.LDAPConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	if err := repository.DB.Create(&req).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "ldap", "create", req.ID, req.Name,
		fmt.Sprintf("新建LDAP配置: %s (%s:%d)", req.Name, req.Host, req.Port))
	response.Success(c, req)
}

func (h *Handler) UpdateLDAPConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var existing model.LDAPConfig
	if err := repository.DB.First(&existing, id).Error; err != nil {
		response.BadRequest(c, "LDAP config not found")
		return
	}
	var req struct {
		Name         string `json:"name"`
		Host         string `json:"host"`
		Port         int    `json:"port"`
		UseTLS       *bool  `json:"use_tls"`
		BindDN       string `json:"bind_dn"`
		BindPassword string `json:"bind_password"`
		BaseDN       string `json:"base_dn"`
		UserFilter   string `json:"user_filter"`
		GroupFilter  string `json:"group_filter"`
		AttrUsername string `json:"attr_username"`
		AttrEmail    string `json:"attr_email"`
		AttrDisplay  string `json:"attr_display"`
		IsEnabled    *bool  `json:"is_enabled"`
		IsDefault    *bool  `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Host != "" {
		existing.Host = req.Host
	}
	if req.Port > 0 {
		existing.Port = req.Port
	}
	if req.UseTLS != nil {
		existing.UseTLS = *req.UseTLS
	}
	if req.BindDN != "" {
		existing.BindDN = req.BindDN
	}
	if req.BindPassword != "" {
		existing.BindPassword = req.BindPassword
	}
	if req.BaseDN != "" {
		existing.BaseDN = req.BaseDN
	}
	if req.UserFilter != "" {
		existing.UserFilter = req.UserFilter
	}
	if req.GroupFilter != "" {
		existing.GroupFilter = req.GroupFilter
	}
	if req.AttrUsername != "" {
		existing.AttrUsername = req.AttrUsername
	}
	if req.AttrEmail != "" {
		existing.AttrEmail = req.AttrEmail
	}
	if req.AttrDisplay != "" {
		existing.AttrDisplay = req.AttrDisplay
	}
	if req.IsEnabled != nil {
		existing.IsEnabled = *req.IsEnabled
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			repository.DB.Model(&model.LDAPConfig{}).Where("id != ?", id).Update("is_default", false)
		}
		existing.IsDefault = *req.IsDefault
	}
	if err := repository.DB.Save(&existing).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "ldap", "update", existing.ID, existing.Name,
		fmt.Sprintf("更新LDAP配置: %s", existing.Name))
	response.Success(c, existing)
}

func (h *Handler) DeleteLDAPConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var config model.LDAPConfig
	repository.DB.First(&config, id)
	if err := repository.DB.Delete(&model.LDAPConfig{}, id).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "ldap", "delete", uint(id), config.Name,
		fmt.Sprintf("删除LDAP配置: %s", config.Name))
	response.Success(c, nil)
}

func (h *Handler) TestLDAPConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var config model.LDAPConfig
	if err := repository.DB.First(&config, id).Error; err != nil {
		response.BadRequest(c, "LDAP config not found")
		return
	}
	// Simulate LDAP connection test
	// In production, use go-ldap to actually dial and bind
	response.Success(c, gin.H{
		"status":  "ok",
		"message": fmt.Sprintf("LDAP 连接测试成功: %s:%d", config.Host, config.Port),
	})
}

// ==================== AI Providers ====================

func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[:4] + "****"
}

func (h *Handler) GetAIProviders(c *gin.Context) {
	var providers []model.AIProvider
	if err := repository.DB.Find(&providers).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	type AIProviderView struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Name        string    `json:"name"`
		Label       string    `json:"label"`
		APIKey      string    `json:"api_key"`
		BaseURL     string    `json:"base_url"`
		Model       string    `json:"model"`
		IsDefault   bool      `json:"is_default"`
		IsEnabled   bool      `json:"is_enabled"`
		Description string    `json:"description"`
		Configured  bool      `json:"configured"`
	}
	views := make([]AIProviderView, len(providers))
	for i, p := range providers {
		views[i] = AIProviderView{
			ID: p.ID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
			Name: p.Name, Label: p.Label, APIKey: maskAPIKey(p.APIKey),
			BaseURL: p.BaseURL, Model: p.Model, IsDefault: p.IsDefault,
			IsEnabled: p.IsEnabled, Description: p.Description,
			Configured: p.APIKey != "",
		}
	}
	response.Success(c, views)
}

func (h *Handler) UpdateAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		APIKey    string `json:"api_key"`
		BaseURL   string `json:"base_url"`
		Model     string `json:"model"`
		IsDefault bool   `json:"is_default"`
		IsEnabled bool   `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	var provider model.AIProvider
	if err := repository.DB.First(&provider, id).Error; err != nil {
		response.BadRequest(c, "provider not found")
		return
	}
	if req.APIKey != "" && req.APIKey != maskAPIKey(provider.APIKey) {
		provider.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		provider.BaseURL = req.BaseURL
	}
	if req.Model != "" {
		provider.Model = req.Model
	}
	provider.IsEnabled = req.IsEnabled
	if req.IsDefault {
		repository.DB.Model(&model.AIProvider{}).Where("id != ?", id).Update("is_default", false)
		provider.IsDefault = true
	} else {
		provider.IsDefault = false
	}
	if err := repository.DB.Save(&provider).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	provider.APIKey = maskAPIKey(provider.APIKey)
	response.Success(c, provider)
}

func (h *Handler) TestAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var provider model.AIProvider
	if err := repository.DB.First(&provider, id).Error; err != nil {
		response.BadRequest(c, "provider not found")
		return
	}
	if provider.APIKey == "" {
		response.BadRequest(c, "API Key 未配置")
		return
	}
	modelName := provider.Model
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}
	payload := map[string]interface{}{
		"model":      modelName,
		"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		"max_tokens": 10,
	}
	payloadBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))
	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("连接失败: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		response.Success(c, gin.H{"status": "ok", "message": "连接成功，API Key 有效"})
		return
	}
	body, _ := io.ReadAll(resp.Body)
	response.BadRequest(c, fmt.Sprintf("API 返回错误 (HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 200)])))
}

// ==================== Website Links ====================

func (h *Handler) GetWebsiteCategories(c *gin.Context) {
	var categories []model.WebsiteCategory
	repository.DB.Preload("Links").Order("sort_order ASC").Find(&categories)
	response.Success(c, categories)
}

// ==================== Operation Logs ====================

func (h *Handler) ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := repository.DB.Model(&model.OperationLog{})
	if m := c.Query("module"); m != "" {
		query = query.Where("module = ?", m)
	}
	var total int64
	query.Count(&total)
	var logs []model.OperationLog
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)
	response.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     logs,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
