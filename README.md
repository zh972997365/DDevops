# DDevops · Jenkins 钉钉发布平台

Go 单进程应用，把 Jenkins 构建收敛为统一发布入口：Web 后台、钉钉机器人、定时任务共用一套构建逻辑与权限规则，前端 Vue3 产物通过 `go:embed` 内嵌进二进制。

## 项目介绍

- **一个构建入口**：Web 手动构建、钉钉指令、定时任务都走 `BuildService`，引用归一、参数选择与历史写入只有一份实现。
- **环境即规则**：测试/预发/生产在「环境管理」中定义分支或 Tag 构建方式、默认引用和 Jenkins 参数名，任务只关联环境 ID。
- **权限一致**：Web 与钉钉共用 `jobScopeFor`，管理员全量，普通用户仅限「可构建项目」。
- **单进程运行**：Web（`8081`）、健康检查（`8080`）、钉钉 Stream、定时调度器同进程启动，无需 Docker 或独立前端服务。

## 技术栈

| 层次 | 技术 |
| --- | --- |
| 后端 | Go 1.25、Gin、sqlx、modernc.org/sqlite（纯 Go、WAL、单连接）、bcrypt |
| 前端 | Vue 3.5、Vue Router 4.5、Pinia 2.3、Vite 5、Tailwind 3.4，无 UI 组件库 |
| 集成 | Jenkins REST API（多实例）、钉钉 Stream SDK、钉钉通讯录 |
| 部署 | 单进程 + `go:embed` 内嵌前端，SQLite 单文件存储 |

仓库未发现 Dockerfile、Makefile、CI 配置与 OpenAPI 文档；接口按 `internal/router` 与 `internal/handler` 实现整理。

## 快速开始

**环境要求**：Go 1.25+；Node.js 18+（仅修改前端时需要）；可访问的 Jenkins 与钉钉机器人（可选）。

```bash
cp .env.example .env      # 填写 Jenkins / 钉钉凭据

cd frontend && npm install && npm run build && cd ..
go run .                  # 或 go build -o ddevops . && ./ddevops
```

| 配置 | 说明 |
| --- | --- |
| `SQLITE_DB_PATH` | SQLite 路径，常用 `data/jenkins.db`（目录需存在） |
| `JENKINS_BASE_URL` / `_USER` / `_TOKEN` | 默认实例 `jenkins` |
| `JENKINS2_*` | 可选第二实例 `jenkins2` |
| `DINGTALK_CLIENT_ID` / `_SECRET` | 留空则不启动钉钉机器人 |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | 仅首次建库时写入，默认 `admin` / `admin123` |
| `WEB_PORT` / `HEALTH_PORT` | 默认 `8081` / `8080` |

访问：Web `http://localhost:8081`，健康检查 `http://localhost:8080/health`。

前端改动必须重新 `npm run build` 并重启 Go 进程；仓库已包含构建产物，仅运行可直接 `go run .`。

## 功能概括

| 页面 | 能力 |
| --- | --- |
| 概括总览 | 今日发布、14 日成功率、运行中、待执行；发布趋势、定时队列、环境矩阵、最近发布、失败待办 |
| 任务管理 | 关键词/环境/服务筛选、排序、分页、批量复制 ID、导出 CSV、详情抽屉、构建、编辑、删除 |
| 任务表单 | 任务 ID、Jenkins 路径、所属服务、环境；钉钉短码自动生成 |
| 定时任务 | 创建（任务搜索、分支/Tag、时间）、状态筛选、倒计时、取消、删除 |
| 发布历史 | 统计卡、任务/环境/状态/日期筛选、分页、失败原因、导出 CSV、构建链接 |
| 环境管理（管理员） | 分支/Tag 规则、默认引用、参数名、拖拽排序、关联任务数与影响提示 |
| 服务管理（管理员） | 多 Jenkins 实例、Token 隐藏、拖拽排序、默认实例不可删 |
| 用户管理（管理员） | 用户 CRUD、重置密码、钉钉选人/同步、可构建项目授权 |

钉钉群 `@机器人` 指令：

| 指令 | 说明 |
| --- | --- |
| `帮助` | 使用帮助卡片 |
| `测试` / `预发` / `生产` | 按环境列出可见任务 |
| `<任务ID或短码> [分支或Tag]` | 立即构建；生产必须给 Tag |
| `定时(YYYYMMDDHHmm) <任务ID或短码> [分支或Tag]` | 创建定时发布 |
| `定时列表` | 自己创建的待执行任务 |
| `取消定时 <任务ID或短码>` | 仅能取消自己创建的 |

- **构建规则**：Tag 模式必须显式填写，空值或非法字符直接拒绝；分支模式留空回落默认引用，未带 `origin/` 自动补齐。调用 `buildWithParameters`，生产 Tag 下拉来自 Jenkins Git Parameter（仅展示最近 10 个）。
- **定时调度**：每分钟扫描 `pending`，到点转 `executing`，结束后转 `completed` / `failed` 并写入历史；同一任务只允许一个待执行计划。
- **权限**：只有 `admin` 和普通用户；普通用户按 `allowed_jobs` 限定任务，空数组即无授权；发布历史普通用户只看自己的记录。
- **界面**：浅/深主题、舒适/紧凑密度、`⌘K` 命令面板、通知面板、响应式布局。

