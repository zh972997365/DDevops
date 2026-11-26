# DDevops · Jenkins 钉钉发布系统

基于钉钉机器人的 Jenkins 持续集成发布系统。单个 Go 进程同时提供 Web 管理后台、钉钉机器人、定时调度器与健康检查；**前端 Vue3 SPA 在编译期嵌入二进制，运行时不依赖任何静态目录，也不需要 Docker**。

> 部署只需一个二进制（或 `go run .`）：不单独起前端、不依赖容器。改了前端代码后，重新构建前端并重新编译/运行 Go 即可。

三个触发入口（Web 手动构建、钉钉指令、定时任务）共用同一个 `BuildService`，构建规则、参数名与发布历史的写入逻辑只有一份实现。Web 与钉钉机器人也共用同一套权限口径。

---

## 1. 功能总览

### 1.1 Web 管理后台

前端使用 history 路由，登录页之外的页面都需要鉴权，管理员页面额外校验角色。

| 页面 | 路径 | 主要能力 |
| --- | --- | --- |
| 概括总览 | `/` | 今日发布、近 14 日趋势、定时队列、环境矩阵、最近发布与失败待办 |
| 任务管理 | `/jobs` | 任务列表、筛选排序、批量导出、详情抽屉、手动触发构建 |
| 新建 / 编辑任务 | `/jobs/new`、`/jobs/:id/edit` | 独立表单页：任务 ID、环境、Jenkins 路径、服务与规则预览 |
| 定时任务 | `/tasks` | 新建（任务下拉支持搜索）、执行时间、按状态分组、取消与删除、按执行时间倒序 |
| 发布历史 | `/history` | 表格展示、统计卡（总数/成功/失败/今日）、任务·环境·状态·日期筛选、分页、失败行标红、构建链接 |
| 环境管理 | `/environments`（管理员） | CRUD、分支/Tag 构建方式、默认分支与参数名、拖拽排序、关联任务数、改动影响确认 |
| 服务管理 | `/servers`（管理员） | 多 Jenkins 实例 CRUD、Token 列表隐藏、拖拽排序、默认实例不可删、配置热更新 |
| 用户管理 | `/permissions`（管理员） | 用户 CRUD、admin 保护、钉钉选人搜索与通讯录同步、可构建项目分配 |

界面为浅色为主的克制玻璃拟态：浅色主题的画布、侧栏、顶栏同为浅色系，用一档中性底与细描边分层；深色主题整体翻转，侧栏与顶栏自然为深色。配色以石墨中性承载信息，低饱和雾钢蓝只用于操作与状态，避免和发布结果抢注意力。支持浅/深主题、舒适/紧凑密度、标准/增强两档动效、`⌘K` 全局搜索、侧栏滑块、详情抽屉、表格排序/筛选/分页/批量选择，以及空态、加载骨架与错误重试。设计令牌集中在 `frontend/src/styles/tokens.css`，组件原语在 `global.css` 与 `components/base`，业务页面不写零散配色；完整规范见 `frontend/DESIGN.md`。

认证：bcrypt 密码摘要（历史明文密码首次登录自动升级）；会话存 `sessions` 表，30 天有效，重启不掉线；改密或删号会踢掉旧会话。

**手动构建的版本 Tag（生产环境）**

生产环境（Tag 构建）的「版本 Tag」字段是一个下拉框，数据**直接来自 Jenkins 任务自身的 Git Parameter**，而不是本系统维护的 GitLab 凭据：

- 取数端点：`GET /job/<任务路径>/api/json`，解析 Git Parameter 插件暴露的 `allValueItems`（与 Jenkins 构建页下拉同源）。
- 插件会在 Jenkins 服务器上用**任务 SCM 配置的 Git 凭据**执行 `git ls-remote`，因此私有仓库也能正常取到，本系统无需额外维护 GitLab Token。
- 下拉仅展示**最近 10 个 Tag、最新在最上方**，纯下拉选择、不手填（如 Jenkins 未返回可用 Tag，下拉禁用并提示重试）。
- 该请求是只读 GET，**不会触发构建**。

### 1.2 钉钉机器人（Stream 长连接）

