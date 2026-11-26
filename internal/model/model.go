package model

import (
	"time"
)

// Job Jenkins 任务。id 既是业务主键也是任务名（唯一、可改名），
// 环境信息只通过 EnvID 关联，任务不保存任何环境名副本。
type Job struct {
	ID        string    `db:"id" json:"id"`
	JobPath   string    `db:"job_path" json:"jobPath"`
	ServerID  string    `db:"server_id" json:"serverId"`
	EnvID     string    `db:"env_id" json:"envId"` // 关联 environments.id，取代对 Name 的硬编码判断
	ShortCode string    `db:"short_code" json:"shortCode"` // 钉钉发布的短码，创建时生成一次后固定；与 ID 改名解耦，避免短码漂移
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Environment 环境配置（测试/预发/生产等），集中承载构建规则，取代散落的硬编码。
type Environment struct {
	ID                 string `db:"id" json:"id"`
	Name               string `db:"name" json:"name"`
	IsProd             bool   `db:"is_prod" json:"isProd"`         // 是否生产环境（决定使用版本标签参数还是分支参数）
	DefaultRef         string `db:"default_ref" json:"defaultRef"` // 默认分支引用，如 origin/test
	DefaultBranchParam string `db:"-" json:"defaultBranchParam"`   // 仅用于前端表单契约，不落库
	DefaultTagParam    string `db:"-" json:"defaultTagParam"`      // 仅用于前端表单契约，不落库
	// param_name 是实际生效的那个参数名的唯一存储位。
	ParamName string    `db:"param_name" json:"-"`
	SortOrder int       `db:"sort_order" json:"sortOrder"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// DefaultRefValue 返回默认分支引用，未配置时回落到 origin/test。
func (e Environment) DefaultRefValue() string {
	if e.DefaultRef == "" {
		return "origin/test"
	}
	return e.DefaultRef
}

type ScheduledTask struct {
	ID           string     `db:"id" json:"id"`
	TaskID       string     `db:"task_id" json:"taskId"`             // Jenkins 任务ID（jobs.id）
	RefValue     string     `db:"ref_value" json:"refValue"`         // 分支/版本
	ScheduleTime time.Time  `db:"schedule_time" json:"scheduleTime"` // 计划执行时间
	CreatedBy    string     `db:"created_by" json:"createdBy"`       // 创建人ID
	CreatedName  string     `db:"created_name" json:"createdName"`   // 创建人名称
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	Status       string     `db:"status" json:"status"` // pending/executing/completed/failed/cancelled
	WebhookURL   string     `db:"webhook_url" json:"webhookUrl"`
	ExecuteTime  *time.Time `db:"execute_time" json:"executeTime"`
	ErrorMessage string     `db:"error_message" json:"errorMessage"` // 读取时用 IFNULL 归一为空串
}

// PublishHistory 发布历史。EnvID 是环境归属的唯一依据（环境名实时从 environments 取），
// TaskID 是任务归属的唯一依据（任务改名时由 JobRepository 级联更新）。
type PublishHistory struct {
	ID           int64     `db:"id" json:"id"`
	TaskID       string    `db:"task_id" json:"taskId"`
	EnvID        string    `db:"env_id" json:"envId"`
	RefValue     string    `db:"ref_value" json:"refValue"`
	TriggerType  string    `db:"trigger_type" json:"triggerType"` // manual/scheduled
	TriggerBy    string    `db:"trigger_by" json:"triggerBy"`     // 触发人ID
	TriggerName  string    `db:"trigger_name" json:"triggerName"` // 触发人名称
	TriggerTime  time.Time `db:"trigger_time" json:"triggerTime"`
	BuildStatus  string    `db:"build_status" json:"buildStatus"` // success/failed
	BuildURL     string    `db:"build_url" json:"buildUrl"`
	ErrorMessage string    `db:"error_message" json:"errorMessage"`
}

type Permission struct {
	UserID      string    `db:"user_id" json:"userId"`
	Username    string    `db:"username" json:"username"`
	UserName    string    `db:"user_name" json:"userName"`
	Password    string    `db:"password" json:"-"`
	WebRole     string    `db:"web_role" json:"webRole"`
	AllowedJobs []string  `db:"-" json:"allowedJobs"` // 允许构建/查看的任务 ID 列表；admin 始终全量，普通用户仅限此列表（空列表即无授权任务）
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

type JenkinsServer struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	BaseURL   string    `db:"base_url" json:"baseUrl"`
	User      string    `db:"user_name" json:"user"`
	Token     string    `db:"token" json:"-"`
	SortOrder int       `db:"sort_order" json:"sortOrder"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Session 登录会话（持久化在 sessions 表，服务重启后仍然有效）。
type Session struct {
	Token     string    `db:"token"`
	Username  string    `db:"username"`
	Name      string    `db:"name"`
	Role      string    `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}
