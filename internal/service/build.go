package service

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"ddevops/internal/model"
	"ddevops/internal/repository"
)

var (
	ErrJobNotFound = errors.New("任务不存在")
	ErrEnvMissing  = errors.New("任务关联的环境配置不存在")
	// 生产环境必须由人指定版本 Tag，绝不静默回落到环境默认分支
	ErrProdTagRequired = errors.New("当前环境按 Tag 构建，必须填写版本 Tag（例：20261105u01）")
	ErrProdTagFormat   = errors.New("版本 Tag 只能包含字母、数字、点、横线、下划线和斜杠")
)

// 版本 Tag 允许的字符集：字母数字 + . - _ /
var prodTagPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// BuildResult 构建触发结果，携带调用方展示卡片/回执所需的信息。
type BuildResult struct {
	Job      model.Job
	Env      model.Environment
	RefValue string
	BuildURL string
}

// BuildService 是「构建」的唯一入口：Web 手动构建、钉钉指令、定时任务三条链路共用。
// 分支归一、环境参数选择、历史记录的实现都只此一份，避免多处行为漂移。
type BuildService struct {
	jobRepo     *repository.JobRepository
	envRepo     *repository.EnvironmentRepository
	historyRepo *repository.HistoryRepository
	jenkinsMgr  *JenkinsManager
}

func NewBuildService(
	jobRepo *repository.JobRepository,
	envRepo *repository.EnvironmentRepository,
	historyRepo *repository.HistoryRepository,
	jenkinsMgr *JenkinsManager,
) *BuildService {
	return &BuildService{
		jobRepo:     jobRepo,
		envRepo:     envRepo,
		historyRepo: historyRepo,
		jenkinsMgr:  jenkinsMgr,
	}
}

// NormalizeRef 归一化分支/版本，规则全部来自 env.Rule()，不再各自解释 isProd。
func NormalizeRef(ref string, env model.Environment) (string, error) {
	rule := env.Rule()
	ref = strings.TrimSpace(ref)

	if ref == "" {
		if rule.RequireRef {
			return "", ErrProdTagRequired
		}
		return rule.DefaultRef, nil
	}

	if rule.RequireRef {
		if !prodTagPattern.MatchString(ref) {
			return "", ErrProdTagFormat
		}
		return ref, nil
	}
	if !strings.HasPrefix(ref, "origin/") {
		return "origin/" + ref, nil
	}
	return ref, nil
}

// Trigger 触发一次构建并记录发布历史。
// 任务/环境预检失败时返回对应哨兵错误且不写历史；已调用 Jenkins 的成败都会落历史。
func (s *BuildService) Trigger(jobID, refValue, triggerType, by, byName string) (BuildResult, error) {
	job, err := s.jobRepo.Get(jobID)
	if err != nil {
		return BuildResult{}, ErrJobNotFound
	}

	env, err := s.envRepo.Get(job.EnvID)
	if err != nil {
		return BuildResult{Job: job}, ErrEnvMissing
	}

	ref, err := NormalizeRef(refValue, env)
	if err != nil {
		return BuildResult{Job: job, Env: env}, err
	}
	buildURL := s.jenkinsMgr.GetBuildURL(job)
	buildErr := s.jenkinsMgr.TriggerBuild(job, env, ref)

	history := model.PublishHistory{
		TaskID:      job.ID,
		EnvID:       job.EnvID,
		RefValue:    ref,
		TriggerType: triggerType,
		TriggerBy:   by,
		TriggerName: byName,
		TriggerTime: time.Now(),
		BuildURL:    buildURL,
	}
	if buildErr != nil {
		history.BuildStatus = "failed"
		history.ErrorMessage = buildErr.Error()
	} else {
		history.BuildStatus = "success"
	}
	if err := s.historyRepo.Create(history); err != nil {
		log.Printf("记录发布历史失败: %v", err)
	}

	return BuildResult{Job: job, Env: env, RefValue: ref, BuildURL: buildURL}, buildErr
}