在群里 `@机器人` 使用，所有指令都会先做权限校验，且权限口径与 Web 完全一致（见 1.5）。

| 指令 | 说明 |
| --- | --- |
| `帮助` / `help` | 帮助卡片（含基本指令、默认分支/版本规则、定时任务、示例） |
| `测试` | 只看测试环境任务，一个任务一条项目符号 |
| `预发` | 只看预发环境任务 |
| `生产` | 只看生产环境任务 |
| `<任务ID或短码> [分支或Tag]` | 立即构建；省略引用时按环境规则回落；任务不在授权范围内返回「无任务权限」卡片 |
| `定时(年月日时分) <任务ID或短码> [分支或Tag]` | 创建定时，如 `定时(202611051800) pop-admin 20261105u01` |
| `定时列表` | 列出**自己创建且授权**的待执行定时任务 |
| `取消定时 <任务ID或短码>` | 只能取消自己创建的定时 |

任务列表使用 ActionCard 的 markdown 正文：`测试` / `预发` / `生产` 分别输出对应环境，一个任务一条项目符号，任务名用真实 ID，短码只放在括号里作为输入提示；不输出汇总、环境计数或页脚说明。卡片沿用旧版视觉：`###` 标题、`---` 分隔线、`#### 🔹` 章节、`**加粗字段**`，环境用 `<font color>` 区分（测试蓝、预发橙、生产绿）。短码保持纯文本，不使用反引号或表格，避免客户端丢失或错行。构建结果仍以卡片回执；缺少版本 Tag 时会单独提示正确指令格式，不会笼统报「构建失败」。

### 1.3 定时调度器

启动即检查一次，之后每分钟扫描 `pending` 任务：到点置 `executing` → 调用 `BuildService` → `completed` / `failed` → 写入发布历史。状态包含 `pending`、`executing`、`completed`、`failed`、`cancelled`。

### 1.4 构建核心

`BuildRule` 是构建规则的唯一来源，由环境配置派生，包含模式、参数名、默认引用、是否必填：

| 模式 | 引用行为 | 传给 Jenkins 的参数 |
| --- | --- | --- |
| Tag 构建 | 必须显式填写版本 Tag（如 `20261105u01`），空值或非法字符直接拒绝 | 环境的标签参数名，如 `Tags` / `GIT_TAG` |
| 分支构建 | 留空使用环境默认分支；未带前缀时自动补 `origin/` | 环境的分支参数名，如 `branch` / `release_branch` |

关键约束：Tag 模式下漏填**不会**静默回落到测试分支。参数名填错会导致 Jenkins 读不到参数、Job 内部校验失败，因此环境编辑页会展示影响面并要求二次确认。

调用 Jenkins 使用官方 `buildWithParameters` 端点，服务地址与任务路径会做斜杠归一，每次构建都写入 `publish_history`。

### 1.5 权限模型

系统只有两种角色：`admin` 与 普通用户，权限在 Web 与钉钉两端完全一致。

**角色能力**

- `admin`：全量权限，不受任务范围限制；可进入用户管理 / 服务管理 / 环境管理三个管理员页面；可删除、编辑所有任务、定时与历史；钉钉通讯录接口仅 admin 可用。
- 普通用户：仅能在「可构建项目」列表内的任务上操作。

**可构建项目（allowed_jobs）**

`permissions.allowed_jobs` 为 JSON 数组，存放该用户可构建 / 查看的任务 ID 列表。

- 显式勾选了项目列表的普通用户：只能构建、在任务列表查看、查看发布历史、创建定时中属于该列表的任务。
- `allowed_jobs` 为**空数组 `[]`**：视为「无任何授权任务」——该用户看不到任何任务、无法构建。旧版「空列表即全量」的向后兼容逻辑已移除。
- 因此，新增的普通用户若未分配项目，会被锁定为无可用任务，需管理员在「用户管理」中为其分配。
- 部分更新语义：`PUT /api/permissions/:userId` 若请求体未提交 `allowedJobs`，沿用旧值，避免「只重置密码」的请求被误写成全量授权。

**Web 与钉钉同一套口径**

`jobScopeFor(role, allowedJobs)` 被 Web API（`jobScope`）与钉钉消息处理（`resolveScope`）同时调用，保证两端行为一致，不出现「Web 能建、钉钉不能」或反之的割裂。

