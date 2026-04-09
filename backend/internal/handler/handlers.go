package handler

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
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

func (h *Handler) HealthCheck(c *gin.Context) {
	// Check database connectivity
	sqlDB, err := repository.DB.DB()
	dbOK := err == nil && sqlDB.Ping() == nil

	// Check admin user exists and has valid password hash
	var admin model.User
	adminOK := false
	adminHashLen := 0
	if err := repository.DB.Where("username = ?", "admin").First(&admin).Error; err == nil {
		adminOK = true
		adminHashLen = len(admin.Password)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"database":       dbOK,
		"admin_exists":   adminOK,
		"admin_hash_len": adminHashLen,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	resp, err := service.Login(req)
	if err != nil {
		// Record failed login attempt
		log := model.OperationLog{
			UserID:     0,
			Username:   req.Username,
			Module:     "auth",
			Action:     "login_failed",
			TargetName: req.Username,
			Detail:     fmt.Sprintf("登录失败: %s (认证方式: %s)", err.Error(), req.AuthType),
			IP:         c.ClientIP(),
		}
		repository.DB.Create(&log)
		// Return HTTP 200 with error code so frontend can read the message
		// (HTTP 401 would be swallowed by Axios interceptor)
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err.Error(),
		})
		return
	}
	// Record successful login
	log := model.OperationLog{
		UserID:     resp.User.ID,
		Username:   resp.User.Username,
		Module:     "auth",
		Action:     "login",
		TargetID:   resp.User.ID,
		TargetName: resp.User.Username,
		Detail:     fmt.Sprintf("用户登录成功 (角色: %s, 认证方式: %s)", resp.User.Role, resp.User.AuthType),
		IP:         c.ClientIP(),
	}
	repository.DB.Create(&log)
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
		IsPublished  bool    `json:"is_published"`
		IronRules    bool    `json:"iron_rules"`
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
		IsPublished:  req.IsPublished,
		IronRules:    req.IronRules,
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
		IsPublished  *bool    `json:"is_published"`
		IronRules    *bool    `json:"iron_rules"`
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
	if req.IsPublished != nil {
		agent.IsPublished = *req.IsPublished
	}
	if req.IronRules != nil {
		agent.IronRules = *req.IronRules
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
	// Record agent usage
	agentName := ""
	if agent, aErr := h.chatService.GetAgent(req.AgentID); aErr == nil && agent != nil {
		agentName = agent.Name
	}
	recordOperationLog(c, "conversation", "create", conv.ID, req.Title,
		fmt.Sprintf("创建对话: %s (智能体: %s)", req.Title, agentName))
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
	// Record message send as agent usage
	contentPreview := req.Content
	if len(contentPreview) > 50 {
		contentPreview = contentPreview[:50] + "..."
	}
	recordOperationLog(c, "conversation", "send_message", uint(convID), contentPreview,
		fmt.Sprintf("发送消息到对话 #%d", convID))
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

func (h *Handler) GetSkill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	sk, err := h.chatService.GetSkill(uint(id))
	if err != nil {
		response.BadRequest(c, "skill not found")
		return
	}
	response.Success(c, sk)
}

func (h *Handler) CreateSkill(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Category    string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	sk := model.Skill{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Category:    req.Category,
		IsActive:    true,
	}
	if err := h.chatService.CreateSkill(&sk); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "skill", "create", sk.ID, sk.Name,
		fmt.Sprintf("创建技能: %s (类型: %s)", sk.Name, sk.Type))
	response.Success(c, sk)
}

func (h *Handler) UpdateSkill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	sk, err := h.chatService.GetSkill(uint(id))
	if err != nil {
		response.BadRequest(c, "skill not found")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	if req.Name != "" {
		sk.Name = req.Name
	}
	if req.Description != "" {
		sk.Description = req.Description
	}
	if req.Type != "" {
		sk.Type = req.Type
	}
	if req.IsActive != nil {
		sk.IsActive = *req.IsActive
	}
	if err := h.chatService.UpdateSkill(sk); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "skill", "update", sk.ID, sk.Name,
		fmt.Sprintf("更新技能: %s", sk.Name))
	response.Success(c, sk)
}

