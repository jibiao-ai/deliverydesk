// Package skill implements a document-based RAG (Retrieval-Augmented Generation) engine
// inspired by the CloudWeGo Eino framework patterns.
//
// Architecture:
//   START{Question, SkillID}
//     ▼
//   [retrieve] → search chunks by TF-IDF similarity
//     ▼
//   [score] → LLM scores each chunk for relevance (0-10)
//     ▼
//   [filter] → keep top-K chunks with score >= threshold
//     ▼
//   [synthesize] → LLM generates cited answer from top chunks
//     ▼
//   END{Answer, Sources, Confidence}
package skill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// ===================== Document Chunk Store (in-memory TF-IDF) =====================

// Chunk represents a piece of text from a document
type Chunk struct {
	ID         string  `json:"id"`
	SkillID    uint    `json:"skill_id"`
	DocID      uint    `json:"doc_id"`
	DocName    string  `json:"doc_name"`
	Content    string  `json:"content"`
	Index      int     `json:"index"`
	TFIDFScore float64 `json:"-"` // computed at query time
}

// ScoredChunk is a chunk with LLM-scored relevance
type ScoredChunk struct {
	Chunk   Chunk  `json:"chunk"`
	Score   int    `json:"score"`   // 0-10 relevance to the question
	Excerpt string `json:"excerpt"` // most relevant sentence
}

// RAGResult is the final output of the RAG pipeline
type RAGResult struct {
	Answer     string        `json:"answer"`
	Sources    []ScoredChunk `json:"sources"`
	Confidence int           `json:"confidence"` // 1-10
	SkillName  string        `json:"skill_name"`
	SkillID    uint          `json:"skill_id"`
	Empty      bool          `json:"empty"` // true if no data found
}

// ChunkStore is an in-memory store for text chunks with TF-IDF retrieval
type ChunkStore struct {
	mu     sync.RWMutex
	chunks map[uint][]Chunk // skillID -> chunks
	idf    map[string]float64
}

var globalStore = &ChunkStore{
	chunks: make(map[uint][]Chunk),
	idf:    make(map[string]float64),
}

// GetStore returns the global chunk store
func GetStore() *ChunkStore {
	return globalStore
}

// IndexDocument splits text into chunks and indexes them for a skill
func (s *ChunkStore) IndexDocument(skillID, docID uint, docName, content string) int {
	chunks := SplitIntoChunks(content, 800)
	s.mu.Lock()
	defer s.mu.Unlock()

	indexed := make([]Chunk, 0, len(chunks))
	for i, text := range chunks {
		indexed = append(indexed, Chunk{
			ID:      fmt.Sprintf("s%d_d%d_c%d", skillID, docID, i),
			SkillID: skillID,
			DocID:   docID,
			DocName: docName,
			Content: text,
			Index:   i,
		})
	}
	s.chunks[skillID] = append(s.chunks[skillID], indexed...)
	s.rebuildIDF()
	return len(indexed)
}

// ClearSkill removes all chunks for a skill
func (s *ChunkStore) ClearSkill(skillID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.chunks, skillID)
	s.rebuildIDF()
}

// RemoveDocument removes all chunks for a specific document from a skill
func (s *ChunkStore) RemoveDocument(skillID, docID uint) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	chunks := s.chunks[skillID]
	if len(chunks) == 0 {
		return 0
	}
	remaining := make([]Chunk, 0, len(chunks))
	removed := 0
	for _, c := range chunks {
		if c.DocID == docID {
			removed++
		} else {
			remaining = append(remaining, c)
		}
	}
	if len(remaining) == 0 {
		delete(s.chunks, skillID)
	} else {
		s.chunks[skillID] = remaining
	}
	if removed > 0 {
		s.rebuildIDF()
	}
	return removed
}

