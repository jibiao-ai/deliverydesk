package repository

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(cfg config.DatabaseConfig) error {
	var db *gorm.DB
	var err error

	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "mysql"
	}

	switch dbDriver {
	case "sqlite":
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "deliverydesk.db"
		}
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open sqlite: %w", err)
		}
		logger.Log.Infof("Using SQLite database: %s", dbPath)

	default: // mysql
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

		logger.Log.Infof("Connecting to MySQL: %s@tcp(%s:%d)/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)

		for i := 0; i < 60; i++ {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
				Logger: gormlogger.Default.LogMode(gormlogger.Info),
			})
			if err == nil {
				// Verify connection actually works
				sqlDB, pingErr := db.DB()
				if pingErr == nil && sqlDB.Ping() == nil {
					break
				}
				if pingErr != nil {
					err = pingErr
				} else {
					err = sqlDB.Ping()
				}
			}
			if i%10 == 0 {
				logger.Log.Warnf("Waiting for MySQL (attempt %d/60): %v", i+1, err)
			}
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL after 60 attempts (2 min): %w", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
		logger.Log.Info("Using MySQL database")
	}

	// Fix database/table collation to avoid "key too long" error (Error 1071)
	// MySQL 8.0 with utf8mb4_unicode_ci can cause index key length > 3072 bytes
	// when GORM creates composite unique indexes with DeletedAt for soft-delete
	logger.Log.Info("Checking database collation...")
	if dbDriver != "sqlite" {
		fixCollationSQL := []string{
			fmt.Sprintf("ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", cfg.DBName),
		}
		for _, sql := range fixCollationSQL {
			if execErr := db.Exec(sql).Error; execErr != nil {
				logger.Log.Warnf("Collation fix SQL warning (non-fatal): %v", execErr)
			}
		}
		// Convert existing tables if they exist
		var tables []string
		db.Raw("SHOW TABLES").Scan(&tables)
		for _, table := range tables {
			alterSQL := fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", table)
			if execErr := db.Exec(alterSQL).Error; execErr != nil {
				logger.Log.Warnf("Table collation fix warning for %s (non-fatal): %v", table, execErr)
			}
		}
		logger.Log.Info("Database collation check completed")
	}

	// Auto migrate - migrate tables one by one for better error diagnosis
	logger.Log.Info("Starting database table migration...")
	migrationModels := map[string]interface{}{
		"User":            &model.User{},
		"LDAPConfig":      &model.LDAPConfig{},
		"Agent":           &model.Agent{},
		"Skill":           &model.Skill{},
		"SkillDocument":   &model.SkillDocument{},
		"AgentSkill":      &model.AgentSkill{},
		"Conversation":    &model.Conversation{},
		"Message":         &model.Message{},
		"TaskLog":         &model.TaskLog{},
		"WebsiteCategory": &model.WebsiteCategory{},
		"WebsiteLink":     &model.WebsiteLink{},
		"AIProvider":      &model.AIProvider{},
		"OperationLog":    &model.OperationLog{},
		"TotpApplication": &model.TotpApplication{},
		"SystemSetting":   &model.SystemSetting{},
		"JiraIssueCache":  &model.JiraIssueCache{},
		"WorktimeUser":    &model.WorktimeUser{},
		"WorktimeCache":   &model.WorktimeCache{},
		"ProjectInfo":     &model.ProjectInfo{},
		"OpsEnvironment":  &model.OpsEnvironment{},
		"KBHistory":       &model.KBHistory{},
	}
	migrationOrder := []string{
		"User", "LDAPConfig", "Agent", "Skill", "SkillDocument", "AgentSkill",
		"Conversation", "Message", "TaskLog",
		"WebsiteCategory", "WebsiteLink", "AIProvider", "OperationLog",
		"TotpApplication", "SystemSetting", "JiraIssueCache", "WorktimeUser", "WorktimeCache",
		"ProjectInfo", "OpsEnvironment", "KBHistory",
	}
	for _, name := range migrationOrder {
		m := migrationModels[name]
		logger.Log.Infof("Migrating table: %s ...", name)
		if migrateErr := db.AutoMigrate(m); migrateErr != nil {
			logger.Log.Warnf("First migration attempt for %s failed: %v", name, migrateErr)
			// If Error 1071 (key too long), try dropping the table and re-creating
			if strings.Contains(migrateErr.Error(), "1071") || strings.Contains(migrateErr.Error(), "key") {
				logger.Log.Warnf("Attempting to drop and recreate table %s to fix key length issue...", name)
				if dropErr := db.Migrator().DropTable(m); dropErr != nil {
					logger.Log.Warnf("Drop table %s warning: %v", name, dropErr)
				}
				if retryErr := db.AutoMigrate(m); retryErr != nil {
					logger.Log.Errorf("Failed to migrate table %s after drop+recreate: %v", name, retryErr)
					return fmt.Errorf("auto migration failed on table %s: %w", name, retryErr)
				}
				logger.Log.Infof("Table %s recreated successfully after drop", name)
			} else {
				return fmt.Errorf("auto migration failed on table %s: %w", name, migrateErr)
			}
		} else {
			logger.Log.Infof("Table %s migrated successfully", name)
		}
	}

	// Explicit column migrations for existing databases
	// Fix: totp_pass was VARCHAR(256) which is too small for TOTP server JSON responses
	if db.Migrator().HasTable(&model.TotpApplication{}) {
		if err := db.Exec("ALTER TABLE totp_applications MODIFY COLUMN totp_pass TEXT").Error; err != nil {
			logger.Log.Warnf("ALTER totp_pass column warning (may already be TEXT): %v", err)
		}
	}

	DB = db
	logger.Log.Info("Database connection established and migrated")

	// Seed default data
	seedDefaultData(db)

	return nil
}