func (h *Handler) DeleteSkill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	sk, _ := h.chatService.GetSkill(uint(id))
	if err := h.chatService.DeleteSkill(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	name := ""
	if sk != nil {
		name = sk.Name
	}
	recordOperationLog(c, "skill", "delete", uint(id), name,
		fmt.Sprintf("删除技能: %s", name))
	response.Success(c, nil)
}

func (h *Handler) UploadSkillDocument(c *gin.Context) {
	skillID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	sk, err := h.chatService.GetSkill(uint(skillID))
	if err != nil {
		response.BadRequest(c, "skill not found")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	// Save file to upload directory
	uploadDir := "/home/user/webapp/backend/uploads/skills"
	os.MkdirAll(uploadDir, 0755)
	fileName := header.Filename
	filePath := fmt.Sprintf("%s/%d_%s", uploadDir, skillID, fileName)
	dst, err := os.Create(filePath)
	if err != nil {
		response.InternalError(c, "failed to save file")
		return
	}
	_, err = io.Copy(dst, file)
	dst.Close()
	if err != nil {
		response.InternalError(c, "failed to write file")
		return
	}

	// Determine file type
	ext := strings.ToLower(filepath.Ext(fileName))
	fileType := strings.TrimPrefix(ext, ".")

	doc := model.SkillDocument{
		SkillID:  uint(skillID),
		FileName: fileName,
		FilePath: filePath,
		FileType: fileType,
		FileSize: header.Size,
		Status:   "pending",
	}
	if err := h.chatService.AddSkillDocument(&doc); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Index document in background
	go func() {
		if err := h.chatService.IndexSkillDocument(&doc); err != nil {
			logger.Log.Errorf("Failed to index document %s: %v", fileName, err)
		}
	}()

	recordOperationLog(c, "skill", "upload_doc", sk.ID, sk.Name,
		fmt.Sprintf("上传文档: %s (技能: %s)", fileName, sk.Name))
	response.Success(c, doc)
}

func (h *Handler) ReindexSkill(c *gin.Context) {
	skillID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.chatService.ReindexSkill(uint(skillID)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "reindex completed"})
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

// ==================== Published Agents (External API) ====================

func (h *Handler) ListPublishedAgents(c *gin.Context) {
	var agents []model.Agent
	repository.DB.Where("is_active = ? AND is_published = ?", true, true).
		Preload("AgentSkills").Preload("AgentSkills.Skill").Find(&agents)
	response.Success(c, agents)
}

func (h *Handler) PublishedAgentChat(c *gin.Context) {
	agentID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var agent model.Agent
	if err := repository.DB.Where("id = ? AND is_published = ? AND is_active = ?", agentID, true, true).
		Preload("AgentSkills").Preload("AgentSkills.Skill").
		First(&agent).Error; err != nil {
		response.BadRequest(c, "agent not found or not published")
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "message is required")
		return
	}

	// Use the same AI response logic
	var provider model.AIProvider
	if err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error; err != nil {
		if err := repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error; err != nil {
			response.InternalError(c, "AI service not configured")
			return
		}
	}

	aiContent := h.chatService.SendMessageToAgent(agent, provider, req.Message)
	response.Success(c, gin.H{
		"agent":   agent.Name,
		"message": aiContent,
	})
}

// ==================== Users (Admin) ====================

func (h *Handler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := repository.DB.Model(&model.User{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?", like, like, like)
	}

	var total int64
	query.Count(&total)

	var users []model.User
	offset := (page - 1) * pageSize
	query.Order("id ASC").Offset(offset).Limit(pageSize).Find(&users)

	response.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     users,
	})
}