// Retrieve finds the most relevant chunks using TF-IDF scoring
func (s *ChunkStore) Retrieve(skillID uint, query string, topK int) []Chunk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chunks, ok := s.chunks[skillID]
	if !ok || len(chunks) == 0 {
		return nil
	}

	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	logger.Log.Infof("Retrieve: query='%s' tokens=%v, searching %d chunks for skill %d", query, queryTerms, len(chunks), skillID)

	// Debug: log a sample chunk's tokens to verify content quality
	if len(chunks) > 0 {
		sampleContent := chunks[0].Content
		runes := []rune(sampleContent)
		if len(runes) > 100 {
			sampleContent = string(runes[:100]) + "..."
		}
		sampleTokens := tokenize(chunks[0].Content)
		tokenPreview := sampleTokens
		if len(tokenPreview) > 20 {
			tokenPreview = tokenPreview[:20]
		}
		logger.Log.Infof("Retrieve: chunk[0] sample content='%s', tokens(first 20)=%v, total_tokens=%d",
			sampleContent, tokenPreview, len(sampleTokens))
	}

	// Calculate TF-IDF score for each chunk
	scored := make([]Chunk, len(chunks))
	copy(scored, chunks)
	positiveCount := 0
	for i := range scored {
		scored[i].TFIDFScore = s.tfidfScore(scored[i].Content, queryTerms)
		if scored[i].TFIDFScore > 0 {
			positiveCount++
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].TFIDFScore > scored[j].TFIDFScore
	})

	if topK > len(scored) {
		topK = len(scored)
	}
	// Only return chunks with positive scores
	result := make([]Chunk, 0, topK)
	for i := 0; i < topK && i < len(scored); i++ {
		if scored[i].TFIDFScore > 0 {
			result = append(result, scored[i])
		}
	}

	// Log scoring results for debugging
	if len(scored) > 0 {
		logger.Log.Infof("Retrieve: top_score=%.6f, positive_chunks=%d/%d, result_count=%d (topK=%d)",
			scored[0].TFIDFScore, positiveCount, len(scored), len(result), topK)
	}

	// If zero results despite having chunks, log diagnostic info
	if len(result) == 0 && len(chunks) > 0 {
		// Check if ANY query term exists in IDF vocabulary
		matchedTerms := 0
		for _, qt := range queryTerms {
			if _, exists := s.idf[qt]; exists {
				matchedTerms++
			}
		}
		logger.Log.Warnf("Retrieve: 0 results! query_terms=%d, matched_in_IDF=%d, total_IDF_terms=%d. "+
			"This usually means chunk content doesn't contain query-relevant text (possibly garbled xlsx parsing).",
			len(queryTerms), matchedTerms, len(s.idf))
	}

	return result
}

// GetChunkCount returns the number of chunks for a skill
func (s *ChunkStore) GetChunkCount(skillID uint) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chunks[skillID])
}

// GetAllSkillStats returns chunk counts for all skills in the store
func (s *ChunkStore) GetAllSkillStats() map[uint]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := make(map[uint]int, len(s.chunks))
	for skillID, chunks := range s.chunks {
		stats[skillID] = len(chunks)
	}
	return stats
}

func (s *ChunkStore) rebuildIDF() {
	totalDocs := 0
	termDocCount := make(map[string]int)
	for _, chunks := range s.chunks {
		for _, chunk := range chunks {
			totalDocs++
			seen := make(map[string]bool)
			for _, term := range tokenize(chunk.Content) {
				if !seen[term] {
					termDocCount[term]++
					seen[term] = true
				}
			}
		}
	}
	s.idf = make(map[string]float64)
	for term, count := range termDocCount {
		s.idf[term] = math.Log(float64(totalDocs+1) / float64(count+1))
	}
}

func (s *ChunkStore) tfidfScore(text string, queryTerms []string) float64 {
	terms := tokenize(text)
	tf := make(map[string]int)
	for _, t := range terms {
		tf[t]++
	}
	total := float64(len(terms))
	if total == 0 {
		return 0
	}
	score := 0.0
	for _, qt := range queryTerms {
		freq := float64(tf[qt])
		idf := s.idf[qt]
		if idf == 0 {
			idf = 1.0
		}
		score += (freq / total) * idf
	}
	return score
}

// ===================== Text Processing =====================

// SplitIntoChunks splits text into chunks of at most chunkSize runes,
// breaking on paragraph or line boundaries where possible.
// For tabular data (xlsx/csv), it detects tab-separated rows and groups
// them with header context for better search quality.
func SplitIntoChunks(text string, chunkSize int) []string {
	// Detect if this is tabular data (has tab-separated lines)
	lines := strings.Split(text, "\n")
	tabLineCount := 0
	for _, line := range lines {
		if strings.Contains(line, "\t") {
			tabLineCount++
		}
	}
	isTabular := tabLineCount > 3 && float64(tabLineCount)/float64(len(lines)) > 0.3

	if isTabular {
		return splitTabularIntoChunks(text, chunkSize)
	}

	// Standard paragraph-based splitting
	var chunks []string
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, s)
		}
		buf.Reset()
	}

	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if buf.Len()+len(para)+2 > chunkSize && buf.Len() > 0 {
			flush()
		}
		if len(para) > chunkSize {
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if buf.Len()+len(line)+1 > chunkSize && buf.Len() > 0 {
					flush()
				}
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(line)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
		}
	}
	flush()
	return chunks
}

