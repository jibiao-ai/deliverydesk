package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/skill"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"gorm.io/gorm"
)

// ==================== Active Stream Tracking (for abort) ====================

var (
	activeStreams   = make(map[uint]context.CancelFunc) // key = conversation ID
	activeStreamsMu sync.Mutex
)

func registerStream(convID uint, cancel context.CancelFunc) {
	activeStreamsMu.Lock()
	defer activeStreamsMu.Unlock()
	// Cancel any existing stream for this conversation
	if prev, ok := activeStreams[convID]; ok {
		prev()
	}
	activeStreams[convID] = cancel
}

func unregisterStream(convID uint) {
	activeStreamsMu.Lock()
	defer activeStreamsMu.Unlock()
	delete(activeStreams, convID)
}

// RegisterStream registers a cancel function for an active streaming conversation.
func RegisterStream(convID uint, cancel context.CancelFunc) {
	registerStream(convID, cancel)
}

// UnregisterStream removes the active stream tracker for a conversation.
func UnregisterStream(convID uint) {
	unregisterStream(convID)
}

// AbortStream cancels the active stream for a conversation. Returns true if aborted.
func AbortStream(convID uint) bool {
	activeStreamsMu.Lock()
	defer activeStreamsMu.Unlock()
	if cancel, ok := activeStreams[convID]; ok {
		cancel()
		delete(activeStreams, convID)
		return true
	}
	return false
}

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

	var skillCount int64
	repository.DB.Model(&model.Skill{}).Where("is_active = ?", true).Count(&skillCount)

	// Recent conversations
	var recentConvs []model.Conversation
	repository.DB.Where("user_id = ?", userID).Order("updated_at DESC").Limit(6).Find(&recentConvs)

	return map[string]interface{}{
		"agents":               agentCount,
		"ai_models":            aiModelCount,
		"conversations":        convCount,
		"website_links":        linkCount,
		"skills":               skillCount,
		"recent_conversations": recentConvs,
	}, nil
}

// GetActivityHeatmap returns daily message counts for the past 6 months
// If userID is 0, returns all system activity; otherwise filters by user
func (s *ChatService) GetActivityHeatmap(userID uint) ([]map[string]interface{}, error) {
	sixMonthsAgo := time.Now().AddDate(0, -6, 0).Format("2006-01-02")

	type DailyCount struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	var results []DailyCount
	query := repository.DB.Model(&model.Message{}).
		Select("DATE(messages.created_at) as date, COUNT(*) as count").
		Joins("JOIN conversations ON conversations.id = messages.conversation_id AND conversations.deleted_at IS NULL").
		Where("messages.created_at >= ? AND messages.deleted_at IS NULL", sixMonthsAgo)

	if userID > 0 {
		query = query.Where("conversations.user_id = ?", userID)
	}

	err := query.Group("DATE(messages.created_at)").
		Order("date ASC").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(results))
	for i, r := range results {
		data[i] = map[string]interface{}{
			"date":  r.Date,
			"count": r.Count,
		}
	}
	return data, nil
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
	err := repository.DB.Preload("Documents").Find(&skills).Error
	return skills, err
}

func (s *ChatService) GetSkill(id uint) (*model.Skill, error) {
	var sk model.Skill
	err := repository.DB.Preload("Documents").First(&sk, id).Error
	return &sk, err
}

func (s *ChatService) CreateSkill(sk *model.Skill) error {
	return repository.DB.Create(sk).Error
}

func (s *ChatService) UpdateSkill(sk *model.Skill) error {
	return repository.DB.Save(sk).Error
}

