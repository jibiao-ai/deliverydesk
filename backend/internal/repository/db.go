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
		"AgentSkill":      &model.AgentSkill{},
		"Conversation":    &model.Conversation{},
		"Message":         &model.Message{},
		"TaskLog":         &model.TaskLog{},
		"WebsiteCategory": &model.WebsiteCategory{},
		"WebsiteLink":     &model.WebsiteLink{},
		"AIProvider":      &model.AIProvider{},
		"OperationLog":    &model.OperationLog{},
	}
	migrationOrder := []string{
		"User", "LDAPConfig", "Agent", "Skill", "AgentSkill",
		"Conversation", "Message", "TaskLog",
		"WebsiteCategory", "WebsiteLink", "AIProvider", "OperationLog",
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
		{Name: "deepseek", Label: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat", IsDefault: false, IsEnabled: true, Description: "深度求索，高性价比国产大模型", IconURL: "deepseek"},
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

	// Seed default agents
	var agentCount int64
	db.Model(&model.Agent{}).Count(&agentCount)
	if agentCount == 0 {
		agents := []model.Agent{
			{
				Name:        "交付黄页智能体",
				Description: "汇聚交付团队常用知识库、文档链接和操作指南的智能体，快速导航到需要的资源",
				SystemPrompt: "你是交付黄页智能助手，帮助交付工程师快速找到所需的文档、工具和资源链接。你熟悉所有常用的交付工具和系统，能够根据用户的需求推荐最合适的资源。",
				Model:       "gpt-4",
				Temperature: 0.3,
				MaxTokens:   4096,
				IsActive:    true,
				CreatedBy:   1,
			},
			{
				Name:        "交付技能智能体",
				Description: "提供云平台交付技能指导，包括安装部署、运维手册、升级操作等专业技能支持",
				SystemPrompt: "你是交付技能专家智能体，专注于 EasyStack 云平台的交付技能指导。你可以帮助工程师解决安装部署、运维管理、升级操作等方面的问题。请基于最佳实践给出详细的操作步骤和建议。",
				Model:       "gpt-4",
				Temperature: 0.2,
				MaxTokens:   8192,
				IsActive:    true,
				CreatedBy:   1,
			},
		}
		for _, a := range agents {
			db.Create(&a)
		}
		logger.Log.Info("Default agents created")
	}

	// Seed default skills
	var skillCount int64
	db.Model(&model.Skill{}).Count(&skillCount)
	if skillCount == 0 {
		skills := []model.Skill{
			{Name: "文档检索", Description: "从知识库中检索交付相关文档和操作手册", Type: "knowledge", IsActive: true},
			{Name: "安装部署指导", Description: "V6.x版本安装部署流程和注意事项", Type: "delivery", IsActive: true},
			{Name: "升级迁移", Description: "云平台版本升级和数据迁移操作指南", Type: "delivery", IsActive: true},
			{Name: "运维巡检", Description: "日常运维巡检和故障排查技能", Type: "ops", IsActive: true},
			{Name: "网络配置", Description: "SDN网络、存储网络等配置包制作技能", Type: "delivery", IsActive: true},
		}
		for _, s := range skills {
			db.Create(&s)
		}
		logger.Log.Info("Default skills created")
	}

	// Seed website links from Excel data
	var catCount int64
	db.Model(&model.WebsiteCategory{}).Count(&catCount)
	if catCount == 0 {
		seedWebsiteLinks(db)
	}

	// Fix typo: 合适费控报销 -> 合思费控报销
	db.Model(&model.WebsiteLink{}).Where("name = ?", "合适费控报销").Update("name", "合思费控报销")
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
