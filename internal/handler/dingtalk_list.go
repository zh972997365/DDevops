package handler

import (
	"fmt"
	"sort"
	"strings"

	"ddevops/internal/model"
)

// 钉钉列表消息渲染。
//
// 用户确认的旧卡片风格：### 标题 + --- 分隔线 + 加粗字段 + 环境颜色。
// 客户端 ActionCard 不渲染行内代码，短码一律纯文本；单个换行会被折叠，
// 所有行使用「行尾两个空格 + 换行」的硬换行。

type listCard struct {
	Title string
	Text  string
}

const mdLineBreak = "  \n"

func mdBullet(b *strings.Builder, format string, args ...any) {
	b.WriteString("- ")
	fmt.Fprintf(b, format, args...)
	b.WriteString(mdLineBreak)
}

func (h *DingTalkHandler) sendTaskListMessage(webhookURL string, scope map[string]bool, restricted bool, filter string) {
	filter, ok := normalizeEnvFilter(filter)
	if !ok {
		text := "### ⚠️ 不支持的环境\n\n" +
			"---\n" +
			"- 可选环境：测试、预发、生产\n" +
			"- 发送 @机器人 帮助 查看指令\n"
		h.sendCardMessage(webhookURL, actionCard("不支持的环境", text, "", ""))
		return
	}

	jobs, err := h.jobRepo.List()
	if err != nil {
		h.sendSimpleCardMessage(webhookURL, "❌ 获取失败", "- **对象**: 任务列表\n- **处理**: 请稍后重试")
		return
	}
	if restricted {
		visible := make([]model.Job, 0, len(jobs))
		for _, j := range jobs {
			if scope[j.ID] {
				visible = append(visible, j)
			}
		}
		jobs = visible
	}

	envs, _ := h.envRepo.List()
	card := buildTaskListCard(jobs, envs, filter)
	h.sendCardMessage(webhookURL, actionCard(card.Title, card.Text, "", ""))
}

func normalizeEnvFilter(s string) (string, bool) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return "", true
	case strings.Contains(s, "测试"):
		return "测试", true
	case strings.Contains(s, "预发"):
		return "预发", true
	case strings.Contains(s, "生产"):
		return "生产", true
	}
	return "", false
}

func envMatches(e model.Environment, filter string) bool {
	switch filter {
	case "测试":
		return !e.IsProd && strings.Contains(e.Name, "测试")
	case "预发":
		return strings.Contains(e.Name, "预发")
	case "生产":
		return e.IsProd
	}
	return true
}

func filterColor(filter string) string {
	switch filter {
	case "生产":
		return envColorProd
	case "预发":
		return envColorStaging
	default:
		return envColorTest
	}
}

// buildTaskListCard 只渲染当前环境的任务：标题 + 环境色 + 每个任务一条项目符号。
// 不输出汇总、环境计数、页脚，也不把多个任务挤在一行。
func buildTaskListCard(jobs []model.Job, envs []model.Environment, filter string) listCard {
	title := "任务列表"
	if filter != "" {
		title = filter + "环境任务"
		matched := make(map[string]bool, len(envs))
		envName := filter + "环境"
		for _, e := range envs {
			if envMatches(e, filter) {
				matched[e.ID] = true
			}
		}
		visible := make([]model.Job, 0, len(jobs))
		for _, j := range jobs {
			if matched[j.EnvID] {
				visible = append(visible, j)
			}
		}
		jobs = visible
		if len(jobs) == 0 {
			return listCard{Title: title, Text: "### 📋 " + title + "\n\n---\n该环境当前没有可构建的任务。"}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "### 📋 %s\n\n---\n", title)
		fmt.Fprintf(&b, "- **项目环境**: <font color='%s'>%s</font>\n", filterColor(filter), envName)
		envOrder := make(map[string]int, len(envs))
		for i, e := range envs {
			envOrder[e.ID] = i
		}
		sorted := append([]model.Job(nil), jobs...)
		sortJobsForList(sorted, envOrder)
		for _, j := range sorted {
			writeJobBullet(&b, j)
		}
		return listCard{Title: title, Text: b.String()}
	}

	if len(jobs) == 0 {
		return listCard{Title: title, Text: "### 📋 任务列表\n\n---\n当前没有可构建的任务。"}
	}
	envOrder := make(map[string]int, len(envs))
	for i, e := range envs {
		envOrder[e.ID] = i
	}
	sorted := append([]model.Job(nil), jobs...)
	sortJobsForList(sorted, envOrder)
	var b strings.Builder
	b.WriteString("### 📋 任务列表\n\n---\n")
	for _, j := range sorted {
		writeJobBullet(&b, j)
	}
	return listCard{Title: title, Text: b.String()}
}

