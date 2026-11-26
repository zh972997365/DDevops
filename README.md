# DDevops · Jenkins 钉钉自动化发布平台

一个 Go 单进程应用：同时提供 Web 管理后台、钉钉机器人、定时发布调度与健康检查。三个触发入口共用同一套构建服务与权限规则，前端 Vue3 产物在编译期通过 `go:embed` 内嵌进二进制，运行时不依赖独立的前端服务或容器。

## 项目介绍

DDevops 把 Jenkins 的参数化构建收敛为统一的发布入口，解决三类问题：

- **发布入口分散**：Web 手动构建、钉钉群指令、定时任务三条链路，统一走 `BuildService`，构建参数、引用归一与历史写入只有一份实现。
- **构建规则硬编码**：测试 / 预发 / 生产等环境在「环境管理」中集中配置，环境决定使用分支参数还是版本 Tag 参数，任务只关联环境 ID，不再各自判断。
- **权限口径不一致**：Web 与钉钉共用 `jobScopeFor` 计算任务范围，管理员全量、普通用户仅限「可构建项目」。

同时支持多个 Jenkins 实例（服务管理）、发布历史与统计、钉钉通讯录选人，以及会话持久化（重启不掉线）。

运行形态：一个进程内启动 Web 服务（默认 `8081`）、健康检查（默认 `8080`）、钉钉 Stream 长连接、定时调度器四个 goroutine。

## 技术栈

| 层次 | 技术 | 版本 | 说明 |
| --- | --- | --- | --- |
| 语言 | Go | 1.25.0 | 单进程服务端 |
| Web 框架 | Gin | v1.12.0 | HTTP 路由与中间件 |
| 数据库访问 | sqlx | v1.4.0 | 结构体映射与事务 |
| 数据库驱动 | modernc.org/sqlite | v1.46.2 | 纯 Go SQLite，无需 CGO；启用 WAL，单连接 |
| 认证 | golang.org/x/crypto | v0.48.0 | bcrypt 密码摘要 |
| 钉钉 | open-dingtalk/dingtalk-stream-sdk-go | v0.9.1 | Stream 模式机器人，无需公网回调地址 |
| 其他后端依赖 | gin-contrib/cors、google/uuid、joho/godotenv、mozillazg/go-pinyin | — | CORS、UUID、`.env` 加载、姓名转拼音 |
| 前端框架 | Vue | ^3.5.13 | Composition API，无 UI 组件库 |
| 路由 / 状态 | Vue Router / Pinia | ^4.5.0 / ^2.3.0 | history 路由、登录态与界面偏好 |
| 构建 | Vite | ^5.4.11 | 产物输出到 `internal/webui/dist` |
| 样式 | Tailwind CSS | ^3.4.17 | 设计令牌见 `frontend/src/styles/tokens.css` |
| 图标 | lucide-vue-next | ^1.0.0 | 线性图标 |

仓库中**未发现** Dockerfile、Makefile、CI 配置与 OpenAPI/Swagger 文档；接口说明依据 `internal/router` 与 `internal/handler` 的实际实现整理。

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 18+（仅在需要重新构建前端时使用；仓库已包含构建好的 `internal/webui/dist`）
- 可访问的 Jenkins 服务及 API Token（可选，缺少时无法触发构建）
- 钉钉企业内部机器人，Stream 模式（可选，未配置时钉钉服务不启动）

### 配置

复制示例配置并按实际环境填写：

```bash
cp .env.example .env
```

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SQLITE_DB_PATH` | `jenkins.db` | SQLite 文件路径；仓库内常用 `data/jenkins.db`，使用前确保目录存在 |
| `JENKINS_BASE_URL` / `JENKINS_USER` / `JENKINS_TOKEN` | 空 | 默认 Jenkins 实例，启动时自动注册为 ID `jenkins` |
| `JENKINS2_BASE_URL` / `JENKINS2_USER` / `JENKINS2_TOKEN` | 空 | 可选的第二个实例，注册为 ID `jenkins2` |
| `DINGTALK_CLIENT_ID` / `DINGTALK_CLIENT_SECRET` | 空 | 钉钉机器人凭据；任一为空时钉钉长连接不启动 |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `admin123` | 仅在 `permissions` 表为空时写入初始管理员 |
| `WEB_PORT` | `8081` | Web 后台端口 |
| `HEALTH_PORT` | `8080` | 健康检查端口 |

`.env` 内含凭据，已被 `.gitignore` 排除，不要提交。

### 安装与启动

前端产物是编译期嵌入的，完整流程如下：

```bash
# 1. 构建前端（输出到 internal/webui/dist）
cd frontend
npm install
npm run build
cd ..