func (h *Handler) CreateUser(c *gin.Context) {
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
		AuthType:    "local", // Manual creation always creates local users; LDAP users are synced via LDAP sync
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
		Name         string  `json:"name"`
		Host         string  `json:"host"`
		Port         int     `json:"port"`
		UseTLS       *bool   `json:"use_tls"`
		BindDN       string  `json:"bind_dn"`
		BindPassword string  `json:"bind_password"`
		BaseDN       string  `json:"base_dn"`
		UserOU       *string `json:"user_ou"` // pointer so we can distinguish empty string from not-sent
		UserFilter   string  `json:"user_filter"`
		GroupFilter  string  `json:"group_filter"`
		AttrUsername string  `json:"attr_username"`
		AttrEmail    string  `json:"attr_email"`
		AttrDisplay  string  `json:"attr_display"`
		IsEnabled    *bool   `json:"is_enabled"`
		IsDefault    *bool   `json:"is_default"`
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
	if req.UserOU != nil {
		existing.UserOU = *req.UserOU // allow clearing (empty string) or setting
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

	// Real LDAP connection test
	conn, err := dialLDAP(config)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("LDAP 连接失败: %v", err))
		return
	}
	defer conn.Close()

	// Try to bind with the configured credentials
	if config.BindDN != "" {
		// Retrieve the bind password (it's stored but hidden from JSON)
		var fullConfig model.LDAPConfig
		repository.DB.Select("bind_password").First(&fullConfig, id)
		if err := conn.Bind(config.BindDN, fullConfig.BindPassword); err != nil {
			response.BadRequest(c, fmt.Sprintf("LDAP Bind 失败: %v", err))
			return
		}
	}

	response.Success(c, gin.H{
		"status":  "ok",
		"message": fmt.Sprintf("LDAP 连接测试成功: %s:%d", config.Host, config.Port),
	})
}

// dialLDAP establishes a connection to the LDAP server
func dialLDAP(config model.LDAPConfig) (*ldap.Conn, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	if config.UseTLS {
		return ldap.DialTLS("tcp", addr, &tls.Config{
			InsecureSkipVerify: true, // Enterprise internal LDAP often uses self-signed certs
		})
	}
	return ldap.Dial("tcp", addr)
}

