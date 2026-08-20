package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// KBHandler handles Jira→Confluence knowledge base generation endpoints
type KBHandler struct{}

func NewKBHandler() *KBHandler {
	return &KBHandler{}
}

// =============== Request / Response structs ===============

type KBGenerateRequest struct {
	IssueKey       string `json:"issue_key" binding:"required"`       // e.g. ECSL2-50377
	JiraServer     string `json:"jira_server"`                        // override system default
	JiraUser       string `json:"jira_user"`                          // override
	JiraToken      string `json:"jira_token"`                         // override
	ConfluenceURL  string `json:"confluence_url" binding:"required"`  // target parent page URL
}

type KBGenerateResponse struct {
	Title       string              `json:"title"`
	PageURL     string              `json:"page_url"`
	PageID      string              `json:"page_id"`
	Preview     string              `json:"preview"`      // HTML preview (for dry-run)
	Content     *KBContentGenerated `json:"content"`      // structured content
}

type KBPreviewRequest struct {
	IssueKey   string `json:"issue_key" binding:"required"`
	JiraServer string `json:"jira_server"`
	JiraUser   string `json:"jira_user"`
	JiraToken  string `json:"jira_token"`
}

type KBContentGenerated struct {
	Title          string       `json:"title"`
	Background     string       `json:"background"`
	Customer       string       `json:"customer"`
	TicketTable    string       `json:"ticket_table"`
	Timeline       []TimelineEv `json:"timeline"`
	ProcessSummary string       `json:"process_summary"`
	TechConclusion string       `json:"tech_conclusion"`
	Result         string       `json:"result"`
	Suggestions    []string     `json:"suggestions"`
}