# 2. 运行（单进程，无需单独启动前端）
go run .

# 也可以编译后运行
go build -o ddevops .
./ddevops
```

若只运行仓库中已有的前端产物，可跳过第 1 步直接 `go run .`。

### 开发模式

后端保持运行，另开一个终端启动 Vite；`/api` 已代理到 `http://localhost:8081`：

```bash
cd frontend && npm run dev
```

Vite 仅用于开发调试，正式运行始终由 Go 进程提供内嵌静态资源。

### 访问

- Web 后台：<http://localhost:8081>
- 健康检查：<http://localhost:8080/health>（返回纯文本 `OK`）
- 初始账号：`admin` / `admin123`（首次登录后请修改密码）

> 任何 `frontend/` 改动都必须重新 `npm run build` 并重启 Go 进程，否则运行的仍是旧界面。

## 功能概括

### Web 管理后台

| 页面 | 路径 | 主要能力 |
| --- | --- | --- |
| 概括总览 | `/` | 今日发布、近 14 日成功率、运行中、待执行；14 日发布趋势；定时队列；环境矩阵（关联任务数）；最近发布；近 14 日失败待接手 |
| 任务管理 | `/jobs` | 关键词 / 环境 / 服务筛选、排序、分页、行选择（复制 ID、导出 CSV）、详情抽屉、构建 / 编辑 / 删除、最近构建信息 |
| 新建 / 编辑任务 | `/jobs/new`、`/jobs/:id/edit` | 任务 ID、Jenkins 路径、所属服务、环境；钉钉短码由后端自动生成 |
| 定时任务 | `/tasks` | 新建（任务搜索、分支/Tag、执行时间）、状态筛选、待执行倒计时、取消、删除、详情 |
| 发布历史 | `/history` | 总数 / 成功 / 失败 / 今日统计；任务、环境、状态、日期筛选；分页；失败原因；导出 CSV；复制构建链接 |
| 环境管理（管理员） | `/environments` | 创建 / 编辑 / 删除、分支或 Tag 构建方式、默认引用、参数名、拖拽排序、关联任务数提示 |
| 服务管理（管理员） | `/servers` | 多 Jenkins 实例 CRUD、Token 隐藏、拖拽排序、默认实例 `jenkins` 不可删除、保存后热更新客户端 |
| 用户管理（管理员） | `/permissions` | 用户 CRUD、重置密码、角色设置、钉钉选人搜索与通讯录同步、可构建项目分配 |

界面能力：浅色 / 深色主题、舒适 / 紧凑密度、标准 / 增强动效、`⌘K` 命令面板、通知面板（今日失败与待执行计划）、侧栏折叠、响应式布局，以及统一的空态、加载骨架与错误重试。

### 钉钉机器人（Stream 长连接）

在群里 `@机器人` 使用，所有指令先做权限校验：

| 指令 | 说明 |
| --- | --- |
| `帮助` / `help` | 返回使用帮助卡片 |
| `测试` / `预发` / `生产` | 按环境列出当前用户可见的任务，一个任务一行 |
| `<任务ID或短码> [分支或Tag]` | 立即构建；分支构建省略引用时回落环境默认分支，生产环境必须提供版本 Tag |
| `定时(YYYYMMDDHHmm) <任务ID或短码> [分支或Tag]` | 创建定时发布，例：`定时(202611051800) pop-admin 20261105u01` |
| `定时列表` | 列出自己创建且授权范围内的待执行定时任务 |
| `取消定时 <任务ID或短码>` | 仅能取消自己创建的定时任务 |

构建结果以 ActionCard 回执，环境用颜色区分（测试蓝 / 预发橙 / 生产绿）；失败时返回 Jenkins 错误摘要。

### 定时调度

- 启动时检查一次，之后每分钟扫描 `pending` 任务；到点改为 `executing`，调用 `BuildService`，结束后改为 `completed` 或 `failed` 并写入发布历史。
- 状态：`pending`、`executing`、`completed`、`failed`、`cancelled`。
- 同一任务同时只允许一个待执行计划；执行时间不得早于当前时间。

