package repository

import (
	"time"

	"ddevops/internal/model"

	"github.com/jmoiron/sqlx"
)

// SessionRepository 管理登录会话（sessions 表），使登录态不依赖进程内存。
type SessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db.db}
}

func (r *SessionRepository) Create(token, username, name, role string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO sessions (token, username, name, role, expires_at) VALUES (?, ?, ?, ?, ?)",
		token, username, name, role, expiresAt,
	)
	return err
}

func (r *SessionRepository) Get(token string) (model.Session, error) {
	var s model.Session
	err := r.db.Get(&s, "SELECT token, username, name, role, created_at, expires_at FROM sessions WHERE token = ?", token)
	return s, err
}

func (r *SessionRepository) Delete(token string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteByUsername 使某用户的所有会话失效（改密码、删账号时调用）。
func (r *SessionRepository) DeleteByUsername(username string) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE username = ?", username)
	return err
}

// DeleteExpired 清理过期会话，启动时调用一次即可。
func (r *SessionRepository) DeleteExpired() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}
