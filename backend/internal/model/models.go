package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents a platform user
type User struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Username    string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password    string         `gorm:"size:256;not null" json:"-"`
	Email       string         `gorm:"size:128" json:"email"`
	DisplayName string         `gorm:"size:128" json:"display_name"`
	Role        string         `gorm:"size:32;default:user" json:"role"` // admin, user
	AuthType    string         `gorm:"size:32;default:local" json:"auth_type"` // local, ldap
	Avatar      string         `gorm:"size:512" json:"avatar"`
}

// LDAPConfig stores LDAP configuration managed by admin
type LDAPConfig struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	Host         string         `gorm:"size:256;not null" json:"host"`
	Port         int            `gorm:"default:389" json:"port"`
	UseTLS       bool           `gorm:"default:false" json:"use_tls"`
	BindDN       string         `gorm:"size:512" json:"bind_dn"`
	BindPassword string         `gorm:"size:256" json:"-"`
	BaseDN       string         `gorm:"size:512;not null" json:"base_dn"`
	UserFilter   string         `gorm:"size:512" json:"user_filter"`
	GroupFilter  string         `gorm:"size:512" json:"group_filter"`
	AttrUsername string         `gorm:"size:64;default:uid" json:"attr_username"`
	AttrEmail    string         `gorm:"size:64;default:mail" json:"attr_email"`
	AttrDisplay  string         `gorm:"size:64;default:cn" json:"attr_display"`
	IsEnabled    bool           `gorm:"default:true" json:"is_enabled"`
	IsDefault    bool           `gorm:"default:false" json:"is_default"`
}

// Agent represents an AI agent configuration
type Agent struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	SystemPrompt string         `gorm:"type:text" json:"system_prompt"`
	Model        string         `gorm:"size:64" json:"model"`
	Temperature  float64        `gorm:"default:0.7" json:"temperature"`
	MaxTokens    int            `gorm:"default:4096" json:"max_tokens"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedBy    uint           `json:"created_by"`
	AgentSkills  []AgentSkill   `gorm:"foreignKey:AgentID" json:"agent_skills,omitempty"`
}

// Skill represents a capability/tool the agent can use
type Skill struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Type        string         `gorm:"size:32" json:"type"`     // delivery, ops, knowledge
	Config      string         `gorm:"type:text" json:"config"` // JSON config
	ToolDefs    string         `gorm:"type:text" json:"tool_defs"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
}

// AgentSkill is the many-to-many join table linking agents to skills
type AgentSkill struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	AgentID   uint      `gorm:"uniqueIndex:idx_agent_skill;not null" json:"agent_id"`
	SkillID   uint      `gorm:"uniqueIndex:idx_agent_skill;not null" json:"skill_id"`
	Skill     Skill     `gorm:"foreignKey:SkillID" json:"skill,omitempty"`
}

// Conversation represents a chat session
type Conversation struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Title     string         `gorm:"size:256" json:"title"`
	UserID    uint           `gorm:"index" json:"user_id"`
	AgentID   uint           `gorm:"index" json:"agent_id"`
	Agent     Agent          `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	ConversationID uint           `gorm:"index;not null" json:"conversation_id"`
	Role           string         `gorm:"size:16;not null" json:"role"` // user, assistant, system
	Content        string         `gorm:"type:longtext;not null" json:"content"`
	TokensUsed     int            `json:"tokens_used"`
	ToolCalls      string         `gorm:"type:text" json:"tool_calls,omitempty"`
}

// WebsiteCategory stores website link categories
type WebsiteCategory struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Icon      string         `gorm:"size:64" json:"icon"`
	SortOrder int            `gorm:"default:0" json:"sort_order"`
	Links     []WebsiteLink  `gorm:"foreignKey:CategoryID" json:"links,omitempty"`
}

// WebsiteLink stores individual website links
type WebsiteLink struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	CategoryID uint           `gorm:"index;not null" json:"category_id"`
	Name       string         `gorm:"size:256;not null" json:"name"`
	URL        string         `gorm:"size:1024;not null" json:"url"`
	Icon       string         `gorm:"size:64" json:"icon"`
	SortOrder  int            `gorm:"default:0" json:"sort_order"`
}

// AIProvider stores AI provider configurations
type AIProvider struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Label       string         `gorm:"size:128;not null" json:"label"`
	APIKey      string         `gorm:"size:512" json:"api_key"`
	BaseURL     string         `gorm:"size:512" json:"base_url"`
	Model       string         `gorm:"size:128" json:"model"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	IsEnabled   bool           `gorm:"default:true" json:"is_enabled"`
	Description string         `gorm:"size:256" json:"description"`
	IconURL     string         `gorm:"size:512" json:"icon_url"`
}

// TaskLog records async tasks processed via RabbitMQ
type TaskLog struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	TaskID    string         `gorm:"size:64;uniqueIndex" json:"task_id"`
	Type      string         `gorm:"size:64" json:"type"`
	Status    string         `gorm:"size:32" json:"status"`
	Input     string         `gorm:"type:text" json:"input"`
	Output    string         `gorm:"type:text" json:"output"`
	Error     string         `gorm:"type:text" json:"error"`
	UserID    uint           `json:"user_id"`
}

// OperationLog records admin operations
type OperationLog struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	UserID     uint           `gorm:"index" json:"user_id"`
	Username   string         `gorm:"size:64" json:"username"`
	Module     string         `gorm:"size:32;index" json:"module"`
	Action     string         `gorm:"size:32" json:"action"`
	TargetID   uint           `json:"target_id"`
	TargetName string         `gorm:"size:128" json:"target_name"`
	Detail     string         `gorm:"type:text" json:"detail"`
	IP         string         `gorm:"size:64" json:"ip"`
}
