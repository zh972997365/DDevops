package repository

import (
	"strings"
	"time"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// HistoryRepository 管理发布历史（publish_history 表）。
type HistoryRepository struct {
	db *sqlx.DB
}

func NewHistoryRepository(db *DB) *HistoryRepository {
	return &HistoryRepository{db: db.db}
}

// historyCols 统一列清单：可空文本列用 IFNULL 归一为空串，
// 这样 model 里可以直接用 string，不必在 handler 层做 sql.NullString 转换。
const historyCols = `id, task_id, env_id, ref_value, trigger_type, trigger_by,
	IFNULL(trigger_name,'') AS trigger_name, trigger_time, build_status,
	IFNULL(build_url,'') AS build_url, IFNULL(error_message,'') AS error_message`

func (r *HistoryRepository) Create(history model.PublishHistory) error {
	_, err := r.db.Exec(
		`INSERT INTO publish_history
		(task_id, env_id, ref_value, trigger_type, trigger_by, trigger_name, build_status, build_url, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		history.TaskID, history.EnvID, history.RefValue, history.TriggerType,
		history.TriggerBy, history.TriggerName, history.BuildStatus, history.BuildURL, history.ErrorMessage,
	)
	return err
}

// GetPublishHistory 分页查询发布历史，支持环境/状态/关键词/日期范围筛选。
// 环境只按 env_id 匹配（环境名由前端用同一份环境列表换算，改名即时生效）。
// taskIDs 非空时仅返回这些任务的历史（受限用户视图）。
// triggerBy 非空时仅返回该执行人的记录（普通用户只能看自己的执行历史）。
// 业务约束：只看近一年。
func (r *HistoryRepository) GetPublishHistory(page, pageSize int, envID, status, search, from, to string, taskIDs []string, triggerBy string) ([]model.PublishHistory, int64, error) {
	var histories []model.PublishHistory
	var total int64

	baseQuery := "FROM publish_history WHERE 1=1"
	args := []interface{}{}

	if envID != "" {
		baseQuery += " AND env_id = ?"
		args = append(args, envID)
	}

	if cond, fargs := taskFilter(taskIDs); cond != "" {
		baseQuery += cond
		args = append(args, fargs...)
	}

	if triggerBy != "" {
		baseQuery += " AND trigger_by = ?"
		args = append(args, triggerBy)
	}

	if status != "" {
		baseQuery += " AND build_status = ?"
		args = append(args, status)
	}

	if search != "" {
		baseQuery += " AND task_id LIKE ?"
		args = append(args, "%"+search+"%")
	}

	// 日期范围按字符串前缀比较，避免 SQLite 无法解析带时区的 DATETIME。
	if from != "" {
		baseQuery += " AND substr(trigger_time, 1, 10) >= ?"
		args = append(args, from)
	}
	if to != "" {
		baseQuery += " AND substr(trigger_time, 1, 10) <= ?"
		args = append(args, to)
	}

	baseQuery += " AND trigger_time >= datetime('now', '-1 year')"

	if err := r.db.Get(&total, "SELECT COUNT(*) "+baseQuery, args...); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return histories, 0, nil
	}

	query := "SELECT " + historyCols + " " + baseQuery + " ORDER BY trigger_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	if err := r.db.Select(&histories, query, args...); err != nil {
		return nil, 0, err
	}
	return histories, total, nil
}

// LatestBuild 某个任务最近一次构建的摘要。
type LatestBuild struct {
	TaskID string    `db:"task_id" json:"-"`
	Status string    `db:"status" json:"status"`
	Time   time.Time `db:"last_time" json:"time"`
	URL    string    `db:"url" json:"url"`
}

// taskFilter 生成 " AND task_id IN (?,?,...)" 片段及其参数。
// taskIDs 为 nil 表示不限制（管理员）；非 nil 的空切片表示「无任何授权任务」，匹配空结果。
func taskFilter(taskIDs []string) (string, []interface{}) {
	if taskIDs == nil {
		return "", nil
	}
	if len(taskIDs) == 0 {
		return " AND 0=1", nil
	}
	ph := strings.Repeat("?,", len(taskIDs))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		args[i] = id
	}
	return " AND task_id IN (" + ph + ")", args
}

// TodayStats 今日构建总数与成功数（按本地日期前缀匹配）。
// taskIDs 非空时只统计这些任务（受限用户视图）；triggerBy 非空时只统计该执行人。
func (r *HistoryRepository) TodayStats(taskIDs []string, triggerBy string) (total int, success int, err error) {
	cond, args := taskFilter(taskIDs)
	if triggerBy != "" {
		cond += " AND trigger_by = ?"
		args = append(args, triggerBy)
	}
	err = r.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN build_status = 'success' THEN 1 ELSE 0 END), 0)
		 FROM publish_history WHERE substr(trigger_time, 1, 10) = date('now', 'localtime')`+cond,
		args...,
	).Scan(&total, &success)
	return
}

// Stats 近一年的构建总数、成功数、失败数，以及今日构建数。
// taskIDs 非空时只统计这些任务（受限用户视图）；triggerBy 非空时只统计该执行人。
func (r *HistoryRepository) Stats(taskIDs []string, triggerBy string) (total, success, failed, today int, err error) {
	cond, args := taskFilter(taskIDs)
	if triggerBy != "" {
		cond += " AND trigger_by = ?"
		args = append(args, triggerBy)
	}
	err = r.db.QueryRow(
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN build_status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN build_status = 'failed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN substr(trigger_time, 1, 10) = date('now', 'localtime') THEN 1 ELSE 0 END), 0)
		 FROM publish_history WHERE trigger_time >= datetime('now', '-1 year')`+cond,
		args...,
	).Scan(&total, &success, &failed, &today)
	return
}

// LatestByJob 返回每个任务最近一次构建，key 为任务 ID。
// triggerBy 非空时只取该执行人的最近一次构建（普通用户只能看自己的）。
func (r *HistoryRepository) LatestByJob(triggerBy string) (map[string]LatestBuild, error) {
	subWhere := ""
	args := []interface{}{}
	if triggerBy != "" {
		subWhere = " WHERE trigger_by = ?"
		args = append(args, triggerBy)
	}
	var rows []LatestBuild
	err := r.db.Select(&rows,
		`SELECT task_id, build_status AS status, trigger_time AS last_time, IFNULL(build_url, '') AS url
		 FROM publish_history
		 WHERE id IN (SELECT MAX(id) FROM publish_history`+subWhere+` GROUP BY task_id)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string]LatestBuild, len(rows))
	for _, row := range rows {
		out[row.TaskID] = row
	}
	return out, nil
}
