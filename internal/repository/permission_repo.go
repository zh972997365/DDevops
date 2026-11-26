package repository

import (
	"encoding/json"
	"time"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// PermissionRepository 管理用户与权限（permissions 表）。
type PermissionRepository struct {
	db *sqlx.DB
}

func NewPermissionRepository(db *DB) *PermissionRepository {
	return &PermissionRepository{db: db.db}
}

const permCols = "user_id, username, user_name, password, web_role, allowed_jobs, created_at, updated_at"

// permRow 是 permissions 表行到 model 的中间结构：allowed_jobs 以 JSON 文本存储，
// 需手动解析为 model.Permission.AllowedJobs 的 []string。
type permRow struct {
	UserID         string    `db:"user_id"`
	Username       string    `db:"username"`
	UserName       string    `db:"user_name"`
	Password       string    `db:"password"`
	WebRole        string    `db:"web_role"`
	AllowedJobsRaw string    `db:"allowed_jobs"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func rowToPerm(r permRow) model.Permission {
	p := model.Permission{
		UserID:    r.UserID,
		Username:  r.Username,
		UserName:  r.UserName,
		Password:  r.Password,
		WebRole:   r.WebRole,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.AllowedJobsRaw != "" && r.AllowedJobsRaw != "null" {
		_ = json.Unmarshal([]byte(r.AllowedJobsRaw), &p.AllowedJobs)
	}
	if p.AllowedJobs == nil {
		p.AllowedJobs = []string{}
	}
	return p
}

func (r *PermissionRepository) List() ([]model.Permission, error) {
	var rows []permRow
	if err := r.db.Select(&rows, "SELECT "+permCols+" FROM permissions ORDER BY user_id"); err != nil {
		return nil, err
	}
	out := make([]model.Permission, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToPerm(row))
	}
	return out, nil
}

func (r *PermissionRepository) Get(userID string) (model.Permission, error) {
	var row permRow
	err := r.db.Get(&row, "SELECT "+permCols+" FROM permissions WHERE user_id = ?", userID)
	return rowToPerm(row), err
}

// GetByUsername 按登录用户名查询（登录名唯一）。
func (r *PermissionRepository) GetByUsername(username string) (model.Permission, error) {
	var row permRow
	err := r.db.Get(&row, "SELECT "+permCols+" FROM permissions WHERE username = ?", username)
	return rowToPerm(row), err
}

func (r *PermissionRepository) Create(perm model.Permission) error {
	aj, _ := json.Marshal(perm.AllowedJobs)
	if len(aj) == 0 {
		aj = []byte("[]")
	}
	_, err := r.db.Exec(
		"INSERT INTO permissions (user_id, username, user_name, password, web_role, allowed_jobs) VALUES (?, ?, ?, ?, ?, ?)",
		perm.UserID, perm.Username, perm.UserName, perm.Password, perm.WebRole, string(aj),
	)
	return err
}

func (r *PermissionRepository) Update(userID string, perm model.Permission) error {
	aj, _ := json.Marshal(perm.AllowedJobs)
	if len(aj) == 0 {
		aj = []byte("[]")
	}
	_, err := r.db.Exec(
		"UPDATE permissions SET username=?, user_name=?, password=?, web_role=?, allowed_jobs=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?",
		perm.Username, perm.UserName, perm.Password, perm.WebRole, string(aj), userID,
	)
	return err
}

func (r *PermissionRepository) Delete(userID string) error {
	_, err := r.db.Exec("DELETE FROM permissions WHERE user_id = ?", userID)
	return err
}

// UpdatePassword 单独更新密码摘要（用于历史明文密码的平滑升级）。
func (r *PermissionRepository) UpdatePassword(userID, hashed string) error {
	_, err := r.db.Exec(
		"UPDATE permissions SET password=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?",
		hashed, userID,
	)
	return err
}

// SeedAdmin 首次启动时若表为空则创建默认管理员。
// password 必须是已哈希的摘要，明文由调用方处理。
func (r *PermissionRepository) SeedAdmin(username, password string) error {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM permissions")
	if err != nil {
		return err
	}
	if count == 0 && username != "" && password != "" {
		return r.Create(model.Permission{
			UserID:      username,
			Username:    username,
			UserName:    "管理员",
			Password:    password,
			WebRole:     "admin",
			AllowedJobs: []string{},
		})
	}
	return nil
}