// splitTabularIntoChunks handles tab-separated tabular data (xlsx, csv).
// It detects header rows and prepends them to each chunk so every chunk
// has context about what each column means. This significantly improves
// TF-IDF matching for queries about specific column values.
func splitTabularIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	var buf strings.Builder
	var currentHeader string

	lines := strings.Split(text, "\n")

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, s)
		}
		buf.Reset()
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect sheet header lines like "--- Sheet: xxx ---"
		if strings.HasPrefix(line, "--- Sheet:") {
			if buf.Len() > 0 {
				flush()
			}
			currentHeader = ""
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}

		// Detect column header rows (first tab-containing row after a sheet marker or start)
		if currentHeader == "" && strings.Contains(line, "\t") {
			// Heuristic: header rows typically contain more CJK/alpha chars than numbers
			cjkCount := 0
			for _, r := range line {
				if isCJK(r) || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					cjkCount++
				}
			}
			if cjkCount > 2 {
				currentHeader = line
			}
		}

		// Check if adding this line would exceed chunk size
		newLen := buf.Len() + len(line) + 1
		if newLen > chunkSize && buf.Len() > 0 {
			flush()
			// Prepend header to new chunk for context
			if currentHeader != "" {
				buf.WriteString(currentHeader)
				buf.WriteByte('\n')
			}
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	flush()
	return chunks
}

// tokenize performs simple CJK-aware tokenization
func tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string

	// Extract CJK bigrams and Latin words
	var word strings.Builder
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isCJK(r) {
			// flush any Latin word
			if word.Len() > 0 {
				tokens = append(tokens, word.String())
				word.Reset()
			}
			// Add single CJK char and bigram
			tokens = append(tokens, string(r))
			if i+1 < len(runes) && isCJK(runes[i+1]) {
				tokens = append(tokens, string(runes[i:i+2]))
			}
		} else if isAlphaNum(r) {
			word.WriteRune(r)
		} else {
			if word.Len() > 0 {
				tokens = append(tokens, word.String())
				word.Reset()
			}
		}
	}
	if word.Len() > 0 {
		tokens = append(tokens, word.String())
	}
	return tokens
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0xF900 && r <= 0xFAFF) // CJK Compatibility Ideographs
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// ===================== RAG Pipeline =====================