func (s *ChatService) DeleteSkill(id uint) error {
	// Clear indexed chunks
	skill.GetStore().ClearSkill(id)
	// Delete documents
	repository.DB.Where("skill_id = ?", id).Delete(&model.SkillDocument{})
	// Remove agent-skill links
	repository.DB.Where("skill_id = ?", id).Delete(&model.AgentSkill{})
	return repository.DB.Delete(&model.Skill{}, id).Error
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

// ==================== Skill Documents ====================

func (s *ChatService) AddSkillDocument(doc *model.SkillDocument) error {
	if err := repository.DB.Create(doc).Error; err != nil {
		return err
	}
	return nil
}

// DeleteSkillDocument removes a document from the database, the filesystem,
// and the in-memory chunk store. It also updates the parent skill's counters.
func (s *ChatService) DeleteSkillDocument(docID uint) error {
	var doc model.SkillDocument
	if err := repository.DB.First(&doc, docID).Error; err != nil {
		return fmt.Errorf("document not found")
	}

	// Remove indexed chunks from in-memory store
	removed := skill.GetStore().RemoveDocument(doc.SkillID, doc.ID)
	logger.Log.Infof("DeleteSkillDocument: removed %d chunks for doc %d (%s) from skill %d",
		removed, doc.ID, doc.FileName, doc.SkillID)

	// Delete the physical file (ignore errors — file may already be gone)
	if doc.FilePath != "" {
		os.Remove(doc.FilePath)
	}

	// Delete from database
	if err := repository.DB.Delete(&model.SkillDocument{}, docID).Error; err != nil {
		return err
	}

	// Update skill doc_count and chunk_count
	s.updateSkillCounts(doc.SkillID)

	return nil
}

// updateSkillCounts recalculates the doc_count and chunk_count for a skill.
func (s *ChatService) updateSkillCounts(skillID uint) {
	var docCount int64
	repository.DB.Model(&model.SkillDocument{}).Where("skill_id = ?", skillID).Count(&docCount)

	var totalChunks int64
	repository.DB.Model(&model.SkillDocument{}).Where("skill_id = ?", skillID).
		Select("COALESCE(SUM(chunks), 0)").Scan(&totalChunks)

	repository.DB.Model(&model.Skill{}).Where("id = ?", skillID).Updates(map[string]interface{}{
		"doc_count":   docCount,
		"chunk_count": totalChunks,
	})
}

func (s *ChatService) IndexSkillDocument(doc *model.SkillDocument) error {
	// Update status
	repository.DB.Model(doc).Update("status", "processing")

	// Parse the document to get text content
	content, err := skill.ParseDocument(doc.FilePath)
	if err != nil {
		repository.DB.Model(doc).Updates(map[string]interface{}{
			"status": "error",
		})
		return err
	}

	// Index the parsed content into in-memory store
	chunks := skill.GetStore().IndexDocument(doc.SkillID, doc.ID, doc.FileName, content)
	if chunks == 0 {
		repository.DB.Model(doc).Updates(map[string]interface{}{
			"status": "error",
		})
		return fmt.Errorf("document %s produced 0 chunks after parsing", doc.FileName)
	}

	// Save content to DB so WarmUp can reload it after server restart
	// This is critical: without persisted content, the knowledge store will be empty after restart
	repository.DB.Model(doc).Updates(map[string]interface{}{
		"status":  "ready",
		"chunks":  chunks,
		"content": content,
	})

	// Update skill chunk count
	totalChunks := skill.GetStore().GetChunkCount(doc.SkillID)
	var docCount int64
	repository.DB.Model(&model.SkillDocument{}).Where("skill_id = ? AND status = ?", doc.SkillID, "ready").Count(&docCount)
	repository.DB.Model(&model.Skill{}).Where("id = ?", doc.SkillID).Updates(map[string]interface{}{
		"doc_count":   docCount,
		"chunk_count": totalChunks,
	})

	logger.Log.Infof("IndexSkillDocument: '%s' for skill %d: %d chunks, content saved to DB (%d bytes)",
		doc.FileName, doc.SkillID, chunks, len(content))
	return nil
}

// IndexSkillDocumentFromContent indexes a document from already-parsed content
func (s *ChatService) IndexSkillDocumentFromContent(doc *model.SkillDocument, content string) error {
	repository.DB.Model(doc).Update("status", "processing")

	chunks := skill.GetStore().IndexDocument(doc.SkillID, doc.ID, doc.FileName, content)

	repository.DB.Model(doc).Updates(map[string]interface{}{
		"status":  "ready",
		"chunks":  chunks,
		"content": content,
	})

	totalChunks := skill.GetStore().GetChunkCount(doc.SkillID)
	var docCount int64
	repository.DB.Model(&model.SkillDocument{}).Where("skill_id = ? AND status = ?", doc.SkillID, "ready").Count(&docCount)
	repository.DB.Model(&model.Skill{}).Where("id = ?", doc.SkillID).Updates(map[string]interface{}{
		"doc_count":   docCount,
		"chunk_count": totalChunks,
	})

	logger.Log.Infof("Indexed document '%s' for skill %d: %d chunks", doc.FileName, doc.SkillID, chunks)
	return nil
}

// ReindexSkill reloads all documents for a skill from the database.
// If a document has no stored content but has a file_path, it will re-parse
// the file and save the content to DB for future warm-up.
func (s *ChatService) ReindexSkill(skillID uint) error {
	skill.GetStore().ClearSkill(skillID)

	var docs []model.SkillDocument
	repository.DB.Where("skill_id = ? AND status = ?", skillID, "ready").Find(&docs)

	for _, doc := range docs {
		if doc.Content != "" {
			skill.GetStore().IndexDocument(skillID, doc.ID, doc.FileName, doc.Content)
		} else if doc.FilePath != "" {
			content, err := skill.ParseDocument(doc.FilePath)
			if err != nil {
				logger.Log.Warnf("ReindexSkill: failed to parse %s: %v", doc.FileName, err)
				continue
			}
			chunks := skill.GetStore().IndexDocument(skillID, doc.ID, doc.FileName, content)
			// Save content to DB for future warm-up
			repository.DB.Model(&doc).Updates(map[string]interface{}{
				"content": content,
				"chunks":  chunks,
			})
			logger.Log.Infof("ReindexSkill: re-parsed and saved content for %s (%d chunks, %d bytes)",
				doc.FileName, chunks, len(content))
		}
	}

	totalChunks := skill.GetStore().GetChunkCount(skillID)
	repository.DB.Model(&model.Skill{}).Where("id = ?", skillID).Update("chunk_count", totalChunks)
	return nil
}

// WarmUpSkillStore loads all skill documents from the database into the in-memory
// ChunkStore. This must be called at server startup so the RAG pipeline has data.
func (s *ChatService) WarmUpSkillStore() {
	var skills []model.Skill
	repository.DB.Where("is_active = ?", true).Find(&skills)

	totalChunks := 0
	totalDocs := 0
	skippedDocs := 0
	reparsedDocs := 0
	for _, sk := range skills {
		var docs []model.SkillDocument
		repository.DB.Where("skill_id = ? AND status = ?", sk.ID, "ready").Find(&docs)
		if len(docs) == 0 {
			logger.Log.Infof("WarmUp: skill '%s' (ID %d, type=%s) has 0 ready documents, skipping", sk.Name, sk.ID, sk.Type)
			continue
		}

		for _, doc := range docs {
			if doc.Content != "" {
				// Validate content quality before indexing
				if isContentLowQuality(doc.Content) {
					logger.Log.Warnf("WarmUp: doc '%s' (skill %d) has low-quality content (possibly garbled xlsx). Attempting re-parse from file...",
						doc.FileName, sk.ID)
					if doc.FilePath != "" {
						content, err := skill.ParseDocument(doc.FilePath)
						if err == nil && !isContentLowQuality(content) {
							n := skill.GetStore().IndexDocument(sk.ID, doc.ID, doc.FileName, content)
							totalChunks += n
							totalDocs++
							reparsedDocs++
							// Update DB with better content
							repository.DB.Model(&doc).Updates(map[string]interface{}{
								"content": content,
								"chunks":  n,
							})
							logger.Log.Infof("WarmUp: re-parsed doc '%s' from file, quality improved (%d bytes, %d chunks)", doc.FileName, len(content), n)
							continue
						}
					}
					// Fall through to index low-quality content anyway (better than nothing)
					logger.Log.Warnf("WarmUp: re-parse failed for doc '%s', using existing low-quality content", doc.FileName)
				}
				n := skill.GetStore().IndexDocument(sk.ID, doc.ID, doc.FileName, doc.Content)
				totalChunks += n
				totalDocs++
				logger.Log.Debugf("WarmUp: loaded doc '%s' from DB content (%d bytes, %d chunks)", doc.FileName, len(doc.Content), n)
			} else if doc.FilePath != "" {
				// Try to parse from file if content not stored in DB
				content, err := skill.ParseDocument(doc.FilePath)
				if err != nil {
					logger.Log.Warnf("WarmUp: doc '%s' (skill %d) has empty content and file parse failed: %v", doc.FileName, sk.ID, err)
					skippedDocs++
					continue
				}
				n := skill.GetStore().IndexDocument(sk.ID, doc.ID, doc.FileName, content)
				totalChunks += n
				totalDocs++
				// Save content to DB so next restart doesn't need the file
				repository.DB.Model(&doc).Updates(map[string]interface{}{
					"content": content,
					"chunks":  n,
				})
				logger.Log.Infof("WarmUp: re-parsed doc '%s' from file, saved content to DB (%d bytes, %d chunks)", doc.FileName, len(content), n)
			} else {
				logger.Log.Warnf("WarmUp: doc '%s' (skill %d) has no content and no file_path, cannot load", doc.FileName, sk.ID)
				skippedDocs++
			}
		}
		chunkCount := skill.GetStore().GetChunkCount(sk.ID)
		repository.DB.Model(&model.Skill{}).Where("id = ?", sk.ID).Update("chunk_count", chunkCount)
		logger.Log.Infof("WarmUp: skill '%s' (ID %d, type=%s) loaded %d chunks from %d docs", sk.Name, sk.ID, sk.Type, chunkCount, len(docs))
	}
	logger.Log.Infof("WarmUp complete: %d documents loaded, %d chunks total, %d skipped, %d re-parsed, across %d active skills",
		totalDocs, totalChunks, skippedDocs, reparsedDocs, len(skills))
}

// isContentLowQuality checks if content appears to be garbled or low-quality
// (e.g., xlsx content where shared strings weren't resolved, producing only numbers)
func isContentLowQuality(content string) bool {
	if len(content) == 0 {
		return true
	}
	runes := []rune(content)
	textChars := 0
	for _, r := range runes {
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			textChars++
		}
	}
	// If less than 5% of content is actual text characters, it's likely garbled
	ratio := float64(textChars) / float64(len(runes))
	return ratio < 0.05 && len(runes) > 50
}