type TimelineEv struct {
	Time    string `json:"time"`
	Author  string `json:"author"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// =============== Jira API structs ===============

type kbJiraIssue struct {
	Key            string                 `json:"key"`
	Fields         map[string]interface{} `json:"fields"`
	RenderedFields map[string]interface{} `json:"renderedFields"`
}

type jiraComment struct {
	Author       map[string]interface{} `json:"author"`
	Body         interface{}            `json:"body"`
	RenderedBody string                 `json:"renderedBody"`
	Created      string                 `json:"created"`
}

type jiraChangelog struct {
	Author  map[string]interface{} `json:"author"`
	Created string                 `json:"created"`
	Items   []jiraChangeItem       `json:"items"`
}

type jiraChangeItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

// =============== Handler methods ===============

// PreviewKB handles POST /api/kb/preview — fetches Jira data + LLM polish, returns structured content
func (h *KBHandler) PreviewKB(c *gin.Context) {
	var req KBPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	server, user, token := h.resolveJiraCredentials(req.JiraServer, req.JiraUser, req.JiraToken)
	if user == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 Jira 认证信息，请前往「系统设置 → Jira 配置」中设置 jira_server / jira_username / jira_password"})
		return
	}

	// 1. Fetch Jira data
	issueKey := strings.TrimSpace(req.IssueKey)
	issue, comments, changelog, err := fetchJiraData(server, issueKey, user, token)
	if err != nil {
		errMsg := fmt.Sprintf("获取 Jira 工单失败 (工单号: %s, 服务器: %s): %s", issueKey, server, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": errMsg})
		return
	}

	// 2. Build raw timeline
	events := buildTimeline(changelog, comments)

	// 3. Extract fields
	fields := issue.Fields
	key := issue.Key
	summary := displayValue(fields["summary"])
	customer := extractCustomer(fields, displayValue(getMapField(fields, "reporter", "displayName")))
	status := displayValue(getMapField(fields, "status", "name"))
	priority := displayValue(getMapField(fields, "priority", "name"))
	resolution := displayValue(getMapField(fields, "resolution", "name"))
	reporter := displayValue(getMapField(fields, "reporter", "displayName"))
	assignee := displayValue(getMapField(fields, "assignee", "displayName"))
	created := displayValue(fields["created"])
	updated := displayValue(fields["updated"])

	descHTML := ""
	if issue.RenderedFields != nil {
		descHTML = displayValue(issue.RenderedFields["description"])
	}
	descText := stripHTML(descHTML)

	// 4. Call LLM for each section
	aiBaseURL, aiKey, aiModel := h.resolveAICredentials()
	if aiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 AI 模型配置，请在系统设置或 AI 模型页配置 API Key"})
		return
	}

	// 4a. Title
	titlePrompt := fmt.Sprintf(`你是资深技术支持工程师，负责将 Jira 工单沉淀为知识库。
根据工单信息生成一个精简、清晰的知识库页面标题。
- 工单号：%s
- 工单标题：%s
- 客户名称：%s
输出要求：
- 以「【知识库】」开头
- 必须包含工单号
- 用一句话概括问题主题：去除环境细节冗余、去除重复措辞
- 长度不超过 40 个汉字
- 只输出标题本身`, key, summary, customer)
	title := callLLM(aiBaseURL, aiKey, aiModel, "你是资深技术支持工程师，输出严格遵守用户要求。", titlePrompt)
	title = strings.TrimSpace(title)
	if title == "" {
		title = fmt.Sprintf("【知识库】%s %s", key, summary)
	}

	// 4b. Ticket info table
	issueURL := fmt.Sprintf("%s/browse/%s", server, key)
	fieldsJSON, _ := json.Marshal(map[string]string{
		"key": key, "summary": summary, "status": status, "priority": priority,
		"resolution": resolution, "customer": customer, "reporter": reporter,
		"assignee": assignee, "created": created, "updated": updated,
		"description": descText,
	})
	ticketPrompt := fmt.Sprintf(`你是资深技术支持工程师，负责将 Jira 工单信息整理为结构化表格。
将工单字段整理为「字段 / 值」两列的结构化表格，输出 Confluence storage 格式 HTML。
输入（JSON）：%s
输出要求：
- 生成 <table><tbody> 结构，首行为表头 <tr><th>字段</th><th>值</th></tr>
- 字段顺序：工单号、标题、状态、优先级、解决结果、客户名称、报告人、经办人、创建时间、更新时间、环境
- 工单号一行的值用 <a href="%s"> 链接包裹
- 值要简洁；「环境」从描述中提取关键点，无则写「（未提供）」
- 只输出表格 HTML`, string(fieldsJSON), issueURL)
	ticketTable := callLLM(aiBaseURL, aiKey, aiModel, "你是资深技术支持工程师，输出严格遵守用户要求。", ticketPrompt)
	ticketTable = cleanCodeBlock(ticketTable)

	// 4c. Process timeline
	timelineRaw := formatTimelineRaw(events)
	processPrompt := fmt.Sprintf(`你是资深技术支持工程师，擅长将冗长的工单处理记录收敛为清晰的时间线和行动总结。
将工单的评论与状态变更记录收敛为「时间线 + 处理内容汇总」两部分。
输入（按时间升序）：
%s
输出要求（严格 JSON）：
{"timeline": [{"time": "YYYY-MM-DD HH:mm", "author": "操作人", "content": "一句话概括"}], "summary": "处理内容汇总（200-400字）"}
规则：
- timeline 按时间升序，content 控制在 60 字以内，去重合并同主题
- 状态变更也要纳入
- summary 必须包含：(1)问题定位过程和关键发现 (2)采取的具体解决动作（命令、配置变更等要明确写出）(3)最终验证结果
- summary 中每一步动作要写清楚具体做了什么，不能用模糊描述如"进行了排查""做了处理"，要写明确的操作如"将PV从Cinder切回NFS-CSI""执行了flock测试命令验证NFS锁机制"
- 只输出 JSON`, timelineRaw)
	processResp := callLLM(aiBaseURL, aiKey, aiModel, "你是资深技术支持工程师，输出严格遵守用户要求。所有结论和动作必须具体、明确，禁止使用模糊表述。", processPrompt)
	processResp = cleanCodeBlock(processResp)

	var processData struct {
		Timeline []TimelineEv `json:"timeline"`
		Summary  string       `json:"summary"`
	}
	if err := json.Unmarshal([]byte(processResp), &processData); err != nil {
		logger.Log.Warnf("KB: LLM process JSON parse error: %v, raw: %s", err, processResp[:kbMin(len(processResp), 200)])
		processData.Timeline = events
		processData.Summary = "（LLM 解析失败，请重试）"
	}

	// 4d. Summary & suggestions
	summaryPrompt := fmt.Sprintf(`你是资深技术支持工程师，擅长从工单处理记录中提炼精准的技术结论。
基于工单处理记录，汇总问题的根因分析、解决方案和改进建议。
输入：
- 工单：%s，状态：%s，解决结果：%s
- 处理内容汇总：%s
- 工单描述：%s
输出要求（严格 JSON）：
{"tech_conclusion": "技术结论（200-400字）", "result": "最终结果（100-200字）", "suggestions": ["建议1","建议2","建议3"]}
规则：
- tech_conclusion 必须包含：(1)问题根因（精确到组件/配置/版本）(2)解决方案（具体的操作步骤或配置变更）(3)影响范围评估
- tech_conclusion 禁止使用"经过排查发现""通过分析得知"等空话，直接写"根因是XXX""解决方案为XXX"
- result 必须写明问题是否彻底解决、验证方式、当前运行状态
- suggestions 每条必须是可执行的具体动作，而非泛泛的"加强监控"，要写成"在Prometheus中增加NFS存储延迟和可用性的告警规则"这样的明确建议
- 只输出 JSON`, key, status, resolution, processData.Summary, descText)
	summaryResp := callLLM(aiBaseURL, aiKey, aiModel, "你是资深技术支持工程师，输出严格遵守用户要求。所有结论必须精确到具体技术细节，禁止泛泛而谈。", summaryPrompt)
	summaryResp = cleanCodeBlock(summaryResp)

	var summaryData struct {
		TechConclusion string   `json:"tech_conclusion"`
		Result         string   `json:"result"`
		Suggestions    []string `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(summaryResp), &summaryData); err != nil {
		logger.Log.Warnf("KB: LLM summary JSON parse error: %v", err)
		summaryData.TechConclusion = "（LLM 解析失败，请重试）"
		summaryData.Result = ""
		summaryData.Suggestions = []string{}
	}

	// Build background
	background := fmt.Sprintf("工单 %s 涉及客户 %s 环境，问题为：%s。创建时间：%s，优先级：%s，当前状态：%s，解决结果：%s。",
		key, customer, summary, created, priority, status, resolution)

	content := &KBContentGenerated{
		Title:          title,
		Background:     background,
		Customer:       customer,
		TicketTable:    ticketTable,
		Timeline:       processData.Timeline,
		ProcessSummary: processData.Summary,
		TechConclusion: summaryData.TechConclusion,
		Result:         summaryData.Result,
		Suggestions:    summaryData.Suggestions,
	}

	// Render preview HTML
	previewHTML := renderStorageHTML(content, descHTML)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": KBGenerateResponse{
			Title:   title,
			Preview: previewHTML,
			Content: content,
		},
	})
}

