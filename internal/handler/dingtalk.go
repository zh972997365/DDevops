package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"ddevops/internal/repository"
	"ddevops/internal/service"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
)

// 钉钉机器人：负责消息接收、命令分发与转发。
//   - 卡片渲染集中在 dingtalk_card.go
//   - 定时相关命令集中在 dingtalk_command.go
type DingTalkHandler struct {
	config   DingTalkConfig
	buildSvc *service.BuildService
	permRepo *repository.PermissionRepository
	jobRepo  *repository.JobRepository
	taskRepo *repository.TaskRepository
	envRepo  *repository.EnvironmentRepository

	activeSessions sync.Map
}

type DingTalkConfig struct {
	ClientID     string
	ClientSecret string
}

// DingTalkDeps 钉钉机器人的依赖聚合。
type DingTalkDeps struct {
	Build        *service.BuildService
	Permissions  *repository.PermissionRepository
	Jobs         *repository.JobRepository
	Tasks        *repository.TaskRepository
	Environments *repository.EnvironmentRepository
}

type SessionInfo struct {
	WebhookURL     string
	ConversationID string
	LastActive     time.Time
	StaffID        string
	StaffName      string
}

func NewDingTalkHandler(config DingTalkConfig, deps DingTalkDeps) *DingTalkHandler {
	return &DingTalkHandler{
		config:   config,
		buildSvc: deps.Build,
		permRepo: deps.Permissions,
		jobRepo:  deps.Jobs,
		taskRepo: deps.Tasks,
		envRepo:  deps.Environments,
	}
}

func (h *DingTalkHandler) Start(ctx context.Context) error {
	cli := client.NewStreamClient(
		client.WithAppCredential(client.NewAppCredentialConfig(
			h.config.ClientID, h.config.ClientSecret,
		)),
	)

	cli.RegisterChatBotCallbackRouter(h.onMessage)

	log.Println("钉钉机器人服务启动中...")
	return cli.Start(ctx)
}

func (h *DingTalkHandler) onMessage(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	// 保存会话信息
	if data.ConversationId != "" && data.SessionWebhook != "" {
		sessionInfo := SessionInfo{
			WebhookURL:     data.SessionWebhook,
			ConversationID: data.ConversationId,
			LastActive:     time.Now(),
			StaffID:        data.SenderStaffId,
			StaffName:      data.SenderNick,
		}
		h.activeSessions.Store(data.ConversationId, sessionInfo)
		log.Printf("保存会话信息: ConversationID=%s", data.ConversationId)
	}

	log.Printf("收到机器人消息: %s", data.Text.Content)
	botIdentifier := data.ChatbotUserId

	if !h.isMentioned(data.AtUsers, botIdentifier) {
		log.Println("消息未@机器人，跳过处理")
		return []byte(`{"code":0}`), nil
	}

	// 获取用户ID
	userID := data.SenderStaffId
	if userID == "" {
		userID = data.SenderId
	}

	// 解析授权范围（与 Web 端 jobScope 共用 jobScopeFor 同一套规则）。
	// exists=false 表示此工号没有对应的授权记录。
	scope, restricted, exists := h.resolveScope(userID)
	if !exists {
		h.sendForbiddenCardMessage(data.SessionWebhook)
		return []byte(`{"code":0}`), nil
	}

	command := h.extractCommand(data.Text.Content)
	log.Printf("提取的命令: %s", command)

	parts := strings.Fields(command)
	if len(parts) < 1 {
		h.sendHelpCardMessage(data.SessionWebhook)
		return []byte(`{"code":0}`), nil
	}

	switch parts[0] {
	case "帮助", "help":
		h.sendHelpCardMessage(data.SessionWebhook)
		return []byte(`{"code":0}`), nil
	case "列表", "list":
		h.sendSimpleCardMessage(data.SessionWebhook, "ℹ️ 指令已调整", "- **状态**: 列表已调整为按环境查看\n- **可用**: 测试 / 预发 / 生产")
		return []byte(`{"code":0}`), nil
	case "测试", "预发", "生产":
		h.sendTaskListMessage(data.SessionWebhook, scope, restricted, parts[0])
		return []byte(`{"code":0}`), nil
	case "定时列表":
		h.handleListScheduledTasks(data, scope, restricted)
		return []byte(`{"code":0}`), nil
	case "取消定时":
		h.handleCancelScheduledTask(data, parts)
		return []byte(`{"code":0}`), nil
	}

	if strings.HasPrefix(command, "定时") {
		h.handleScheduledCommand(data, command, scope, restricted)
		return []byte(`{"code":0}`), nil
	}

	// 处理构建命令：任务查询、分支归一、触发与历史记录统一交给 BuildService。
	// 生产环境使用版本标签，非生产环境自动补 origin/ 前缀，未指定则用环境默认分支。
	// 首参既可以是完整任务ID，也可以是创建时分配的短码（P01/T02…）。
	taskID := h.resolveJobID(parts[0])
	refInput := ""
	if len(parts) > 1 {
		refInput = strings.Join(parts[1:], " ")
	}

	// 任务级授权：受限用户只能构建授权范围内的任务。
	if restricted && !scope[taskID] {
		h.sendNoTaskPermissionCardMessage(data.SessionWebhook, taskID)
		return []byte(`{"code":0}`), nil
	}

	res, err := h.buildSvc.Trigger(taskID, refInput, "manual", userID, data.SenderNick)
	switch {
	case errors.Is(err, service.ErrJobNotFound):
		log.Printf("未知任务ID: [%s]", taskID)
		h.sendUnknownTaskCardMessage(data.SessionWebhook, taskID)
	case errors.Is(err, service.ErrEnvMissing):
		var b strings.Builder
		fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(taskID))
		b.WriteString("- **原因**: 关联的环境配置不存在\n")
		b.WriteString("- **处理**: 请联系管理员\n")
		h.sendSimpleCardMessage(data.SessionWebhook, "⚠️ 环境配置缺失", b.String())
	case errors.Is(err, service.ErrProdTagRequired), errors.Is(err, service.ErrProdTagFormat):
		var b strings.Builder
		fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(taskID))
		b.WriteString("- **要求**: 生产环境必须指定版本 Tag\n")
		fmt.Fprintf(&b, "- **示例**: %s 20261105u01\n", mdCell(taskID))
		h.sendSimpleCardMessage(data.SessionWebhook, "⚠️ 生产环境需要版本 Tag", b.String())
	case err != nil:
		log.Printf("Jenkins构建失败: %v", err)
		h.sendBuildFailureCardMessage(data.SessionWebhook, taskID, res.RefValue, err.Error(), res.BuildURL, res.Env, res.Job.ShortCode)
	default:
		h.sendBuildSuccessCardMessage(data.SessionWebhook, taskID, res.RefValue, res.BuildURL, res.Env, res.Job.ShortCode)
	}

	return []byte(`{"code":0}`), nil
}

