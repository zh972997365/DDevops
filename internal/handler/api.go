package handler

import (
	"database/sql"
	"ddevops/internal/auth"
	"ddevops/internal/model"
	"ddevops/internal/repository"
	"ddevops/internal/service"
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Deps API 层依赖聚合，避免长串位置参数。
type Deps struct {
	Jobs         *repository.JobRepository
	Tasks        *repository.TaskRepository
	History      *repository.HistoryRepository
	Permissions  *repository.PermissionRepository
	Servers      *repository.ServerRepository
	Environments *repository.EnvironmentRepository
	Jenkins      *service.JenkinsManager
	Build        *service.BuildService
	Auth         *auth.AuthService
	DingTalk     *service.DingTalkService
}

type APIHandler struct {
	jobRepo     *repository.JobRepository
	taskRepo    *repository.TaskRepository
	historyRepo *repository.HistoryRepository
	permRepo    *repository.PermissionRepository
	serverRepo  *repository.ServerRepository
	envRepo     *repository.EnvironmentRepository
	jenkinsMgr  *service.JenkinsManager
	buildSvc    *service.BuildService
	authSvc     *auth.AuthService
	dingtalkSvc *service.DingTalkService
}

func NewAPIHandler(d Deps) *APIHandler {
	return &APIHandler{
		jobRepo:     d.Jobs,
		taskRepo:    d.Tasks,
		historyRepo: d.History,
		permRepo:    d.Permissions,
		serverRepo:  d.Servers,
		envRepo:     d.Environments,
		jenkinsMgr:  d.Jenkins,
		buildSvc:    d.Build,
		authSvc:     d.Auth,
		dingtalkSvc: d.DingTalk,
	}
}

// 当前登录用户（由 AuthMiddleware 写入上下文）。
func currentUser(c *gin.Context) (username, name, role string) {
	if v, ok := c.Get("username"); ok {
		username, _ = v.(string)
	}
	if v, ok := c.Get("name"); ok {
		name, _ = v.(string)
	}
	if v, ok := c.Get("role"); ok {
		role, _ = v.(string)
	}
	if name == "" {
		name = username
	}
	return
}

// jobScopeFor 由角色与授权任务列表计算任务维度的可见/可构建范围，
// Web 端与钉钉机器人共用同一套规则，避免出现两套不一致的口径。
// admin 始终全量（返回 restricted=false）；普通用户仅限 allowedJobs，列表为空即无任何授权任务。
func jobScopeFor(role string, allowedJobs []string) (scope map[string]bool, restricted bool) {
	if role == "admin" {
		return nil, false
	}
	if len(allowedJobs) == 0 {
		return map[string]bool{}, true
	}
	m := make(map[string]bool, len(allowedJobs))
	for _, id := range allowedJobs {
		m[id] = true
	}
	return m, true
}

// jobScope 返回当前登录用户在任务维度的可见/可构建范围。
// 找不到授权记录时按「无任何权限」收口（fail-closed），避免越权放行。
func (h *APIHandler) jobScope(c *gin.Context) (allowed map[string]bool, restricted bool) {
	_, _, role := currentUser(c)
	if role == "admin" {
		return nil, false
	}
	username, _, _ := currentUser(c)
	perm, err := h.permRepo.GetByUsername(username)
	if err != nil {
		return map[string]bool{}, true
	}
	return jobScopeFor(perm.WebRole, perm.AllowedJobs)
}

// scopeTaskIDs 是 jobScope 的切片形态，便于传给仓储做 IN 过滤。
// 返回 nil 表示不受限（管理员）；非 nil 空切片表示「无任何授权任务」（受限但无授权）。
func (h *APIHandler) scopeTaskIDs(c *gin.Context) []string {
	scope, restricted := h.jobScope(c)
	if !restricted {
		return nil
	}
	ids := make([]string, 0, len(scope))
	for id := range scope {
		ids = append(ids, id)
	}
	return ids
}

// historyFilter 返回当前用户在发布历史维度的可见范围。
// admin 不受限（返回 nil, ""）；普通用户只看到自己执行的记录（trigger_by = 自己的登录账号），
// 同时仍受 allowedJobs 任务范围约束（taskIDs 非空时进一步限制任务集合，空切片表示无任务）。
// 这与 jobScope 是两套独立的维度：allowedJobs 限制「看哪些任务」，triggerBy 限制「看谁执行的」。
func (h *APIHandler) historyFilter(c *gin.Context) (taskIDs []string, triggerBy string) {
	_, _, role := currentUser(c)
	if role == "admin" {
		return nil, ""
	}
	username, _, _ := currentUser(c)
	return h.scopeTaskIDs(c), username
}

// Jobs API
func (h *APIHandler) ListJobs(c *gin.Context) {
	jobs, err := h.jobRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if scope, restricted := h.jobScope(c); restricted {
		filtered := jobs[:0]
		for _, j := range jobs {
			if scope[j.ID] {
				filtered = append(filtered, j)
			}
		}
		jobs = filtered
	}
	if jobs == nil {
		jobs = []model.Job{}
	}
	c.JSON(200, jobs)
}

// validateJob 校验任务配置，返回错误提示（空串表示通过）。
func (h *APIHandler) validateJob(job model.Job) string {
	if job.ID == "" || job.JobPath == "" {
		return "任务ID和Jenkins路径不能为空"
	}
	if job.EnvID == "" {
		return "请选择所属环境"
	}
	if _, err := h.envRepo.Get(job.EnvID); err != nil {
		return "所选环境不存在"
	}
	return ""
}

func (h *APIHandler) CreateJob(c *gin.Context) {
	var job model.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	job.ID = strings.TrimSpace(job.ID)
	if msg := h.validateJob(job); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	if _, err := h.jobRepo.Get(job.ID); err == nil {
		c.JSON(400, gin.H{"error": repository.ErrJobIDExists.Error()})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := h.jobRepo.Create(job); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"message": "创建成功"})
}