// writeJobBullet 任务名用真实 ID，短码只作为输入提示放在括号里。
func writeJobBullet(b *strings.Builder, j model.Job) {
	name := mdCell(j.ID)
	code := jobCode(j)
	if code != j.ID {
		fmt.Fprintf(b, "- **%s**（%s）\n", name, code)
		return
	}
	fmt.Fprintf(b, "- **%s**\n", name)
}

func sortJobsForList(jobs []model.Job, envOrder map[string]int) {
	unknown := len(envOrder) + 1
	sort.SliceStable(jobs, func(i, j int) bool {
		oi, ok := envOrder[jobs[i].EnvID]
		if !ok {
			oi = unknown
		}
		oj, ok := envOrder[jobs[j].EnvID]
		if !ok {
			oj = unknown
		}
		if oi != oj {
			return oi < oj
		}
		ci, cj := jobCode(jobs[i]), jobCode(jobs[j])
		if ci != cj {
			return ci < cj
		}
		return jobs[i].ID < jobs[j].ID
	})
}

func jobCode(j model.Job) string {
	if j.ShortCode != "" {
		return j.ShortCode
	}
	return j.ID
}

// buildScheduleListCard 一个待执行定时任务一行：时间 / 短码 / 环境 / 版本。
// 环境按测试蓝、预发橙、生产绿着色。
func buildScheduleListCard(tasks []model.ScheduledTask, envNames, codes map[string]string) listCard {
	var b strings.Builder
	b.WriteString("### 🗓 定时任务\n\n---\n")
	if len(tasks) == 0 {
		b.WriteString("当前没有待执行的定时任务。")
		return listCard{Title: "定时任务", Text: b.String()}
	}
	for _, t := range tasks {
		code := codes[t.TaskID]
		if code == "" {
			code = t.TaskID
		}
		env := envNames[t.TaskID]
		if env == "" {
			env = "未配置"
		}
		ref := t.RefValue
		if ref == "" {
			ref = "默认"
		}
		name := t.TaskID
		if name == "" {
			name = code
		}
		if code != name {
			fmt.Fprintf(&b, "- **%s** / **%s**（%s） / <font color='%s'>%s</font> / %s\n",
				t.ScheduleTime.Format("01-02 15:04"), mdCell(name), mdCell(code), colorByName(env), mdCell(env), mdCell(ref))
		} else {
			fmt.Fprintf(&b, "- **%s** / **%s** / <font color='%s'>%s</font> / %s\n",
				t.ScheduleTime.Format("01-02 15:04"), mdCell(name), colorByName(env), mdCell(env), mdCell(ref))
		}
	}
	return listCard{Title: "定时任务", Text: b.String()}
}

func colorByName(name string) string {
	switch {
	case strings.Contains(name, "生产"):
		return envColorProd
	case strings.Contains(name, "预发"):
		return envColorStaging
	default:
		return envColorTest
	}
}

func (h *DingTalkHandler) jobCodeMap() map[string]string {
	jobs, err := h.jobRepo.List()
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(jobs))
	for _, j := range jobs {
		out[j.ID] = jobCode(j)
	}
	return out
}

// mdCell 清理单行文本里的换行和竖线，避免客户端折行错位。
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
