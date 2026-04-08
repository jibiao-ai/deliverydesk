package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	AI       AIConfig
	LDAP     LDAPConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

type AIConfig struct {
	Provider string // openai, azure, local
	APIKey   string
	BaseURL  string
	Model    string
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

type LDAPConfig struct {
	Enabled    bool
	Host       string
	Port       int
	UseTLS     bool
	BindDN     string
	BindPassword string
	BaseDN     string
	UserFilter string
	GroupFilter string
	Attributes LDAPAttributes
}

type LDAPAttributes struct {
	Username    string
	Email       string
	DisplayName string
	MemberOf    string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "mysql"),
			Port:     getEnvInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "deliverydesk"),
			Password: getEnv("DB_PASSWORD", "deliverydesk123"),
			DBName:   getEnv("DB_NAME", "deliverydesk"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "rabbitmq"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "/"),
		},
		AI: AIConfig{
			Provider: getEnv("AI_PROVIDER", "openai"),
			APIKey:   getEnv("AI_API_KEY", ""),
			BaseURL:  getEnv("AI_BASE_URL", "https://api.openai.com/v1"),
			Model:    getEnv("AI_MODEL", "gpt-4"),
		},
		LDAP: LDAPConfig{
			Enabled:      getEnvBool("LDAP_ENABLED", false),
			Host:         getEnv("LDAP_HOST", "ldap.example.com"),
			Port:         getEnvInt("LDAP_PORT", 389),
			UseTLS:       getEnvBool("LDAP_USE_TLS", false),
			BindDN:       getEnv("LDAP_BIND_DN", "cn=admin,dc=example,dc=com"),
			BindPassword: getEnv("LDAP_BIND_PASSWORD", ""),
			BaseDN:       getEnv("LDAP_BASE_DN", "dc=example,dc=com"),
			UserFilter:   getEnv("LDAP_USER_FILTER", "(uid=%s)"),
			GroupFilter:  getEnv("LDAP_GROUP_FILTER", "(memberUid=%s)"),
			Attributes: LDAPAttributes{
				Username:    getEnv("LDAP_ATTR_USERNAME", "uid"),
				Email:       getEnv("LDAP_ATTR_EMAIL", "mail"),
				DisplayName: getEnv("LDAP_ATTR_DISPLAY_NAME", "cn"),
				MemberOf:    getEnv("LDAP_ATTR_MEMBER_OF", "memberOf"),
			},
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
