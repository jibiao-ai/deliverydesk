package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"gorm.io/gorm"
)

type ChatService struct{}

func NewChatService() *ChatService {
	return &ChatService{}
}

// ==================== Dashboard ====================

func (s *ChatService) GetDashboardStats(userID uint) (map[string]interface{}, error) {
	var agentCount int64
	repository.DB.Model(&model.Agent{}).Where("is_active = ?", true).Count(&agentCount)

	var aiModelCount int64
	repository.DB.Model(&model.AIProvider{}).Where("api_key != '' AND is_enabled = ?", true).Count(&aiModelCount)

	var convCount int64
	repository.DB.Model(&model.Conversation{}).Where("user_id = ?", userID).Count(&convCount)

	var linkCount int64
	repository.DB.Model(&model.WebsiteLink{}).Count(&linkCount)

	// Recent conversations
	var recentConvs []model.Conversation
	repository.DB.Where("user_id = ?", userID).Order("updated_at DESC").Limit(6).Find(&recentConvs)

	return map[string]interface{}{
		"agents":               agentCount,
		"ai_models":            aiModelCount,
		"conversations":        convCount,
		"website_links":        linkCount,
		"recent_conversations": recentConvs,
	}, nil
}

// ==================== Agents ====================

func (s *ChatService) GetAgents() ([]model.Agent, error) {
	var agents []model.Agent
	err := repository.DB.Preload("AgentSkills").Preload("AgentSkills.Skill").Find(&agents).Error
	return agents, err
}

func (s *ChatService) GetAgent(id uint) (*model.Agent, error) {
	var agent model.Agent
	err := repository.DB.Preload("AgentSkills").Preload("AgentSkills.Skill").First(&agent, id).Error
	return &agent, err
}

func (s *ChatService) CreateAgent(agent *model.Agent) error {
	return repository.DB.Create(agent).Error
}

func (s *ChatService) UpdateAgent(agent *model.Agent) error {
	return repository.DB.Save(agent).Error
}

func (s *ChatService) DeleteAgent(id uint) error {
	repository.DB.Where("agent_id = ?", id).Delete(&model.AgentSkill{})
	return repository.DB.Delete(&model.Agent{}, id).Error
}

func (s *ChatService) UpdateAgentSkills(agentID uint, skillIDs []uint) error {
	repository.DB.Where("agent_id = ?", agentID).Delete(&model.AgentSkill{})
	for _, sid := range skillIDs {
		as := model.AgentSkill{AgentID: agentID, SkillID: sid}
		repository.DB.Create(&as)
	}
	return nil
}

// ==================== Skills ====================

func (s *ChatService) GetSkills() ([]model.Skill, error) {
	var skills []model.Skill
	err := repository.DB.Find(&skills).Error
	return skills, err
}

func (s *ChatService) GetSkillsByAgent(agentID uint) ([]model.Skill, error) {
	var agentSkills []model.AgentSkill
	repository.DB.Where("agent_id = ?", agentID).Preload("Skill").Find(&agentSkills)
	skills := make([]model.Skill, 0, len(agentSkills))
	for _, as := range agentSkills {
		skills = append(skills, as.Skill)
	}
	return skills, nil
}

// ==================== Conversations ====================

func (s *ChatService) GetConversations(userID uint) ([]model.Conversation, error) {
	var convs []model.Conversation
	err := repository.DB.Where("user_id = ?", userID).Order("updated_at DESC").Find(&convs).Error
	return convs, err
}

func (s *ChatService) CreateConversation(userID, agentID uint, title string) (*model.Conversation, error) {
	conv := model.Conversation{
		Title:   title,
		UserID:  userID,
		AgentID: agentID,
	}
	err := repository.DB.Create(&conv).Error
	return &conv, err
}

func (s *ChatService) DeleteConversation(id, userID uint) error {
	repository.DB.Where("conversation_id = ?", id).Delete(&model.Message{})
	return repository.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Conversation{}).Error
}

// ==================== Messages ====================

func (s *ChatService) GetMessages(convID, userID uint) ([]model.Message, error) {
	var conv model.Conversation
	if err := repository.DB.First(&conv, convID).Error; err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	var msgs []model.Message
	err := repository.DB.Where("conversation_id = ?", convID).Order("created_at ASC").Find(&msgs).Error
	return msgs, err
}

func (s *ChatService) SendMessage(convID, userID uint, content string) (*model.Message, *model.Message, error) {
	var conv model.Conversation
	if err := repository.DB.Preload("Agent").First(&conv, convID).Error; err != nil {
		return nil, nil, err
	}

	// Save user message
	userMsg := model.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
	}
	repository.DB.Create(&userMsg)

	// Get AI response
	aiContent := s.getAIResponse(conv.Agent, content, convID)

	// Save assistant message
	assistantMsg := model.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        aiContent,
	}
	repository.DB.Create(&assistantMsg)

	// Update conversation timestamp
	repository.DB.Model(&conv).Update("updated_at", time.Now())

	return &userMsg, &assistantMsg, nil
}

func (s *ChatService) getAIResponse(agent model.Agent, userContent string, convID uint) string {
	// Get AI provider config
	var provider model.AIProvider
	if err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error; err != nil {
		if err := repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error; err != nil {
			return "抱歉，AI 服务未配置。请管理员前往「模型配置」页面配置 AI 提供商的 API Key。"
		}
	}

	// Build messages for the API call
	messages := []map[string]string{}
	if agent.SystemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": agent.SystemPrompt})
	}

	// Get recent messages for context
	var recentMsgs []model.Message
	repository.DB.Where("conversation_id = ?", convID).Order("created_at DESC").Limit(10).Find(&recentMsgs)
	for i := len(recentMsgs) - 1; i >= 0; i-- {
		messages = append(messages, map[string]string{
			"role":    recentMsgs[i].Role,
			"content": recentMsgs[i].Content,
		})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userContent})

	modelName := agent.Model
	if modelName == "" {
		modelName = provider.Model
	}

	payload := map[string]interface{}{
		"model":      modelName,
		"messages":   messages,
		"max_tokens": agent.MaxTokens,
	}
	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		logger.Log.Errorf("Create AI request failed: %v", err)
		return "AI 请求创建失败"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Errorf("AI request failed: %v", err)
		return "AI 服务请求失败，请稍后重试"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		logger.Log.Errorf("AI API error (HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 500)]))
		return fmt.Sprintf("AI 服务返回错误 (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "AI 响应解析失败"
	}
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return "AI 未返回内容"
}

// ==================== Website Links ====================

func (s *ChatService) GetWebsiteCategories() ([]model.WebsiteCategory, error) {
	var categories []model.WebsiteCategory
	err := repository.DB.Preload("Links", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Order("sort_order ASC").Find(&categories).Error
	return categories, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