### 构建规则

构建规则由环境配置派生（`Environment.Rule()`）：

| 模式 | 引用要求 | 默认引用 | 传给 Jenkins 的参数名 |
| --- | --- | --- | --- |
| Tag 构建（`isProd=true`） | 必须显式提供，空值或非法字符直接拒绝 | 无回退 | 环境的 `param_name`，未配置时 `Tags` |
| 分支构建（`isProd=false`） | 可省略，省略时使用默认引用；未带 `origin/` 前缀会自动补齐 | `default_ref`，未配置时 `origin/test` | 环境的 `param_name`，未配置时 `branch` |

- 调用 Jenkins 使用 `buildWithParameters`；服务地址与任务路径会做斜杠归一。
- 网络错误最多重试 3 次；Jenkins 返回 5xx 时会尝试补齐任务的全部参数后重试一次，并从错误页提取可读描述。
- 生产环境版本 Tag 下拉来自 Jenkins 任务自身的 Git Parameter（`api/json` 的 `allValueItems`），前端展示最近 10 个、最新在最上方；该请求只读，不会触发构建。

### 权限模型

- 角色只有 `admin` 与普通用户；管理员可进入环境 / 服务 / 用户管理，且不受任务范围限制。
- 普通用户通过 `allowed_jobs`（JSON 数组）限定可构建与可见的任务；空数组表示**没有任何授权任务**，不再兼容「空列表即全量」。
- Web 使用登录账号（`username`），钉钉使用工号（`user_id`）查询同一张 `permissions` 表，两端共用 `jobScopeFor`。
- 发布历史：管理员看全部（含执行人）；普通用户只看自己触发的记录。定时任务：管理员看全部，普通用户只看自己创建的。
- 密码使用 bcrypt；历史明文密码在首次成功登录后自动升级为摘要。

### 数据库

SQLite 单文件，启动时建表并执行幂等迁移，共 9 张表：

`jobs`（任务）、`environments`（环境与构建规则）、`servers`（Jenkins 实例）、`scheduled_tasks`（定时任务）、`publish_history`（发布历史）、`permissions`（用户与授权）、`sessions`（登录会话）、`dingtalk_users`（通讯录缓存）、`dingtalk_sync_meta`（同步时间）。

## API 接口

### 通用约定

- 基础路径：`/api`；请求与响应均为 JSON（健康检查除外）。
- 登录后需在请求头携带**原始 token**：`Authorization: <token>`，不是 `Bearer <token>`。
- 登录接口之外的所有 `/api` 路由都要求登录；管理员接口额外校验角色，失败返回 `403`。
- 错误响应统一为 `{"error": "说明"}`。
- 时间字段为本地时间的 RFC3339 / `YYYY-MM-DD HH:mm` 字符串（按接口标注）。

| 状态码 | 含义 |
| --- | --- |
| `200` | 查询或更新成功 |
| `201` | 创建成功 |
| `400` | 参数错误、业务约束不满足（时间早于当前、重复定时、环境仍被任务引用等） |
| `401` | 未登录或会话过期 |
| `403` | 权限不足（非管理员或任务不在授权范围） |
| `404` | 资源不存在 |
| `500` | 服务端错误 |

### 认证

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `POST` | `/api/login` | 公开 | `{"username":"...","password":"..."}` | `{"token":"...","role":"admin\|user"}` |
| `POST` | `/api/logout` | 登录 | 无 | `{"message":"已退出"}` |

