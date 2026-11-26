package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ddevops/internal/model"
)

// 钉钉卡片渲染与发送。所有消息出口都收敛在本文件，命令层只负责业务判断。
//
// 视觉基线（用户确认的旧卡片风格）：
//   - ### 标题 + --- 分隔线；
//   - #### 🔹 章节 + **加粗字段**；
//   - 环境用 <font color> 区分：测试蓝、预发橙、生产绿；
//   - 底部按钮只用于构建详情。

const (
	envColorTest    = "#0099FF"
	envColorStaging = "#FF9900"
	envColorProd    = "#00CC66"
)

func envColor(e model.Environment) string {
	switch {
	case e.IsProd:
		return envColorProd
	case strings.Contains(e.Name, "预发"):
		return envColorStaging
	default:
		return envColorTest
	}
}

func (h *DingTalkHandler) sendBuildSuccessCardMessage(webhookURL, taskID, refValue, jobURL string, env model.Environment, shortCode string) {
	card := h.makeBuildStatusCard(taskID, refValue, jobURL, "构建成功", env, "", shortCode)
	h.sendCardMessage(webhookURL, card)
}

func (h *DingTalkHandler) sendBuildFailureCardMessage(webhookURL, taskID, refValue, errorMsg, jobURL string, env model.Environment, shortCode string) {
	card := h.makeBuildStatusCard(taskID, refValue, jobURL, "构建失败", env, errorMsg, shortCode)
	h.sendCardMessage(webhookURL, card)
}

func (h *DingTalkHandler) sendUnknownTaskCardMessage(webhookURL, taskID string) {
	text := fmt.Sprintf(
		"### ❌ 未知任务ID\n\n"+
			"---\n"+
			"- **任务名称**: %s\n"+
			"- **状态**: 不存在\n"+
			"- **处理**: 发送 @机器人 帮助 查看指令",
		mdCell(taskID),
	)
	h.sendCardMessage(webhookURL, actionCard("未知任务ID", text, "", ""))
}

func (h *DingTalkHandler) sendForbiddenCardMessage(webhookURL string) {
	text := "### 🔐 权限拒绝通知\n\n" +
		"---\n" +
		"⚠️ 您当前没有使用权限。\n\n" +
		"---\n" +
		"#### 📌 可能原因：\n" +
		"- 您的账号未在许可名单中\n" +
		"- 需要联系管理员添加权限\n" +
		"---\n"
	h.sendCardMessage(webhookURL, actionCard("🚫 权限不足", text, "", ""))
}

func (h *DingTalkHandler) sendNoTaskPermissionCardMessage(webhookURL, taskID string) {
	text := fmt.Sprintf(
		"### 🔐 权限不足\n\n"+
			"---\n"+
			"- **任务名称**: %s\n"+
			"- **原因**: 不在你的可构建项目范围内\n"+
			"- **处理**: 联系管理员分配，或发送 @机器人 帮助 查看",
		mdCell(taskID),
	)
	h.sendCardMessage(webhookURL, actionCard("🚫 无任务权限", text, "", ""))
}

// sendScheduleConfirmationMessage 传入的是展示用短码（无短码时回退完整任务 ID）。
func (h *DingTalkHandler) sendScheduleConfirmationMessage(webhookURL, taskRef, envName, refValue string, scheduleTime time.Time) {
	if envName == "" {
		envName = "未配置环境"
	}
	if refValue == "" {
		refValue = "默认分支"
	}
	text := fmt.Sprintf(
		"### ⏰ 定时任务设置成功\n\n"+
			"---\n"+
			"- **项目环境**: %s\n"+
			"- **任务名称**: %s\n"+
			"- **分支版本**: %s\n"+
			"- **执行时间**: %s\n"+
			"- **任务状态**: 🟡 等待执行\n"+
			"---\n"+
			"任务将在 **%s** 自动执行。\n\n"+
			"可以 @机器人 取消定时 %s 取消此任务。",
		envName, taskRef, refValue, scheduleTime.Format("2006-01-02 15:04:05"),
		scheduleTime.Format("2006-01-02 15:04:05"), taskRef,
	)
	h.sendCardMessage(webhookURL, actionCard("✅ 定时任务已设置", text, "", ""))
}

func (h *DingTalkHandler) sendInvalidScheduleFormatMessage(webhookURL string) {
	text := "### ⚠️ 定时命令格式不正确\n\n" +
		"---\n" +
		"#### 🔹 正确格式\n" +
		"- @机器人 定时(年月日时分) 任务名称 版本号\n\n" +
		"#### 🔹 示例\n" +
		"- @机器人 定时(202611051800) P01 20261105u01\n" +
		"---\n" +
		"#### 🔹 说明\n" +
		"- 年月日时分: 202611051800\n" +
		"- 生产环境必须提供版本 Tag\n"
	h.sendCardMessage(webhookURL, actionCard("❌ 定时格式错误", text, "", ""))
}