// PublishKB handles POST /api/kb/publish — publishes generated content to Confluence
func (h *KBHandler) PublishKB(c *gin.Context) {
	var req struct {
		Content       *KBContentGenerated `json:"content" binding:"required"`
		ConfluenceURL string              `json:"confluence_url" binding:"required"` // parent page URL
		JiraServer    string              `json:"jira_server"`
		JiraUser      string              `json:"jira_user"`
		JiraToken     string              `json:"jira_token"`
		IssueKey      string              `json:"issue_key"` // for attachments
		DescHTML      string              `json:"desc_html"` // original description HTML
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}

	server, user, token := h.resolveJiraCredentials(req.JiraServer, req.JiraUser, req.JiraToken)
	if user == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 Jira/Confluence 认证信息，请前往「系统设置 → Jira 配置」中设置"})
		return
	}

	// Parse Confluence URL to extract spaceKey and parentId
	spaceKey, parentID, err := parseConfluenceURL(req.ConfluenceURL, server)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无法解析 Confluence URL: " + err.Error()})
		return
	}

	// Resolve space ID
	spaceID, err := resolveSpaceID(server, spaceKey, user, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "获取 Confluence 空间失败: " + err.Error()})
		return
	}

	// Render full HTML
	bodyHTML := renderStorageHTML(req.Content, req.DescHTML)

	// Create page as child of parent
	pageID, pageURL, err := createConfluencePage(server, spaceID, parentID, req.Content.Title, bodyHTML, user, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建 Confluence 页面失败: " + err.Error()})
		return
	}

	// Upload attachments from Jira if issue key provided
	if req.IssueKey != "" {
		go func() {
			h.uploadJiraAttachments(server, req.IssueKey, pageID, user, token)
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": KBGenerateResponse{
			Title:   req.Content.Title,
			PageURL: pageURL,
			PageID:  pageID,
		},
		"message": "知识库页面已发布",
	})
}