// ==================== Conversations ====================

func (s *ChatService) GetConversations(userID uint) ([]model.Conversation, error) {
	var convs []model.Conversation
	err := repository.DB.Where("user_id = ?", userID).
		Preload("Agent").
		Order("updated_at DESC").Find(&convs).Error
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
	if err := repository.DB.Preload("Agent").Preload("Agent.AgentSkills").Preload("Agent.AgentSkills.Skill").First(&conv, convID).Error; err != nil {
		return nil, nil, err
	}

	// Save user message
	userMsg := model.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
	}
	repository.DB.Create(&userMsg)

	// Get AI response (with skill-aware RAG)
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

// isDeepSeekModel checks if the model name is a DeepSeek V4 model
func isDeepSeekModel(modelName string) bool {
	m := strings.ToLower(modelName)
	return strings.HasPrefix(m, "deepseek-v4-") ||
		m == "deepseek-chat" || m == "deepseek-reasoner"
}

// isDeepSeekProvider checks if the provider is DeepSeek based on name or base URL
func isDeepSeekProvider(provider model.AIProvider) bool {
	return strings.ToLower(provider.Name) == "deepseek" ||
		strings.Contains(strings.ToLower(provider.BaseURL), "deepseek.com")
}

// buildDeepSeekEndpoint ensures the correct endpoint for DeepSeek V4 API
// DeepSeek V4 uses https://api.deepseek.com/chat/completions (no /v1 prefix)
func buildDeepSeekEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	// Remove trailing /v1 if present (legacy format)
	base = strings.TrimSuffix(base, "/v1")
	return base + "/chat/completions"
}

func (s *ChatService) getAIResponse(agent model.Agent, userContent string, convID uint) string {
	// Get AI provider config
	var provider model.AIProvider
	if err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error; err != nil {
		if err := repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error; err != nil {
			return "抱歉，AI 服务未配置。请管理员前往「模型配置」页面配置 AI 提供商的 API Key。"
		}
	}

	aiConfig := skill.AIConfig{
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
		Model:   provider.Model,
	}

	modelName := agent.Model
	if modelName == "" {
		modelName = provider.Model
	} else {
		aiConfig.Model = modelName
	}

	// Check if agent has ops skills with tool execution capability (e.g., totp-query)
	// This handles skills that have ToolDefs but NO document chunks (skipped by RAG)
	if hasOpsToolSkill(agent, "totp-query") {
		toolResult := s.executeTotpToolCall(userContent)
		if toolResult != "" {
			return toolResult
		}
	}

	// Check if agent has ops-env-warranty skill for warranty/maintenance queries
	if hasOpsToolSkill(agent, "ops-env-warranty") {
		toolResult := s.executeOpsEnvToolCall(userContent)
		if toolResult != "" {
			return toolResult
		}
	}

	// Check if agent has delivery skills with indexed documents - use RAG
	if agent.IronRules || hasIndexedSkills(agent) {
		ragResult := s.runSkillRAG(agent, aiConfig, userContent)
		if ragResult != "" {
			return ragResult
		}
	}

	// Standard AI response (no RAG)
	return s.standardAIResponse(agent, provider, modelName, userContent, convID)
}

