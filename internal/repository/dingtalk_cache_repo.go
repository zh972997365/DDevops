package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// DingTalkCacheRepository 缓存钉钉员工列表与同步时间（dingtalk_users / dingtalk_sync_meta 表）。
type DingTalkCacheRepository struct {
	db *sqlx.DB
}

func NewDingTalkCacheRepository(db *DB) *DingTalkCacheRepository {
	return &DingTalkCacheRepository{db: db.db}
}

// SaveAll 覆盖保存员工列表并记录同步时间。
func (r *DingTalkCacheRepository) SaveAll(users map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM dingtalk_users"); err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO dingtalk_users (user_id, name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for id, name := range users {
		if _, err := stmt.Exec(id, name); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(
		"INSERT INTO dingtalk_sync_meta (id, synced_at) VALUES (1, CURRENT_TIMESTAMP) " +
			"ON CONFLICT(id) DO UPDATE SET synced_at=CURRENT_TIMESTAMP"); err != nil {
		return err
	}
	return tx.Commit()
}

// LoadAll 加载缓存的员工列表及同步时间。
func (r *DingTalkCacheRepository) LoadAll() (map[string]string, time.Time, error) {
	users := make(map[string]string)
	rows, err := r.db.Query("SELECT user_id, name FROM dingtalk_users")
	if err != nil {
		return users, time.Time{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err == nil {
			users[id] = name
		}
	}
	var syncedAt time.Time
	_ = r.db.Get(&syncedAt, "SELECT synced_at FROM dingtalk_sync_meta WHERE id = 1")
	return users, syncedAt, nil
}