// =============== Helper: Jira data fetching ===============

func fetchJiraData(server, issueKey, user, token string) (*kbJiraIssue, []jiraComment, []jiraChangelog, error) {
	// Normalize inputs
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	issueKey = strings.TrimSpace(issueKey)

	// Try API v3 first, fall back to v2 on 404
	apiVersion := "3"
	issueURL := fmt.Sprintf("%s/rest/api/%s/issue/%s?expand=renderedFields", server, apiVersion, issueKey)
	logger.Log.Infof("[KB] Fetching Jira issue: %s (user: %s)", issueURL, user)
	issueBody, err := jiraHTTP("GET", issueURL, user, token, nil)
	if err != nil {
		// If v3 returns 404, try v2
		if strings.Contains(err.Error(), "404") {
			logger.Log.Infof("[KB] API v3 returned 404, trying v2...")
			apiVersion = "2"
			issueURL = fmt.Sprintf("%s/rest/api/%s/issue/%s?expand=renderedFields", server, apiVersion, issueKey)
			issueBody, err = jiraHTTP("GET", issueURL, user, token, nil)
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("issue [%s]: %w", issueURL, err)
		}
	}
	var issue kbJiraIssue
	if err := json.Unmarshal(issueBody, &issue); err != nil {
		return nil, nil, nil, fmt.Errorf("parse issue: %w", err)
	}
	logger.Log.Infof("[KB] Issue fetched OK: %s (using API v%s)", issue.Key, apiVersion)

	// Fetch comments (using same API version that worked for the issue)
	comments := []jiraComment{}
	startAt := 0
	for {
		commURL := fmt.Sprintf("%s/rest/api/%s/issue/%s/comment?startAt=%d&maxResults=100&expand=renderedBody", server, apiVersion, issueKey, startAt)
		body, err := jiraHTTP("GET", commURL, user, token, nil)
		if err != nil {
			logger.Log.Warnf("[KB] Fetch comments failed at startAt=%d: %v", startAt, err)
			break
		}
		var resp struct {
			Comments []jiraComment `json:"comments"`
			Total    int           `json:"total"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			break
		}
		comments = append(comments, resp.Comments...)
		if startAt+len(resp.Comments) >= resp.Total {
			break
		}
		startAt += len(resp.Comments)
	}
	logger.Log.Infof("[KB] Fetched %d comments", len(comments))

	// Fetch changelog (using same API version)
	changelogs := []jiraChangelog{}
	startAt = 0
	for {
		clURL := fmt.Sprintf("%s/rest/api/%s/issue/%s/changelog?startAt=%d&maxResults=100", server, apiVersion, issueKey, startAt)
		body, err := jiraHTTP("GET", clURL, user, token, nil)
		if err != nil {
			logger.Log.Warnf("[KB] Fetch changelog failed at startAt=%d: %v", startAt, err)
			break
		}
		var resp struct {
			Values []jiraChangelog `json:"values"`
			Total  int            `json:"total"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			break
		}
		changelogs = append(changelogs, resp.Values...)
		if startAt+len(resp.Values) >= resp.Total {
			break
		}
		startAt += len(resp.Values)
	}
	logger.Log.Infof("[KB] Fetched %d changelog entries", len(changelogs))

	return &issue, comments, changelogs, nil
}

func jiraHTTP(method, url, user, token string, body []byte) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody[:kbMin(len(respBody), 500)]))
	}
	return respBody, nil
}

// =============== Helper: Confluence publishing ===============