// hasIndexedSkills checks if the agent has ANY skill type with indexed document chunks.
// Previously this only checked "delivery" type skills, causing knowledge/community skills
// to be completely ignored by the RAG pipeline.
func hasIndexedSkills(agent model.Agent) bool {
	for _, as := range agent.AgentSkills {
		if as.Skill.IsActive && skill.GetStore().GetChunkCount(as.Skill.ID) > 0 {
			return true
		}
	}
	return false
}

// ==================== Tool Execution for Ops Skills ====================

// caseNumberRegex matches common CSE/ECSL case number patterns
var caseNumberRegex = regexp.MustCompile(`(?i)(ECSL\d?-\d+|ECSDESK-\d+|ECS[A-Z]*-\d+|CSE-\d+|CASE-\d+)`)

// hasOpsToolSkill checks if the agent has an ops skill with totp-query category
func hasOpsToolSkill(agent model.Agent, category string) bool {
	for _, as := range agent.AgentSkills {
		if as.Skill.IsActive && as.Skill.Category == category && as.Skill.ToolDefs != "" {
			return true
		}
	}
	return false
}

// executeTotpToolCall handles the "totp-query" skill execution.
// When a case number is detected in the user message and the agent has the totp-query skill,
// this function calls the TOTP service to generate Roller OTP and dynamic password,
// then returns a formatted result string that can be directly sent to the user.
func (s *ChatService) executeTotpToolCall(userContent string) string {
	// Extract case number from user message
	matches := caseNumberRegex.FindStringSubmatch(userContent)
	if len(matches) == 0 {
		return ""
	}
	caseNumber := strings.ToUpper(matches[1])
	logger.Log.Infof("ToolExec: detected case number '%s' in message, executing TOTP query", caseNumber)

	totpSvc := NewTotpService()

	// Step 1: Look up issue from Jira
	issueInfo, err := totpSvc.CheckIssueFromJira(caseNumber)
	if err != nil {
		logger.Log.Warnf("ToolExec: Jira lookup failed for %s: %v", caseNumber, err)
		return fmt.Sprintf("## 🔐 双因子查询结果\n\n**Case号**: %s\n\n❌ **查询失败**: %s\n\n> 请检查Case号是否正确，或联系管理员确认Jira连接状态。", caseNumber, err.Error())
	}

	customer := issueInfo["customer"]
	project := issueInfo["project"]
	summary := issueInfo["summary"]

	if customer == "" || project == "" {
		return fmt.Sprintf("## 🔐 双因子查询结果\n\n**Case号**: %s\n**摘要**: %s\n\n❌ **无法生成密码**: 未从该Case中解析到客户名称和项目名称。\n\n> 请确认该Case号关联了正确的客户和项目信息。", caseNumber, summary)
	}

	// Step 2: Generate Roller OTP
	rollerPass, rollerTs, rollerErr := totpSvc.QuickGenerateTotp(customer, project, "", "roller")

	// Step 3: Generate dynamic password (tries V5, V6, V611 automatically)
	dynamicPass, dynamicTs, dynamicErr := totpSvc.QuickGenerateTotp(customer, project, "", "dynamic")

	// Build formatted response
	var sb strings.Builder
	sb.WriteString("## 🔐 双因子查询结果\n\n")
	sb.WriteString(fmt.Sprintf("| 字段 | 值 |\n|------|------|\n"))
	sb.WriteString(fmt.Sprintf("| **Case号** | %s |\n", caseNumber))
	if summary != "" {
		sb.WriteString(fmt.Sprintf("| **摘要** | %s |\n", summary))
	}
	sb.WriteString(fmt.Sprintf("| **客户** | %s |\n", customer))
	sb.WriteString(fmt.Sprintf("| **项目** | %s |\n", project))
	sb.WriteString("\n")

	sb.WriteString("### 🎯 Roller 双因子 (OTP)\n\n")
	if rollerErr == nil {
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", rollerPass))
		sb.WriteString(fmt.Sprintf("> 生成时间: %s | 有效期30秒，请尽快使用\n\n", rollerTs))
	} else {
		sb.WriteString(fmt.Sprintf("❌ 生成失败: %s\n\n", rollerErr.Error()))
	}

	sb.WriteString("### 🔑 动态密码\n\n")
	if dynamicErr == nil {
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", dynamicPass))
		sb.WriteString(fmt.Sprintf("> 生成时间: %s\n\n", dynamicTs))
	} else {
		sb.WriteString(fmt.Sprintf("❌ 生成失败: %s\n\n", dynamicErr.Error()))
	}

	sb.WriteString("---\n*由「双因子申请」技能自动生成*")

	logger.Log.Infof("ToolExec: TOTP query successful for case %s (customer=%s, project=%s)", caseNumber, customer, project)
	return sb.String()
}

