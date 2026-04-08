package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/internal/model"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

		for i := 0; i < 30; i++ {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
			if err == nil {
				break
			}
			logger.Log.Warnf("Failed to connect to database (attempt %d/30): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to database after retries: %w", err)
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

	// Auto migrate
	err = db.AutoMigrate(
		&model.User{},
		&model.LDAPConfig{},
		&model.Agent{},
		&model.Skill{},
		&model.AgentSkill{},
		&model.Conversation{},
		&model.Message{},
		&model.TaskLog{},
		&model.WebsiteCategory{},
		&model.WebsiteLink{},
		&model.AIProvider{},
		&model.OperationLog{},
	)
	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	DB = db
	logger.Log.Info("Database connection established and migrated")

	// Seed default data
	seedDefaultData(db)

	return nil
}

func seedDefaultData(db *gorm.DB) {
	// Ensure default admin user
	const adminPlainPassword = "Admin@2024!"
	adminHashBytes, _ := bcrypt.GenerateFromPassword([]byte(adminPlainPassword), 10)
	adminPasswordHash := string(adminHashBytes)

	var admin model.User
	result := db.Where("username = ?", "admin").First(&admin)
	if result.Error != nil {
		admin = model.User{
			Username:    "admin",
			Password:    adminPasswordHash,
			Email:       "admin@deliverydesk.local",
			DisplayName: "管理员",
			Role:        "admin",
			AuthType:    "local",
		}
		db.Create(&admin)
		logger.Log.Info("Default admin user created")
	}

	// Seed default AI providers
	defaultProviders := []model.AIProvider{
		{Name: "openai", Label: "OpenAI", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o", IsDefault: true, IsEnabled: true, Description: "OpenAI GPT 系列模型"},
		{Name: "deepseek", Label: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat", IsDefault: false, IsEnabled: true, Description: "深度求索 DeepSeek 系列模型"},
		{Name: "qwen", Label: "通义千问", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Model: "qwen-plus", IsDefault: false, IsEnabled: true, Description: "阿里云通义千问系列模型"},
		{Name: "glm", Label: "智谱 GLM", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Model: "glm-4", IsDefault: false, IsEnabled: true, Description: "智谱 AI GLM 系列模型"},
		{Name: "siliconflow", Label: "硅基流动 SiliconFlow", BaseURL: "https://api.siliconflow.cn/v1", Model: "Qwen/Qwen2.5-7B-Instruct", IsDefault: false, IsEnabled: true, Description: "硅基流动，支持多种开源模型"},
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
}

func seedWebsiteLinks(db *gorm.DB) {
	type linkData struct {
		Name string
		URL  string
	}
	type categoryData struct {
		Name  string
		Icon  string
		Links []linkData
	}

	categories := []categoryData{
		{
			Name: "常用工具&系统",
			Icon: "Wrench",
			Links: []linkData{
				{Name: "商业存储对接包", URL: "http://octopus.easystack.io/"},
				{Name: "拓扑制作系统", URL: "http://lic.easystack.cn/topoweb/login"},
				{Name: "Redmine系统", URL: "https://redmine.easystack.cn/login"},
				{Name: "Jira系统", URL: "https://easystack.atlassian.net/jira/for-you"},
				{Name: "Confluence系统", URL: "https://easystack.atlassian.net/wiki"},
				{Name: "VPN工具", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1099467131/01-VPN"},
				{Name: "企业邮箱", URL: "https://mail.qiye.163.com/static/login/"},
				{Name: "企业网盘", URL: "https://pan.easystack.io/"},
			},
		},
		{
			Name: "网络配置包制作",
			Icon: "Network",
			Links: []linkData{
				{Name: "neutron-az、多生产(业务)网配包制作", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1659437057/09-"},
				{Name: "云杉网络对接包", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1764688072/10-"},
			},
		},
		{
			Name: "常用网站",
			Icon: "Globe",
			Links: []linkData{
				{Name: "Confluence系统使用介绍", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1099499945/04-Confluence"},
				{Name: "交付黄页", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1102644405/05-"},
				{Name: "合适费控报销", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1659373029/07-"},
			},
		},
		{
			Name: "云产品资料",
			Icon: "Cloud",
			Links: []linkData{
				{Name: "V6.0.1/V6.1.1标准安装介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1695711233"},
				{Name: "V6.0.1/V6.0.2/V6.1.1产品升级介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1128104392"},
				{Name: "V6.2.1标准安装介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2205548547/V621"},
				{Name: "V6.2.1产品升级介质", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2226094087/V621"},
			},
		},
		{
			Name: "安装部署手册/勘误",
			Icon: "BookOpen",
			Links: []linkData{
				{Name: "V6.2.1安装部署手册", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/2114355900/V621"},
				{Name: "V6.0.1/V6.1.1安装部署手册", URL: "https://easystack.atlassian.net/wiki/spaces/PM/pages/1042744182"},
				{Name: "实施阶段文档模版", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1847656703/00-"},
				{Name: "云平台勘误", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/folder/2801336339"},
				{Name: "万博迁移软件相关", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1908670626/00-"},
			},
		},
		{
			Name: "运维",
			Icon: "Settings",
			Links: []linkData{
				{Name: "驻场运维规范", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/2486763531/09-"},
				{Name: "技能培训", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/2486927470/03-"},
				{Name: "SDN相关服务问题", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1682866530/SDN"},
				{Name: "V6.2.1标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/2245297715/ECS+V621"},
				{Name: "V6.1.1标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1682669599/ECS+V611"},
				{Name: "V6标准变更/勘误", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/1110181031/ECS+V6"},
				{Name: "V6运维手册库", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/873759399/ECS+V6"},
			},
		},
		{
			Name: "专项交付",
			Icon: "Target",
			Links: []linkData{
				{Name: "邮储专项交付", URL: "https://easystack.atlassian.net/wiki/spaces/delivery/pages/1660649520/02-"},
			},
		},
		{
			Name: "镜像相关",
			Icon: "HardDrive",
			Links: []linkData{
				{Name: "标准镜像制作相关", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/85951587"},
				{Name: "标准镜像维护列表", URL: "https://easystack.atlassian.net/wiki/spaces/ESK/pages/85951522/03-"},
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
				SortOrder:  j,
			}
			db.Create(&websiteLink)
		}
	}
	logger.Log.Info("Website links seeded from Excel data")
}
