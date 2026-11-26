package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ddevops/internal/auth"
	"ddevops/internal/config"
	"ddevops/internal/handler"
	"ddevops/internal/repository"
	"ddevops/internal/router"
	"ddevops/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := repository.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer func(db *repository.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	// 结构/数据迁移统一入口
	if err := db.Migrate(); err != nil {
		log.Printf("数据迁移存在异常（服务继续启动）: %v", err)
	}

	jobRepo := repository.NewJobRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	serverRepo := repository.NewServerRepository(db)
	envRepo := repository.NewEnvironmentRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	dingtalkCacheRepo := repository.NewDingTalkCacheRepository(db)

	if err := sessionRepo.DeleteExpired(); err != nil {
		log.Printf("清理过期会话失败: %v", err)
	}

	if err := serverRepo.SeedDefaultServer(cfg.Jenkins.BaseURL, cfg.Jenkins.User, cfg.Jenkins.Token); err != nil {
		log.Printf("初始化默认服务器失败: %v", err)
	}
	if err := serverRepo.SeedServer("jenkins2", "Jenkins2", cfg.Jenkins2.BaseURL, cfg.Jenkins2.User, cfg.Jenkins2.Token); err != nil {
		log.Printf("初始化 Jenkins2 服务器失败: %v", err)
	}

	// 初始管理员密码以 bcrypt 摘要入库。
	adminPassword, err := auth.HashPassword(cfg.Admin.Password)
	if err != nil {
		log.Fatalf("管理员密码初始化失败: %v", err)
	}
	if err := permRepo.SeedAdmin(cfg.Admin.Username, adminPassword); err != nil {
		log.Printf("初始化管理员失败: %v", err)
	}

	jenkinsMgr := service.NewJenkinsManager(serverRepo, cfg.Jenkins)
	authSvc := auth.NewAuthService(permRepo, sessionRepo)
	dingtalkSvc := service.NewDingTalkService(cfg.DingTalk.ClientID, cfg.DingTalk.ClientSecret, dingtalkCacheRepo)

	buildSvc := service.NewBuildService(jobRepo, envRepo, historyRepo, jenkinsMgr)
	scheduler := service.NewScheduler(taskRepo, buildSvc)

	dingtalkHandler := handler.NewDingTalkHandler(
		handler.DingTalkConfig{
			ClientID:     cfg.DingTalk.ClientID,
			ClientSecret: cfg.DingTalk.ClientSecret,
		},
		handler.DingTalkDeps{
			Build:        buildSvc,
			Permissions:  permRepo,
			Jobs:         jobRepo,
			Tasks:        taskRepo,
			Environments: envRepo,
		},
	)

	apiHandler := handler.NewAPIHandler(handler.Deps{
		Jobs:         jobRepo,
		Tasks:        taskRepo,
		History:      historyRepo,
		Permissions:  permRepo,
		Servers:      serverRepo,
		Environments: envRepo,
		Jenkins:      jenkinsMgr,
		Build:        buildSvc,
		Auth:         authSvc,
		DingTalk:     dingtalkSvc,
	})

	engine := router.SetupRouter(apiHandler, authSvc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scheduler.Start(ctx)

	go func() {
		addr := ":" + cfg.WebPort
		log.Printf("Web服务启动在 http://localhost%s", addr)
		if err := engine.Run(addr); err != nil {
			log.Printf("Web服务启动失败: %v", err)
		}
	}()

	go func() {
		if err := router.StartHealthCheck(":" + cfg.HealthPort); err != nil {
			log.Printf("健康检查服务启动失败: %v", err)
		}
	}()

	go func() {
		if err := dingtalkHandler.Start(ctx); err != nil {
			log.Printf("钉钉机器人服务异常: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("服务正在关闭...")
	time.Sleep(2 * time.Second)
}