### 任务

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/jobs` | 登录 | 无 | 任务数组，普通用户已按授权过滤 |
| `GET` | `/api/jobs/:id/git-tags` | 登录 + 任务授权 | 路径参数 `id` | `{"tags":[],"available":true,"source":"jenkins"}`；不可用时返回 `available:false` 与 `reason` |
| `POST` | `/api/jobs/:id/build` | 登录 + 任务授权 | `{"refValue":"分支或Tag"}` | `{"message":"构建已触发","buildUrl":"..."}` |
| `POST` | `/api/jobs` | 管理员 | `{"id","jobPath","serverId","envId"}` | `201 {"message":"创建成功"}`，短码自动生成 |
| `PUT` | `/api/jobs/:id` | 管理员 | 同上，`id` 可改为新任务名 | `{"message":"..."}`；改名会同步定时任务与历史 |
| `DELETE` | `/api/jobs/:id` | 管理员 | 无 | `{"message":"删除成功"}` |

任务对象字段：`id`、`jobPath`、`serverId`、`envId`、`shortCode`、`createdAt`、`updatedAt`。短码按环境生成：生产 `P`、预发 `S`、其余 `T`，后接两位序号（如 `P01`）。

### 定时任务

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/tasks` | 登录 | 查询参数 `status`（可选，`pending` / `executing` / `completed` / `failed` / `cancelled`） | 定时任务数组 |
| `POST` | `/api/tasks` | 登录 + 任务授权 | `{"taskId":"...","refValue":"...","scheduleTime":"YYYY-MM-DD HH:mm"}` | `201 {"message":"定时任务创建成功"}` |
| `POST` | `/api/tasks/:id/cancel` | 创建者本人或管理员 | 无 | `{"message":"已取消"}` |
| `DELETE` | `/api/tasks/:id` | 管理员 | 无 | `{"message":"删除成功"}` |

定时任务对象字段：`id`、`taskId`、`refValue`、`scheduleTime`、`createdBy`、`createdName`、`createdAt`、`status`、`webhookUrl`、`executeTime`、`errorMessage`。

### 发布历史与总览

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/history` | 登录 | `page`（默认 1）、`size`（默认 20，最大 200）、`envId`、`status`（`success` / `failed`）、`search`（任务 ID 模糊匹配）、`from` / `to`（`YYYY-MM-DD`） | `{"data":[],"total":0,"page":1,"size":20}` |
| `GET` | `/api/history/stats` | 登录 | 无 | `{"total":0,"success":0,"failed":0,"today":0,"rate":0}`（近一年） |
| `GET` | `/api/overview` | 登录 | 无 | `{"jobs":{"total":0},"today":{"builds":0,"success":0,"rate":0},"running":0,"pending":0,"latestBuilds":{}}` |

补充说明：

- `/api/history` 仍接受旧参数 `env`（环境名），服务端会换算为 `env_id`；结果固定只查近一年，按触发时间倒序。
- `stats` 与 `overview` 中的 `rate` 是 0–1 的比例，由前端换算为百分比展示。
- `latestBuilds` 以任务 ID 为键，值为 `{"status","time","url"}`。
- 历史记录字段：`id`、`taskId`、`envId`、`refValue`、`triggerType`（`manual` / `scheduled`）、`triggerBy`、`triggerName`、`triggerTime`、`buildStatus`、`buildUrl`、`errorMessage`。

### 环境

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/environments` | 登录 | 无 | 环境数组，每项额外包含 `jobCount` |
| `POST` | `/api/environments` | 管理员 | `{"name","isProd","defaultRef","defaultBranchParam","defaultTagParam"}` | `201 {"message":"创建成功"}` |
| `PUT` | `/api/environments/:id` | 管理员 | 同上 | `{"message":"更新成功"}` |
| `PUT` | `/api/environments/reorder` | 管理员 | `{"ids":["..."]}`，必须与现有环境集合完全一致 | `{"message":"顺序已保存"}` |
| `DELETE` | `/api/environments/:id` | 管理员 | 无 | 仍有关联任务时返回 `400` |

环境对象字段：`id`、`name`、`isProd`、`defaultRef`、`defaultBranchParam`、`defaultTagParam`、`sortOrder`、`createdAt`、`updatedAt`、`jobCount`。实际生效的参数名只保存一份（`param_name`）：Tag 模式取 `defaultTagParam`，分支模式取 `defaultBranchParam`；参数名只允许字母、数字、下划线且不能以数字开头。

首次建库会预置三个环境：

| 名称 | 构建方式 | 默认引用 | 参数名 |
| --- | --- | --- | --- |
| 测试环境 | 分支 | `origin/test` | `branch` |
| 预发环境 | 分支 | `origin/release` | `release_branch` |
| 生产环境 | Tag | `origin/master` | `Tags` |

