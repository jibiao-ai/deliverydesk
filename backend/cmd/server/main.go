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
	h := handler.NewHandler(chatService)

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

			// Skills
			auth.GET("/skills", h.ListSkills)
			auth.GET("/agents/:id/skills", h.GetAgentSkills)

			// AI Providers
			auth.GET("/ai-providers", h.GetAIProviders)
			auth.POST("/ai-providers", h.CreateAIProvider)
			auth.PUT("/ai-providers/:id", h.UpdateAIProvider)
			auth.DELETE("/ai-providers/:id", h.DeleteAIProvider)
			auth.POST("/ai-providers/:id/test", h.TestAIProvider)

			// Website Links
			auth.GET("/website-categories", h.GetWebsiteCategories)

			// Admin routes
			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.GET("/users", h.ListUsers)
				admin.POST("/users", h.CreateUser)
				admin.PUT("/users/:id", h.UpdateUser)
				admin.DELETE("/users/:id", h.DeleteUser)

				// LDAP Configuration
				admin.GET("/ldap-configs", h.ListLDAPConfigs)
				admin.POST("/ldap-configs", h.CreateLDAPConfig)
				admin.PUT("/ldap-configs/:id", h.UpdateLDAPConfig)
				admin.DELETE("/ldap-configs/:id", h.DeleteLDAPConfig)
				admin.POST("/ldap-configs/:id/test", h.TestLDAPConfig)

				admin.GET("/operation-logs", h.ListOperationLogs)
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
