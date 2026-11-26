package config

import (
	"ddevops/internal/service"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath     string
	Jenkins    service.JenkinsConfig
	Jenkins2   service.JenkinsConfig
	DingTalk   DingTalkConfig
	Admin      AdminConfig
	WebPort    string
	HealthPort string
}

type AdminConfig struct {
	Username string
	Password string
}

type DingTalkConfig struct {
	ClientID     string
	ClientSecret string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用环境变量")
	}

	return &Config{
		DBPath: getEnv("SQLITE_DB_PATH", "jenkins.db"),
		Jenkins: service.JenkinsConfig{
			BaseURL: getEnv("JENKINS_BASE_URL", ""),
			User:    getEnv("JENKINS_USER", ""),
			Token:   getEnv("JENKINS_TOKEN", ""),
		},
		Jenkins2: service.JenkinsConfig{
			BaseURL: getEnv("JENKINS2_BASE_URL", ""),
			User:    getEnv("JENKINS2_USER", ""),
			Token:   getEnv("JENKINS2_TOKEN", ""),
		},
		DingTalk: DingTalkConfig{
			ClientID:     getEnv("DINGTALK_CLIENT_ID", ""),
			ClientSecret: getEnv("DINGTALK_CLIENT_SECRET", ""),
		},
		Admin: AdminConfig{
			Username: getEnv("ADMIN_USERNAME", "admin"),
			Password: getEnv("ADMIN_PASSWORD", "admin123"),
		},
		WebPort:    getEnv("WEB_PORT", "8081"),
		HealthPort: getEnv("HEALTH_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
