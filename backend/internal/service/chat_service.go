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
		"model":      modelName,
		"messages":   messages,
		"max_tokens": agent.MaxTokens,
		"stream":     true,
	}
	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))

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
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				token := chunk.Choices[0].Delta.Content
				accumulated.WriteString(token)
				onToken(token)
			}
		}
		resp.Body.Close()

		result := accumulated.String()
		if result != "" {
			return result, nil
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
		"model":      modelName,
		"messages":   messages,
		"max_tokens": agent.MaxTokens,
	}
	if agent.Temperature > 0 {
		payload["temperature"] = agent.Temperature
	}

	payloadBytes, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(provider.BaseURL, "/"))

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
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = "AI 响应解析失败"
			continue
		}
		if len(result.Choices) > 0 {
			return result.Choices[0].Message.Content
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
		"model":      aiConfig.Model,
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