func parseConfluenceURL(rawURL, jiraServer string) (spaceKey, parentID string, err error) {
	// Expected: https://xxx.atlassian.net/wiki/spaces/PSBC/pages/3495297025
	// or https://xxx.atlassian.net/wiki/spaces/PSBC/pages/3495297025/title
	re := regexp.MustCompile(`/wiki/spaces/([^/]+)/pages/(\d+)`)
	m := re.FindStringSubmatch(rawURL)
	if len(m) >= 3 {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("URL 格式无法识别，请提供 /wiki/spaces/{spaceKey}/pages/{pageId} 格式")
}

func resolveSpaceID(server, spaceKey, user, token string) (string, error) {
	url := fmt.Sprintf("%s/wiki/api/v2/spaces?keys=%s&limit=1", server, spaceKey)
	body, err := jiraHTTP("GET", url, user, token, nil)
	if err != nil {
		return "", err
	}
	var resp struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Results) == 0 {
		return "", fmt.Errorf("未找到空间 key=%s", spaceKey)
	}
	return resp.Results[0].ID, nil
}

func createConfluencePage(server, spaceID, parentID, title, body, user, token string) (string, string, error) {
	payload := map[string]interface{}{
		"spaceId":  spaceID,
		"status":   "current",
		"title":    title,
		"parentId": parentID,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          body,
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/wiki/api/v2/pages", server)
	respBody, err := jiraHTTP("POST", url, user, token, payloadBytes)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		ID    string `json:"id"`
		Links struct {
			WebUI string `json:"webui"`
		} `json:"_links"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", err
	}
	pageURL := server + resp.Links.WebUI
	if resp.Links.WebUI == "" {
		pageURL = fmt.Sprintf("%s/wiki/api/v2/pages/%s", server, resp.ID)
	}
	return resp.ID, pageURL, nil
}

// uploadJiraAttachments downloads from Jira and uploads to Confluence page
func (h *KBHandler) uploadJiraAttachments(server, issueKey, pageID, user, token string) {
	// Fetch issue to get attachments list (try v3, fallback v2)
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	issueURL := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=attachment", server, issueKey)
	body, err := jiraHTTP("GET", issueURL, user, token, nil)
	if err != nil {
		logger.Log.Warnf("KB attachment fetch failed: %v", err)
		return
	}
	var issue struct {
		Fields struct {
			Attachment []struct {
				Filename string `json:"filename"`
				Content  string `json:"content"`
			} `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &issue); err != nil {
		return
	}

	tmpDir := filepath.Join(os.TempDir(), "kb_attachments_"+issueKey)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	for _, att := range issue.Fields.Attachment {
		if att.Filename == "" || att.Content == "" {
			continue
		}
		// Download from Jira
		req, _ := http.NewRequest("GET", att.Content, nil)
		req.SetBasicAuth(user, token)
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		localPath := filepath.Join(tmpDir, att.Filename)
		os.WriteFile(localPath, data, 0644)

		// Upload to Confluence
		uploadAttachmentToConfluence(server, pageID, localPath, att.Filename, user, token)
	}
}

func uploadAttachmentToConfluence(server, pageID, filepath_, filename, user, token string) {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	fileData, err := os.ReadFile(filepath_)
	if err != nil {
		return
	}

	// Build multipart form
	var buf bytes.Buffer
	boundary := "----KBUploadBoundary" + fmt.Sprintf("%d", time.Now().UnixNano())
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", filename))
	buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", mimeType))
	buf.Write(fileData)
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"pageId\"\r\n\r\n%s", pageID))
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	url := fmt.Sprintf("%s/wiki/api/v2/attachments", server)
	req, _ := http.NewRequest("POST", url, &buf)
	req.SetBasicAuth(user, token)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("KB: attachment upload failed %s: %v", filename, err)
		return
	}
	resp.Body.Close()
}

// =============== Helper: credential resolution ===============

func (h *KBHandler) resolveJiraCredentials(reqServer, reqUser, reqToken string) (string, string, string) {
	server := strings.TrimSpace(reqServer)
	user := strings.TrimSpace(reqUser)
	token := strings.TrimSpace(reqToken)

	// Fall back to system settings (using same keys as ops_env_handlers.go for consistency)
	if server == "" || user == "" || token == "" {
		var settings []model.SystemSetting
		repository.DB.Where("category = ?", "jira").Find(&settings)
		for _, s := range settings {
			val := strings.TrimSpace(s.Value)
			switch s.Key {
			case "jira_server":
				if server == "" {
					server = val
				}
			case "jira_username":
				if user == "" {
					user = val
				}
			case "jira_password", "jira_token", "jira_api_token":
				if token == "" {
					token = val
				}
			}
		}
	}

	// Fallback to environment/config (same as ops_env_handlers.go)
	if server == "" || user == "" || token == "" {
		cfg := config.Load()
		if server == "" {
			server = cfg.Jira.Server
		}
		if user == "" {
			user = cfg.Jira.Username
		}
		if token == "" {
			token = cfg.Jira.APIToken
		}
	}

	if server == "" {
		server = "https://easystack.atlassian.net"
	}
	server = strings.TrimRight(server, "/")

	logger.Log.Debugf("[KB] Resolved Jira credentials: server=%q, user=%q, token_len=%d", server, user, len(token))
	return server, user, token
}