**数据可见性（按角色）**

- 发布历史：admin 看所有用户、所有环境的执行记录（含执行人）；普通用户只看到自己触发的记录。
- 定时任务：admin 看全部；普通用户只看自己创建的。

**会话**

bcrypt 密码；会话存 `sessions` 表，30 天 TTL，重启 / 多副本有效；改密或删号会踢掉旧会话。

---

## 2. 架构

```
main
 ├── config          读取 .env / 环境变量
 ├── repository      SQLite 数据访问（幂等迁移在此）
 ├── service         Jenkins / 钉钉 / 调度器 / BuildService
 ├── handler         API + 钉钉消息与卡片
 ├── router          路由、SPA 托管、健康检查
 └── middleware      鉴权
frontend/src         Vue3 + Vite SPA，产物经 internal/webui (go:embed) 内嵌
```

技术栈：Go 1.25 + Gin、SQLite（纯 Go 驱动、WAL、单连接）、Vue3 + Vite5 + Pinia（无 UI 框架）、钉钉 Stream SDK、bcrypt。

进程启动 4 个 goroutine：

1. Web 服务（`WEB_PORT`，默认 `8081`）
2. 健康检查（`HEALTH_PORT`，默认 `8080`，`GET /health`）
3. 钉钉 Stream 长连接
4. 定时调度器

### 目录

| 路径 | 内容 |
| --- | --- |
| `internal/auth` | 密码摘要与会话校验 |
| `internal/config` | 配置加载 |
| `internal/model` | 数据模型与 `BuildRule` |
| `internal/repository` | 建表、幂等迁移、各实体 CRUD |
| `internal/service` | Jenkins 客户端、构建服务、调度器、钉钉服务 |
| `internal/handler` | HTTP API 与钉钉消息/卡片渲染 |
| `internal/router` | 路由注册、SPA 回退、健康检查 |
| `internal/webui` | `go:embed` 嵌入前端产物 |
| `frontend/src` | views / components / composables / api / stores / router |

---

## 3. 快速开始

### 3.1 环境要求

- Go 1.25+
- Node.js 18+（仅构建前端时需要）
- 可访问的 Jenkins 服务与 API Token
- 钉钉企业内部机器人（使用 Stream 模式）

### 3.2 配置

在项目根目录创建 `.env`（不创建则使用环境变量与以下默认值）：

```dotenv
# 默认 Jenkins 服务（自动注册 id=jenkins）
JENKINS_BASE_URL=https://jenkins.example.com
JENKINS_USER=your_username
JENKINS_TOKEN=your_api_token

# 第二套 Jenkins 服务（可选，自动注册 id=jenkins2）
JENKINS2_BASE_URL=
JENKINS2_USER=
JENKINS2_TOKEN=

# 钉钉 Stream 机器人
DINGTALK_CLIENT_ID=your_client_id
DINGTALK_CLIENT_SECRET=your_client_secret

# 首次启动写入的初始管理员
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123

# 数据库与端口
SQLITE_DB_PATH=data/jenkins.db
WEB_PORT=8081
HEALTH_PORT=8080
```

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SQLITE_DB_PATH` | `jenkins.db` | SQLite 文件路径，测试可用 `:memory:`；当前项目 `.env` 指向 `data/jenkins.db` |
| `JENKINS_*` | 空 | 默认实例连接信息 |
| `JENKINS2_*` | 空 | 可选的第二实例 |
| `DINGTALK_*` | 空 | 留空时钉钉服务不启动 |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `admin123` | 仅首次启动时 seed |
| `WEB_PORT` / `HEALTH_PORT` | `8081` / `8080` | Web 与健康检查端口 |

生产环境请务必修改初始管理员密码。

### 3.3 构建与启动

前端产物是编译期嵌入的，**任何 `frontend/` 改动都要先重新构建前端，再重新运行 Go**：

```bash
# 1) 构建前端（输出到 ../internal/webui/dist，随后被 go:embed 内嵌）
cd frontend
npm install
npm run build

# 2) 回到根目录，直接运行（推荐，单进程即可，无需单独起前端、无需 Docker）
cd ..
go run .

