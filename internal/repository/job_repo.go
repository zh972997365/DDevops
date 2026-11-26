package repository

import (
	"ddevops/internal/model"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// shortCodePrefix 按环境推断短码前缀：生产 P、预发 S、其余 T。
// 环境名可能自定义，判断是否预发用"预发/pre/stag"三种常见写法兜底。
func shortCodePrefix(env model.Environment) string {
	if env.IsProd {
		return "P"
	}
	name := strings.ToLower(env.Name)
	if strings.Contains(name, "预发") || strings.Contains(name, "pre") || strings.Contains(name, "stag") {
		return "S"
	}
	return "T"
}

// nextShortCode 在给定前缀下分配下一个两位序号短码（P01、T07 …）。
// 连接被限制为单连接，查询最大值与随后写入串行，序号不会重复。
func nextShortCode(db *sqlx.DB, prefix string) (string, error) {
	var codes []string
	if err := db.Select(&codes, "SELECT short_code FROM jobs WHERE short_code LIKE ?", prefix+"%"); err != nil {
		return "", err
	}
	max := 0
	for _, code := range codes {
		if n, e := strconv.Atoi(strings.TrimPrefix(code, prefix)); e == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s%02d", prefix, max+1), nil
}

// ErrJobIDExists 任务改名时，新 ID 已被其他任务占用。
var ErrJobIDExists = errors.New("任务 ID 已存在")

// JobRepository 管理 Jenkins 任务配置（jobs 表）。
type JobRepository struct {
	db *sqlx.DB
}

func NewJobRepository(db *DB) *JobRepository {
	return &JobRepository{db: db.db}
}

// jobCols 显式列清单：老库可能残留已废弃的 name 列，
// 用 SELECT * 会让 sqlx 因未映射字段直接报错。
const jobCols = `id, job_path, server_id, env_id, short_code, created_at, updated_at`

func (r *JobRepository) List() ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Select(&jobs, "SELECT "+jobCols+" FROM jobs ORDER BY id")
	return jobs, err
}

func (r *JobRepository) Get(id string) (model.Job, error) {
	var job model.Job
	err := r.db.Get(&job, "SELECT "+jobCols+" FROM jobs WHERE id = ?", id)
	return job, err
}

func (r *JobRepository) Create(job model.Job) error {
	// 短码只在创建时生成一次；调用方（API）已校验环境存在，这里读取环境推断前缀。
	var env model.Environment
	if err := r.db.Get(&env, "SELECT id, name, is_prod FROM environments WHERE id = ?", job.EnvID); err != nil {
		return fmt.Errorf("读取任务环境以生成短码失败: %v", err)
	}
	code, err := nextShortCode(r.db, shortCodePrefix(env))
	if err != nil {
		return fmt.Errorf("生成任务短码失败: %v", err)
	}
	job.ShortCode = code

	_, err = r.db.Exec(
		"INSERT INTO jobs (id, job_path, server_id, env_id, short_code) VALUES (?, ?, ?, ?, ?)",
		job.ID, job.JobPath, job.ServerID, job.EnvID, job.ShortCode,
	)
	return err
}

// GetByShortCode 按短码查任务（别名入口，大小写不敏感）。
func (r *JobRepository) GetByShortCode(code string) (model.Job, error) {
	var job model.Job
	err := r.db.Get(&job, "SELECT "+jobCols+" FROM jobs WHERE short_code = ?", strings.ToUpper(strings.TrimSpace(code)))
	return job, err
}

// BackfillShortCodes 为存量任务补短码：按各自环境前缀分配，幂等（已有短码的跳过）。
func (r *JobRepository) BackfillShortCodes() (int, error) {
	jobs, err := r.List()
	if err != nil {
		return 0, err
	}
	envByID := map[string]model.Environment{}
	rows, err := r.db.Query("SELECT id, name, is_prod FROM environments")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var e model.Environment
		if err := rows.Scan(&e.ID, &e.Name, &e.IsProd); err != nil {
			return 0, err
		}
		envByID[e.ID] = e
	}

	// 先按前缀分组收集待补任务，避免边查边改导致序号重复。
	grouped := map[string][]string{}
	for _, job := range jobs {
		if job.ShortCode != "" {
			continue
		}
		prefix := shortCodePrefix(envByID[job.EnvID])
		grouped[prefix] = append(grouped[prefix], job.ID)
	}

	count := 0
	for prefix, ids := range grouped {
		for _, id := range ids {
			code, err := nextShortCode(r.db, prefix)
			if err != nil {
				return count, err
			}
			if _, err := r.db.Exec("UPDATE jobs SET short_code = ? WHERE id = ?", code, id); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func (r *JobRepository) Update(id string, job model.Job) error {
	if job.ID == id {
		_, err := r.db.Exec(
			"UPDATE jobs SET id=?, job_path=?, server_id=?, env_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			job.ID, job.JobPath, job.ServerID, job.EnvID, id,
		)
		return err
	}
	return r.renameID(id, job)
}

// renameID 主键改名时同步两张关联表的 task_id：jobs.id 同时被定时任务和发布历史引用，
// 只改 jobs 会让这些记录变成孤儿。全部放在事务里，要么一起改、要么都不改。
func (r *JobRepository) renameID(oldID string, job model.Job) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var cnt int
	if err := tx.Get(&cnt, "SELECT COUNT(1) FROM jobs WHERE id = ?", job.ID); err != nil {
		return err
	}
	if cnt > 0 {
		return ErrJobIDExists
	}

	if _, err := tx.Exec(
		"UPDATE jobs SET id=?, job_path=?, server_id=?, env_id=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		job.ID, job.JobPath, job.ServerID, job.EnvID, oldID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE scheduled_tasks SET task_id=? WHERE task_id=?", job.ID, oldID); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE publish_history SET task_id=? WHERE task_id=?", job.ID, oldID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *JobRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}