// resolveJobID 把输入解析为完整任务ID：既能识别短码（P01/T02…），也兼容原完整ID。
// 短码优先按精确匹配查询，查不到时原样返回，交给下游的 ErrJobNotFound 统一提示。
func (h *DingTalkHandler) resolveJobID(input string) string {
	job, err := h.jobRepo.GetByShortCode(input)
	if err != nil {
		return input
	}
	return job.ID
}

// envNameMap 构建 任务ID -> 环境名 映射，避免循环中反复查库。
// 环境名实时取自 environments 表：任务只存 env_id，不存任何环境名副本。
func (h *DingTalkHandler) envNameMap() map[string]string {
	jobs, err := h.jobRepo.List()
	if err != nil {
		log.Printf("获取任务列表失败: %v", err)
		return map[string]string{}
	}
	envs, err := h.envRepo.List()
	if err != nil {
		log.Printf("获取环境列表失败: %v", err)
		return map[string]string{}
	}
	nameByEnvID := make(map[string]string, len(envs))
	for _, e := range envs {
		nameByEnvID[e.ID] = e.Name
	}
	m := make(map[string]string, len(jobs))
	for _, j := range jobs {
		m[j.ID] = nameByEnvID[j.EnvID]
	}
	return m
}

// resolveScope 按钉钉工号解析任务维度的授权范围，与 Web 端 jobScope 共用 jobScopeFor 同一套规则。
// exists=false 表示此工号没有对应的授权记录（尚未被纳入用户管理）。
func (h *DingTalkHandler) resolveScope(staffID string) (scope map[string]bool, restricted, exists bool) {
	perm, err := h.permRepo.Get(staffID)
	if err != nil {
		return nil, false, false
	}
	scope, restricted = jobScopeFor(perm.WebRole, perm.AllowedJobs)
	return scope, restricted, true
}

// 检查是否@了机器人
func (h *DingTalkHandler) isMentioned(atUsers []chatbot.BotCallbackDataAtUserModel, botID string) bool {
	for _, user := range atUsers {
		if user.DingtalkId == botID || user.StaffId == botID {
			return true
		}
	}
	return false
}

// 提取命令文本
func (h *DingTalkHandler) extractCommand(content string) string {
	re := regexp.MustCompile(`^@\S+\s*`)
	return strings.TrimSpace(re.ReplaceAllString(content, ""))
}
