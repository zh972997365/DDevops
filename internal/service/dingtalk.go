package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"ddevops/internal/repository"

	"github.com/mozillazg/go-pinyin"
)

type DingTalkUser struct {
	UserID string `json:"userid"`
	Name   string `json:"name"`
	Pinyin string `json:"pinyin"`
}

// ToPinyin 将中文姓名转为小写拼音，如 张昊 -> zhanghao
func ToPinyin(name string) string {
	if name == "" {
		return ""
	}
	a := pinyin.NewArgs()
	a.Style = pinyin.Normal
	parts := pinyin.LazyPinyin(name, a)
	return strings.ToLower(strings.Join(parts, ""))
}

type DingTalkService struct {
	mu           sync.RWMutex
	users        map[string]string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	token        string
	tokenExpire  time.Time
	syncing      bool
	cache        *repository.DingTalkCacheRepository
}

func NewDingTalkService(clientID, clientSecret string, cache *repository.DingTalkCacheRepository) *DingTalkService {
	s := &DingTalkService{
		users:        make(map[string]string),
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 20 * time.Second},
		cache:        cache,
	}
	if cache != nil {
		if users, syncedAt, err := cache.LoadAll(); err == nil && len(users) > 0 {
			s.users = users
			log.Printf("从数据库加载钉钉员工缓存: %d 人 (同步于 %s)", len(users), syncedAt.Format("2006-01-02 15:04:05"))
		}
	}
	return s
}

func (s *DingTalkService) getToken() (string, error) {
	if s.token != "" && time.Now().Before(s.tokenExpire) {
		return s.token, nil
	}
	body, _ := json.Marshal(map[string]string{
		"appKey":    s.clientID,
		"appSecret": s.clientSecret,
	})
	resp, err := s.httpClient.Post("https://api.dingtalk.com/v1.0/oauth2/accessToken", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("请求token失败: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.AccessToken != "" {
		s.token = result.AccessToken
		s.tokenExpire = time.Now().Add(7000 * time.Second)
		return s.token, nil
	}
	// 回退旧版接口
	return s.getTokenOld()
}

func (s *DingTalkService) getTokenOld() (string, error) {
	u := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s", s.clientID, s.clientSecret)
	resp, err := s.httpClient.Get(u)
	if err != nil {
		return "", fmt.Errorf("请求token失败: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析token失败: %v", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("token错误: %s", result.ErrMsg)
	}
	s.token = result.AccessToken
	s.tokenExpire = time.Now().Add(7000 * time.Second)
	return s.token, nil
}

// listSubDepts 获取子部门ID列表
func (s *DingTalkService) listSubDepts(token string, deptID int64) []int64 {
	u := fmt.Sprintf("https://oapi.dingtalk.com/topapi/v2/department/listsub?access_token=%s", token)
	body := fmt.Sprintf(`{"dept_id":%d}`, deptID)
	resp, err := s.httpClient.Post(u, "application/json", strings.NewReader(body))
	if err != nil {
		log.Printf("获取部门%d子部门失败: %v", deptID, err)
		return nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Result  []struct {
			DeptID int64 `json:"dept_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("解析部门%d子部门失败: %v", deptID, err)
		return nil
	}
	if result.ErrCode != 0 {
		log.Printf("部门%d子部门接口错误: %s", deptID, result.ErrMsg)
		return nil
	}
	var ids []int64
	for _, d := range result.Result {
		ids = append(ids, d.DeptID)
	}
	return ids
}

// listDeptUsers 获取部门用户（分页）
func (s *DingTalkService) listDeptUsers(token string, deptID int64, out map[string]string) {
	cursor := 0
	for {
		u := fmt.Sprintf("https://oapi.dingtalk.com/topapi/v2/user/list?access_token=%s", token)
		body := fmt.Sprintf(`{"dept_id":%d,"cursor":%d,"size":100}`, deptID, cursor)
		resp, err := s.httpClient.Post(u, "application/json", strings.NewReader(body))
		if err != nil {
			log.Printf("部门%d拉取用户失败: %v", deptID, err)
			return
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			Result  struct {
				List []struct {
					UserID string `json:"userid"`
					Name   string `json:"name"`
				} `json:"list"`
				HasMore bool `json:"has_more"`
			} `json:"result"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			log.Printf("解析部门%d用户失败: %v", deptID, err)
			return
		}
		if result.ErrCode != 0 {
			log.Printf("部门%d用户接口错误: %s", deptID, result.ErrMsg)
			return
		}
		for _, item := range result.Result.List {
			if item.UserID != "" && item.Name != "" {
				out[item.UserID] = item.Name
			}
		}
		if !result.Result.HasMore {
			break
		}
		cursor += 100
	}
}

// collectDeptUsers 递归收集部门及子部门的用户
func (s *DingTalkService) collectDeptUsers(token string, deptID int64, out map[string]string, visited map[int64]bool) {
	if visited[deptID] {
		return
	}
	visited[deptID] = true

	s.listDeptUsers(token, deptID, out)
	for _, sub := range s.listSubDepts(token, deptID) {
		s.collectDeptUsers(token, sub, out, visited)
	}
}

// SyncUsers 从钉钉拉取全量员工
func (s *DingTalkService) SyncUsers() error {
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		return nil
	}
	s.syncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.syncing = false
		s.mu.Unlock()
	}()

	token, err := s.getToken()
	if err != nil {
		log.Printf("同步员工失败(获取token): %v", err)
		return err
	}

	result := make(map[string]string)
	s.collectDeptUsers(token, 1, result, make(map[int64]bool))

	if len(result) == 0 {
		log.Printf("同步员工：未获取到任何员工，请检查应用通讯录数据权限范围是否为「全部员工」")
		return fmt.Errorf("未获取到员工数据")
	}

	s.mu.Lock()
	s.users = result
	s.mu.Unlock()

	if s.cache != nil {
		if err := s.cache.SaveAll(result); err != nil {
			log.Printf("保存钉钉员工缓存失败: %v", err)
		}
	}

	log.Printf("钉钉员工同步完成，共 %d 人", len(result))
	return nil
}

// SearchUsers 按姓名 / 拼音 / 用户ID 模糊搜索，结果按姓名排序，保证展示顺序稳定。
func (s *DingTalkService) SearchUsers(keyword string) ([]DingTalkUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kw := strings.ToLower(keyword)
	users := make([]DingTalkUser, 0, len(s.users))
	for userID, name := range s.users {
		py := ToPinyin(name)
		if kw == "" || strings.Contains(strings.ToLower(name), kw) || strings.Contains(userID, kw) || strings.Contains(py, kw) {
			users = append(users, DingTalkUser{UserID: userID, Name: name, Pinyin: py})
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Name < users[j].Name })
	return users, nil
}

func (s *DingTalkService) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}