// executeOpsEnvToolCall handles the "ops-env-warranty" skill execution.
// When the user's message appears to be querying warranty/maintenance info for a customer,
// project, or CSE number, this function queries the OpsEnvironment database and returns
// a formatted result with CSE number, node count, maintain dates, SLA, etc.
func (s *ChatService) executeOpsEnvToolCall(userContent string) string {
	// Extract search keyword from user message
	// Strategy: detect CSE keys directly, or use the entire message as search keyword
	// after removing common query prefixes
	searchKeyword := extractOpsEnvSearchKeyword(userContent)
	if searchKeyword == "" {
		return ""
	}

	logger.Log.Infof("ToolExec: OpsEnv warranty query detected, keyword='%s'", searchKeyword)

	// Query the database directly (same logic as ListOpsEnvironments handler)
	var items []model.OpsEnvironment
	query := repository.DB.Model(&model.OpsEnvironment{}).
		Where("env_type != ? AND env_type != ?", "POC", "poc").
		Where("customer_name LIKE ? OR project_name LIKE ? OR cse_name LIKE ? OR cse_key LIKE ?",
			"%"+searchKeyword+"%", "%"+searchKeyword+"%", "%"+searchKeyword+"%", "%"+searchKeyword+"%")

	var total int64
	query.Count(&total)

	if total == 0 {
		return fmt.Sprintf("## 🔍 运维环境维保查询\n\n**查询关键字**: %s\n\n❌ 未找到匹配的运维环境记录。\n\n> 请尝试使用客户名称、项目名称或CSE编号进行查询。", searchKeyword)
	}

	// Limit to 20 results to avoid overwhelming output
	limit := 20
	if total < int64(limit) {
		limit = int(total)
	}
	err := query.Order("maintain_end ASC").Limit(limit).Find(&items).Error
	if err != nil {
		logger.Log.Warnf("ToolExec: OpsEnv DB query failed: %v", err)
		return fmt.Sprintf("## 🔍 运维环境维保查询\n\n**查询关键字**: %s\n\n❌ 数据库查询失败: %s", searchKeyword, err.Error())
	}

	// Build formatted Markdown table response
	var sb strings.Builder
	sb.WriteString("## 🔍 运维环境维保查询结果\n\n")
	sb.WriteString(fmt.Sprintf("**查询关键字**: %s | **匹配数量**: %d 条", searchKeyword, total))
	if total > int64(limit) {
		sb.WriteString(fmt.Sprintf("（显示前 %d 条）", limit))
	}
	sb.WriteString("\n\n")

	// Table header
	sb.WriteString("| CSE编号 | 客户 | 项目 | 节点数 | 维保开始 | 维保结束 | 状态 | SLA | 运维区域 |\n")
	sb.WriteString("|---------|------|------|--------|----------|----------|------|-----|----------|\n")

	for _, item := range items {
		// Format status with emoji
		statusEmoji := "🟢"
		switch {
		case strings.Contains(item.Status, "Discarded") || strings.Contains(item.Status, "弃用"):
			statusEmoji = "⚫"
		case strings.Contains(item.Status, "Done") || strings.Contains(item.Status, "完成"):
			statusEmoji = "🔵"
		case strings.Contains(item.Status, "Progress") || strings.Contains(item.Status, "进行"):
			statusEmoji = "🟢"
		}

		// Check if warranty is expiring soon (within 90 days)
		maintainEnd := item.MaintainEnd
		if maintainEnd != "" {
			if t, err := time.Parse("2006-01-02", maintainEnd); err == nil {
				daysLeft := int(time.Until(t).Hours() / 24)
				if daysLeft < 0 {
					maintainEnd = fmt.Sprintf("⚠️ **已过期** %s", item.MaintainEnd)
				} else if daysLeft <= 90 {
					maintainEnd = fmt.Sprintf("⏰ %s (%d天)", item.MaintainEnd, daysLeft)
				}
			}
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %s %s | %s | %s |\n",
			item.CSEKey,
			truncStr(item.CustomerName, 12),
			truncStr(item.ProjectName, 15),
			item.NodeCount,
			item.MaintainStart,
			maintainEnd,
			statusEmoji, item.Status,
			item.SLA,
			item.OpsRegion,
		))
	}

	// Add summary stats
	sb.WriteString("\n---\n")
	var totalNodes int
	var expiredCount, expiringCount, activeCount int
	now := time.Now()
	for _, item := range items {
		totalNodes += item.NodeCount
		if item.MaintainEnd != "" {
			if t, err := time.Parse("2006-01-02", item.MaintainEnd); err == nil {
				daysLeft := int(t.Sub(now).Hours() / 24)
				if daysLeft < 0 {
					expiredCount++
				} else if daysLeft <= 90 {
					expiringCount++
				} else {
					activeCount++
				}
			}
		}
	}
	sb.WriteString(fmt.Sprintf("**汇总**: 共 %d 个环境 | 总节点数 %d | ", limit, totalNodes))
	if expiredCount > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ 已过保 %d | ", expiredCount))
	}
	if expiringCount > 0 {
		sb.WriteString(fmt.Sprintf("⏰ 90天内到期 %d | ", expiringCount))
	}
	sb.WriteString(fmt.Sprintf("🟢 正常 %d", activeCount))
	sb.WriteString("\n\n*由「过保维保查询」技能自动生成*")

	logger.Log.Infof("ToolExec: OpsEnv query returned %d results for keyword '%s'", len(items), searchKeyword)
	return sb.String()
}