func (h *DingTalkHandler) sendPastScheduleTimeMessage(webhookURL string, scheduleTime time.Time) {
	text := fmt.Sprintf(
		"### ⚠️ 不能设置过去的时间\n\n"+
			"---\n"+
			"- 您设置的时间 (%s) 已经过去\n\n"+
			"- 请设置一个未来的时间\n"+
			"---\n",
		scheduleTime.Format("2006-01-02 15:04:05"),
	)
	h.sendCardMessage(webhookURL, actionCard("❌ 时间设置错误", text, "", ""))
}

// sendSimpleCardMessage 标题会作为正文首行，保证卡片有明确的层级。
func (h *DingTalkHandler) sendSimpleCardMessage(webhookURL, title, text string) {
	h.sendCardMessage(webhookURL, actionCard(title, "### "+title+"\n\n---\n"+text, "", ""))
}

// sendHelpCardMessage 帮助卡片：旧版章节样式，例子用真实短码。
func (h *DingTalkHandler) sendHelpCardMessage(webhookURL string) {
	branchExample, tagExample := "T01", "P01"
	if jobs, err := h.jobRepo.List(); err == nil {
		branchFound, tagFound := false, false
		for _, j := range jobs {
			env, e := h.envRepo.Get(j.EnvID)
			if e != nil {
				continue
			}
			code := jobCode(j)
			if env.IsProd && !tagFound {
				tagExample = code
				tagFound = true
			} else if !env.IsProd && !branchFound {
				branchExample = code
				branchFound = true
			}
			if branchFound && tagFound {
				break
			}
		}
	}

	text := helpCardText(branchExample, tagExample)
	h.sendCardMessage(webhookURL, actionCard("📚 使用帮助", text, "", ""))
}

func helpCardText(branchExample, tagExample string) string {
	return fmt.Sprintf(
		"### 📚 使用帮助\n\n"+
			"---\n"+
			"#### 🔹 基本指令\n"+
			"- @机器人 [任务ID] [分支或版本号]\n"+
			"- **说明**：分支/版本号可选，不指定则使用环境默认分支；生产环境必须指定版本 Tag。\n\n"+
			"---\n"+
			"#### 🔹 环境查看\n"+
			"- @机器人 测试\n"+
			"- @机器人 预发\n"+
			"- @机器人 生产\n\n"+
			"---\n"+
			"#### 🔹 定时任务\n"+
			"- @机器人 定时(年月日时分) [任务ID] [版本号]\n"+
			"- @机器人 定时列表\n"+
			"- @机器人 取消定时 [任务ID]\n\n"+
			"---\n"+
			"#### 🔹 使用示例\n"+
			"- 分支构建: @机器人 %s\n"+
			"- 生产发布: @机器人 %s 20261105u01\n"+
			"- 定时发布: @机器人 定时(202611051800) %s 20261105u01\n",
		branchExample, tagExample, tagExample,
	)
}

func (h *DingTalkHandler) makeBuildStatusCard(taskID, refValue, jobURL, statusText string, env model.Environment, errorMsg, shortCode string) map[string]interface{} {
	envName := env.Name
	if envName == "" {
		envName = "未配置环境"
	}
	ref := refValue
	if ref == "" {
		ref = "默认分支"
	}

	titleTask := strings.Join(strings.Fields(taskID), " ")
	if titleTask == "" {
		titleTask = shortCode
	}
	header := fmt.Sprintf("### %s %s", titleTask, statusText)
	if statusText == "构建失败" {
		header = "### ❌ " + titleTask + " 构建失败"
	}

	var b strings.Builder
	b.WriteString(header + "\n\n---\n")
	fmt.Fprintf(&b, "- **项目环境**: <font color='%s'>%s</font>\n", envColor(env), envName)
	fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(taskID))
	fmt.Fprintf(&b, "- **项目分支**: %s\n", mdCell(ref))
	fmt.Fprintf(&b, "- **触发时间**: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	b.WriteString("---\n")
	if statusText == "构建失败" && errorMsg != "" {
		b.WriteString("\n**错误详情**\n")
		b.WriteString(briefError(errorMsg) + "\n")
	}

	title := fmt.Sprintf("%s %s", titleTask, statusText)
	return actionCard(title, b.String(), "👉 查看任务详情", jobURL)
}

func actionCard(title, text, buttonTitle, buttonURL string) map[string]interface{} {
	action := map[string]interface{}{
		"title": title,
		"text":  text,
	}
	if buttonTitle != "" && buttonURL != "" {
		action["btnOrientation"] = "0"
		action["singleTitle"] = buttonTitle
		action["singleURL"] = buttonURL
	}
	return map[string]interface{}{
		"msgtype":    "actionCard",
		"actionCard": action,
	}
}

func (h *DingTalkHandler) sendCardMessage(webhookURL string, card map[string]interface{}) {
	if webhookURL == "" {
		log.Println("发送卡片消息失败：webhookURL为空")
		return
	}
	jsonData, err := json.Marshal(card)
	if err != nil {
		log.Printf("JSON编码失败: %v", err)
		return
	}
	resp, err := http.Post(webhookURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("发送卡片消息失败: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("钉钉卡片消息响应状态: %d", resp.StatusCode)
}

func briefError(msg string) string {
	text := strings.Join(strings.Fields(msg), " ")
	if runes := []rune(text); len(runes) > 160 {
		return string(runes[:160]) + "…"
	}
	return text
}
