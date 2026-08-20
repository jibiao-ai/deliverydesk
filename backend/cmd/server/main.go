package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/internal/handler"
	"github.com/jibiao-ai/deliverydesk/internal/middleware"
	"github.com/jibiao-ai/deliverydesk/internal/mq"
	"github.com/jibiao-ai/deliverydesk/internal/repository"
	"github.com/jibiao-ai/deliverydesk/internal/service"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

func main() {
	logger.Init()
	logger.Log.Info("Starting DeliveryDesk - Cloud Delivery Workbench...")

	cfg := config.Load()
	gin.SetMode(cfg.Server.Mode)

	// Initialize database (with extended retry for container orchestration)
	var dbErr error
	for attempt := 1; attempt <= 3; attempt++ {
		dbErr = repository.InitDB(cfg.Database)
		if dbErr == nil {
			break
		}
		logger.Log.Warnf("Database init attempt %d/3 failed: %v", attempt, dbErr)
		if attempt < 3 {
			logger.Log.Info("Waiting 10 seconds before retry...")
			time.Sleep(10 * time.Second)
		}
	}
	if dbErr != nil {
		logger.Log.Fatalf("Failed to initialize database after 3 attempts: %v", dbErr)
	}

	// Initialize RabbitMQ
	rabbitMQ := mq.NewRabbitMQ(cfg.RabbitMQ)
	if err := rabbitMQ.Connect(); err != nil {
		logger.Log.Warnf("Failed to connect to RabbitMQ (will continue without MQ): %v", err)
	} else {
		defer rabbitMQ.Close()
		rabbitMQ.Consume(mq.QueueAgentTask, func(msg mq.TaskMessage) error {
			logger.Log.Infof("Processing task: %s (type: %s)", msg.ID, msg.Type)
			return nil
		})
	}

	// Initialize services
	chatService := service.NewChatService()

	// Warm up skill knowledge store (load indexed documents into memory for RAG)
	logger.Log.Info("Warming up skill knowledge store...")
	chatService.WarmUpSkillStore()

	h := handler.NewHandler(chatService)
	totpH := handler.NewTotpHandler()
	worktimeH := handler.NewWorktimeHandler()
	projectH := handler.NewProjectHandler()
	opsEnvH := handler.NewOpsEnvHandler()
	kbH := handler.NewKBHandler()
	wbsH := &handler.WBSHandler{}

	// Start periodic Jira sync (every 2 hours)
	jiraSvc := service.GetJiraService()
	jiraSvc.StartPeriodicSync(2 * time.Hour)

	// Start monthly auto-fetch for worktime (fetches last month's data on the 1st)
	worktimeSvc := service.GetWorktimeService()
	worktimeSvc.StartMonthlyAutoFetch()

	// Start periodic project sync from Redmine (daily at 2:00 AM)
	projectSvc := service.GetProjectService()
	projectSvc.StartPeriodicSync()

	// Start periodic OpsEnvironment sync from Jira CSE (every 6 hours)
	handler.StartPeriodicOpsEnvSync(6 * time.Hour)

	// Setup Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "not found"})
	})

	// API routes
	api := r.Group("/api")
	{
		api.POST("/login", h.Login)
		api.GET("/health", h.HealthCheck)

		// Published agents external API (no auth required)
		api.GET("/published-agents", h.ListPublishedAgents)
		api.POST("/published-agents/:id/chat", h.PublishedAgentChat)

		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.GET("/profile", h.GetProfile)
			auth.GET("/dashboard", h.GetDashboard)

			// Agents
			auth.GET("/agents", h.ListAgents)
			auth.GET("/agents/:id", h.GetAgent)
			auth.POST("/agents", h.CreateAgent)
			auth.PUT("/agents/:id", h.UpdateAgent)
			auth.DELETE("/agents/:id", h.DeleteAgent)

			// Conversations
			auth.GET("/conversations", h.ListConversations)
			auth.POST("/conversations", h.CreateConversation)
			auth.DELETE("/conversations/:id", h.DeleteConversation)

			// Messages
			auth.GET("/conversations/:id/messages", h.GetMessages)
			auth.POST("/conversations/:id/messages", h.SendMessage)
			auth.POST("/conversations/:id/messages/stream", h.SendMessageStream)
			auth.POST("/conversations/:id/abort", h.AbortStream)

			// Skills
			auth.GET("/skills", h.ListSkills)
			auth.GET("/skills/:id", h.GetSkill)
			auth.POST("/skills", h.CreateSkill)
			auth.PUT("/skills/:id", h.UpdateSkill)
			auth.DELETE("/skills/:id", h.DeleteSkill)
			auth.POST("/skills/:id/upload", h.UploadSkillDocument)
			auth.POST("/skills/:id/upload-multi", h.UploadSkillDocuments)
			auth.POST("/skills/:id/reindex", h.ReindexSkill)
			auth.DELETE("/skills/:id/documents/:docId", h.DeleteSkillDocument)
			auth.GET("/agents/:id/skills", h.GetAgentSkills)

			// AI Providers
			auth.GET("/ai-providers", h.GetAIProviders)
			auth.POST("/ai-providers", h.CreateAIProvider)
			auth.PUT("/ai-providers/:id", h.UpdateAIProvider)
			auth.DELETE("/ai-providers/:id", h.DeleteAIProvider)
			auth.POST("/ai-providers/:id/test", h.TestAIProvider)

			// Website Links
			auth.GET("/website-categories", h.GetWebsiteCategories)

			// TOTP Applications (all authenticated users)
			auth.POST("/totp/apply", totpH.CreateTotpApplication)
			auth.GET("/totp/my-applications", totpH.ListMyApplications)
			auth.GET("/totp/check-issue", totpH.CheckIssue)
			auth.GET("/totp/quick-query", totpH.QuickQueryTotp)
			auth.GET("/totp/jira-cache", totpH.ListJiraCache)
			auth.GET("/totp/admins", totpH.GetAdminList)

			// Worktime Management (all authenticated users)
			auth.GET("/worktime/stats", worktimeH.GetWorktimeStats)
			auth.GET("/worktime/export", worktimeH.ExportWorktime)
			auth.GET("/worktime/users", worktimeH.ListWorktimeUsers)
			auth.POST("/worktime/users", worktimeH.AddWorktimeUser)
			auth.DELETE("/worktime/users/:id", worktimeH.RemoveWorktimeUser)
			auth.POST("/worktime/users/batch", worktimeH.BatchAddWorktimeUsers)

			// Project Management (all authenticated users)
			auth.GET("/projects/stats", projectH.GetProjectStats)
			auth.GET("/projects/list", projectH.GetProjectList)
			auth.GET("/projects/pre-delivery", projectH.GetPreDeliveryList)
			auth.POST("/projects/sync", projectH.SyncProjects)

		// Ops Environment routes
			auth.GET("/ops-env/list", opsEnvH.ListOpsEnvironments)
			auth.GET("/ops-env/stats", opsEnvH.GetOpsEnvStats)
			auth.GET("/ops-env/quick-query", opsEnvH.QuickQueryOpsEnv)
			auth.GET("/ops-env/calendar", opsEnvH.GetOpsEnvCalendar)
		auth.GET("/ops-env/regions", opsEnvH.GetRegions)
		auth.GET("/ops-env/diagnose", opsEnvH.DiagnoseOpsEnv)
		auth.GET("/ops-env/top-customers", opsEnvH.GetOpsEnvTopCustomers)
		auth.GET("/ops-env/top-nodes", opsEnvH.GetOpsEnvTopNodes)
		auth.POST("/ops-env/sync", opsEnvH.SyncOpsEnvironments)

		// Knowledge Base (Jira → Confluence) routes
		auth.POST("/kb/preview", kbH.PreviewKB)
		auth.POST("/kb/publish", kbH.PublishKB)
		auth.GET("/kb/history", kbH.ListKBHistory)

		// WBS Service routes
		auth.GET("/wbs/catalog", wbsH.GetCatalog)
		auth.POST("/wbs/orders", wbsH.SaveOrder)
		auth.GET("/wbs/orders", wbsH.ListOrders)
		auth.GET("/wbs/orders/:id", wbsH.GetOrder)
		auth.GET("/wbs/orders/:id/export", wbsH.ExportExcel)
		auth.DELETE("/wbs/orders/:id", wbsH.DeleteOrder)

			// Admin routes
			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.GET("/users", h.ListUsers)
				admin.GET("/users/stats", h.GetUserStats)
				admin.POST("/users", h.CreateUser)
				admin.PUT("/users/:id", h.UpdateUser)
				admin.DELETE("/users/:id", h.DeleteUser)

				// LDAP Configuration
				admin.GET("/ldap-configs", h.ListLDAPConfigs)
				admin.POST("/ldap-configs", h.CreateLDAPConfig)
				admin.POST("/ldap-configs/sync-users", h.SyncLDAPUsers)
				admin.PUT("/ldap-configs/:id", h.UpdateLDAPConfig)
				admin.DELETE("/ldap-configs/:id", h.DeleteLDAPConfig)
				admin.POST("/ldap-configs/:id/test", h.TestLDAPConfig)
				admin.GET("/ldap-configs/:id/diagnose", h.DiagnoseLDAP)

				admin.GET("/operation-logs", h.ListOperationLogs)

				// TOTP Audit (admin only)
				admin.GET("/totp/pending-reviews", totpH.ListPendingReviews)
				admin.GET("/totp/all", totpH.ListAllApplications)
				admin.POST("/totp/audit", totpH.AuditApplication)
				admin.POST("/totp/sync-jira", totpH.SyncJiraIssues)

				// System Settings (admin only)
				admin.GET("/settings", totpH.GetSettings)
				admin.PUT("/settings", totpH.UpdateSettings)

				// Diagnostic endpoint for skill/RAG debugging
				admin.GET("/diagnose/skills", h.DiagnoseSkills)
				admin.POST("/diagnose/skills/reindex-all", h.ReindexAllSkills)
			}
		}
	}

	port := cfg.Server.Port
	logger.Log.Infof("Server starting on port %s", port)

	go func() {
		if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
			logger.Log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")
}