### Jenkins 服务

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/servers` | 管理员 | 无 | 服务数组，不返回 `token` |
| `GET` | `/api/servers/options` | 登录 | 无 | `[{"id","name"}]` |
| `POST` | `/api/servers` | 管理员 | `{"id","name","baseUrl","user","token"}` | `201 {"message":"创建成功"}` |
| `PUT` | `/api/servers/:id` | 管理员 | 同上；`token` 留空表示保持原值 | `{"message":"更新成功"}` |
| `PUT` | `/api/servers/reorder` | 管理员 | `{"ids":["..."]}` | `{"message":"顺序已保存"}` |
| `DELETE` | `/api/servers/:id` | 管理员 | 无 | ID 为 `jenkins` 的默认实例不可删除 |

服务对象字段：`id`、`name`、`baseUrl`、`user`、`sortOrder`、`createdAt`。

### 用户与权限

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/permissions` | 管理员 | 无 | 用户数组（不返回密码摘要） |
| `POST` | `/api/permissions` | 管理员 | `{"userId","username","userName","password","webRole","allowedJobs":[]}` | `201 {"message":"创建成功"}` |
| `PUT` | `/api/permissions/:userId` | 管理员 | 同上；`password` 留空不改，`allowedJobs` 缺省沿用旧值 | `{"message":"更新成功"}` |
| `DELETE` | `/api/permissions/:userId` | 管理员 | 无 | `admin` 账号不可删除 |

- `userId` 同时是钉钉工号；`username` 是 Web 登录名。
- `webRole` 取值 `admin` / `user`，缺省为 `user`。
- 修改密码或删除用户会使该用户的全部会话立即失效。

### 钉钉通讯录（管理员）

| 方法 | 路径 | 权限 | 参数 | 返回 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/dingtalk/users` | 管理员 | `name`（姓名、拼音或用户 ID 模糊搜索） | `[{"userid","name","pinyin"}]` |
| `POST` | `/api/dingtalk/sync` | 管理员 | 无 | `{"message":"同步成功","count":0}` |

### 健康检查

| 方法 | 路径 | 端口 | 返回 |
| --- | --- | --- | --- |
| `GET` | `/health` | `HEALTH_PORT`（默认 8080） | 纯文本 `OK` |

## 项目结构

```text
DDevops/
├── main.go                      # 入口：初始化 DB、仓储、服务、路由，启动四个 goroutine
├── go.mod / go.sum              # Go 模块与依赖
├── .env.example                 # 配置模板（复制为 .env）
├── README.md
├── data/                        # SQLite 数据文件目录（数据库本身被 .gitignore 排除）
├── internal/
│   ├── auth/                    # bcrypt 密码校验、会话签发与失效
│   ├── config/                  # 从 .env / 环境变量加载配置
│   ├── model/                   # 领域模型与 BuildRule（唯一构建规则来源）
│   ├── repository/              # SQLite 访问、建表、幂等迁移（jobs / tasks / history / ...）
│   ├── service/                 # Jenkins 客户端、BuildService、调度器、钉钉通讯录
│   ├── handler/                 # HTTP API、钉钉消息分发、卡片与列表渲染
│   ├── middleware/              # AuthMiddleware / AdminMiddleware
│   ├── router/                  # /api 路由、SPA 回退、静态资源缓存策略、健康检查
│   └── webui/                   # go:embed 内嵌 dist；webui.go
└── frontend/
    ├── package.json             # vue / vite / tailwind / pinia
    ├── vite.config.js           # 构建输出到 ../internal/webui/dist，开发代理 /api
    ├── tailwind.config.js
    ├── DESIGN.md                # 前端设计规范
    ├── index.html
    ├── public/                  # brand-mark.svg、favicon.ico
    └── src/
        ├── main.js / App.vue
        ├── router/              # 路由与登录 / 管理员守卫
        ├── stores/              # auth（登录态）、ui（主题 / 密度 / 动效 / 搜索）
        ├── api/                 # 按资源封装的接口调用
        ├── views/               # 9 个页面：总览、任务、任务表单、定时、历史、环境、服务、用户、登录
        ├── components/
        │   ├── base/            # 按钮、输入、下拉、表格、分页、抽屉、弹窗、确认框
        │   ├── data/            # DataTable、KPI、状态 / 环境徽标、空态、错误态
        │   ├── feature/         # 构建弹窗、任务选择器、钉钉选人
        │   ├── layout/          # 侧栏、顶栏、命令面板、通知、页头
        │   └── ui/              # CountUp 等少量展示组件
        ├── composables/         # 防抖、倒计时、拖拽排序、权限、Toast
        ├── constants/           # 导航、状态、环境、角色常量
        ├── utils/               # 请求封装、格式化、CSV 导出、校验
        └── styles/              # tokens.css（设计令牌）、global.css
```