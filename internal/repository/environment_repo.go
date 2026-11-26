package repository

import (
	"database/sql"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// 显式列清单：只查真正落库的字段；老库残留的历史列由迁移删除，不影响读取。
const envCols = "id, name, is_prod, default_ref, param_name, sort_order, created_at, updated_at"

// EnvironmentRepository 管理环境配置（environments 表）。
type EnvironmentRepository struct {
	db *sqlx.DB
}

func NewEnvironmentRepository(db *DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db.db}
}

func (r *EnvironmentRepository) List() ([]model.Environment, error) {
	var envs []model.Environment
	err := r.db.Select(&envs, "SELECT "+envCols+" FROM environments ORDER BY sort_order, created_at, id")
	for i := range envs {
		envs[i].ExpandParamName()
	}
	return envs, err
}

func (r *EnvironmentRepository) Get(id string) (model.Environment, error) {
	var env model.Environment
	err := r.db.Get(&env, "SELECT "+envCols+" FROM environments WHERE id = ?", id)
	env.ExpandParamName()
	return env, err
}

func (r *EnvironmentRepository) GetByName(name string) (model.Environment, error) {
	var env model.Environment
	err := r.db.Get(&env, "SELECT "+envCols+" FROM environments WHERE name = ?", name)
	env.ExpandParamName()
	return env, err
}

func (r *EnvironmentRepository) Create(env model.Environment) error {
	env.SyncParamName()
	_, err := r.db.Exec(
		"INSERT INTO environments (id, name, is_prod, default_ref, param_name) VALUES (?, ?, ?, ?, ?)",
		env.ID, env.Name, env.IsProd, env.DefaultRef, env.ParamName,
	)
	return err
}

func (r *EnvironmentRepository) Update(id string, env model.Environment) error {
	env.SyncParamName()
	_, err := r.db.Exec(
		"UPDATE environments SET name=?, is_prod=?, default_ref=?, param_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		env.Name, env.IsProd, env.DefaultRef, env.ParamName, id,
	)
	return err
}

func (r *EnvironmentRepository) Delete(id string) error {
	res, err := r.db.Exec("DELETE FROM environments WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// HasJobs 判断某环境是否还有关联的 Jenkins 任务（用于删除保护）。
func (r *EnvironmentRepository) HasJobs(envID string) (bool, error) {
	var cnt int
	err := r.db.Get(&cnt, "SELECT COUNT(1) FROM jobs WHERE env_id = ?", envID)
	return cnt > 0, err
}

// UpdateOrder 按传入的 ID 顺序重排环境，在事务里一次性提交。
func (r *EnvironmentRepository) UpdateOrder(ids []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.Exec("UPDATE environments SET sort_order = ? WHERE id = ?", i+1, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// JobCounts 返回每个环境关联的任务数，用于环境编辑页展示改动影响面。
func (r *EnvironmentRepository) JobCounts() (map[string]int, error) {
	var rows []struct {
		EnvID string `db:"env_id"`
		Cnt   int    `db:"cnt"`
	}
	if err := r.db.Select(&rows, "SELECT env_id, COUNT(*) AS cnt FROM jobs GROUP BY env_id"); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.EnvID] = row.Cnt
	}
	return out, nil
}
