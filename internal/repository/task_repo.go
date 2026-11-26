package repository

import (
	"strings"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// TaskRepository 管理定时任务（scheduled_tasks 表）。
type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db.db}
}

// taskCols 统一列清单，可空文本列用 IFNULL 归一为空串。
const taskCols = `id, task_id, IFNULL(ref_value,'') AS ref_value, schedule_time,
	created_by, IFNULL(created_name,'') AS created_name, created_at, status,
	IFNULL(webhook_url,'') AS webhook_url, execute_time, IFNULL(error_message,'') AS error_message`

func (r *TaskRepository) Create(task model.ScheduledTask) error {
	_, err := r.db.Exec(
		`INSERT INTO scheduled_tasks
		(id, task_id, ref_value, schedule_time, created_by, created_name, webhook_url, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.TaskID, task.RefValue, task.ScheduleTime,
		task.CreatedBy, task.CreatedName, task.WebhookURL, task.Status,
	)
	return err
}

// UpdateStatus 更新任务状态与执行时间（错误信息为空时写入 NULL）。
func (r *TaskRepository) UpdateStatus(id, status, errorMsg string) error {
	var errMsg interface{}
	if errorMsg != "" {
		errMsg = errorMsg
	}
	_, err := r.db.Exec(
		"UPDATE scheduled_tasks SET status=?, execute_time=CURRENT_TIMESTAMP, error_message=? WHERE id=?",
		status, errMsg, id,
	)
	return err
}

// GetPending 获取待执行/执行中的任务（调度器用）。
func (r *TaskRepository) GetPending() ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.db.Select(&tasks,
		"SELECT "+taskCols+" FROM scheduled_tasks WHERE status IN ('pending', 'executing') ORDER BY schedule_time")
	return tasks, err
}

// Get 按主键取单条定时任务。
func (r *TaskRepository) Get(id string) (model.ScheduledTask, error) {
	var task model.ScheduledTask
	err := r.db.Get(&task, "SELECT "+taskCols+" FROM scheduled_tasks WHERE id = ?", id)
	return task, err
}

// Cancel 取消待执行的定时任务；返回受影响行数以便调用方判断是否真的取消成功。
func (r *TaskRepository) Cancel(id string) (int64, error) {
	res, err := r.db.Exec(
		"UPDATE scheduled_tasks SET status='cancelled', execute_time=CURRENT_TIMESTAMP WHERE id=? AND status='pending'",
		id,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *TaskRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM scheduled_tasks WHERE id = ?", id)
	return err
}

// List 按状态获取定时任务列表；status 为空或 "all" 表示全部。
// createdBy 非空时仅返回该创建人的任务（普通用户只看自己创建的定时任务），
// 为空表示不过滤（管理员视图、调度器业务校验等场景使用）。
func (r *TaskRepository) List(status, createdBy string) ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask

	where := []string{}
	args := []interface{}{}
	if status != "" && status != "all" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if createdBy != "" {
		where = append(where, "created_by = ?")
		args = append(args, createdBy)
	}

	query := "SELECT " + taskCols + " FROM scheduled_tasks"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY schedule_time"

	if err := r.db.Select(&tasks, query, args...); err != nil {
		return nil, err
	}
	return tasks, nil
}
