package service

import (
	"context"
	"errors"
	"log"
	"time"

	"ddevops/internal/model"
	"ddevops/internal/repository"
)

type Scheduler struct {
	taskRepo *repository.TaskRepository
	buildSvc *BuildService
	ticker   *time.Ticker
}

func NewScheduler(taskRepo *repository.TaskRepository, buildSvc *BuildService) *Scheduler {
	return &Scheduler{
		taskRepo: taskRepo,
		buildSvc: buildSvc,
		ticker:   time.NewTicker(time.Minute),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	defer s.ticker.Stop()

	log.Println("⏰ 定时任务调度器已启动")
	s.checkTasks()

	for {
		select {
		case <-s.ticker.C:
			s.checkTasks()
		case <-ctx.Done():
			log.Println("⏰ 定时任务调度器停止")
			return
		}
	}
}

func (s *Scheduler) checkTasks() {
	tasks, err := s.taskRepo.GetPending()
	if err != nil {
		log.Printf("获取待执行任务失败: %v", err)
		return
	}

	now := time.Now()
	for _, task := range tasks {
		// GetPending 同时取回 executing，这里只处理真正到点的 pending，避免重复触发。
		if task.Status == "pending" && !task.ScheduleTime.After(now) {
			go s.executeTask(task)
		}
	}
}

// executeTask 执行单条定时任务；构建编排与历史记录统一由 BuildService 负责。
func (s *Scheduler) executeTask(task model.ScheduledTask) {
	log.Printf("执行定时任务: %s", task.TaskID)

	if err := s.taskRepo.UpdateStatus(task.ID, "executing", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
	}

	_, err := s.buildSvc.Trigger(task.TaskID, task.RefValue, "scheduled", task.CreatedBy, task.CreatedName)

	var status, errMsg string
	switch {
	case errors.Is(err, ErrJobNotFound):
		status, errMsg = "failed", "任务配置不存在"
	case errors.Is(err, ErrEnvMissing):
		status, errMsg = "failed", "任务关联的环境配置不存在"
	case err != nil:
		status, errMsg = "failed", err.Error()
	default:
		status = "completed"
	}

	if err := s.taskRepo.UpdateStatus(task.ID, status, errMsg); err != nil {
		log.Printf("更新任务状态失败: %v", err)
	}
	if status == "failed" {
		log.Printf("定时任务执行失败: %s, 原因: %s", task.TaskID, errMsg)
	} else {
		log.Printf("定时任务执行成功: %s", task.TaskID)
	}
}
