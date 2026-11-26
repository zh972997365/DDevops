package repository

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接（WAL 模式，单连接）。
type DB struct {
	db *sqlx.DB
}

func NewDB(dbPath string) (*DB, error) {
	db, err := sqlx.Open("sqlite", dbPath+"?_pragma=journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("创建表失败: %v", err)
	}

	return &DB{db: db}, nil
}

func createTables(db *sqlx.DB) error {
	// 注意：SQLite不支持在Exec中执行多条SQL语句，需要分开执行
	// 也不能使用 "--" 注释，会引发语法错误

	// 创建 jobs 表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id VARCHAR(50) PRIMARY KEY,
			job_path TEXT NOT NULL,
			server_id VARCHAR(50) DEFAULT '',
			env_id VARCHAR(50) DEFAULT '',
			short_code VARCHAR(10) DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建jobs表失败: %v", err)
	}

	// 创建 scheduled_tasks 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduled_tasks (
			id VARCHAR(50) PRIMARY KEY,
			task_id VARCHAR(50) NOT NULL,
			ref_value TEXT,
			schedule_time DATETIME NOT NULL,
			created_by VARCHAR(100) NOT NULL,
			created_name VARCHAR(100),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(20) DEFAULT 'pending',
			webhook_url TEXT,
			execute_time DATETIME,
			error_message TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("创建scheduled_tasks表失败: %v", err)
	}

	// 创建 publish_history 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS publish_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id VARCHAR(50) NOT NULL,
			env_id VARCHAR(50) DEFAULT '',
			ref_value TEXT NOT NULL,
			trigger_type VARCHAR(20) NOT NULL,
			trigger_by VARCHAR(100) NOT NULL,
			trigger_name VARCHAR(100),
			trigger_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			build_status VARCHAR(20) NOT NULL,
			build_url TEXT,
			error_message TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("创建publish_history表失败: %v", err)
	}

	// 创建 permissions 表（完整列，兼容从零建库；存量库由 MigratePermissions 补齐）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			user_id VARCHAR(100) PRIMARY KEY,
			username VARCHAR(100),
			user_name VARCHAR(100),
			password TEXT,
			web_role VARCHAR(20) DEFAULT 'user',
			allowed_jobs TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建permissions表失败: %v", err)
	}

	// 创建 sessions 表（登录会话持久化，重启与多副本下均有效）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			username VARCHAR(100) NOT NULL,
			name VARCHAR(100) DEFAULT '',
			role VARCHAR(20) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("创建sessions表失败: %v", err)
	}

	// 创建索引 - scheduled_tasks
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_status
		ON scheduled_tasks(status)
	`)
	if err != nil {
		return fmt.Errorf("创建索引idx_scheduled_tasks_status失败: %v", err)
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_schedule_time
		ON scheduled_tasks(schedule_time)
	`)
	if err != nil {
		return fmt.Errorf("创建索引idx_scheduled_tasks_schedule_time失败: %v", err)
	}

	// 创建索引 - publish_history
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_publish_history_trigger_time
		ON publish_history(trigger_time)
	`)
	if err != nil {
		return fmt.Errorf("创建索引idx_publish_history_trigger_time失败: %v", err)
	}

	// 创建索引 - publish_history 状态字段
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS idx_publish_history_build_status
	ON publish_history(build_status)
`)
	if err != nil {
		return fmt.Errorf("创建索引idx_publish_history_build_status失败: %v", err)
	}

	// 创建索引 - publish_history 任务ID字段（用于搜索）
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_publish_history_task_id
		ON publish_history(task_id)
	`)
	if err != nil {
		return fmt.Errorf("创建索引idx_publish_history_task_id失败: %v", err)
	}

	// env_id 索引在 MigrateHistoryEnv 中创建：存量库此时还没有该列，必须先迁移再加索引。

	// 创建 servers 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			base_url TEXT NOT NULL,
			user_name VARCHAR(100) NOT NULL,
			token TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建servers表失败: %v", err)
	}

	// 创建 dingtalk_users 表（缓存同步的钉钉员工）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS dingtalk_users (
			user_id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(100) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("创建dingtalk_users表失败: %v", err)
	}

	// 创建 dingtalk_sync_meta 表（记录最后同步时间）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS dingtalk_sync_meta (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			synced_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("创建dingtalk_sync_meta表失败: %v", err)
	}

	// 创建 environments 表（环境配置：测试/预发/生产等，集中承载构建规则）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS environments (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			is_prod INTEGER DEFAULT 0,
			default_ref VARCHAR(100) DEFAULT '',
			param_name VARCHAR(50) NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("创建environments表失败: %v", err)
	}

	return nil
}

func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Migrate 一次性执行全部结构/数据迁移。所有步骤均幂等，可重复执行。
// 单步失败只记录日志并继续，避免一处历史包袱阻塞启动。
func (d *DB) Migrate() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"jobs.server_id", d.MigrateJobs},
		{"servers.id", d.MigrateServerID},
		{"permissions", d.MigratePermissions},
		{"jobs.env", d.MigrateEnv},
		{"jobs.env_param", d.MigrateDropEnvParam},
		{"jobs.params", d.MigrateDropJobParams},
		{"environments", d.MigrateEnvironments},
		{"publish_history.env_id", d.MigrateHistoryEnv},
		{"sort_order", d.MigrateSortOrder},
		{"environments.param_name", d.MigrateEnvironmentParamName},
		{"legacy_columns", d.MigrateDropLegacyColumns},
		{"permissions.allowed_jobs", d.MigratePermissionAllowedJobs},
		{"jobs.short_code", d.MigrateShortCode},
	}

	var firstErr error
	for _, step := range steps {
		if err := step.fn(); err != nil {
			log.Printf("数据迁移[%s]失败: %v", step.name, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// MigrateShortCode 为 jobs 增加 short_code 列并回填存量任务。
// 幂等：列已存在则跳过加列；已有短码的任务不再重复分配。
func (d *DB) MigrateShortCode() error {
	var cnt int
	if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('jobs') WHERE name='short_code'"); err != nil {
		return err
	}
	if cnt == 0 {
		if _, err := d.db.Exec("ALTER TABLE jobs ADD COLUMN short_code VARCHAR(10) DEFAULT ''"); err != nil {
			return err
		}
	}
	if _, err := NewJobRepository(d).BackfillShortCodes(); err != nil {
		return fmt.Errorf("回填任务短码失败: %v", err)
	}
	return nil
}

// MigrateJobs 添加 server_id 列（兼容旧数据）。
func (d *DB) MigrateJobs() error {
	_, err := d.db.Exec("ALTER TABLE jobs ADD COLUMN server_id VARCHAR(50) DEFAULT ''")
	if err != nil {
		// 列可能已存在，忽略错误
	}
	_, err = d.db.Exec("UPDATE jobs SET server_id = '' WHERE server_id IS NULL OR server_id = ''")
	return err
}

// MigrateServerID 将旧 "default" 服务器 ID 迁移为 "jenkins"。
func (d *DB) MigrateServerID() error {
	_, err := d.db.Exec("UPDATE servers SET id = 'jenkins', name = 'Jenkins' WHERE id = 'default'")
	if err != nil {
		return err
	}
	_, err = d.db.Exec("UPDATE jobs SET server_id = 'jenkins' WHERE server_id = '' OR server_id = 'default'")
	return err
}

// MigratePermissions 为 permissions 表添加 password/web_role/username 列，删除 permission_lv。
func (d *DB) MigratePermissions() error {
	_, err := d.db.Exec("ALTER TABLE permissions ADD COLUMN password TEXT DEFAULT ''")
	if err != nil {
	}
	_, err = d.db.Exec("ALTER TABLE permissions ADD COLUMN web_role VARCHAR(20) DEFAULT 'user'")
	if err != nil {
	}
	_, err = d.db.Exec("ALTER TABLE permissions ADD COLUMN username VARCHAR(100) DEFAULT ''")
	if err != nil {
	}
	// 旧数据 username 为空时用 user_id 兜底
	_, err = d.db.Exec("UPDATE permissions SET username = user_id WHERE username IS NULL OR username = ''")
	if err != nil {
	}
	_, err = d.db.Exec("ALTER TABLE permissions DROP COLUMN permission_lv")
	if err != nil {
	}
	return nil
}

// MigratePermissionAllowedJobs 为老库 permissions 表添加 allowed_jobs 列（JSON 数组，空为 "[]"）。
func (d *DB) MigratePermissionAllowedJobs() error {
	_, err := d.db.Exec("ALTER TABLE permissions ADD COLUMN allowed_jobs TEXT DEFAULT '[]'")
	if err != nil {
		// 列可能已存在，忽略
	}
	return nil
}

// hasColumn 判断表是否存在指定列（表不存在也返回 false）。
func (d *DB) hasColumn(table, col string) (bool, error) {
	var cnt int
	if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('"+table+"') WHERE name=?", col); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// MigrateEnv 为 jobs 表添加 env 列，并根据 name 回填历史数据（test/pre/prod）。
// jobs.name 已在新版本删除，这里必须做列存在判断，否则新库直接报 no such column。
func (d *DB) MigrateEnv() error {
	_, err := d.db.Exec("ALTER TABLE jobs ADD COLUMN env VARCHAR(20) DEFAULT ''")
	if err != nil {
		// 列可能已存在，忽略错误
	}
	hasName, err := d.hasColumn("jobs", "name")
	if err != nil || !hasName {
		return nil
	}
	if _, err = d.db.Exec("UPDATE jobs SET env='test' WHERE name='测试环境' AND (env IS NULL OR env='')"); err != nil {
		return err
	}
	if _, err = d.db.Exec("UPDATE jobs SET env='pre' WHERE name='预发环境' AND (env IS NULL OR env='')"); err != nil {
		return err
	}
	if _, err = d.db.Exec("UPDATE jobs SET env='prod' WHERE name='生产环境' AND (env IS NULL OR env='')"); err != nil {
		return err
	}
	return nil
}

// MigrateDropEnvParam 删除 jobs 表中遗留的 env_param 死字段（构建触发从未使用）。
// 字段已从模型与读写中移除；此处仅对存量库做一次无害清理。
func (d *DB) MigrateDropEnvParam() error {
	var cnt int
	if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('jobs') WHERE name='env_param'"); err != nil {
		// 极旧驱动不支持 pragma 表函数时跳过，不影响主流程
		return nil
	}
	if cnt == 0 {
		return nil
	}
	if _, err := d.db.Exec("ALTER TABLE jobs DROP COLUMN env_param"); err != nil {
		// 个别 SQLite 版本不支持 DROP COLUMN，忽略，留空列无害
		return nil
	}
	return nil
}

// MigrateDropJobParams 删除 jobs 表中遗留的 branch_param / tag_param 死字段。
// 自 v1.3 起分支/标签参数名改由 environments 配置提供，任务不再存储；此处仅对存量库做一次无害清理。
func (d *DB) MigrateDropJobParams() error {
	for _, col := range []string{"branch_param", "tag_param"} {
		var cnt int
		if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('jobs') WHERE name=?", col); err != nil {
			// 极旧驱动不支持 pragma 表函数时跳过，不影响主流程
			return nil
		}
		if cnt == 0 {
			continue
		}
		if _, err := d.db.Exec("ALTER TABLE jobs DROP COLUMN " + col); err != nil {
			// 个别 SQLite 版本不支持 DROP COLUMN，忽略，留空列无害
		}
	}
	return nil
}

// dropEnvironmentKeyColumn 删除 v1.3 遗留的环境标识列（幂等）。
// 该列原本只用于把旧库的 jobs.env(test/pre/prod) 映射到环境，现改为按 id 拼接映射。
// 老库的 key 带 NOT NULL，若不删除，后续 INSERT 环境会因缺值直接失败。
func (d *DB) dropEnvironmentKeyColumn() error {
	var cnt int
	if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('environments') WHERE name='key'"); err != nil {
		return err
	}
	if cnt == 0 {
		return nil
	}
	if _, err := d.db.Exec("ALTER TABLE environments DROP COLUMN key"); err != nil {
		return fmt.Errorf("删除 environments.key 失败: %v", err)
	}
	return nil
}

// MigrateEnvironments 初始化 environments 表数据并完成 jobs.env_id 迁移。
//   - 删除 v1.3 遗留的 key 列（老库有、新库建表时已无），之后所有 SQL 都不再依赖它。
//   - 首次启动时预置 测试/预发/生产 三个环境（含默认分支与参数名）。
//   - 为 jobs 增加 env_id 列，并按旧 env 标识 / name 回填历史数据。
//   - 迁移完成后删除遗留的 env 列（SQLite 不支持则忽略，留空列无害）。
func (d *DB) MigrateEnvironments() error {
	// 必须在预置/回填之前删列：老库的 key 是 NOT NULL，不删会让后续 INSERT 失败
	if err := d.dropEnvironmentKeyColumn(); err != nil {
		return err
	}

	// 预置环境（仅当表为空时）
	var cnt int
	if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM environments"); err != nil {
		return err
	}
	if cnt == 0 {
		seeds := []struct {
			id, name, defRef, paramName string
			isProd                      bool
		}{
			{"env-test", "测试环境", "origin/test", "branch", false},
			{"env-pre", "预发环境", "origin/release", "release_branch", false},
			{"env-prod", "生产环境", "origin/master", "Tags", true},
		}
		for _, s := range seeds {
			if _, err := d.db.Exec(
				"INSERT INTO environments (id, name, is_prod, default_ref, param_name) VALUES (?, ?, ?, ?, ?)",
				s.id, s.name, s.isProd, s.defRef, s.paramName,
			); err != nil {
				return err
			}
		}
	}

	// jobs 增加 env_id 列（列可能已存在，忽略错误）
	if _, err := d.db.Exec("ALTER TABLE jobs ADD COLUMN env_id VARCHAR(50) DEFAULT ''"); err != nil {
	}

	// 旧库按 env 标识映射（仅当 env 列存在）。
	// 历史 env 值是 test/pre/prod，预置环境 id 固定为 env-test/env-pre/env-prod，
	// 所以用 id 拼接即可，不再依赖已删除的 key 列。
	var envCol int
	if err := d.db.Get(&envCol, "SELECT COUNT(1) FROM pragma_table_info('jobs') WHERE name='env'"); err == nil && envCol > 0 {
		if _, err := d.db.Exec(
			"UPDATE jobs SET env_id = (SELECT id FROM environments WHERE id = 'env-' || jobs.env) WHERE (env_id IS NULL OR env_id = '')",
		); err != nil {
			return err
		}
	}
	// 按 name 兜底映射（仅老库有 jobs.name 时执行；新库该列已删除）
	if hasName, err := d.hasColumn("jobs", "name"); err == nil && hasName {
		if _, err := d.db.Exec(
			"UPDATE jobs SET env_id = (SELECT id FROM environments WHERE name = jobs.name) WHERE (env_id IS NULL OR env_id = '')",
		); err != nil {
			return err
		}
	}

	// 删除遗留 env 列（SQLite 不支持则忽略）
	var colCnt int
	if err := d.db.Get(&colCnt, "SELECT COUNT(1) FROM pragma_table_info('jobs') WHERE name='env'"); err == nil && colCnt > 0 {
		if _, err := d.db.Exec("ALTER TABLE jobs DROP COLUMN env"); err != nil {
			// 忽略，留空列无害
		}
	}
	return nil
}

// MigrateHistoryEnv 为 publish_history 增加 env_id 列并回填存量数据。
// 优先按 task_id 关联 jobs.env_id；老库若还有 task_name 快照，再按环境名兜底。
// 该函数必须早于 MigrateDropLegacyColumns（后者会删除 task_name）。
func (d *DB) MigrateHistoryEnv() error {
	if _, err := d.db.Exec("ALTER TABLE publish_history ADD COLUMN env_id VARCHAR(50) DEFAULT ''"); err != nil {
		// 列已存在，忽略
	}
	if _, err := d.db.Exec(`
		UPDATE publish_history SET env_id = (SELECT env_id FROM jobs WHERE jobs.id = publish_history.task_id)
		WHERE (env_id IS NULL OR env_id = '') AND EXISTS (SELECT 1 FROM jobs WHERE jobs.id = publish_history.task_id)
	`); err != nil {
		return err
	}
	if hasTaskName, err := d.hasColumn("publish_history", "task_name"); err == nil && hasTaskName {
		if _, err := d.db.Exec(`
			UPDATE publish_history SET env_id = (SELECT id FROM environments WHERE environments.name = publish_history.task_name)
			WHERE (env_id IS NULL OR env_id = '') AND EXISTS (SELECT 1 FROM environments WHERE environments.name = publish_history.task_name)
		`); err != nil {
			return err
		}
	}
	// 列就绪后再建索引（存量库此前无 env_id，索引无法提前创建）。
	if _, err := d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_publish_history_env_id ON publish_history(env_id)
	`); err != nil {
		return fmt.Errorf("创建索引idx_publish_history_env_id失败: %v", err)
	}
	return nil
}

