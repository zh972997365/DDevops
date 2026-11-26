package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"ddevops/internal/model"
)

func testEnvs() []model.Environment {
	return []model.Environment{
		{ID: "env-test", Name: "测试环境", IsProd: false, DefaultRef: "origin/test", DefaultBranchParam: "branch"},
		{ID: "env-prod", Name: "生产环境", IsProd: true, DefaultTagParam: "Tags"},
	}
}

func makeJobs(envID, prefix string, n int) []model.Job {
	out := make([]model.Job, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, model.Job{
			ID:        fmt.Sprintf("pop-%02d", i),
			EnvID:     envID,
			ShortCode: fmt.Sprintf("%s%02d", prefix, i),
		})
	}
	return out
}

// 客户端 ActionCard 不渲染行内代码，正文里出现反引号会导致短码整段丢失。
func hasInlineCode(s string) bool {
	return strings.Contains(s, "`")
}

func TestTaskListEnvFilter(t *testing.T) {
	envs := testEnvs()
	jobs := append(makeJobs("env-prod", "P", 4), makeJobs("env-test", "T", 5)...)
	testCard := buildTaskListCard(jobs, envs, "测试")
	if testCard.Title != "测试环境任务" || strings.Contains(testCard.Text, "**P01**") {
		t.Fatalf("test filter leaked prod tasks: %#v", testCard)
	}
	if !strings.Contains(testCard.Text, "（T01）") || strings.Contains(testCard.Text, "（P01）") || !strings.Contains(testCard.Text, "测试环境") {
		t.Fatalf("test filter missing test tasks: %s", testCard.Text)
	}
	prodCard := buildTaskListCard(jobs, envs, "生产")
	if prodCard.Title != "生产环境任务" || strings.Contains(prodCard.Text, "**T01**") {
		t.Fatalf("prod filter leaked test tasks: %#v", prodCard)
	}
	if !strings.Contains(prodCard.Text, "（P01）") || strings.Contains(prodCard.Text, "（T01）") || !strings.Contains(prodCard.Text, "<font color=") {
		t.Fatalf("prod filter missing prod tasks or color: %s", prodCard.Text)
	}
	if got, ok := normalizeEnvFilter("生产环境"); !ok || got != "生产" {
		t.Fatalf("normalize 生产环境 = %q, %v", got, ok)
	}
	if _, ok := normalizeEnvFilter("未知"); ok {
		t.Fatal("unknown filter should be rejected")
	}
}

func TestHelpCardUsesLegacySections(t *testing.T) {
	text := helpCardText("T01", "P01")
	if hasInlineCode(text) {
		t.Fatalf("help card should not use inline code: %s", text)
	}
	for _, want := range []string{
		"### 📚 使用帮助",
		"#### 🔹 基本指令",
		"#### 🔹 环境查看",
		"#### 🔹 定时任务",
		"#### 🔹 使用示例",
		"@机器人 测试",
		"@机器人 预发",
		"@机器人 生产",
		"@机器人 定时列表",
		"P01",
		"T01",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help card missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "列表：查看全部任务") || strings.Contains(text, "共 ") {
		t.Fatalf("help should not mention removed 列表 command or summary: %s", text)
	}
}

func TestBuildStatusCardUsesLegacyStyle(t *testing.T) {
	card := (&DingTalkHandler{}).makeBuildStatusCard(
		"线上-8003", "8003crm20261125u01", "https://example.com/build/1",
		"构建成功", model.Environment{Name: "生产环境", IsProd: true}, "", "P01",
	)
	action, _ := card["actionCard"].(map[string]interface{})
	text, _ := action["text"].(string)
	if !strings.HasPrefix(text, "### 线上-8003 构建成功") {
		t.Fatalf("missing header: %q", text)
	}
	if !strings.Contains(text, "<font color='#00CC66'>生产环境</font>") {
		t.Fatalf("environment should be green: %q", text)
	}
	if !strings.Contains(text, "- **项目环境**:") || !strings.Contains(text, "- **项目分支**:") {
		t.Fatalf("missing bold fields: %q", text)
	}
	if strings.Contains(text, "任务信息") {
		t.Fatalf("status card should not add a section heading: %q", text)
	}
	if action["singleTitle"] != "👉 查看任务详情" {
		t.Fatalf("missing detail button: %#v", action)
	}
}