// AIConfig holds the AI provider configuration for the RAG pipeline
type AIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// RunRAG executes the full RAG pipeline: retrieve → score → filter → synthesize
func RunRAG(config AIConfig, skillID uint, skillName, question string, ironRules bool) RAGResult {
	store := GetStore()

	// Step 1: Retrieve top candidates via TF-IDF
	candidates := store.Retrieve(skillID, question, 20)
	logger.Log.Infof("RAG Step1 [skill=%s, id=%d]: TF-IDF retrieved %d candidates from %d chunks for question: %s",
		skillName, skillID, len(candidates), store.GetChunkCount(skillID), question)

	if len(candidates) == 0 {
		// Log sample chunk content for debugging
		store.mu.RLock()
		chunks := store.chunks[skillID]
		if len(chunks) > 0 {
			sample := chunks[0].Content
			runes := []rune(sample)
			if len(runes) > 100 {
				sample = string(runes[:100]) + "..."
			}
			logger.Log.Warnf("RAG Step1: 0 candidates but %d chunks exist. Sample chunk[0]: '%s'", len(chunks), sample)
			// Debug: show what query tokens look like
			queryTokens := tokenize(question)
			logger.Log.Warnf("RAG Step1: query tokens: %v", queryTokens)
		}
		store.mu.RUnlock()

		return RAGResult{
			Answer:     "无有效数据，无法判断。该技能知识库中没有与您的问题相关的文档内容。",
			SkillName:  skillName,
			SkillID:    skillID,
			Confidence: 0,
			Empty:      true,
		}
	}

	// Log top candidate scores for debugging
	for i, c := range candidates {
		if i >= 3 {
			break
		}
		preview := c.Content
		runes := []rune(preview)
		if len(runes) > 80 {
			preview = string(runes[:80]) + "..."
		}
		logger.Log.Infof("RAG Step1: candidate[%d] score=%.4f doc=%s: %s", i, c.TFIDFScore, c.DocName, preview)
	}

	// Step 2: Score each candidate with LLM (parallel, max 5 concurrent)
	scored := scoreChunks(config, candidates, question, 5)

	// Log LLM scoring results
	scoredAbove0 := 0
	maxScore := 0
	for _, sc := range scored {
		if sc.Score > 0 {
			scoredAbove0++
		}
		if sc.Score > maxScore {
			maxScore = sc.Score
		}
	}
	logger.Log.Infof("RAG Step2 [skill=%s]: LLM scored %d candidates, %d with score>0, max_score=%d",
		skillName, len(scored), scoredAbove0, maxScore)

	// Step 3: Filter top-K with score >= 2 (lowered from 3 to be more inclusive)
	filtered := filterTopK(scored, 5, 2)
	logger.Log.Infof("RAG Step3 [skill=%s]: filtered to %d chunks (minScore=2)", skillName, len(filtered))

	if len(filtered) == 0 {
		// If LLM scoring failed (all scores 0), fall back to using TF-IDF top candidates directly
		if scoredAbove0 == 0 && len(candidates) > 0 {
			logger.Log.Warnf("RAG Step3: all LLM scores are 0 (likely API failure), falling back to TF-IDF top candidates")
			// Use top TF-IDF candidates directly without LLM scoring
			directChunks := candidates
			if len(directChunks) > 5 {
				directChunks = directChunks[:5]
			}
			// Create synthetic scored chunks with TF-IDF score mapped to 1-10
			for _, c := range directChunks {
				filtered = append(filtered, ScoredChunk{
					Chunk:   c,
					Score:   5, // default moderate relevance
					Excerpt: "",
				})
			}
			logger.Log.Infof("RAG Step3: using %d TF-IDF candidates as fallback", len(filtered))
		}
	}

	if len(filtered) == 0 {
		return RAGResult{
			Answer:     "无有效数据，无法判断。知识库中的文档内容与您的问题关联度较低。",
			SkillName:  skillName,
			SkillID:    skillID,
			Confidence: 0,
			Empty:      true,
		}
	}

	// Step 4: Synthesize answer from top chunks
	answer, confidence := synthesize(config, filtered, question, skillName, ironRules)
	logger.Log.Infof("RAG Step4 [skill=%s]: synthesized answer with confidence=%d, length=%d",
		skillName, confidence, len(answer))

	return RAGResult{
		Answer:     answer,
		Sources:    filtered,
		SkillName:  skillName,
		SkillID:    skillID,
		Confidence: confidence,
		Empty:      false,
	}
}

func scoreChunks(config AIConfig, chunks []Chunk, question string, maxConcurrent int) []ScoredChunk {
	results := make([]ScoredChunk, len(chunks))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i, chunk := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, c Chunk) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = scoreOneChunk(config, c, question)
		}(i, chunk)
	}
	wg.Wait()
	return results
}

func scoreOneChunk(config AIConfig, chunk Chunk, question string) ScoredChunk {
	// Truncate chunk for scoring prompt
	content := chunk.Content
	if utf8.RuneCountInString(content) > 500 {
		runes := []rune(content)
		content = string(runes[:500]) + "..."
	}

	prompt := fmt.Sprintf(`请评估以下文档片段与问题的相关性。

问题: %s

文档片段 (来源: %s):
%s

请仅返回JSON格式，不要返回其他内容:
{"score": <0-10>, "excerpt": "<最相关的一句话或短语，如果score为0则为空字符串>"}

评分标准: 0=完全无关, 3=略有关联, 7=明显相关, 10=直接回答问题。`,
		question, chunk.DocName, content)

	respContent := callAI(config, prompt, 0.1, 200)
	if respContent == "" {
		logger.Log.Warnf("RAG scoreOneChunk: LLM returned empty for chunk %s (doc=%s)", chunk.ID, chunk.DocName)
		return ScoredChunk{Chunk: chunk, Score: 0}
	}

	// Parse JSON response
	respContent = strings.TrimSpace(respContent)
	respContent = strings.TrimPrefix(respContent, "```json")
	respContent = strings.TrimPrefix(respContent, "```")
	respContent = strings.TrimSuffix(respContent, "```")
	respContent = strings.TrimSpace(respContent)

	var sr struct {
		Score   int    `json:"score"`
		Excerpt string `json:"excerpt"`
	}
	if err := json.Unmarshal([]byte(respContent), &sr); err != nil {
		logger.Log.Warnf("RAG scoreOneChunk: failed to parse LLM response for chunk %s: %v, raw: %s", chunk.ID, err, respContent[:min(len(respContent), 100)])
		return ScoredChunk{Chunk: chunk, Score: 0}
	}
	logger.Log.Debugf("RAG scoreOneChunk: chunk %s scored %d", chunk.ID, sr.Score)
	return ScoredChunk{Chunk: chunk, Score: sr.Score, Excerpt: sr.Excerpt}
}