// SyncLDAPUsers pulls users from all enabled LDAP configurations into the platform
func (h *Handler) SyncLDAPUsers(c *gin.Context) {
	// Get all enabled LDAP configurations
	var ldapConfigs []model.LDAPConfig
	if err := repository.DB.Where("is_enabled = ?", true).Find(&ldapConfigs).Error; err != nil {
		response.InternalError(c, "Failed to load LDAP configs: "+err.Error())
		return
	}
	if len(ldapConfigs) == 0 {
		response.BadRequest(c, "没有已启用的LDAP配置")
		return
	}

	newUsers := 0
	updatedUsers := 0
	var syncErrors []string

	for _, ldapCfg := range ldapConfigs {
		// Retrieve bind password (excluded from JSON serialization)
		var fullCfg model.LDAPConfig
		repository.DB.First(&fullCfg, ldapCfg.ID)

		// Connect to LDAP server
		conn, err := dialLDAP(ldapCfg)
		if err != nil {
			errMsg := fmt.Sprintf("连接 %s:%d 失败: %v", ldapCfg.Host, ldapCfg.Port, err)
			logger.Log.Warnf("LDAP sync: %s", errMsg)
			syncErrors = append(syncErrors, errMsg)
			continue
		}

		// Bind with service account
		if fullCfg.BindDN != "" && fullCfg.BindPassword != "" {
			if err := conn.Bind(fullCfg.BindDN, fullCfg.BindPassword); err != nil {
				conn.Close()
				errMsg := fmt.Sprintf("Bind %s 失败: %v", ldapCfg.Name, err)
				logger.Log.Warnf("LDAP sync: %s", errMsg)
				syncErrors = append(syncErrors, errMsg)
				continue
			}
		}

		// Build search filter
		searchFilter := "(objectClass=person)"
		if ldapCfg.UserFilter != "" {
			// Replace %s placeholder with * for listing all users
			filter := strings.ReplaceAll(ldapCfg.UserFilter, "%s", "*")
			searchFilter = filter
		}

		// Determine attribute names
		attrUsername := ldapCfg.AttrUsername
		if attrUsername == "" {
			attrUsername = "uid"
		}
		attrEmail := ldapCfg.AttrEmail
		if attrEmail == "" {
			attrEmail = "mail"
		}
		attrDisplay := ldapCfg.AttrDisplay
		if attrDisplay == "" {
			attrDisplay = "cn"
		}

		// Determine the search base: use UserOU if specified, otherwise use BaseDN
		searchBase := ldapCfg.BaseDN
		if ldapCfg.UserOU != "" {
			searchBase = ldapCfg.UserOU
		}

		// Search for users with paging to handle large directories (> 100 entries).
		// NOTE: Do NOT include paging control in the search request controls,
		// SearchWithPaging adds its own paging control internally.
		searchReq := ldap.NewSearchRequest(
			searchBase,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			searchFilter,
			[]string{attrUsername, attrEmail, attrDisplay},
			nil, // no manual controls — SearchWithPaging manages paging
		)

		result, err := conn.SearchWithPaging(searchReq, 500)
		conn.Close()

		if err != nil {
			errMsg := fmt.Sprintf("搜索 %s 失败: %v", ldapCfg.Name, err)
			logger.Log.Warnf("LDAP sync: %s", errMsg)
			syncErrors = append(syncErrors, errMsg)
			continue
		}

		logger.Log.Infof("LDAP search returned %d entries from %s (search base: %s, filter: %s)",
			len(result.Entries), ldapCfg.Name, searchBase, searchFilter)

		domain := extractDomainFromBaseDN(ldapCfg.BaseDN)

		// Process each LDAP entry
		skippedEntries := 0
		for _, entry := range result.Entries {
			username := entry.GetAttributeValue(attrUsername)
			if username == "" {
				skippedEntries++
				continue
			}
			email := entry.GetAttributeValue(attrEmail)
			if email == "" {
				email = username + "@" + domain
			}
			displayName := entry.GetAttributeValue(attrDisplay)
			if displayName == "" {
				displayName = username
			}

			// Check if user already exists in DB
			var existing model.User
			dbResult := repository.DB.Where("username = ? AND auth_type = ?", username, "ldap").First(&existing)
			if dbResult.Error != nil {
				// Create new LDAP user
				newUser := model.User{
					Username:    username,
					Password:    "",
					Email:       email,
					DisplayName: displayName,
					Role:        "user",
					AuthType:    "ldap",
				}
				if err := repository.DB.Create(&newUser).Error; err != nil {
					logger.Log.Warnf("Failed to create LDAP user %s: %v", username, err)
					continue
				}
				newUsers++
				logger.Log.Infof("Synced new LDAP user: %s (%s) from %s", username, displayName, ldapCfg.Name)
			} else {
				// Update existing user's email/display name from LDAP
				needsUpdate := false
				if existing.Email != email {
					existing.Email = email
					needsUpdate = true
				}
				if existing.DisplayName != displayName {
					existing.DisplayName = displayName
					needsUpdate = true
				}
				if needsUpdate {
					repository.DB.Model(&existing).Updates(map[string]interface{}{
						"email":        existing.Email,
						"display_name": existing.DisplayName,
					})
					updatedUsers++
				}
			}
		}

		logger.Log.Infof("LDAP sync from %s (search base: %s): found %d entries, skipped %d (empty username)", ldapCfg.Name, searchBase, len(result.Entries), skippedEntries)
	}

	// Count total LDAP users in the platform
	var totalLDAPUsers int64
	repository.DB.Model(&model.User{}).Where("auth_type = ?", "ldap").Count(&totalLDAPUsers)

	recordOperationLog(c, "ldap", "sync_users", 0, "",
		fmt.Sprintf("同步LDAP用户: 新增 %d 人, 更新 %d 人, 总计 %d 个LDAP用户", newUsers, updatedUsers, totalLDAPUsers))

	result := gin.H{
		"new_users":        newUsers,
		"updated_users":    updatedUsers,
		"total_ldap_users": totalLDAPUsers,
	}
	if len(syncErrors) > 0 {
		result["errors"] = syncErrors
	}

	response.Success(c, result)
}