// MigrateSortOrder 给环境/服务表加排序字段，并按现有顺序做一次初始化。
// ALTER 在列已存在时会报错，忽略即可；所有步骤幂等。
func (d *DB) MigrateSortOrder() error {
	for _, table := range []string{"environments", "servers"} {
		_, _ = d.db.Exec("ALTER TABLE " + table + " ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0")
		// 仅在全部为 0（首次迁移）时按当前展示顺序编号，避免覆盖用户已保存的顺序
		var nonZero int
		if err := d.db.Get(&nonZero, "SELECT COUNT(*) FROM "+table+" WHERE sort_order <> 0"); err != nil {
			return err
		}
		if nonZero > 0 {
			continue
		}
		var ids []string
		if err := d.db.Select(&ids, "SELECT id FROM "+table+" ORDER BY created_at, id"); err != nil {
			return err
		}
		for i, id := range ids {
			if _, err := d.db.Exec("UPDATE "+table+" SET sort_order = ? WHERE id = ?", i+1, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// MigrateEnvironmentParamName 把成对的分支/Tag 参数名收敛成实际生效的单字段。
// 幂等：列已存在时忽略 ALTER；只在 param_name 全空时回填，不覆盖用户后续修改。
func (d *DB) MigrateEnvironmentParamName() error {
	_, _ = d.db.Exec("ALTER TABLE environments ADD COLUMN param_name VARCHAR(50) NOT NULL DEFAULT ''")

	var filled int
	if err := d.db.Get(&filled, "SELECT COUNT(*) FROM environments WHERE param_name <> ''"); err != nil {
		return err
	}
	if filled > 0 {
		return nil
	}
	// 老库从成对的旧参数名列读回自定义值；新库已不带这两列，直接给内置默认值。
	// 旧列在 MigrateDropLegacyColumns 中于本步骤之后删除。
	var legacy int
	if err := d.db.Get(&legacy, "SELECT COUNT(1) FROM pragma_table_info('environments') WHERE name='default_branch_param'"); err != nil {
		return err
	}
	if legacy == 0 {
		_, err := d.db.Exec(`UPDATE environments SET param_name = CASE
			WHEN is_prod = 1 THEN 'Tags' ELSE 'branch' END`)
		return err
	}
	_, err := d.db.Exec(`UPDATE environments SET param_name = CASE
		WHEN is_prod = 1 THEN COALESCE(NULLIF(default_tag_param, ''), 'Tags')
		ELSE COALESCE(NULLIF(default_branch_param, ''), 'branch') END`)
	return err
}

// MigrateDropLegacyColumns 删除历次重构遗留的列（幂等）。
// 全部放在迁移链最后，确保 Backfill/回填类步骤还能读到旧数据。
// jobs.name / publish_history.task_name / scheduled_tasks.task_name 都是「环境名快照」的
// 历史残留：环境一改名，这些副本就与实际配置对不上，展示时统一改用 env_id 实时取。
func (d *DB) MigrateDropLegacyColumns() error {
	// 先删引用废弃列的索引，否则 SQLite 的 DROP COLUMN 会直接失败。
	for _, idx := range []string{"idx_jobs_name", "idx_publish_history_task_name"} {
		_, _ = d.db.Exec("DROP INDEX IF EXISTS " + idx)
	}
	legacy := map[string][]string{
		"environments":    {"key", "default_branch_param", "default_tag_param"},
		"servers":         {"branch_param", "tag_param"},
		"jobs":            {"name"},
		"publish_history": {"task_name"},
		"scheduled_tasks": {"task_name"},
	}
	for table, cols := range legacy {
		for _, col := range cols {
			var cnt int
			if err := d.db.Get(&cnt, "SELECT COUNT(1) FROM pragma_table_info('"+table+"') WHERE name=?", col); err != nil {
				return err
			}
			if cnt == 0 {
				continue
			}
			if _, err := d.db.Exec("ALTER TABLE " + table + " DROP COLUMN " + col); err != nil {
				return fmt.Errorf("删除 %s.%s 失败: %v", table, col, err)
			}
		}
	}
	return nil
}