func filterTopK(scored []ScoredChunk, maxK, minScore int) []ScoredChunk {
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	var top []ScoredChunk
	for _, c := range scored {
		if c.Score < minScore {
			break
		}
		top = append(top, c)
		if len(top) == maxK {
			break
		}
	}
	return top
}

func synthesize(config AIConfig, topK []ScoredChunk, question, skillName string, ironRules bool) (string, int) {
	var sb strings.Builder
	sb.WriteString("你是交付专家智能体，请根据以下文档摘录回答问题。\n\n")

	if ironRules {
		sb.WriteString("【铁律规则 - 必须严格遵守】\n")
		sb.WriteString("1. 所有指标/标签/数值必须来自下面提供的文档摘录，禁止编造数据\n")
		sb.WriteString("2. 如果文档数据不足，不要推断根因或编造趋势\n")
		sb.WriteString("3. 阈值必须来自文档内容，禁止自定义阈值\n")
		sb.WriteString("4. 回答必须引用具体的文档来源和对应的环境信息\n")
		sb.WriteString("5. 如果数据为空，直接回复\"无有效数据，无法判断\"\n")
		sb.WriteString("6. 请在回答末尾给出1-10的置信度评分（格式：[置信度: X/10]）\n")
		sb.WriteString("7. 置信度低于7时请标注\"[低置信度警告]\"标签\n\n")
	}

	sb.WriteString(fmt.Sprintf("技能来源: %s\n", skillName))
	sb.WriteString(fmt.Sprintf("问题: %s\n\n", question))
	sb.WriteString("文档摘录:\n")

	for i, c := range topK {
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Chunk.Content
			if utf8.RuneCountInString(excerpt) > 300 {
				excerpt = string([]rune(excerpt)[:300]) + "..."
			}
		}
		fmt.Fprintf(&sb, "[%d] (来源: %s, 相关度: %d/10)\n%s\n\n", i+1, c.Chunk.DocName, c.Score, excerpt)
	}

	sb.WriteString("请提供清晰、准确的回答。引用来源时使用 [1] [2] 等编号标注。")
	if ironRules {
		sb.WriteString("\n\n在回答最后一行，请用以下格式给出置信度: [置信度: X/10]")
	}

	answer := callAI(config, sb.String(), 0.3, 4096)
	if answer == "" {
		return "AI 服务请求失败，请稍后重试", 0
	}

	// Extract confidence score
	confidence := 7
	if ironRules {
		if idx := strings.LastIndex(answer, "[置信度:"); idx >= 0 {
			tail := answer[idx:]
			var c int
			if _, err := fmt.Sscanf(tail, "[置信度: %d/10]", &c); err == nil && c >= 1 && c <= 10 {
				confidence = c
			}
		}
	}

	return answer, confidence
}

func callAI(config AIConfig, prompt string, temperature float64, maxTokens int) string {
	payload := map[string]interface{}{
		"model":       config.Model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	payloadBytes, _ := json.Marshal(payload)

	endpoint := fmt.Sprintf("%s/chat/completions", strings.TrimRight(config.BaseURL, "/"))
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		logger.Log.Errorf("RAG: create request failed: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Errorf("RAG: request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		logger.Log.Errorf("RAG: API error (HTTP %d): %s", resp.StatusCode, string(body[:min(len(body), 300)]))
		return ""
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