func seedDefaultData(db *gorm.DB) {
	// Ensure default admin user with correct password
	const adminPlainPassword = "Admin@2024!"
	adminHashBytes, err := bcrypt.GenerateFromPassword([]byte(adminPlainPassword), 10)
	if err != nil {
		logger.Log.Errorf("Failed to hash admin password: %v", err)
		return
	}
	adminPasswordHash := string(adminHashBytes)

	var admin model.User
	result := db.Where("username = ?", "admin").First(&admin)
	if result.Error != nil {
		// Admin does not exist, create it
		admin = model.User{
			Username:    "admin",
			Password:    adminPasswordHash,
			Email:       "admin@deliverydesk.local",
			DisplayName: "管理员",
			Role:        "admin",
			AuthType:    "local",
		}
		if err := db.Create(&admin).Error; err != nil {
			logger.Log.Errorf("Failed to create admin user: %v", err)
			return
		}
		logger.Log.Infof("Default admin user created (id=%d, hash_len=%d)", admin.ID, len(admin.Password))
	} else {
		// Admin exists — always force reset password to ensure it works
		// This fixes issues where the password hash was corrupted or stored incorrectly
		admin.Password = adminPasswordHash
		if err := db.Save(&admin).Error; err != nil {
			logger.Log.Errorf("Failed to reset admin password: %v", err)
			return
		}
		logger.Log.Infof("Admin password has been reset to default (id=%d, hash_len=%d)", admin.ID, len(admin.Password))
	}

	// Seed default AI providers (13 vendors)
	defaultProviders := []model.AIProvider{
		{Name: "openai", Label: "OpenAI", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o", IsDefault: true, IsEnabled: true, Description: "GPT-4o / GPT-4 / GPT-3.5 系列", IconURL: "openai"},
		{Name: "deepseek", Label: "DeepSeek", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", IsDefault: false, IsEnabled: true, Description: "DeepSeek V4，支持 deepseek-v4-flash / deepseek-v4-pro，1M 上下文", IconURL: "deepseek"},
		{Name: "qwen", Label: "通义千问", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Model: "qwen-plus", IsDefault: false, IsEnabled: true, Description: "阿里云 Qwen-Plus / Qwen-Max 系列", IconURL: "qwen"},
		{Name: "glm", Label: "智谱 GLM", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Model: "glm-4", IsDefault: false, IsEnabled: true, Description: "智谱 AI GLM-4 / GLM-4-Flash 系列", IconURL: "glm"},
		{Name: "minimax", Label: "MiniMax", BaseURL: "https://api.minimax.chat/v1", Model: "abab6.5s-chat", IsDefault: false, IsEnabled: true, Description: "MiniMax abab 系列", IconURL: "minimax"},
		{Name: "siliconflow", Label: "硅基流动", BaseURL: "https://api.siliconflow.cn/v1", Model: "Qwen/Qwen2.5-7B-Instruct", IsDefault: false, IsEnabled: true, Description: "支持 Qwen / DeepSeek / GLM 开源模型推理", IconURL: "siliconflow"},
		{Name: "moonshot", Label: "Moonshot (Kimi)", BaseURL: "https://api.moonshot.cn/v1", Model: "moonshot-v1-8k", IsDefault: false, IsEnabled: true, Description: "超长上下文，8k / 32k / 128k", IconURL: "moonshot"},
		{Name: "ernie", Label: "百度文心一言", BaseURL: "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat", Model: "ernie-4.5-8k", IsDefault: false, IsEnabled: true, Description: "ERNIE 4.5 / 4.0 / Speed 系列", IconURL: "ernie"},
		{Name: "doubao", Label: "火山引擎（豆包）", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", Model: "doubao-pro-4k", IsDefault: false, IsEnabled: true, Description: "字节豆包 doubao-pro / lite 系列", IconURL: "doubao"},
		{Name: "hunyuan", Label: "腾讯混元", BaseURL: "https://hunyuan.tencentcloudapi.com", Model: "hunyuan-pro", IsDefault: false, IsEnabled: true, Description: "混元 pro / standard 系列", IconURL: "hunyuan"},
		{Name: "baichuan", Label: "百川智能", BaseURL: "https://api.baichuan-ai.com/v1", Model: "Baichuan4", IsDefault: false, IsEnabled: true, Description: "Baichuan4 / Baichuan3-Turbo 系列", IconURL: "baichuan"},
		{Name: "anthropic", Label: "Anthropic Claude", BaseURL: "https://api.anthropic.com/v1", Model: "claude-3-5-sonnet-20241022", IsDefault: false, IsEnabled: true, Description: "claude-3-5-sonnet / haiku / opus", IconURL: "anthropic"},
		{Name: "gemini", Label: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta", Model: "gemini-2.0-flash", IsDefault: false, IsEnabled: true, Description: "gemini-2.0-flash / 1.5-pro 系列", IconURL: "gemini"},
	}
	for _, p := range defaultProviders {
		var existing model.AIProvider
		if err := db.Where("name = ?", p.Name).First(&existing).Error; err != nil {
			db.Create(&p)
		}
	}
	logger.Log.Info("Default AI providers seeded")

	// Migrate existing DeepSeek provider to V4 API format
	// The V4 API removed the /v1 prefix and uses new model names
	var dsProvider model.AIProvider
	if err := db.Where("name = ?", "deepseek").First(&dsProvider).Error; err == nil {
		needsUpdate := false
		// Fix base URL: remove /v1 suffix
		if strings.HasSuffix(dsProvider.BaseURL, "/v1") {
			dsProvider.BaseURL = strings.TrimSuffix(dsProvider.BaseURL, "/v1")
			needsUpdate = true
		}
		// Migrate deprecated model names to V4
		if dsProvider.Model == "deepseek-chat" {
			dsProvider.Model = "deepseek-v4-flash"
			needsUpdate = true
		} else if dsProvider.Model == "deepseek-reasoner" {
			dsProvider.Model = "deepseek-v4-flash"
			needsUpdate = true
		}
		// Update description to reflect V4
		if !strings.Contains(dsProvider.Description, "V4") {
			dsProvider.Description = "DeepSeek V4，支持 deepseek-v4-flash / deepseek-v4-pro，1M 上下文"
			needsUpdate = true
		}
		if needsUpdate {
			db.Save(&dsProvider)
			logger.Log.Info("DeepSeek provider migrated to V4 API format")
		}
	}

	// Seed default skills (delivery skill + community skills)
	var skillCount int64
	db.Model(&model.Skill{}).Count(&skillCount)
	if skillCount == 0 {
		skills := []model.Skill{
			{
				Name:        "交付技能的skills",
				Description: "基于交付文档知识库的技能，可以帮助用户编写实施方案、回答交付相关问题（交付边界、兼容性等）",
				Type:        "delivery",
				Category:    "delivery-skill",
				IsActive:    true,
			},
		}
		for _, s := range skills {
			db.Create(&s)
		}
		logger.Log.Info("Default delivery skill created")
	}

	// Seed community skills (k8s-operator, openstack-operator)
	seedCommunitySkills(db)

	// Seed ops skills (双因子申请, 过保维保查询)
	seedOpsSkills(db)

	// Seed open-source market skills (PDF, document processing, etc.)
	seedMarketSkills(db)

	// Clean up deprecated knowledge skills (张雪峰考研, 乔布斯)
	cleanupDeprecatedSkills(db)

	// Seed default system settings
	seedSystemSettings(db)

	// Seed default agents
	var agentCount int64
	db.Model(&model.Agent{}).Count(&agentCount)
	if agentCount == 0 {
		// Get the delivery skill ID for linking
		var deliverySkill model.Skill
		db.Where("category = ?", "delivery-skill").First(&deliverySkill)

		agents := []model.Agent{
			{
				Name:        "交付黄页智能体",
				Description: "汇聚交付团队常用知识库、文档链接和操作指南的智能体，快速导航到需要的资源",
				SystemPrompt: "你是交付黄页智能助手，帮助交付工程师快速找到所需的文档、工具和资源链接。你熟悉所有常用的交付工具和系统，能够根据用户的需求推荐最合适的资源。",
				Model:       "",
				Temperature: 0.3,
				MaxTokens:   4096,
				IsActive:    true,
				CreatedBy:   1,
			},
			{
				Name:         "交付专家",
				Description:  "交付专家智能体 - 连接交付技能知识库，提供实施方案编写、交付边界确认、兼容性查询等专业服务。严格遵循铁律规则，所有回答基于真实文档数据。",
				SystemPrompt: deliveryExpertSystemPrompt(),
				Model:        "",
				Temperature:  0.2,
				MaxTokens:    8192,
				IsActive:     true,
				IsPublished:  true,
				IronRules:    true,
				CreatedBy:    1,
			},
		}
		for _, a := range agents {
			db.Create(&a)
		}
		// Link delivery expert to delivery skill
		if deliverySkill.ID > 0 {
			var expert model.Agent
			db.Where("name = ?", "交付专家").First(&expert)
			if expert.ID > 0 {
				db.Create(&model.AgentSkill{AgentID: expert.ID, SkillID: deliverySkill.ID})
				// Also link community skills
				var k8sSkill, osSkill model.Skill
				db.Where("category = ?", "k8s-operator").First(&k8sSkill)
				db.Where("category = ?", "openstack-operator").First(&osSkill)
				if k8sSkill.ID > 0 {
					db.Create(&model.AgentSkill{AgentID: expert.ID, SkillID: k8sSkill.ID})
				}
				if osSkill.ID > 0 {
					db.Create(&model.AgentSkill{AgentID: expert.ID, SkillID: osSkill.ID})
				}
			}
		}
		logger.Log.Info("Default agents created (including 交付专家)")
	}

	// Seed 运维专家 agent (idempotent — only if it doesn't exist yet)
	seedOpsExpertAgent(db)

	// Seed 双因子申请智能体 agent (idempotent — only if it doesn't exist yet)
	seedTotpAgent(db)

	// Seed 过保维保查询智能体 agent (idempotent — only if it doesn't exist yet)
	seedOpsEnvWarrantyAgent(db)

	// Seed website links from Excel data
	var catCount int64
	db.Model(&model.WebsiteCategory{}).Count(&catCount)
	if catCount == 0 {
		seedWebsiteLinks(db)
	}

	// Fix typo: 合适费控报销 -> 合思费控报销
	db.Model(&model.WebsiteLink{}).Where("name = ?", "合适费控报销").Update("name", "合思费控报销")

	// Seed default worktime users (idempotent - only if table is empty)
	var worktimeUserCount int64
	db.Model(&model.WorktimeUser{}).Count(&worktimeUserCount)
	if worktimeUserCount == 0 {
		defaultWorktimeUsers := []string{
			"王利东", "郭建华", "潘牧阳", "樊继标", "任超伟", "林飞飞", "何哲",
			"王志峰", "汪翔", "梁兆", "李智豪", "刘沪宁", "刘涛", "孟竹青",
			"岳金虎", "何梓潇", "孙洪峰", "强有明", "王长庆", "王秋鹤", "樊亚州",
			"陈雷", "贾书林", "王春春", "王明福", "孙奇", "尚玉龙", "张杰",
			"陈京川", "赵起", "傅友权", "冯伯伟", "万德良", "曹瑞", "李阳宇",
		}
		for _, name := range defaultWorktimeUsers {
			db.FirstOrCreate(&model.WorktimeUser{}, model.WorktimeUser{Name: name})
		}
		logger.Log.Infof("Seeded %d default worktime users", len(defaultWorktimeUsers))
	}
}

func seedWebsiteLinks(db *gorm.DB) {
	type linkData struct {
		Name string
		URL  string
		Icon string
	}
	type categoryData struct {
		Name  string
		Icon  string
		Links []linkData
	}

	categories := []categoryData{
		{
			Name: "日常系统",
			Icon: "Monitor",
			Links: []linkData{
				{Name: "商业存储对接包", URL: "http://octopus.easystack.io/", Icon: "database"},
				{Name: "拓扑制作系统", URL: "http://lic.easystack.cn/topoweb/login", Icon: "network"},
				{Name: "Redmine系统", URL: "https://redmine.easystack.cn/login", Icon: "redmine"},
				{Name: "Jira系统", URL: "https://easystack.atlassian.net/jira/for-you", Icon: "jira"},
				{Name: "Confluence系统", URL: "https://easystack.atlassian.net/wiki", Icon: "confluence"},
				{Name: "VPN工具", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1099467131/01-VPN", Icon: "vpn"},
				{Name: "企业邮箱", URL: "https://mail.qiye.163.com/static/login/", Icon: "email"},
				{Name: "企业网盘", URL: "https://pan.easystack.io/", Icon: "cloud-storage"},
			},
		},
		{
			Name: "产品资料",
			Icon: "Package",
			Links: []linkData{
				{Name: "V6.0.1/V6.1.1标准安装介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1695711233", Icon: "download"},
				{Name: "V6.0.1/V6.0.2/V6.1.1产品升级介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1128104392", Icon: "upgrade"},
				{Name: "V6.2.1标准安装介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2205548547/V621", Icon: "download"},
				{Name: "V6.2.1产品升级介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2226094087/V621", Icon: "upgrade"},
			},
		},
		{
			Name: "常用网站",
			Icon: "Globe",
			Links: []linkData{
				{Name: "Confluence使用介绍", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1099499945/04-Confluence", Icon: "confluence"},
				{Name: "交付黄页", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1102644405/05-", Icon: "book"},
				{Name: "合思费控报销", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1659373029/07-", Icon: "receipt"},
			},
		},
		{
			Name: "交付标准",
			Icon: "ClipboardCheck",
			Links: []linkData{
				{Name: "V6.2.1安装部署手册", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2114355900/V621", Icon: "manual"},
				{Name: "V6.0.1/V6.1.1安装部署手册", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1042744182", Icon: "manual"},
				{Name: "实施阶段文档模版", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1847656703/00-", Icon: "template"},
				{Name: "云平台勘误", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/folder/2801336339", Icon: "alert"},
				{Name: "万博迁移软件相关", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1908670626/00-", Icon: "migrate"},
				{Name: "neutron-az、多生产(业务)网配包制作", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1659437057/09-", Icon: "network"},
				{Name: "云杉网络对接包", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1764688072/10-", Icon: "network"},
				{Name: "标准镜像制作相关", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/85951587", Icon: "image"},
				{Name: "标准镜像维护列表", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/85951522/03-", Icon: "image"},
				{Name: "邮储专项交付", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1660649520/02-", Icon: "bank"},
			},
		},
		{
			Name: "运维规范",
			Icon: "ShieldCheck",
			Links: []linkData{
				{Name: "驻场运维规范", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/2486763531/09-", Icon: "standard"},
				{Name: "技能培训", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/2486927470/03-", Icon: "training"},
				{Name: "SDN相关服务问题", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1682866530/SDN", Icon: "network"},
				{Name: "V6.2.1标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/2245297715/ECS+V621", Icon: "change"},
				{Name: "V6.1.1标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1682669599/ECS+V611", Icon: "change"},
				{Name: "V6标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1110181031/ECS+V6", Icon: "change"},
				{Name: "V6运维手册库", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/873759399/ECS+V6", Icon: "book"},
			},
		},
	}

	for i, cat := range categories {
		category := model.WebsiteCategory{
			Name:      cat.Name,
			Icon:      cat.Icon,
			SortOrder: i,
		}
		db.Create(&category)

		for j, link := range cat.Links {
			websiteLink := model.WebsiteLink{
				CategoryID: category.ID,
				Name:       link.Name,
				URL:        link.URL,
				Icon:       link.Icon,
				SortOrder:  j,
			}
			db.Create(&websiteLink)
		}
	}
	logger.Log.Info("Website links seeded (5 categories)")
}

func seedCommunitySkills(db *gorm.DB) {
	communityDefs := []struct {
		Name        string
		Description string
		Category    string
		ToolDefs    string
	}{
		{
			Name:        "k8s-operator",
			Description: "Kubernetes 集群管理技能 - 提供 K8S 集群运维、故障排查、资源管理、Pod调度、网络策略、存储管理等操作指导",
			Category:    "k8s-operator",
			ToolDefs:    `[{"name":"k8s_cluster_status","description":"获取K8S集群状态"},{"name":"k8s_pod_diagnosis","description":"诊断Pod异常"},{"name":"k8s_resource_check","description":"检查资源配额"},{"name":"k8s_yaml_generator","description":"生成K8S YAML配置"}]`,
		},
		{
			Name:        "openstack-operator",
			Description: "OpenStack 云平台管理技能 - 提供 OpenStack 部署运维、计算/网络/存储服务管理、故障排查、性能调优等操作指导",
			Category:    "openstack-operator",
			ToolDefs:    `[{"name":"os_service_status","description":"检查OpenStack服务状态"},{"name":"os_compute_diagnosis","description":"诊断计算服务问题"},{"name":"os_network_diagnosis","description":"诊断网络问题"},{"name":"os_compatibility_check","description":"检查兼容性"}]`,
		},
		{
			Name:        "sre-operator",
			Description: "SRE 站点可靠性工程技能 - 提供 SLO/SLI 定义、故障管理、容量规划、变更管理、自动化运维、监控告警、事件响应等 SRE 实践指导",
			Category:    "sre-operator",
			ToolDefs:    `[{"name":"sre_slo_calculator","description":"计算SLO和错误预算"},{"name":"sre_incident_response","description":"引导事件响应流程"},{"name":"sre_capacity_planning","description":"容量规划和预测"},{"name":"sre_change_risk","description":"变更风险评估"},{"name":"sre_toil_analysis","description":"Toil分析和自动化消除"},{"name":"sre_postmortem_guide","description":"事后复盘引导"}]`,
		},
	}
	for _, def := range communityDefs {
		var existing model.Skill
		if err := db.Where("category = ?", def.Category).First(&existing).Error; err != nil {
			skill := model.Skill{
				Name:        def.Name,
				Description: def.Description,
				Type:        "community",
				Category:    def.Category,
				ToolDefs:    def.ToolDefs,
				IsActive:    true,
			}
			db.Create(&skill)
			logger.Log.Infof("Community skill '%s' seeded", def.Name)
		}
	}
}

// seedOpsSkills seeds operations skills (双因子申请, 过保维保查询)
func seedOpsSkills(db *gorm.DB) {
	opsDefs := []struct {
		Name         string
		Description  string
		Category     string
		ToolDefs     string
		SystemPrompt string
	}{
		{
			Name:        "双因子申请",
			Description: "双因子认证管理技能 - 支持调用【双因子管理】能力，输入一个 Case 号自动查询并返回 Roller 双因子验证码和动态密码（OTP）。适用于运维环境登录认证场景。",
			Category:    "totp-query",
			ToolDefs:    `[{"name":"totp_query_by_case","description":"根据Case号查询双因子密码，返回Roller双因子和动态OTP密码","parameters":{"case_number":"CSE案例编号"}},{"name":"totp_generate_otp","description":"生成当前时刻的Roller动态密码(OTP)","parameters":{}},{"name":"totp_list_recent","description":"列出最近的双因子申请记录","parameters":{"limit":"返回数量，默认10"}}]`,
			SystemPrompt: `你是「双因子申请」智能体，专门帮助用户快速获取运维环境的双因子认证密码。

## 核心功能
当用户输入一个 Case 号（如 ECSL2-50579、ECSDESK-1234 等格式），系统会自动：
1. 从 Jira 查询该 Case 对应的客户名称和项目名称
2. 调用 TOTP 系统生成 Roller 双因子验证码（OTP，30秒有效）
3. 调用 TOTP 服务器生成动态密码

## 使用方式
直接输入 Case 号即可，例如：
- ECSL2-50579
- ECSDESK-12345

系统会自动返回查询结果，包括 Roller OTP 和动态密码。

## 注意事项
- Roller OTP 有效期为 30 秒，请尽快使用
- 动态密码由 TOTP 服务器实时生成
- 如果查询失败，请确认 Case 号格式正确且存在于 Jira 系统中`,
		},
		{
			Name:        "过保维保查询",
			Description: "运维环境维保信息查询技能 - 支持调用【运维环境】接口，查询客户或项目的维保信息。可返回 CSE 编号、项目名称、客户名称、节点数、维保到期时间、当前状态、SLA 等级等关键信息。支持按客户名、项目名模糊搜索。",
			Category:    "ops-env-warranty",
			ToolDefs:    `[{"name":"ops_env_query","description":"查询运维环境维保信息，支持按客户名或项目名搜索","parameters":{"search":"客户名称或项目名称关键字","status":"状态过滤: in_progress/done/discarded/all"}},{"name":"ops_env_expiring","description":"查询即将到期的维保环境（按维保结束日期排序）","parameters":{"days":"未来N天内到期，默认30"}},{"name":"ops_env_stats","description":"获取运维环境统计概览（状态分布、区域分布、节点总数）","parameters":{}}]`,
			SystemPrompt: `你是「过保维保查询」智能体，帮助运维人员快速查询客户环境的维保状态。

## 核心功能
输入客户名称、项目名称或 CSE 编号，系统自动查询运维环境数据库并返回：
- CSE 编号、客户名、项目名
- 节点数量
- 维保开始/结束时间（过期标红告警）
- 当前状态（进行中/已完成/已弃用）
- SLA 等级、运维区域

## 使用方式
直接输入查询关键字即可，例如：
- 客户名称：中国移动、招商银行
- 项目名称：私有云项目
- CSE 编号：CSE-1234

系统会自动模糊匹配并返回维保信息表格。`,
		},
	}
	for _, def := range opsDefs {
		var existing model.Skill
		if err := db.Where("category = ?", def.Category).First(&existing).Error; err != nil {
			skill := model.Skill{
				Name:         def.Name,
				Description:  def.Description,
				Type:         "ops",
				Category:     def.Category,
				ToolDefs:     def.ToolDefs,
				SystemPrompt: def.SystemPrompt,
				IsActive:     true,
			}
			db.Create(&skill)
			logger.Log.Infof("Ops skill '%s' seeded", def.Name)
		} else if existing.SystemPrompt == "" && def.SystemPrompt != "" {
			// Update existing skill with SystemPrompt if it was missing
			db.Model(&existing).Update("system_prompt", def.SystemPrompt)
			logger.Log.Infof("Ops skill '%s' SystemPrompt updated", def.Name)
		}
	}
}

// seedMarketSkills seeds basic open-source market skills (PDF processing, text extraction, etc.)
func seedMarketSkills(db *gorm.DB) {
	marketDefs := []struct {
		Name        string
		Description string
		Category    string
		Type        string
		ToolDefs    string
	}{
		{
			Name:        "PDF文档解析",
			Description: "PDF 文档智能解析技能 - 支持上传 PDF 文件并自动提取文本内容、表格数据、图片描述等。适用于合同解析、报告提取、技术文档阅读等场景。",
			Category:    "pdf-parser",
			Type:        "community",
			ToolDefs:    `[{"name":"pdf_extract_text","description":"从PDF文件中提取纯文本内容","parameters":{"file_path":"PDF文件路径"}},{"name":"pdf_extract_tables","description":"从PDF中识别并提取表格数据","parameters":{"file_path":"PDF文件路径","page":"指定页码，默认全部"}},{"name":"pdf_summarize","description":"对PDF文档生成摘要","parameters":{"file_path":"PDF文件路径","max_length":"摘要最大字数"}}]`,
		},
		{
			Name:        "Excel数据分析",
			Description: "Excel/CSV 数据分析技能 - 支持上传 Excel 或 CSV 文件，进行数据统计、筛选、透视分析和可视化图表生成。适用于运营数据分析、项目进度统计等场景。",
			Category:    "excel-analyzer",
			Type:        "community",
			ToolDefs:    `[{"name":"excel_read_sheet","description":"读取Excel工作表内容","parameters":{"file_path":"文件路径","sheet_name":"工作表名称，默认第一个"}},{"name":"excel_statistics","description":"对指定列进行统计分析（求和/均值/最大/最小/计数）","parameters":{"file_path":"文件路径","column":"列名","operation":"统计操作"}},{"name":"excel_filter","description":"按条件筛选数据行","parameters":{"file_path":"文件路径","column":"列名","condition":"筛选条件"}}]`,
		},
		{
			Name:        "Markdown文档助手",
			Description: "Markdown 文档编写与转换技能 - 支持 Markdown 格式文档的智能编写、格式转换（MD→HTML/PDF）、目录生成、文档模板套用等功能。",
			Category:    "markdown-helper",
			Type:        "community",
			ToolDefs:    `[{"name":"md_generate_toc","description":"为Markdown文档自动生成目录结构","parameters":{"content":"Markdown内容"}},{"name":"md_to_html","description":"将Markdown转换为HTML","parameters":{"content":"Markdown内容"}},{"name":"md_template","description":"根据模板生成Markdown文档","parameters":{"template_type":"模板类型: meeting_notes/tech_doc/changelog/runbook","context":"文档上下文信息"}}]`,
		},
		{
			Name:        "日志分析",
			Description: "日志智能分析技能 - 支持解析各类系统日志（syslog、journalctl、K8S events、OpenStack logs），自动识别错误模式、异常时间段和根因关联。",
			Category:    "log-analyzer",
			Type:        "ops",
			ToolDefs:    `[{"name":"log_parse","description":"解析日志文本，识别错误和警告","parameters":{"log_content":"日志内容","log_format":"日志格式: syslog/json/k8s/openstack"}},{"name":"log_timeline","description":"按时间线梳理事件链","parameters":{"log_content":"日志内容","time_range":"时间范围"}},{"name":"log_root_cause","description":"分析日志中的根因关联","parameters":{"error_message":"错误信息","log_content":"上下文日志"}}]`,
		},
		{
			Name:        "网络诊断",
			Description: "网络故障诊断技能 - 提供网络连通性检测、DNS 解析、路由追踪、端口扫描等网络排查能力。适用于运维场景下的网络问题快速定位。",
			Category:    "network-diagnosis",
			Type:        "ops",
			ToolDefs:    `[{"name":"net_connectivity","description":"检测目标主机连通性","parameters":{"host":"目标主机IP或域名","port":"端口号"}},{"name":"net_dns_resolve","description":"DNS解析查询","parameters":{"domain":"域名","record_type":"记录类型: A/AAAA/CNAME/MX/NS"}},{"name":"net_traceroute","description":"路由追踪","parameters":{"host":"目标主机"}},{"name":"net_port_scan","description":"端口扫描","parameters":{"host":"目标主机","port_range":"端口范围，如1-1024"}}]`,
		},
	}
	for _, def := range marketDefs {
		var existing model.Skill
		if err := db.Where("category = ?", def.Category).First(&existing).Error; err != nil {
			skill := model.Skill{
				Name:        def.Name,
				Description: def.Description,
				Type:        def.Type,
				Category:    def.Category,
				ToolDefs:    def.ToolDefs,
				IsActive:    true,
			}
			db.Create(&skill)
			logger.Log.Infof("Market skill '%s' seeded", def.Name)
		}
	}
}

// cleanupDeprecatedSkills removes deprecated skills (张雪峰考研, 乔布斯) from the database
func cleanupDeprecatedSkills(db *gorm.DB) {
	deprecatedCategories := []string{"zhangxuefeng-skill", "steve-jobs-skill"}
	for _, cat := range deprecatedCategories {
		var skill model.Skill
		if err := db.Where("category = ?", cat).First(&skill).Error; err == nil {
			// Remove agent-skill associations first
			db.Where("skill_id = ?", skill.ID).Delete(&model.AgentSkill{})
			// Delete the skill (soft delete)
			db.Delete(&skill)
			logger.Log.Infof("Deprecated skill '%s' (category=%s) removed", skill.Name, cat)
		}
	}
}



func deliveryExpertSystemPrompt() string {
	return `你是「交付专家」智能体，专注于 EasyStack 云平台的交付服务。你连接了交付技能知识库、K8S管理技能和OpenStack管理技能。

## 核心能力
1. **实施方案编写**: 基于知识库中的模板和标准，帮助用户编写项目实施方案
2. **交付边界确认**: 根据产品规划文档确认交付范围和边界
3. **兼容性查询**: 根据兼容性列表回答硬件/软件兼容性问题
4. **安装部署指导**: 提供 ECF V6.2.1 安装部署步骤和注意事项
5. **新功能特性**: 解答 EHV 计算虚拟化和镜像服务的新功能特性

## 铁律规则（必须严格遵守）
1. 所有指标、标签、数值必须来自技能知识库的具体结果，禁止编造数据
2. 如果没有技能数据，不要推断根因或编造趋势
3. 阈值必须来自具体技能数据，禁止自定义阈值
4. 回答必须引用具体的技能和绑定的环境信息
5. 如果数据为空，直接回复"无有效数据，无法判断"
6. 每个回答需要给出1-10的置信度评分，低于7分需标注"[低置信度警告]"
7. 遵守 token 限制，失败时最多重试5次

## 回答格式
- 引用来源: [1] [2] 标注引用的文档片段
- 置信度: [置信度: X/10]
- 技能来源: 标注数据来自哪个技能
- 低置信度时: 添加 [低置信度警告] 标签`
}

func seedOpsExpertAgent(db *gorm.DB) {
	var existing model.Agent
	if err := db.Where("name = ?", "运维专家").First(&existing).Error; err == nil {
		return // already exists
	}

	agent := model.Agent{
		Name:        "运维专家",
		Description: "运维专家智能体 - 融合 Kubernetes、OpenStack 和 SRE 站点可靠性工程三大技能，提供全栈运维能力：集群管理、云平台运维、故障排查、SLO管理、事件响应、容量规划等专业服务。",
		SystemPrompt: opsExpertSystemPrompt(),
		Model:       "",
		Temperature: 0.3,
		MaxTokens:   8192,
		IsActive:    true,
		IsPublished: false,
		IronRules:   false,
		CreatedBy:   1,
	}
	if err := db.Create(&agent).Error; err != nil {
		logger.Log.Warnf("Failed to seed 运维专家 agent: %v", err)
		return
	}

	// Link to community skills: k8s-operator, openstack-operator, sre-operator
	categories := []string{"k8s-operator", "openstack-operator", "sre-operator"}
	for _, cat := range categories {
		var sk model.Skill
		if err := db.Where("category = ?", cat).First(&sk).Error; err == nil {
			db.Create(&model.AgentSkill{AgentID: agent.ID, SkillID: sk.ID})
		}
	}
	logger.Log.Info("运维专家 agent seeded with k8s/openstack/sre skills")
}

func seedTotpAgent(db *gorm.DB) {
	var existing model.Agent
	if err := db.Where("name = ?", "双因子申请智能体").First(&existing).Error; err == nil {
		// Agent already exists — ensure it has the totp-query skill linked
		var totpSkill model.Skill
		if err := db.Where("category = ?", "totp-query").First(&totpSkill).Error; err == nil {
			var link model.AgentSkill
			if err := db.Where("agent_id = ? AND skill_id = ?", existing.ID, totpSkill.ID).First(&link).Error; err != nil {
				db.Create(&model.AgentSkill{AgentID: existing.ID, SkillID: totpSkill.ID})
				logger.Log.Info("双因子申请智能体: linked totp-query skill")
			}
		}
		return
	}

	agent := model.Agent{
		Name:        "双因子申请智能体",
		Description: "输入 Case 号（如 ECSL2-50579），自动查询并返回 Roller 双因子验证码和动态密码。",
		SystemPrompt: `你是「双因子申请」智能体，帮助运维人员快速获取双因子认证密码。

用户只需输入一个 Case 号（如 ECSL2-50579、ECSDESK-1234），系统会自动：
1. 从 Jira 查询该 Case 对应的客户和项目
2. 生成 Roller 双因子验证码（OTP，30秒有效期）
3. 生成动态密码

请直接输入 Case 号开始查询。`,
		Model:       "",
		Temperature: 0.2,
		MaxTokens:   4096,
		IsActive:    true,
		IsPublished: true,
		IronRules:   false,
		CreatedBy:   1,
	}
	if err := db.Create(&agent).Error; err != nil {
		logger.Log.Warnf("Failed to seed 双因子申请智能体: %v", err)
		return
	}

	// Link to totp-query skill
	var totpSkill model.Skill
	if err := db.Where("category = ?", "totp-query").First(&totpSkill).Error; err == nil {
		db.Create(&model.AgentSkill{AgentID: agent.ID, SkillID: totpSkill.ID})
	}
	logger.Log.Info("双因子申请智能体 seeded with totp-query skill")
}

func seedOpsEnvWarrantyAgent(db *gorm.DB) {
	var existing model.Agent
	if err := db.Where("name = ?", "过保维保查询智能体").First(&existing).Error; err == nil {
		// Agent already exists — ensure it has the ops-env-warranty skill linked
		var opsSkill model.Skill
		if err := db.Where("category = ?", "ops-env-warranty").First(&opsSkill).Error; err == nil {
			var link model.AgentSkill
			if err := db.Where("agent_id = ? AND skill_id = ?", existing.ID, opsSkill.ID).First(&link).Error; err != nil {
				db.Create(&model.AgentSkill{AgentID: existing.ID, SkillID: opsSkill.ID})
				logger.Log.Info("过保维保查询智能体: linked ops-env-warranty skill")
			}
		}
		return
	}

	agent := model.Agent{
		Name:        "过保维保查询智能体",
		Description: "输入客户名称、项目名称或CSE编号，自动查询并返回运维环境维保信息（节点数、维保到期、SLA等）。",
		SystemPrompt: `你是「过保维保查询」智能体，帮助运维人员快速查询客户环境的维保状态。

用户输入客户名称、项目名称或 CSE 编号，系统会自动查询运维环境数据库并返回维保信息表格，包括：
- CSE 编号、客户名、项目名
- 节点数量
- 维保开始/结束时间（过期告警、即将到期提醒）
- 当前状态、SLA 等级、运维区域

请直接输入要查询的关键字开始。`,
		Model:       "",
		Temperature: 0.2,
		MaxTokens:   4096,
		IsActive:    true,
		IsPublished: true,
		IronRules:   false,
		CreatedBy:   1,
	}
	if err := db.Create(&agent).Error; err != nil {
		logger.Log.Warnf("Failed to seed 过保维保查询智能体: %v", err)
		return
	}

	// Link to ops-env-warranty skill
	var opsSkill model.Skill
	if err := db.Where("category = ?", "ops-env-warranty").First(&opsSkill).Error; err == nil {
		db.Create(&model.AgentSkill{AgentID: agent.ID, SkillID: opsSkill.ID})
	}
	logger.Log.Info("过保维保查询智能体 seeded with ops-env-warranty skill")
}

func opsExpertSystemPrompt() string {
	return `你是「运维专家」智能体，一个全栈运维领域的资深专家。你融合了 Kubernetes 集群管理、OpenStack 云平台运维和 SRE 站点可靠性工程三大核心技能。

## 核心能力

### Kubernetes 运维
- 集群部署、升级、扩缩容、高可用配置
- 工作负载管理: Deployment、StatefulSet、DaemonSet、Job/CronJob
- 网络管理: Service、Ingress、NetworkPolicy 配置和故障排查
- 存储管理: PV、PVC、StorageClass 配置和扩容
- 安全管理: RBAC、ServiceAccount、Secret、SecurityContext
- 监控: Prometheus/Grafana 配置和告警规则
- 故障排查: Pod 异常诊断（CrashLoopBackOff、OOMKilled、网络不通等）

### OpenStack 云平台运维
- 计算服务 (Nova): 虚拟机管理、热迁移/冷迁移、调度策略
- 网络服务 (Neutron): 虚拟网络、路由、浮动IP、安全组
- 存储服务 (Cinder/Swift): 块存储、对象存储、快照管理
- 认证服务 (Keystone): 用户管理、LDAP对接
- EasyStack 特有: ECS平台部署、ECF兼容性、EHV虚拟化

### SRE 站点可靠性工程
- SLO/SLI 管理: 定义和监控服务级别目标、错误预算策略
- 事件管理: 事件响应流程、事后复盘(Postmortem)、根因分析
- 容量规划: 资源利用率分析、容量预测、扩缩容策略
- 变更管理: 发布策略(金丝雀/蓝绿/滚动)、风险评估、回滚方案
- 自动化运维: Toil 消除、IaC、ChatOps
- 监控告警: 监控体系设计、告警规则优化、可观测性建设
- 混沌工程: 故障注入、弹性测试、GameDay 演练

## 工具链熟练
- 容器编排: Kubernetes、Docker、Helm
- 云平台: OpenStack、EasyStack ECS
- 监控: Prometheus、Grafana、AlertManager、Zabbix
- 日志: ELK/EFK、Loki
- 追踪: Jaeger、OpenTelemetry
- IaC: Terraform、Ansible、Pulumi
- CI/CD: Jenkins、GitLab CI、ArgoCD

## 操作规范
- 所有操作前确认当前环境状态
- 变更操作提供回滚方案
- 生产环境操作遵循审批流程
- 提供具体命令和配置示例
- 记录操作日志用于审计`
}

// seedSystemSettings seeds default system settings (idempotent)
func seedSystemSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Category: "jira", Key: "jira_server", Value: "https://easystack.atlassian.net", Label: "Jira 服务器地址", ValueType: "text", SortOrder: 1},
		{Category: "jira", Key: "jira_username", Value: "esoncall@easystack.cn", Label: "Jira 用户名", ValueType: "text", SortOrder: 2},
		{Category: "jira", Key: "jira_password", Value: "", Label: "Jira API Token", ValueType: "password", SortOrder: 3},
		{Category: "totp", Key: "totp_server", Value: "http://lic.easystack.cn", Label: "TOTP 服务器地址", ValueType: "text", SortOrder: 1},
		{Category: "totp", Key: "totp_auth_user", Value: "totp", Label: "TOTP 认证用户", ValueType: "text", SortOrder: 2},
		{Category: "totp", Key: "totp_auth_pass", Value: "Totp@2013", Label: "TOTP 认证密码", ValueType: "password", SortOrder: 3},
		{Category: "totp", Key: "roller_secret", Value: "6GAF6NXNYCT75FV3APTJU4R5XZJ7X6I4", Label: "Roller OTP Secret", ValueType: "password", SortOrder: 4},
		{Category: "totp", Key: "auto_approve", Value: "false", Label: "自动审批(跳过审核)", ValueType: "boolean", SortOrder: 5},
		{Category: "wechat", Key: "corp_id", Value: "", Label: "企业微信 Corp ID", ValueType: "text", SortOrder: 1},
		{Category: "wechat", Key: "secret", Value: "", Label: "企业微信 Secret", ValueType: "password", SortOrder: 2},
		{Category: "wechat", Key: "app_id", Value: "", Label: "应用 ID", ValueType: "text", SortOrder: 3},
	}

	for _, s := range defaults {
		var existing model.SystemSetting
		if err := db.Where("category = ? AND `key` = ?", s.Category, s.Key).First(&existing).Error; err != nil {
			db.Create(&s)
		} else if existing.Value == "" && s.Value != "" {
			// Update empty settings with new defaults (e.g., Jira credentials)
			db.Model(&existing).Update("value", s.Value)
		}
	}
	logger.Log.Info("Default system settings seeded")
}