# 或者编译为二进制后运行
# go build -o ddevops .
# ./ddevops
```

开发时可启动 Vite 调试前端（`http://localhost:5173`），配置里已把 `/api` 代理到 `http://localhost:8081`，后端服务需同时保持运行：

```bash
cd frontend && npm run dev
```

注意 Vite 只用于开发调试，正式运行始终由 Go 进程提供内嵌的静态资源。

启动后：

- Web 后台：`http://localhost:8081`
- 健康检查：`http://localhost:8080/health`
- 默认账号：`admin` / `admin123`

---

## 4. 配置要点

### 4.1 环境管理

环境是构建规则的载体，每个任务关联一个环境。环境表单包含：

- **环境名称**：仅用于展示（钉钉卡片文案、历史筛选），不直接改变构建参数
- **构建方式**：分支构建 / Tag 构建，决定是否强制填写版本 Tag
- **默认分支**：分支构建留空时使用
- **分支参数名 / 标签参数名**：真正传给 Jenkins 的参数名

不同 Jenkins 上 Job 定义的参数名可能不同（例如有的叫 `Tags`、有的叫 `GIT_TAG`）。遇到这种情况，新建另一个环境并分别配置参数名即可，不需要改代码。参数名只允许字母、数字、下划线，且不能以数字开头。

编辑环境时，页面会显示关联任务数；改动会影响这些任务**下一次及以后**的构建，正在执行与历史记录不受影响。检测到构建相关字段变化时会逐条列出并要求确认。

### 4.2 任务与服务

- 任务的「任务路径」应指向可构建的路径，例如 `/view/test/job/xxx/build`；系统会换算为 `buildWithParameters`
- 服务地址的末尾斜杠与任务路径的开头斜杠会自动归一，不会拼出 `//`
- 服务列表不返回 Token；编辑时 Token 留空表示保持原值
- 任务 ID 既是主键也是任务名，创建后仍可修改。改名会在事务里同步 `scheduled_tasks.task_id` 与 `publish_history.task_id`，定时任务和历史不会断链。前端会二次确认，因为钉钉指令需改用新 ID、旧 ID 立即失效
- 任务表不保存环境名副本：环境名一律按 `env_id` 实时读取，环境改名后任务列表、定时任务、发布历史、钉钉卡片立即同步
- 新 ID 不能与已有任务重复，重复时返回 400
- 有任务关联的环境不允许删除

### 4.3 发布历史

服务端分页，按触发时间倒序，硬过滤近一年；支持任务、环境、状态、日期范围筛选，记录包含触发方式（手动/定时/钉钉）、触发人、分支或 Tag、结果与构建链接。

环境列只按 `env_id` 归属，环境名由前端用当前环境配置换算，因此历史里不会出现改名前的旧环境名。

### 4.4 用户管理与授权

- 用户管理页可增删改用户、搜索 / 同步钉钉通讯录（`admin` 专属）。
- 每个普通用户通过「可构建项目」勾选其能操作的项目列表；未勾选即空列表，该用户登录后无任何可用任务。
- `admin` 始终全量，不受可构建项目限制，也不能被删除。
- 钉钉用户与 Web 用户是两套身份体系：钉钉侧通过工号解析 Web 权限（`resolveScope`），因此同一人在钉钉端享有的任务范围取决于其 Web 账号的 `allowed_jobs` 配置。

---

## 5. 运维说明

- **单进程运行**：只需运行一个 Go 进程（`go run .` 或编译后的二进制），前端已被内嵌，不要在前面再单独部署一份前端静态服务。
- **改前端必须重建**：任何前端改动后，必须 `npm run build` 重建并重启 Go 进程，否则看到的仍是旧界面。
- **生产敏感任务**：涉及生产发布的 Jenkins 任务（如 `线上-8080` 等）请在 Jenkins 侧自行触发/验证，避免通过本系统误触发。
- **健康检查**：`GET /health` 用于存活探测，默认端口 `HEALTH_PORT`(8080)。
- **数据库**：SQLite 单文件（当前为 `data/jenkins.db`），备份时直接拷贝该文件即可（建议先停服或确保 WAL 已落盘）。