func (h *KBHandler) resolveAICredentials() (string, string, string) {
	// 1) Check system default AI provider
	var provider model.AIProvider
	if err := repository.DB.Where("is_default = ? AND is_enabled = ?", true, true).First(&provider).Error; err == nil {
		if provider.APIKey != "" {
			baseURL := provider.BaseURL
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			return baseURL, provider.APIKey, provider.Model
		}
	}

	// 2) Fallback: any enabled provider with API key
	if err := repository.DB.Where("is_enabled = ? AND api_key != ''", true).First(&provider).Error; err == nil {
		baseURL := provider.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return baseURL, provider.APIKey, provider.Model
	}

	// 3) Environment variable fallback
	return getEnvOr("AI_BASE_URL", "https://api.openai.com/v1"),
		os.Getenv("AI_API_KEY"),
		getEnvOr("AI_MODEL", "gpt-4")
}

func getEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// =============== Helper: LLM call ===============

func callLLM(baseURL, apiKey, model, system, user string) string {
	// Ensure baseURL has /chat/completions
	url := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.3,
		"max_tokens":  4000,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Warnf("KB LLM call failed: %v", err)
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		logger.Log.Warnf("KB LLM HTTP %d: %s", resp.StatusCode, string(body[:kbMin(len(body), 300)]))
		return ""
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Choices) == 0 {
		return ""
	}
	return result.Choices[0].Message.Content
}

// =============== Helper: data extraction ===============

