package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"ddevops/internal/model"
	"ddevops/internal/repository"
)

type JenkinsConfig struct {
	BaseURL string
	User    string
	Token   string
}

type jenkinsClient struct {
	config JenkinsConfig
	client *http.Client

	paramMu    sync.Mutex
	paramCache map[string]paramCacheEntry
}

type paramCacheEntry struct {
	names []string
	at    time.Time
}

func newJenkinsClient(config JenkinsConfig) *jenkinsClient {
	return &jenkinsClient{
		client:     &http.Client{Timeout: 30 * time.Second},
		config:     config,
		paramCache: make(map[string]paramCacheEntry),
	}
}

// joinJenkinsURL 拼接服务地址与任务路径，消除双方斜杠重复导致的 "//" 异常路由。
// Jenkins 遇到 "//view/..." 会返回 500 错误页，且日志里只显示"Sign in"，很难排查。
func joinJenkinsURL(base, jobPath string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	jobPath = strings.TrimSpace(jobPath)
	if jobPath == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(jobPath, "/")
}

// jobBasePath 去掉构建端点后缀，得到任务本身的路径。
func jobBasePath(jobPath string) string {
	path := strings.TrimRight(strings.TrimSpace(jobPath), "/")
	if strings.HasSuffix(path, "/buildWithParameters") {
		return strings.TrimSuffix(path, "/buildWithParameters")
	}
	return strings.TrimSuffix(path, "/build")
}

// buildWithParametersPath 把历史配置里的 ".../build" 换成官方参数化构建端点。
// 新版 Jenkins 上表单端点 /build 配合 json 参数会返回 500 错误页，buildWithParameters 才是稳定入口。
func buildWithParametersPath(jobPath string) string {
	return jobBasePath(jobPath) + "/buildWithParameters"
}

func (c *jenkinsClient) TriggerBuild(job model.Job, env model.Environment, refValue string) error {
	jobURL := joinJenkinsURL(c.config.BaseURL, buildWithParametersPath(job.JobPath))
	paramName := env.Rule().ParamName

	formData := url.Values{}
	formData.Set(paramName, refValue)

	status, body, err := c.postForm(jobURL, formData)
	if err != nil {
		return err
	}

	// Jenkins 的 buildWithParameters 会遍历任务上「所有」参数定义，为本次没提交的参数计算默认值。
	// 一旦某个参数定义已损坏（例如 extended-choice 的 type 字段缺失），这一步就抛 NPE，
	// 整页只回一句 "A problem occurred..." + Logging ID，与参数值无关，怎么改 Tag 都构建不了。
	// Jenkins 页面能正常构建，是因为浏览器把每个参数的字段都提交了；这里补一次同样的全量提交。
	if status >= http.StatusInternalServerError {
		if filled, ok := c.fullParamForm(job.JobPath, paramName, refValue); ok {
			log.Printf("Jenkins 返回 %d，疑似参数定义异常，改为提交全部 %d 个参数后重试", status, len(filled))
			status, body, err = c.postForm(jobURL, filled)
			if err != nil {
				return err
			}
		}
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return errors.New(jenkinsErrorDetail(body, status))
	}

	return nil
}