func extractDomainFromBaseDN(baseDN string) string {
	parts := strings.Split(baseDN, ",")
	var domain []string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(strings.ToLower(kv[0])) == "dc" {
			domain = append(domain, strings.TrimSpace(kv[1]))
		}
	}
	if len(domain) > 0 {
		return strings.Join(domain, ".")
	}
	return "example.com"
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
		IconURL     string    `json:"icon_url"`
		Configured  bool      `json:"configured"`
	}
	views := make([]AIProviderView, len(providers))
	for i, p := range providers {
		views[i] = AIProviderView{
			ID: p.ID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
			Name: p.Name, Label: p.Label, APIKey: maskAPIKey(p.APIKey),
			BaseURL: p.BaseURL, Model: p.Model, IsDefault: p.IsDefault,
			IsEnabled: p.IsEnabled, Description: p.Description,
			IconURL: p.IconURL, Configured: p.APIKey != "",
		}
	}
	response.Success(c, views)
}

func (h *Handler) CreateAIProvider(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Label       string `json:"label" binding:"required"`
		APIKey      string `json:"api_key"`
		BaseURL     string `json:"base_url" binding:"required"`
		Model       string `json:"model" binding:"required"`
		IsDefault   bool   `json:"is_default"`
		IsEnabled   bool   `json:"is_enabled"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写必要字段: name, label, base_url, model")
		return
	}
	// Check uniqueness
	var existing model.AIProvider
	if err := repository.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		response.BadRequest(c, fmt.Sprintf("厂商标识 '%s' 已存在", req.Name))
		return
	}
	provider := model.AIProvider{
		Name:        req.Name,
		Label:       req.Label,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		Model:       req.Model,
		IsDefault:   req.IsDefault,
		IsEnabled:   req.IsEnabled,
		Description: req.Description,
		IconURL:     req.IconURL,
	}
	if req.IsDefault {
		repository.DB.Model(&model.AIProvider{}).Where("1=1").Update("is_default", false)
	}
	if err := repository.DB.Create(&provider).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "ai_provider", "create", provider.ID, provider.Label,
		fmt.Sprintf("新增AI模型厂商: %s (%s)", provider.Label, provider.Name))
	provider.APIKey = maskAPIKey(provider.APIKey)
	response.Success(c, provider)
}

func (h *Handler) DeleteAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var provider model.AIProvider
	if err := repository.DB.First(&provider, id).Error; err != nil {
		response.BadRequest(c, "provider not found")
		return
	}
	if err := repository.DB.Delete(&model.AIProvider{}, id).Error; err != nil {
		response.InternalError(c, err.Error())
		return
	}
	recordOperationLog(c, "ai_provider", "delete", uint(id), provider.Label,
		fmt.Sprintf("删除AI模型厂商: %s (%s)", provider.Label, provider.Name))
	response.Success(c, nil)
}

func (h *Handler) UpdateAIProvider(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		Label       string `json:"label"`
		APIKey      string `json:"api_key"`
		BaseURL     string `json:"base_url"`
		Model       string `json:"model"`
		IsDefault   bool   `json:"is_default"`
		IsEnabled   bool   `json:"is_enabled"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
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
	if req.Label != "" {
		provider.Label = req.Label
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
	if req.Description != "" {
		provider.Description = req.Description
	}
	if req.IconURL != "" {
		provider.IconURL = req.IconURL
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
	recordOperationLog(c, "ai_provider", "update", provider.ID, provider.Label,
		fmt.Sprintf("更新AI模型厂商: %s", provider.Label))
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
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
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