func buildTimeline(changelog []jiraChangelog, comments []jiraComment) []TimelineEv {
	var events []TimelineEv

	for _, h := range changelog {
		author := "未知"
		if h.Author != nil {
			if dn, ok := h.Author["displayName"].(string); ok {
				author = dn
			}
		}
		for _, item := range h.Items {
			if item.Field == "status" || item.Field == "resolution" {
				events = append(events, TimelineEv{
					Time:    h.Created,
					Author:  author,
					Type:    "状态变更",
					Content: fmt.Sprintf("%s: %s → %s", item.Field, item.FromString, item.ToString),
				})
			}
		}
	}

	for _, c := range comments {
		author := "未知"
		if c.Author != nil {
			if dn, ok := c.Author["displayName"].(string); ok {
				author = dn
			}
		}
		contentText := stripHTML(c.RenderedBody)
		if len(contentText) > 200 {
			contentText = contentText[:200] + "..."
		}
		events = append(events, TimelineEv{
			Time:    c.Created,
			Author:  author,
			Type:    "评论",
			Content: contentText,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Time < events[j].Time
	})
	return events
}

func formatTimelineRaw(events []TimelineEv) string {
	var lines []string
	for _, e := range events {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s", e.Time, e.Author, e.Type, e.Content))
	}
	return strings.Join(lines, "\n")
}

func extractCustomer(fields map[string]interface{}, fallback string) string {
	// 1. Check custom fields with customer-related names
	keywords := []string{"客户", "customer", "组织", "organization", "company", "租户", "tenant", "账号", "account"}
	for k, v := range fields {
		kl := strings.ToLower(k)
		for _, w := range keywords {
			if strings.Contains(kl, w) {
				val := displayValue(v)
				if val != "" {
					return val
				}
			}
		}
	}

	// 2. Try to extract from summary (title)
	// Common patterns in EasyStack Jira:
	//   "中国邮政储蓄银行-廊坊2023ES测试云1(CSE-4595)-容器平台部署时使用nfs存储时，prometheus pod无法正常启动"
	//   Customer name is everything before the (CSE-XXXX) or (ECSL-XXXX) pattern
	summary := displayValue(fields["summary"])
	if summary != "" {
		// Pattern 1: Extract everything before (CSE-XXXX) or (ECSL2-XXXX) etc.
		reCSE := regexp.MustCompile(`^(.+?)\s*\([A-Z][A-Z0-9]*-\d+\)`)
		if m := reCSE.FindStringSubmatch(summary); len(m) >= 2 {
			customer := strings.TrimRight(m[1], "- ")
			if len(customer) >= 2 {
				return customer
			}
		}

		// Pattern 2: "客户名-问题描述" where problem description starts with Chinese tech keywords
		reTech := regexp.MustCompile(`^(.+?)[-—]\s*(?:容器|云平台|虚拟|网络|存储|集群|节点|服务|部署|升级|扩容|备份|监控|告警|迁移|安装)`)
		if m := reTech.FindStringSubmatch(summary); len(m) >= 2 {
			customer := strings.TrimRight(m[1], "- ")
			if len(customer) >= 2 {
				return customer
			}
		}

		// Pattern 3: Simple first segment before "-" (original logic, relaxed length)
		re := regexp.MustCompile(`^([^(（]+)-`)
		if m := re.FindStringSubmatch(summary); len(m) >= 2 && len(m[1]) >= 2 {
			return strings.TrimSpace(m[1])
		}
	}

	// 3. Check "客户名称" field specifically (some Jira instances use this)
	if cf, ok := fields["customfield_10601"]; ok {
		val := displayValue(cf)
		if val != "" {
			return val
		}
	}

	if fallback != "" {
		return fallback
	}
	return "（未识别）"
}

func displayValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		for _, k := range []string{"displayName", "name", "value", "emailAddress"} {
			if s, ok := val[k].(string); ok && s != "" {
				return s
			}
		}
		return ""
	case []interface{}:
		var parts []string
		for _, item := range val {
			s := displayValue(item)
			if s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

func getMapField(fields map[string]interface{}, key, subKey string) interface{} {
	if v, ok := fields[key]; ok && v != nil {
		if m, ok := v.(map[string]interface{}); ok {
			return m[subKey]
		}
	}
	return nil
}

func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(s, " ")
	// Collapse whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func cleanCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		re := regexp.MustCompile("(?s)^```[a-zA-Z]*\n?")
		s = re.ReplaceAllString(s, "")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// =============== Helper: HTML rendering ===============

func renderStorageHTML(content *KBContentGenerated, descHTML string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<h1>%s</h1>", html.EscapeString(content.Title)))

	b.WriteString("<h2>一、问题背景</h2>")
	b.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(content.Background)))

	b.WriteString("<h2>二、工单信息</h2>")
	b.WriteString(content.TicketTable)

	b.WriteString("<h2>三、客户名称</h2>")
	b.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(content.Customer)))

	b.WriteString("<h2>四、问题现象</h2>")
	if descHTML != "" {
		b.WriteString(descHTML)
	} else {
		b.WriteString("<p>（无描述）</p>")
	}

	b.WriteString("<h2>五、解决过程</h2>")
	b.WriteString("<h3>5.1 时间线</h3>")
	b.WriteString(renderTimelineTable(content.Timeline))
	b.WriteString("<h3>5.2 处理内容汇总</h3>")
	b.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(content.ProcessSummary)))

	b.WriteString("<h2>六、问题总结</h2>")
	b.WriteString("<h3>技术结论</h3>")
	b.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(content.TechConclusion)))
	b.WriteString("<h3>结果</h3>")
	b.WriteString(fmt.Sprintf("<p>%s</p>", html.EscapeString(content.Result)))

	b.WriteString("<h2>七、改进建议</h2>")
	b.WriteString("<ol>")
	for _, s := range content.Suggestions {
		b.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(s)))
	}
	b.WriteString("</ol>")

	return b.String()
}

func renderTimelineTable(timeline []TimelineEv) string {
	var b strings.Builder
	b.WriteString("<table><tbody>")
	b.WriteString("<tr><th>时间</th><th>操作人</th><th>内容</th></tr>")
	for _, e := range timeline {
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(e.Time), html.EscapeString(e.Author), html.EscapeString(e.Content)))
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

func kbMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
