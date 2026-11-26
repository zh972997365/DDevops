package repository

import (
	"database/sql"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// ServerRepository 管理多 Jenkins 服务（servers 表）。
type ServerRepository struct {
	db *sqlx.DB
}

func NewServerRepository(db *DB) *ServerRepository {
	return &ServerRepository{db: db.db}
}

// 显式列清单：只查模型真正需要的列，后续加列也不会影响读取。
const serverCols = `id, name, base_url, user_name, token, sort_order, created_at, updated_at`

func (r *ServerRepository) List() ([]model.JenkinsServer, error) {
	var servers []model.JenkinsServer
	err := r.db.Select(&servers, "SELECT "+serverCols+" FROM servers ORDER BY sort_order, name")
	return servers, err
}

func (r *ServerRepository) Get(id string) (model.JenkinsServer, error) {
	var server model.JenkinsServer
	err := r.db.Get(&server, "SELECT "+serverCols+" FROM servers WHERE id = ?", id)
	return server, err
}

func (r *ServerRepository) Create(server model.JenkinsServer) error {
	_, err := r.db.Exec(
		"INSERT INTO servers (id, name, base_url, user_name, token) VALUES (?, ?, ?, ?, ?)",
		server.ID, server.Name, server.BaseURL, server.User, server.Token,
	)
	return err
}

// Update 更新服务配置。token 传空表示「不修改」——列表接口从不返回 token，
// 编辑表单里也永远是空的，若直接覆盖会把已保存的凭据清空，导致 Jenkins 调用全部失败。
// 同时不允许改写主键 id（过去这里写的是 SET id=?，一旦前端漏传就会把 id 置空）。
func (r *ServerRepository) Update(id string, server model.JenkinsServer) error {
	var (
		res sql.Result
		err error
	)
	if server.Token == "" {
		res, err = r.db.Exec(
			"UPDATE servers SET name=?, base_url=?, user_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			server.Name, server.BaseURL, server.User, id,
		)
	} else {
		res, err = r.db.Exec(
			"UPDATE servers SET name=?, base_url=?, user_name=?, token=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			server.Name, server.BaseURL, server.User, server.Token, id,
		)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ServerRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM servers WHERE id = ?", id)
	return err
}

// SeedDefaultServer 从环境变量写入默认 Jenkins 服务器（不覆盖已存在的）。
func (r *ServerRepository) SeedDefaultServer(baseURL, user, token string) error {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM servers WHERE id = ?", "jenkins")
	if err != nil {
		return err
	}
	if count == 0 && baseURL != "" {
		_, err = r.db.Exec(
			"INSERT INTO servers (id, name, base_url, user_name, token) VALUES (?, ?, ?, ?, ?)",
			"jenkins", "Jenkins", baseURL, user, token,
		)
	}
	return err
}

// SeedServer 按需写入第二套 Jenkins 服务器（不覆盖已存在的）。
func (r *ServerRepository) SeedServer(id, name, baseURL, user, token string) error {
	if baseURL == "" {
		return nil
	}
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM servers WHERE id = ?", id)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = r.db.Exec(
			"INSERT INTO servers (id, name, base_url, user_name, token) VALUES (?, ?, ?, ?, ?)",
			id, name, baseURL, user, token,
		)
	}
	return err
}

// UpdateOrder 按传入的 ID 顺序重排服务，在事务里一次性提交。
func (r *ServerRepository) UpdateOrder(ids []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.Exec("UPDATE servers SET sort_order = ? WHERE id = ?", i+1, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