// postForm 提交参数化构建请求，网络错误重试三次（每次重建请求体，Body 只能读一次）。
func (c *jenkinsClient) postForm(jobURL string, form url.Values) (int, []byte, error) {
	log.Printf("Jenkins请求: POST %s  参数=%s", jobURL, form.Encode())

	var (
		resp *http.Response
		err  error
	)
	for attempt := 1; attempt <= 3; attempt++ {
		var req *http.Request
		req, err = http.NewRequest("POST", jobURL, strings.NewReader(form.Encode()))
		if err != nil {
			return 0, nil, fmt.Errorf("创建请求失败: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(c.config.User, c.config.Token)

		resp, err = c.client.Do(req)
		if err == nil {
			break
		}
		log.Printf("请求失败(尝试 %d): %v", attempt, err)
		if attempt < 3 {
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		return 0, nil, fmt.Errorf("多次尝试后失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// 失败时响应常是一整页 HTML，日志只留摘要避免刷屏
	if len(body) > 300 {
		log.Printf("Jenkins响应: %d (body %d bytes)", resp.StatusCode, len(body))
	} else {
		log.Printf("Jenkins响应: %d %s", resp.StatusCode, string(body))
	}
	return resp.StatusCode, body, nil
}

// fullParamForm 构造"除目标参数外其余参数传空"的全量表单。
// 空值足以跳过 Jenkins 对默认值的计算，同时不改变任务本身的参数语义。
func (c *jenkinsClient) fullParamForm(jobPath, paramName, refValue string) (url.Values, bool) {
	names, err := c.parameterNames(jobPath)
	if err != nil {
		log.Printf("读取任务参数定义失败: %v", err)
		return nil, false
	}

	filled := url.Values{}
	for _, name := range names {
		if name == "" || name == paramName {
			continue
		}
		filled.Set(name, "")
	}
	if len(filled) == 0 {
		return nil, false
	}
	filled.Set(paramName, refValue)
	return filled, true
}

// parameterNames 从任务 config.xml 里解析参数定义名。
// 不用 api/json：损坏的参数定义在 JSON 序列化阶段就被跳过，恰恰是它让构建端点崩掉，
// 靠 API 拿到的列表会漏掉真正需要补字段的那个参数。
func (c *jenkinsClient) parameterNames(jobPath string) ([]string, error) {
	basePath := jobBasePath(jobPath)

	c.paramMu.Lock()
	if entry, ok := c.paramCache[basePath]; ok && time.Since(entry.at) < 10*time.Minute {
		c.paramMu.Unlock()
		return entry.names, nil
	}
	c.paramMu.Unlock()

	configURL := joinJenkinsURL(c.config.BaseURL, basePath) + "/config.xml"
	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.config.User, c.config.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("读取 config.xml 返回 %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	names := parseParameterNames(raw)
	c.paramMu.Lock()
	c.paramCache[basePath] = paramCacheEntry{names: names, at: time.Now()}
	c.paramMu.Unlock()
	return names, nil
}

var (
	paramDefsPattern = regexp.MustCompile(`(?s)<parameterDefinitions>(.*?)</parameterDefinitions>`)
	paramNamePattern = regexp.MustCompile(`<name>(?:<!\[CDATA\[)?(.*?)(?:\]\]>)?</name>`)
)

// parseParameterNames 取出 parameterDefinitions 里的参数名，去重保序。
func parseParameterNames(configXML []byte) []string {
	block := paramDefsPattern.FindSubmatch(configXML)
	if len(block) < 2 {
		return nil
	}

	var names []string
	seen := make(map[string]bool)
	for _, m := range paramNamePattern.FindAllSubmatch(block[1], -1) {
		name := strings.TrimSpace(html.UnescapeString(string(m[1])))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

var (
	htmlTagPattern   = regexp.MustCompile(`<[^>]*>`)
	errorDescPattern = regexp.MustCompile(`(?s)<div[^>]*id="error-description"[^>]*>(.*?)</div>`)
)

// jenkinsErrorDetail 从 Jenkins 错误页中提取可读描述。
// Jenkins 出错时返回的是整页 HTML，直接展示会把导航、搜索框等噪音一起抛给用户。
func jenkinsErrorDetail(body []byte, status int) string {
	raw := string(body)
	// 优先取错误描述区；没有该结构时（例如纯 JSON 错误）再退回整页
	segment := raw
	if m := errorDescPattern.FindStringSubmatch(raw); len(m) > 1 && strings.TrimSpace(m[1]) != "" {
		segment = m[1]
	}
	text := htmlTagPattern.ReplaceAllString(segment, " ")
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > 200 {
		text = string(runes[:200]) + "…"
	}
	if text == "" {
		return fmt.Sprintf("Jenkins 返回 %d（无响应内容）", status)
	}
	msg := fmt.Sprintf("Jenkins 返回 %d：%s", status, text)
	// 服务端内部错误时 Jenkins 只给一个 Logging ID，用户无从下手，补一句可执行的方向
	if status >= http.StatusInternalServerError {
		msg += "（服务端内部错误，常见原因是任务配置里残留了无效参数，可到 Jenkins 该任务的配置页检查参数定义）"
	}
	return msg
}

func (c *jenkinsClient) GetBuildURL(jobPath string) string {
	return joinJenkinsURL(c.config.BaseURL, jobBasePath(jobPath))
}

type JenkinsManager struct {
	mu         sync.RWMutex
	clients    map[string]*jenkinsClient
	serverRepo *repository.ServerRepository
}

func NewJenkinsManager(serverRepo *repository.ServerRepository, defaultCfg JenkinsConfig) *JenkinsManager {
	m := &JenkinsManager{
		clients:    make(map[string]*jenkinsClient),
		serverRepo: serverRepo,
	}
	if defaultCfg.BaseURL != "" {
		m.clients["jenkins"] = newJenkinsClient(defaultCfg)
	}
	return m
}

func (m *JenkinsManager) getClient(serverID string) (*jenkinsClient, error) {
	if serverID == "" {
		serverID = "jenkins"
	}

	m.mu.RLock()
	client, ok := m.clients[serverID]
	m.mu.RUnlock()
	if ok {
		return client, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok = m.clients[serverID]
	if ok {
		return client, nil
	}

	server, err := m.serverRepo.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("Jenkins服务器 %s 不存在: %v", serverID, err)
	}

	client = newJenkinsClient(JenkinsConfig{
		BaseURL: server.BaseURL,
		User:    server.User,
		Token:   server.Token,
	})
	m.clients[serverID] = client
	return client, nil
}

func (m *JenkinsManager) TriggerBuild(job model.Job, env model.Environment, refValue string) error {
	client, err := m.getClient(job.ServerID)
	if err != nil {
		return err
	}
	return client.TriggerBuild(job, env, refValue)
}

func (m *JenkinsManager) GetBuildURL(job model.Job) string {
	client, err := m.getClient(job.ServerID)
	if err != nil {
		return ""
	}
	return client.GetBuildURL(job.JobPath)
}

func (m *JenkinsManager) ReloadServer(serverID string) {
	m.mu.Lock()
	delete(m.clients, serverID)
	m.mu.Unlock()
}

// ---- 经 Jenkins REST API 拉取 Git Parameter 的 Tag 列表 ----

// canonicalJobPath 把存储的 job_path（可能带 /view/<视图>/ 前缀与 /build 动作段）
// 规整为 Jenkins REST API 的规范任务路径 /job/...（文件夹任务为 /job/<folder>/job/<name>）。
func canonicalJobPath(jobPath string) string {
	p := strings.TrimRight(jobPath, "/")
	p = strings.TrimPrefix(p, "/")
	if strings.HasSuffix(p, "/buildWithParameters") {
		p = strings.TrimSuffix(p, "/buildWithParameters")
	} else if strings.HasSuffix(p, "/build") {
		p = strings.TrimSuffix(p, "/build")
	}
	p = strings.TrimRight(p, "/")
	if strings.HasPrefix(p, "view/") {
		if idx := strings.Index(p, "/job/"); idx >= 0 {
			p = p[idx+1:] // 去掉 /view/<视图>/ 前缀，保留其后的 /job/...
		}
	}
	return "/" + strings.TrimLeft(p, "/")
}

// GetJobGitTags 经由 Jenkins 的 Git Parameter 类插件端点拉取该任务可用的 Git 引用（tag/branch）。
// 该端点与 Jenkins 构建页下拉同源：插件会在 Jenkins 服务器上用任务 SCM 配置的 Git 凭据执行
// git ls-remote，因此 private 仓库也能正常取到（无需本系统额外维护 GitLab 认证）。
// 返回 (tags, ok, err)：ok=true 表示成功取到；否则 err 说明原因，调用方应优雅降级为手动输入。
func (m *JenkinsManager) GetJobGitTags(job model.Job) ([]string, bool, error) {
	client, err := m.getClient(job.ServerID)
	if err != nil {
		return nil, false, err
	}
	return client.getJobGitTags(job.JobPath)
}

func (c *jenkinsClient) getJobGitTags(jobPath string) ([]string, bool, error) {
	apiURL := joinJenkinsURL(c.config.BaseURL, canonicalJobPath(jobPath)+"/api/json")

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.SetBasicAuth(c.config.User, c.config.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("读取 Jenkins REST API 返回 %d", resp.StatusCode)
	}
	// 注意：Git Parameter 插件的 allValueItems 是 @Exported 的计算型字段，
	// 仅在「完整 api/json（不带 tree 过滤）」时才会被真正调用并填充 tag 列表。
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, false, err
	}

	var doc struct {
		Property []struct {
			ParameterDefinitions []struct {
				Name          string `json:"name"`
				Type          string `json:"type"`
				AllValueItems struct {
					Values []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"values"`
					Errors []string `json:"errors"`
				} `json:"allValueItems"`
			} `json:"parameterDefinitions"`
		} `json:"property"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false, fmt.Errorf("解析 Jenkins 响应失败: %v", err)
	}

	type candidate struct {
		name   string
		values []string
	}
	var tagCands, anyCands []candidate
	for _, prop := range doc.Property {
		for _, pd := range prop.ParameterDefinitions {
			if !strings.Contains(pd.Type, "GitParameterDefinition") {
				continue
			}
			if len(pd.AllValueItems.Errors) > 0 {
				continue
			}
			vals := make([]string, 0, len(pd.AllValueItems.Values))
			for _, v := range pd.AllValueItems.Values {
				t := v.Value
				if t == "" {
					t = v.Name
				}
				if t != "" {
					vals = append(vals, t)
				}
			}
			if len(vals) == 0 {
				continue
			}
			c := candidate{name: pd.Name, values: vals}
			anyCands = append(anyCands, c)
			if strings.Contains(strings.ToUpper(pd.Name), "TAG") {
				tagCands = append(tagCands, c)
			}
		}
	}

	if len(tagCands) > 0 {
		return tagCands[0].values, true, nil
	}
	if len(anyCands) > 0 {
		return anyCands[0].values, true, nil
	}
	return nil, false, fmt.Errorf("该任务未配置 Git Parameter，或插件未返回 Tag 列表")
}