## API 接口

- 基础路径 `/api`；除登录外都需要请求头 `Authorization: <token>`（原始 token，非 Bearer）。
- 管理员接口额外校验角色；错误统一返回 `{"error": "说明"}`。
- 任务、定时、历史对象的字段与 `internal/model/model.go` 一致，下表省略。

| 方法与路径 | 权限 | 参数 → 返回 |
| --- | --- | --- |
| `POST /api/login` | 公开 | `{username,password}` → `{token,role}` |
| `POST /api/logout` | 登录 | — → `{message}` |
| `GET /api/jobs` | 登录 | — → 授权范围内的任务数组 |
| `POST /api/jobs` | 管理员 | `{id,jobPath,serverId,envId}` → `201` |
| `PUT /api/jobs/:id` | 管理员 | 同创建；`id` 可改名 |
| `DELETE /api/jobs/:id` | 管理员 | — → `{message}` |
| `GET /api/jobs/:id/git-tags` | 登录 + 授权 | — → `{tags,available,reason?}` |
| `POST /api/jobs/:id/build` | 登录 + 授权 | `{refValue}` → `{message,buildUrl}` |
| `GET /api/tasks` | 登录 | `?status=` → 定时任务数组 |
| `POST /api/tasks` | 登录 + 授权 | `{taskId,refValue,scheduleTime}`，`YYYY-MM-DD HH:mm` |
| `POST /api/tasks/:id/cancel` | 创建者 / 管理员 | — → `{message}` |
| `DELETE /api/tasks/:id` | 管理员 | — → `{message}` |
| `GET /api/history` | 登录 | `page,size,envId,status,search,from,to` → `{data,total,page,size}` |
| `GET /api/history/stats` | 登录 | — → `{total,success,failed,today,rate}`，`rate` 为 0–1 |
| `GET /api/overview` | 登录 | — → `{jobs,today,running,pending,latestBuilds}` |
| `GET /api/environments` | 登录 | — → 环境数组，含 `jobCount` |
| `POST /api/environments` | 管理员 | `{name,isProd,defaultRef,defaultBranchParam,defaultTagParam}` |
| `PUT /api/environments/:id` | 管理员 | 同上 → `{message}` |
| `DELETE /api/environments/:id` | 管理员 | 有关联任务时返回 `400` |
| `PUT /api/environments/reorder` | 管理员 | `{ids:[...]}` 全量顺序 |
| `GET /api/servers` | 管理员 | — → 服务数组（不含 Token） |
| `GET /api/servers/options` | 登录 | — → `[{id,name}]` |
| `POST /api/servers` | 管理员 | `{id,name,baseUrl,user,token}` |
| `PUT /api/servers/:id` | 管理员 | 同上；Token 留空保持原值 |
| `DELETE /api/servers/:id` | 管理员 | 默认实例 `jenkins` 不可删 |
| `PUT /api/servers/reorder` | 管理员 | `{ids:[...]}` 全量顺序 |
| `GET /api/permissions` | 管理员 | — → 用户数组（不含密码） |
| `POST /api/permissions` | 管理员 | `{userId,username,userName,password,webRole,allowedJobs}` |
| `PUT /api/permissions/:userId` | 管理员 | 同上；密码留空不改 |
| `DELETE /api/permissions/:userId` | 管理员 | `admin` 账号不可删除 |
| `GET /api/dingtalk/users` | 管理员 | `?name=` → `[{userid,name,pinyin}]` |
| `POST /api/dingtalk/sync` | 管理员 | — → `{message,count}` |
| `GET /health` | 公开 | 独立端口 `8080`，返回纯文本 `OK` |

## 项目结构

```text
DDevops/
├── main.go                 # 初始化 DB、仓储、服务与路由
├── internal/
│   ├── config/             # .env 配置
│   ├── model/              # 领域模型、BuildRule
│   ├── repository/         # SQLite 访问、建表与幂等迁移
│   ├── service/            # Jenkins、BuildService、调度器、钉钉
│   ├── handler/            # HTTP API、钉钉消息与卡片
│   ├── middleware/         # 登录 / 管理员校验
│   ├── router/             # 路由、SPA 托管、健康检查
│   └── webui/              # go:embed 内嵌前端产物
├── frontend/
│   ├── DESIGN.md           # 前端设计规范
│   └── src/
│       ├── views/          # 9 个页面
│       ├── components/     # base / data / feature / layout / ui
│       ├── api/ stores/ composables/ utils/ constants/ styles/
│       └── router/
└── data/                   # SQLite 数据文件
```

数据库共 9 张表：`jobs`、`environments`、`servers`、`scheduled_tasks`、`publish_history`、`permissions`、`sessions`、`dingtalk_users`、`dingtalk_sync_meta`。