func (h *APIHandler) UpdateJob(c *gin.Context) {
	id := c.Param("id")
	var job model.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	job.ID = strings.TrimSpace(job.ID)
	if msg := h.validateJob(job); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	if err := h.jobRepo.Update(id, job); err != nil {
		if errors.Is(err, repository.ErrJobIDExists) {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if job.ID != id {
		c.JSON(200, gin.H{"message": "任务已更新，关联定时任务与发布历史已同步"})
		return
	}
	c.JSON(200, gin.H{"message": "更新成功"})
}

func (h *APIHandler) DeleteJob(c *gin.Context) {
	if err := h.jobRepo.Delete(c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "删除成功"})
}

// BuildJob 手动触发构建（构建编排、分支归一与历史记录统一由 BuildService 完成）。
func (h *APIHandler) BuildJob(c *gin.Context) {
	var req struct {
		RefValue string `json:"refValue"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	// 权限校验：受限用户只能构建授权范围内的任务。
	if scope, restricted := h.jobScope(c); restricted && !scope[c.Param("id")] {
		c.JSON(403, gin.H{"error": "无权构建该任务：不在您的授权项目范围内"})
		return
	}

	username, name, _ := currentUser(c)
	res, err := h.buildSvc.Trigger(c.Param("id"), req.RefValue, "manual", username, name)
	switch {
	case errors.Is(err, service.ErrJobNotFound):
		c.JSON(404, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrEnvMissing):
		c.JSON(400, gin.H{"error": "任务关联的环境配置不存在，请先检查环境配置"})
	case errors.Is(err, service.ErrProdTagRequired), errors.Is(err, service.ErrProdTagFormat):
		c.JSON(400, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(500, gin.H{"error": err.Error()})
	default:
		c.JSON(200, gin.H{"message": "构建已触发", "buildUrl": res.BuildURL})
	}
}

// GetJobGitTags 返回指定任务对应 Git 仓库的最新 Tag 列表，供生产环境构建时下拉选择。
// 经由 Jenkins 的 Git Parameter 类插件端点拉取（与构建页「Build with Parameters」下拉同源，
// 复用任务 SCM 配置的 Git 凭据，private 仓库亦可）。任意环节不满足都优雅降级：
// 返回 available=false 与原因，前端回退到手动输入。
func (h *APIHandler) GetJobGitTags(c *gin.Context) {
	id := c.Param("id")

	// 与构建同理：受限用户只能查询授权范围内的任务。
	if scope, restricted := h.jobScope(c); restricted && !scope[id] {
		c.JSON(403, gin.H{"error": "无权访问该任务：不在您的授权项目范围内"})
		return
	}

	job, err := h.jobRepo.Get(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	// 经 Jenkins 的 Git Parameter 类插件拉取（复用任务 SCM 的 Git 凭据，private 仓库亦可）。
	tags, ok, jerr := h.jenkinsMgr.GetJobGitTags(job)
	if ok && len(tags) > 0 {
		c.JSON(200, gin.H{"tags": tags, "available": true, "source": "jenkins"})
		return
	}
	if jerr != nil {
		log.Printf("经 Jenkins 获取 Tag 失败：%v", jerr)
	}
	c.JSON(200, gin.H{
		"tags":      []string{},
		"available": false,
		"reason":    "无法从 Jenkins 获取 Tag：请确认该任务构建页确实有 tag 下拉（Git Parameter / List Git Branches 插件），且凭据有权访问仓库",
	})
}

// CreateTask 创建定时任务
func (h *APIHandler) CreateTask(c *gin.Context) {
	var req struct {
		TaskID       string `json:"taskId"`
		RefValue     string `json:"refValue"`
		ScheduleTime string `json:"scheduleTime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}
	if req.TaskID == "" || req.ScheduleTime == "" {
		c.JSON(400, gin.H{"error": "任务ID和执行时间不能为空"})
		return
	}

	// 权限校验：受限用户不能为授权范围外的任务创建定时计划。
	if scope, restricted := h.jobScope(c); restricted && !scope[req.TaskID] {
		c.JSON(403, gin.H{"error": "无权为该任务创建定时计划：不在授权项目范围内"})
		return
	}

	scheduleTime, err := time.ParseInLocation("2006-01-02 15:04", req.ScheduleTime, time.Local)
	if err != nil {
		c.JSON(400, gin.H{"error": "时间格式错误，请使用 YYYY-MM-DD HH:mm"})
		return
	}
	if scheduleTime.Before(time.Now()) {
		c.JSON(400, gin.H{"error": "执行时间不能早于当前时间"})
		return
	}

	if _, err := h.jobRepo.Get(req.TaskID); err != nil {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	// 同一任务同时只允许一个待执行的定时计划（此处需看全部用户，不能按创建人过滤）。
	if tasks, _ := h.taskRepo.List("pending", ""); hasPending(tasks, req.TaskID) {
		c.JSON(400, gin.H{"error": "该任务已有一个待执行的定时计划"})
		return
	}

	username, name, _ := currentUser(c)
	task := model.ScheduledTask{
		ID:           uuid.New().String(),
		TaskID:       req.TaskID,
		RefValue:     req.RefValue,
		ScheduleTime: scheduleTime,
		CreatedBy:    username,
		CreatedName:  name,
		Status:       "pending",
	}

	if err := h.taskRepo.Create(task); err != nil {
		c.JSON(500, gin.H{"error": "保存定时任务失败"})
		return
	}
	c.JSON(201, gin.H{"message": "定时任务创建成功"})
}

func hasPending(tasks []model.ScheduledTask, taskID string) bool {
	for _, t := range tasks {
		if t.TaskID == taskID {
			return true
		}
	}
	return false
}

// ListTasks 获取定时任务列表（模型字段已是可直接序列化的形态）
// 可见性规则与发布历史一致：管理员看全部（含创建人），普通用户只看自己创建的。
// createdBy 维度与 jobScope（任务范围）正交叠加。
func (h *APIHandler) ListTasks(c *gin.Context) {
	username, _, role := currentUser(c)
	createdBy := ""
	if role != "admin" {
		createdBy = username
	}
	tasks, err := h.taskRepo.List(c.Query("status"), createdBy)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []model.ScheduledTask{}
	}
	if scope, restricted := h.jobScope(c); restricted {
		filtered := tasks[:0]
		for _, t := range tasks {
			if scope[t.TaskID] {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	c.JSON(200, tasks)
}

// CancelTask 取消定时任务：仅创建者本人或管理员可操作。
func (h *APIHandler) CancelTask(c *gin.Context) {
	id := c.Param("id")
	task, err := h.taskRepo.Get(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "定时任务不存在"})
		return
	}

	username, _, role := currentUser(c)
	if task.CreatedBy != username && role != "admin" {
		c.JSON(403, gin.H{"error": "只能取消自己创建的定时任务"})
		return
	}
	if task.Status != "pending" {
		c.JSON(400, gin.H{"error": "该任务已不在待执行状态"})
		return
	}

	affected, err := h.taskRepo.Cancel(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if affected == 0 {
		c.JSON(400, gin.H{"error": "取消失败，任务状态已变更"})
		return
	}
	c.JSON(200, gin.H{"message": "已取消"})
}

func (h *APIHandler) DeleteTask(c *gin.Context) {
	if err := h.taskRepo.Delete(c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "删除成功"})
}

// ListHistory 获取发布历史（按 env_id 归属环境，任务改名不影响历史）
// HistoryStats 发布历史统计汇总。
func (h *APIHandler) HistoryStats(c *gin.Context) {
	taskIDs, by := h.historyFilter(c)
	total, success, failed, today, err := h.historyRepo.Stats(taskIDs, by)
	if err != nil {
		log.Printf("查询发布历史统计失败: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	rate := 0.0
	if total > 0 {
		rate = float64(success) / float64(total)
	}
	c.JSON(200, gin.H{
		"total":   total,
		"success": success,
		"failed":  failed,
		"today":   today,
		"rate":    rate,
	})
}

func (h *APIHandler) ListHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}

	envID := c.Query("envId")

	// 兼容按环境名传参的旧调用：统一换算成 env_id 再查，避免出现"名称副本"。
	if envID == "" {
		if envName := c.Query("env"); envName != "" {
			if env, err := h.envRepo.GetByName(envName); err == nil {
				envID = env.ID
			}
		}
	}

	// 普通用户只展示自己执行的记录（historyFilter 内部对非 admin 附加 trigger_by 过滤）；
	// admin 不受限，可看到所有用户、所有环境的执行记录及执行人。
	taskIDs, by := h.historyFilter(c)
	histories, total, err := h.historyRepo.GetPublishHistory(
		page, size, envID,
		c.Query("status"), c.Query("search"), c.Query("from"), c.Query("to"), taskIDs, by,
	)
	if err != nil {
		log.Printf("查询发布历史失败: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if histories == nil {
		histories = []model.PublishHistory{}
	}
	c.JSON(200, gin.H{"data": histories, "total": total, "page": page, "size": size})
}

// GetOverview 首页概览：任务总数、今日构建、运行中与每个任务的最近构建。
// 供任务列表的统计卡与"最近构建"列使用，避免前端多次请求拼接。
func (h *APIHandler) GetOverview(c *gin.Context) {
	jobs, err := h.jobRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 受限用户只统计授权范围内任务，否则列表只有 2 条而"任务总数"显示 70，口径自相矛盾。
	scope, restricted := h.jobScope(c)
	// historyFilter 同时给出「可看的任务集合」与「只看自己执行记录」的约束，
	// 普通用户的历史统计与"最近构建"才会与历史页一致。
	taskIDs, by := h.historyFilter(c)
	if restricted {
		filtered := make([]model.Job, 0, len(jobs))
		for _, j := range jobs {
			if scope[j.ID] {
				filtered = append(filtered, j)
			}
		}
		jobs = filtered
	}

	builds, success, err := h.historyRepo.TodayStats(taskIDs, by)
	if err != nil {
		builds, success = 0, 0
	}

	latest, err := h.historyRepo.LatestByJob(by)
	if err != nil {
		latest = map[string]repository.LatestBuild{}
	}
	if restricted {
		for id := range latest {
			if !scope[id] {
				delete(latest, id)
			}
		}
	}

	// 普通用户只看自己创建的定时任务（by 即其账号，admin 为空表示不过滤），
	// 与 /api/tasks 列表口径一致；同时仍叠加 jobScope 的任务范围约束。
	tasks, _ := h.taskRepo.List("", by)
	var running, pending int
	for _, t := range tasks {
		if restricted && !scope[t.TaskID] {
			continue
		}
		switch t.Status {
		case "executing":
			running++
		case "pending":
			pending++
		}
	}

	rate := 0.0
	if builds > 0 {
		rate = float64(success) / float64(builds)
	}

	c.JSON(200, gin.H{
		"jobs":         gin.H{"total": len(jobs)},
		"today":        gin.H{"builds": builds, "success": success, "rate": rate},
		"running":      running,
		"pending":      pending,
		"latestBuilds": latest,
	})
}

// Environment API（列出对所有登录用户开放；增删改仅管理员）
// environmentVO 环境配置 + 关联任务数（影响面）。
type environmentVO struct {
	model.Environment
	JobCount int `json:"jobCount"`
}

func (h *APIHandler) ListEnvironments(c *gin.Context) {
	envs, err := h.envRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	counts, err := h.envRepo.JobCounts()
	if err != nil {
		counts = map[string]int{}
	}
	resp := make([]environmentVO, 0, len(envs))
	for _, e := range envs {
		resp = append(resp, environmentVO{Environment: e, JobCount: counts[e.ID]})
	}
	c.JSON(200, resp)
}

var jenkinsParamPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateEnvironment 只校验真正生效的那个参数名（调用前需先 SyncParamName）。
// 环境标识不参与构建，这里不再做格式限制，仅由 Create/Update 保证非空与唯一。
func validateEnvironment(env model.Environment) string {
	if !jenkinsParamPattern.MatchString(env.ParamName) {
		return "参数名格式不合法：只能用字母、数字、下划线，且不能以数字开头（例：Tags / GIT_TAG / release_branch）"
	}
	return ""
}

// fillEnvParamDefaults 与 Create 保持一致，避免编辑时把参数名清空导致构建参数缺失。
func fillEnvParamDefaults(env *model.Environment) {
	if env.DefaultBranchParam == "" {
		env.DefaultBranchParam = "branch"
	}
	if env.DefaultTagParam == "" {
		env.DefaultTagParam = "Tags"
	}
}

func (h *APIHandler) CreateEnvironment(c *gin.Context) {
	var env model.Environment
	if err := c.ShouldBindJSON(&env); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env.Name = strings.TrimSpace(env.Name)
	if env.Name == "" {
		c.JSON(400, gin.H{"error": "环境名称不能为空"})
		return
	}
	env.ID = uuid.New().String()
	fillEnvParamDefaults(&env)
	env.SyncParamName()
	if msg := validateEnvironment(env); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	if err := h.envRepo.Create(env); err != nil {
		c.JSON(500, gin.H{"error": "创建环境失败: " + err.Error()})
		return
	}
	c.JSON(201, gin.H{"message": "创建成功"})
}

func (h *APIHandler) UpdateEnvironment(c *gin.Context) {
	id := c.Param("id")
	var env model.Environment
	if err := c.ShouldBindJSON(&env); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	env.Name = strings.TrimSpace(env.Name)
	if env.Name == "" {
		c.JSON(400, gin.H{"error": "环境名称不能为空"})
		return
	}
	// 确认环境存在，避免更新不存在的 id 时静默返回成功
	if _, err := h.envRepo.Get(id); err != nil {
		c.JSON(404, gin.H{"error": "环境不存在"})
		return
	}

	fillEnvParamDefaults(&env)
	env.SyncParamName()
	if msg := validateEnvironment(env); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	if err := h.envRepo.Update(id, env); err != nil {
		c.JSON(500, gin.H{"error": "更新环境失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "更新成功"})
}

func (h *APIHandler) DeleteEnvironment(c *gin.Context) {
	id := c.Param("id")
	has, err := h.envRepo.HasJobs(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if has {
		c.JSON(400, gin.H{"error": "该环境仍有关联的 Jenkins 任务，无法删除（请先删除相关任务）"})
		return
	}
	if err := h.envRepo.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, gin.H{"error": "环境不存在"})
			return
		}
		c.JSON(500, gin.H{"error": "删除环境失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "删除成功"})
}

// Server API
func (h *APIHandler) ListServers(c *gin.Context) {
	servers, err := h.serverRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// 隐藏 token
	type ServerVO struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		BaseURL   string    `json:"baseUrl"`
		User      string    `json:"user"`
		SortOrder int       `json:"sortOrder"`
		CreatedAt time.Time `json:"createdAt"`
	}
	resp := make([]ServerVO, len(servers))
	for i, s := range servers {
		resp[i] = ServerVO{ID: s.ID, Name: s.Name, BaseURL: s.BaseURL, User: s.User, SortOrder: s.SortOrder, CreatedAt: s.CreatedAt}
	}
	c.JSON(200, resp)
}

// ListServerOptions 返回服务器精简列表（仅 id + name），对所有登录用户开放。
// 任务列表需要显示所属服务器名称，但 base_url / 账号 / token 属于管理员配置，不应下发给普通用户。
func (h *APIHandler) ListServerOptions(c *gin.Context) {
	servers, err := h.serverRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	type serverOption struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	resp := make([]serverOption, len(servers))
	for i, s := range servers {
		resp[i] = serverOption{ID: s.ID, Name: s.Name}
	}
	c.JSON(200, resp)
}

func (h *APIHandler) CreateServer(c *gin.Context) {
	var server model.JenkinsServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if server.ID == "" {
		c.JSON(400, gin.H{"error": "服务器ID不能为空"})
		return
	}
	if err := h.serverRepo.Create(server); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	h.jenkinsMgr.ReloadServer(server.ID)
	c.JSON(201, gin.H{"message": "创建成功"})
}

func (h *APIHandler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	var server model.JenkinsServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if server.Name == "" || server.BaseURL == "" || server.User == "" {
		c.JSON(400, gin.H{"error": "名称、Base URL 与用户名不能为空"})
		return
	}
	if err := h.serverRepo.Update(id, server); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, gin.H{"error": "服务不存在"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	h.jenkinsMgr.ReloadServer(id)
	c.JSON(200, gin.H{"message": "更新成功"})
}

func (h *APIHandler) DeleteServer(c *gin.Context) {
	id := c.Param("id")
	if id == "jenkins" {
		c.JSON(400, gin.H{"error": "默认服务不能删除"})
		return
	}
	if err := h.serverRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	h.jenkinsMgr.ReloadServer(id)
	c.JSON(200, gin.H{"message": "删除成功"})
}

// Permissions API
func (h *APIHandler) ListPermissions(c *gin.Context) {
	perms, err := h.permRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, perms)
}

type permissionReq struct {
	UserID      string   `json:"userId"`
	Username    string   `json:"username"`
	UserName    string   `json:"userName"`
	Password    string   `json:"password"`
	WebRole     string   `json:"webRole"`
	AllowedJobs []string `json:"allowedJobs"`
}

func (r permissionReq) validate() string {
	switch {
	case r.UserID == "":
		return "用户ID不能为空"
	case r.Username == "":
		return "登录用户名不能为空"
	case r.UserName == "":
		return "用户名称不能为空"
	}
	return ""
}

func (h *APIHandler) CreatePermission(c *gin.Context) {
	var req permissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	if req.Password == "" {
		c.JSON(400, gin.H{"error": "登录密码不能为空"})
		return
	}
	if req.WebRole == "" {
		req.WebRole = "user"
	}
	if _, err := h.permRepo.GetByUsername(req.Username); err == nil {
		c.JSON(400, gin.H{"error": "该登录用户名已存在"})
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(500, gin.H{"error": "密码加密失败"})
		return
	}
	perm := model.Permission{
		UserID:      req.UserID,
		Username:    req.Username,
		UserName:    req.UserName,
		Password:    hashed,
		WebRole:     req.WebRole,
		AllowedJobs: req.AllowedJobs,
	}
	if err := h.permRepo.Create(perm); err != nil {
		c.JSON(500, gin.H{"error": "创建失败，用户ID可能已存在"})
		return
	}
	c.JSON(201, gin.H{"message": "创建成功"})
}

func (h *APIHandler) UpdatePermission(c *gin.Context) {
	userID := c.Param("userId")
	var req permissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if msg := req.validate(); msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	// 先取原记录：既用于「未提交字段沿用旧值」，也让更新不存在的用户尽早返回 404。
	existing, err := h.permRepo.Get(userID)
	if err != nil {
		c.JSON(404, gin.H{"error": "用户不存在"})
		return
	}

	if req.WebRole == "" {
		req.WebRole = existing.WebRole
	}
	if req.WebRole == "" {
		req.WebRole = "user"
	}
	if userID == "admin" {
		req.WebRole = "admin"
	}
	if found, err := h.permRepo.GetByUsername(req.Username); err == nil && found.UserID != userID {
		c.JSON(400, gin.H{"error": "该登录用户名已被占用"})
		return
	}

	// 密码留空表示不修改；非空则重新哈希，并踢掉该用户已有会话。
	password := req.Password
	passwordChanged := false
	if password == "" {
		password = existing.Password
	} else {
		hashed, err := auth.HashPassword(password)
		if err != nil {
			c.JSON(500, gin.H{"error": "密码加密失败"})
			return
		}
		password, passwordChanged = hashed, true
	}

	// allowedJobs 未随请求提交时沿用旧值：只重置密码的请求若被当成「清空」，
	// 语义会从「仅 N 个项目」变成「全部项目」，属于权限放大。
	allowedJobs := req.AllowedJobs
	if allowedJobs == nil {
		allowedJobs = existing.AllowedJobs
	}

	perm := model.Permission{Username: req.Username, UserName: req.UserName, Password: password, WebRole: req.WebRole, AllowedJobs: allowedJobs}
	if err := h.permRepo.Update(userID, perm); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if passwordChanged {
		h.authSvc.LogoutUser(req.Username)
	}
	c.JSON(200, gin.H{"message": "更新成功"})
}

func (h *APIHandler) DeletePermission(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "admin" {
		c.JSON(400, gin.H{"error": "不能删除管理员账号"})
		return
	}
	perm, _ := h.permRepo.Get(userID)
	if err := h.permRepo.Delete(userID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if perm.Username != "" {
		h.authSvc.LogoutUser(perm.Username)
	}
	c.JSON(200, gin.H{"message": "删除成功"})
}

// Auth API
func (h *APIHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}
	token, role, ok := h.authSvc.Login(req.Username, req.Password)
	if !ok {
		c.JSON(401, gin.H{"error": "用户名或密码错误"})
		return
	}
	c.JSON(200, gin.H{"token": token, "role": role})
}

func (h *APIHandler) Logout(c *gin.Context) {
	h.authSvc.Logout(c.GetHeader("Authorization"))
	c.JSON(200, gin.H{"message": "已退出"})
}

// DingTalk API
func (h *APIHandler) SearchDingTalkUsers(c *gin.Context) {
	users, err := h.dingtalkSvc.SearchUsers(c.Query("name"))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, users)
}

// SyncDingTalkUsers 手动同步钉钉员工
func (h *APIHandler) SyncDingTalkUsers(c *gin.Context) {
	if err := h.dingtalkSvc.SyncUsers(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "同步成功", "count": h.dingtalkSvc.Size()})
}

// sameIDSet 校验拖拽提交的 ID 列表与库中记录完全一致，避免漏项或夹带不存在的 ID。
func sameIDSet(ids []string, existing []string) bool {
	if len(ids) != len(existing) {
		return false
	}
	set := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		set[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := set[id]; !ok {
			return false
		}
		if _, dup := seen[id]; dup {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func bindOrder(c *gin.Context) ([]string, bool) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return nil, false
	}
	return req.IDs, true
}

// ReorderEnvironments 保存环境卡片的拖拽顺序。
func (h *APIHandler) ReorderEnvironments(c *gin.Context) {
	ids, ok := bindOrder(c)
	if !ok {
		return
	}
	envs, err := h.envRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	all := make([]string, 0, len(envs))
	for _, e := range envs {
		all = append(all, e.ID)
	}
	if !sameIDSet(ids, all) {
		c.JSON(400, gin.H{"error": "顺序列表与现有环境不一致，请刷新后重试"})
		return
	}
	if err := h.envRepo.UpdateOrder(ids); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "顺序已保存"})
}

// ReorderServers 保存服务卡片的拖拽顺序。
func (h *APIHandler) ReorderServers(c *gin.Context) {
	ids, ok := bindOrder(c)
	if !ok {
		return
	}
	servers, err := h.serverRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	all := make([]string, 0, len(servers))
	for _, s := range servers {
		all = append(all, s.ID)
	}
	if !sameIDSet(ids, all) {
		c.JSON(400, gin.H{"error": "顺序列表与现有服务不一致，请刷新后重试"})
		return
	}
	if err := h.serverRepo.UpdateOrder(ids); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "顺序已保存"})
}