// extractOpsEnvSearchKeyword extracts a meaningful search keyword from the user message.
// It handles:
// 1. Direct CSE keys (CSE-1234)
// 2. Customer/project names after stripping common query prefixes
// 3. Returns empty string if the message doesn't look like an ops-env query
func extractOpsEnvSearchKeyword(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	// Pattern 1: Direct CSE key reference
	cseRegex := regexp.MustCompile(`(?i)(CSE-\d+)`)
	if matches := cseRegex.FindStringSubmatch(msg); len(matches) > 0 {
		return strings.ToUpper(matches[1])
	}

	// Pattern 2: Strip common query prefixes to extract keyword
	// Common patterns: "查一下XX的维保", "XX客户维保信息", "查询XX", etc.
	prefixes := []string{
		"查一下", "查询", "帮我查", "查", "搜索", "搜",
		"看一下", "看看", "找一下", "找",
		"请查询", "请查", "帮我查询", "帮查",
	}
	suffixes := []string{
		"的维保信息", "的维保", "的过保信息", "的过保", "维保信息",
		"的环境", "环境信息", "的节点", "节点信息",
		"过保了吗", "过保没", "过保情况", "维保情况",
		"的信息", "信息", "情况",
	}

	cleaned := msg
	for _, p := range prefixes {
		cleaned = strings.TrimPrefix(cleaned, p)
	}
	for _, s := range suffixes {
		cleaned = strings.TrimSuffix(cleaned, s)
	}
	cleaned = strings.TrimSpace(cleaned)

	// If cleaned is the same as original (no prefix/suffix stripped) and it's too long,
	// it might not be a warranty query — skip it
	if cleaned == msg && len([]rune(cleaned)) > 20 {
		return ""
	}

	// If cleaned is empty after stripping, the user just typed "维保信息" etc. — skip
	if cleaned == "" {
		return ""
	}

	// Final check: ensure the keyword is reasonable (not too short for single char, not too long)
	runeLen := len([]rune(cleaned))
	if runeLen < 1 || runeLen > 30 {
		return ""
	}

	return cleaned
}

// truncStr truncates a string to maxLen runes with ellipsis
func truncStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

func (s *ChatService) runSkillRAG(agent model.Agent, aiConfig skill.AIConfig, question string) string {
	var allResults []skill.RAGResult

	skillCount := 0
	totalChunks := 0
	for _, as := range agent.AgentSkills {
		sk := as.Skill
		if !sk.IsActive {
			logger.Log.Debugf("RAG: skipping inactive skill '%s' (ID %d)", sk.Name, sk.ID)
			continue
		}
		chunkCount := skill.GetStore().GetChunkCount(sk.ID)
		if chunkCount == 0 {
			logger.Log.Warnf("RAG: skill '%s' (ID %d) has 0 chunks in memory, skipping. Type=%s", sk.Name, sk.ID, sk.Type)
			continue
		}

		logger.Log.Infof("RAG: searching skill '%s' (ID %d, type=%s, chunks=%d) for question: %s",
			sk.Name, sk.ID, sk.Type, chunkCount, question)
		skillCount++
		totalChunks += chunkCount

		result := skill.RunRAG(aiConfig, sk.ID, sk.Name, question, agent.IronRules)
		if !result.Empty {
			logger.Log.Infof("RAG: skill '%s' returned result with confidence %d", sk.Name, result.Confidence)
			allResults = append(allResults, result)
		} else {
			logger.Log.Infof("RAG: skill '%s' returned empty (no relevant data found)", sk.Name)
		}
	}

	logger.Log.Infof("RAG summary: agent '%s' (ID %d) searched %d skills (%d total chunks), got %d results",
		agent.Name, agent.ID, skillCount, totalChunks, len(allResults))

	if len(allResults) == 0 {
		if agent.IronRules {
			return "无有效数据，无法判断。当前绑定的技能知识库中没有与您的问题相关的文档内容。\n\n[置信度: 0/10]\n[低置信度警告]"
		}
		return "" // fall through to standard AI response
	}

	// Combine results from multiple skills
	if len(allResults) == 1 {
		return allResults[0].Answer
	}

	var sb strings.Builder
	sb.WriteString("综合多个技能知识库的查询结果：\n\n")
	for _, r := range allResults {
		sb.WriteString(fmt.Sprintf("### 来自技能「%s」的回答\n", r.SkillName))
		sb.WriteString(r.Answer)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// getKnowledgeContext retrieves relevant chunks from all active skills and returns
// them as context text to inject into the system prompt. This ensures the AI
// prioritizes knowledge base content even when the full RAG synthesis is skipped.
func (s *ChatService) getKnowledgeContext(agent model.Agent, question string) string {
	var contextParts []string
	for _, as := range agent.AgentSkills {
		sk := as.Skill
		if !sk.IsActive {
			continue
		}
		chunks := skill.GetStore().Retrieve(sk.ID, question, 5)
		if len(chunks) == 0 {
			continue
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("【来自技能知识库「%s」的参考资料】\n", sk.Name))
		for i, c := range chunks {
			content := c.Content
			runes := []rune(content)
			if len(runes) > 400 {
				content = string(runes[:400]) + "..."
			}
			sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, content))
		}
		contextParts = append(contextParts, sb.String())
	}
	if len(contextParts) == 0 {
		return ""
	}
	return strings.Join(contextParts, "\n\n")
}

// StreamCallback is called for each token chunk during streaming
type StreamCallback func(token string)