func TestTaskListOnePerLine(t *testing.T) {
	envs := testEnvs()
	jobs := append(makeJobs("env-prod", "P", 5), makeJobs("env-test", "T", 7)...)
	card := buildTaskListCard(jobs, envs, "")
	if hasInlineCode(card.Text) {
		t.Fatalf("card must not use inline code: %s", card.Text)
	}
	if strings.Contains(card.Text, "共 ") || strings.Contains(card.Text, "短码见") {
		t.Fatalf("list should not print summary/footer: %s", card.Text)
	}
	if !strings.HasPrefix(card.Text, "### 📋 任务列表") || !strings.Contains(card.Text, "---") {
		t.Fatalf("list needs header and divider: %s", card.Text)
	}
	for i := 1; i <= 7; i++ {
		if !strings.Contains(card.Text, fmt.Sprintf("（T%02d）", i)) {
			t.Fatalf("missing T%02d: %s", i, card.Text)
		}
	}
	for i := 1; i <= 5; i++ {
		if !strings.Contains(card.Text, fmt.Sprintf("（P%02d）", i)) {
			t.Fatalf("missing P%02d: %s", i, card.Text)
		}
	}
	for _, line := range strings.Split(card.Text, "\n") {
		if strings.HasPrefix(line, "- **") {
			if strings.Count(line, "（T")+strings.Count(line, "（P") > 1 {
				t.Fatalf("more than one task on line %q", line)
			}
		}
	}
}

func TestTaskListNoPagination(t *testing.T) {
	card := buildTaskListCard(makeJobs("env-prod", "P", 4), testEnvs(), "生产")
	if strings.Contains(card.Text, "下一页") || strings.Contains(card.Text, "第 1/") {
		t.Fatalf("list should not paginate: %s", card.Text)
	}
}

func TestTaskListEmptyDoesNotPanic(t *testing.T) {
	card := buildTaskListCard(nil, testEnvs(), "测试")
	if !strings.Contains(card.Text, "没有可构建的任务") {
		t.Fatalf("unexpected empty card: %s", card.Text)
	}
}

func TestBuildScheduleListCard(t *testing.T) {
	tasks := make([]model.ScheduledTask, 0, 10)
	for i := 1; i <= 10; i++ {
		tasks = append(tasks, model.ScheduledTask{
			ID:           fmt.Sprintf("t-%d", i),
			TaskID:       fmt.Sprintf("job-%d", i),
			RefValue:     "20261105u01",
			ScheduleTime: time.Date(2026, 11, 24, 18, i, 0, 0, time.Local),
			Status:       "pending",
		})
	}
	envNames := map[string]string{"job-1": "生产环境"}
	codes := map[string]string{"job-1": "P01"}
	card := buildScheduleListCard(tasks, envNames, codes)
	if hasInlineCode(card.Text) {
		t.Fatalf("schedule card should stay plain text: %s", card.Text)
	}
	if strings.Contains(card.Text, "|") {
		t.Fatalf("schedule card must not use a table: %s", card.Text)
	}
	for _, want := range []string{"### 🗓 定时任务", "<font color=", "P01", "生产环境", "20261105u01"} {
		if !strings.Contains(card.Text, want) {
			t.Fatalf("schedule card missing %q: %s", want, card.Text)
		}
	}
	if strings.Contains(card.Text, "下一页") || strings.Contains(card.Text, "第 1/") {
		t.Fatalf("schedule card should not paginate: %s", card.Text)
	}
}

func TestBuildScheduleListEmpty(t *testing.T) {
	card := buildScheduleListCard(nil, nil, nil)
	if !strings.Contains(card.Text, "没有待执行") {
		t.Fatalf("unexpected empty schedule card: %s", card.Text)
	}
}

func TestMDCellEscapesTableSeparator(t *testing.T) {
	if got := mdCell("a|b\nc"); got != "a\\|b c" {
		t.Fatalf("mdCell=%q", got)
	}
}