// SendMessageStream is like SendMessage but streams the AI response via a callback.
// It respects the context for cancellation (abort).
func (s *ChatService) SendMessageStream(ctx context.Context, convID, userID uint, content string, onToken StreamCallback) (*model.Message, *model.Message, error) {
	var conv model.Conversation
	if err := repository.DB.Preload("Agent").Preload("Agent.AgentSkills").Preload("Agent.AgentSkills.Skill").First(&conv, convID).Error; err != nil {
		return nil, nil, err
	}

	// Save user message
	userMsg := model.Message{
		ConversationID: convID,
		Role:           "user",
		Content:        content,
	}
	repository.DB.Create(&userMsg)

	// Check for context cancellation before starting AI call
	select {
	case <-ctx.Done():
		return &userMsg, nil, ctx.Err()
	default:
	}

	// Get AI provider
	var provider model.AIProvider
	if err := repository.DB.Where("is_default = ? AND is_enabled = ? AND api_key != ''", true, true).First(&provider).Error; err != nil {
		if err := repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error; err != nil {
			errContent := "AI service not configured"
			onToken(errContent)
			asstMsg := model.Message{ConversationID: convID, Role: "assistant", Content: errContent}
			repository.DB.Create(&asstMsg)
			return &userMsg, &asstMsg, nil
		}
	}

	aiConfig := skill.AIConfig{BaseURL: provider.BaseURL, APIKey: provider.APIKey, Model: provider.Model}
	modelNameStr := conv.Agent.Model
	if modelNameStr == "" {
		modelNameStr = provider.Model
	} else {
		aiConfig.Model = modelNameStr
	}

	// Tool execution check for ops skills (e.g., totp-query) - before RAG
	if hasOpsToolSkill(conv.Agent, "totp-query") {
		toolResult := s.executeTotpToolCall(content)
		if toolResult != "" {
			onToken(toolResult)
			asstMsg := model.Message{ConversationID: convID, Role: "assistant", Content: toolResult}
			repository.DB.Create(&asstMsg)
			repository.DB.Model(&conv).Update("updated_at", time.Now())
			return &userMsg, &asstMsg, nil
		}
	}

	// Tool execution check for ops-env-warranty skill - before RAG
	if hasOpsToolSkill(conv.Agent, "ops-env-warranty") {
		toolResult := s.executeOpsEnvToolCall(content)
		if toolResult != "" {
			onToken(toolResult)
			asstMsg := model.Message{ConversationID: convID, Role: "assistant", Content: toolResult}
			repository.DB.Create(&asstMsg)
			repository.DB.Model(&conv).Update("updated_at", time.Now())
			return &userMsg, &asstMsg, nil
		}
	}

	// RAG check - if RAG produces a result, stream it all at once
	if conv.Agent.IronRules || hasIndexedSkills(conv.Agent) {
		ragResult := s.runSkillRAG(conv.Agent, aiConfig, content)
		if ragResult != "" {
			onToken(ragResult)
			asstMsg := model.Message{ConversationID: convID, Role: "assistant", Content: ragResult}
			repository.DB.Create(&asstMsg)
			repository.DB.Model(&conv).Update("updated_at", time.Now())
			return &userMsg, &asstMsg, nil
		}
	}

	// Streaming AI response
	aiContent, err := s.streamAIResponse(ctx, conv.Agent, provider, modelNameStr, content, convID, onToken)
	if err != nil {
		// Context cancelled (aborted) - save partial content
		if aiContent == "" {
			aiContent = "[回复已中断]"
		} else {
			aiContent += "\n\n[回复已中断]"
		}
	}

	asstMsg := model.Message{ConversationID: convID, Role: "assistant", Content: aiContent}
	repository.DB.Create(&asstMsg)
	repository.DB.Model(&conv).Update("updated_at", time.Now())

	return &userMsg, &asstMsg, nil
}

// streamAIResponse calls the OpenAI-compatible API with stream=true and
// invokes onToken for each delta. Returns the full accumulated content.
func (s *ChatService) streamAIResponse(ctx context.Context, agent model.Agent, provider model.AIProvider, modelName, userContent string, convID uint, onToken StreamCallback) (string, error) {
	// Build messages with knowledge context
	messages := []map[string]string{}

	systemPrompt := agent.SystemPrompt
	knowledgeCtx := s.getKnowledgeContext(agent, userContent)
	if knowledgeCtx != "" {
		if systemPrompt == "" {
			systemPrompt = "你是一个智能助手。"
		}
		systemPrompt += "\n\n【知识库参考资料 - 请优先基于以下内容回答问题，如果参考资料与问题相关请引用】\n" + knowledgeCtx
		logger.Log.Infof("StreamAI: injected knowledge context into system prompt for agent '%s'", agent.Name)
	}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	var recentMsgs []model.Message
	repository.DB.Where("conversation_id = ?", convID).Order("created_at DESC").Limit(10).Find(&recentMsgs)
	for i := len(recentMsgs) - 1; i >= 0; i-- {
		messages = append(messages, map[string]string{"role": recentMsgs[i].Role, "content": recentMsgs[i].Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userContent})

	payload := map[string]interface{}{
		"model":    modelName,
		"messages": messages,
		"stream":   true,
	}

	// DeepSeek V4 specific: add thinking parameter and handle endpoint
	if isDeepSeekModel(modelName) || isDeepSeekProvider(provider) {
		payload["thinking"] = map[string]string{"type": "disabled"}
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	} else {
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	}
	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)
	var endpoint string
	if isDeepSeekModel(modelName) || isDeepSeekProvider(provider) {
		endpoint = buildDeepSeekEndpoint(provider.BaseURL)
	} else {
		endpoint = fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))
	}

	maxRetries := 1
	if agent.IronRules {
		maxRetries = 5
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)

		client := &http.Client{Timeout: 180 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logger.Log.Errorf("Stream AI request failed (attempt %d): %v", attempt+1, err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
			logger.Log.Errorf("Stream AI API error (attempt %d): %v", attempt+1, lastErr)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		// Parse SSE stream from upstream
		var accumulated strings.Builder
		var reasoningAccumulated strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				resp.Body.Close()
				return accumulated.String(), ctx.Err()
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				// Handle reasoning_content (DeepSeek V4 thinking mode fallback)
				if chunk.Choices[0].Delta.ReasoningContent != "" {
					reasoningAccumulated.WriteString(chunk.Choices[0].Delta.ReasoningContent)
				}
				if chunk.Choices[0].Delta.Content != "" {
					token := chunk.Choices[0].Delta.Content
					accumulated.WriteString(token)
					onToken(token)
				}
			}
		}
		resp.Body.Close()

		result := accumulated.String()
		if result != "" {
			return result, nil
		}
		// If content is empty but reasoning exists (DeepSeek V4 thinking mode), use reasoning as fallback
		if reasoning := reasoningAccumulated.String(); reasoning != "" {
			onToken(reasoning)
			return reasoning, nil
		}
		lastErr = fmt.Errorf("AI returned empty stream")
	}

	errMsg := fmt.Sprintf("AI streaming failed after %d attempts: %v", maxRetries, lastErr)
	onToken(errMsg)
	return errMsg, lastErr
}

func (s *ChatService) standardAIResponse(agent model.Agent, provider model.AIProvider, modelName, userContent string, convID uint) string {
	// Build messages for the API call
	messages := []map[string]string{}

	// Build system prompt with knowledge context if available
	systemPrompt := agent.SystemPrompt
	knowledgeCtx := s.getKnowledgeContext(agent, userContent)
	if knowledgeCtx != "" {
		if systemPrompt == "" {
			systemPrompt = "你是一个智能助手。"
		}
		systemPrompt += "\n\n【知识库参考资料 - 请优先基于以下内容回答问题，如果参考资料与问题相关请引用】\n" + knowledgeCtx
		logger.Log.Infof("StandardAI: injected knowledge context into system prompt for agent '%s'", agent.Name)
	}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
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

	payload := map[string]interface{}{
		"model":    modelName,
		"messages": messages,
	}

	// DeepSeek V4 specific: add thinking parameter and handle max_tokens
	if isDeepSeekModel(modelName) || isDeepSeekProvider(provider) {
		// DeepSeek V4 defaults to thinking mode enabled; disable it for normal chat
		// to get faster responses unless using the "pro" model
		payload["thinking"] = map[string]string{"type": "disabled"}
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	} else {
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	}

	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)

	// Build the correct endpoint
	var endpoint string
	if isDeepSeekModel(modelName) || isDeepSeekProvider(provider) {
		endpoint = buildDeepSeekEndpoint(provider.BaseURL)
	} else {
		endpoint = fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))
	}

	// Retry up to 5 times on failure (Iron Rule #7)
	maxRetries := 1
	if agent.IronRules {
		maxRetries = 5
	}

	var lastErr string
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			lastErr = fmt.Sprintf("AI 请求创建失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)

		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Sprintf("AI 服务请求失败: %v", err)
			logger.Log.Errorf("AI request failed (attempt %d): %v", attempt+1, err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			lastErr = fmt.Sprintf("AI 服务返回错误 (HTTP %d)", resp.StatusCode)
			logger.Log.Errorf("AI API error (HTTP %d, attempt %d): %s", resp.StatusCode, attempt+1, string(body[:min(len(body), 500)]))
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		var result struct {
			Choices []struct {
				Message struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = "AI 响应解析失败"
			continue
		}
		if len(result.Choices) > 0 {
			content := result.Choices[0].Message.Content
			if content != "" {
				return content
			}
			// Fallback to reasoning_content if content is empty (DeepSeek V4 thinking mode)
			if result.Choices[0].Message.ReasoningContent != "" {
				return result.Choices[0].Message.ReasoningContent
			}
		}
		lastErr = "AI 未返回内容"
	}

	return lastErr
}

// ==================== Website Links ====================

// SendMessageToAgent handles a single message to a published agent (stateless)
func (s *ChatService) SendMessageToAgent(agent model.Agent, provider model.AIProvider, message string) string {
	aiConfig := skill.AIConfig{
		BaseURL: provider.BaseURL,
		APIKey:  provider.APIKey,
		Model:   provider.Model,
	}

	if agent.Model != "" {
		aiConfig.Model = agent.Model
	}

	// Try RAG first if agent has indexed skills
	if agent.IronRules || hasIndexedSkills(agent) {
		ragResult := s.runSkillRAG(agent, aiConfig, message)
		if ragResult != "" {
			return ragResult
		}
	}

	// Standard AI response with knowledge context
	messages := []map[string]string{}
	systemPrompt := agent.SystemPrompt
	knowledgeCtx := s.getKnowledgeContext(agent, message)
	if knowledgeCtx != "" {
		if systemPrompt == "" {
			systemPrompt = "你是一个智能助手。"
		}
		systemPrompt += "\n\n【知识库参考资料 - 请优先基于以下内容回答问题，如果参考资料与问题相关请引用】\n" + knowledgeCtx
	}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": message})

	payload := map[string]interface{}{
		"model":    aiConfig.Model,
		"messages": messages,
	}

	// DeepSeek V4 specific: add thinking parameter and handle endpoint
	if isDeepSeekModel(aiConfig.Model) || isDeepSeekProvider(provider) {
		payload["thinking"] = map[string]string{"type": "disabled"}
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	} else {
		if agent.MaxTokens > 0 {
			payload["max_tokens"] = agent.MaxTokens
		}
	}
	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)
	var endpoint string
	if isDeepSeekModel(aiConfig.Model) || isDeepSeekProvider(provider) {
		endpoint = buildDeepSeekEndpoint(provider.BaseURL)
	} else {
		endpoint = fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return "AI 请求创建失败"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "AI 服务请求失败，请稍后重试"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("AI 服务返回错误 (HTTP %d)", resp.StatusCode)
	}

	// Parse response - handle both standard and DeepSeek V4 reasoning_content
	var result struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "AI 响应解析失败"
	}
	if len(result.Choices) > 0 {
		content := result.Choices[0].Message.Content
		// If the model returned reasoning content (thinking mode), append it as context
		if reasoning := result.Choices[0].Message.ReasoningContent; reasoning != "" && content == "" {
			// If content is empty but reasoning exists, use reasoning as fallback
			return reasoning
		}
		if content != "" {
			return content
		}
	}
	return "AI 未返回内容"
}

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
