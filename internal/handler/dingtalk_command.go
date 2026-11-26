package handler

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ddevops/internal/model"

	"github.com/google/uuid"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
)

// 定时相关命令：定时 / 定时列表 / 取消定时。
// 命令解析与业务校验在本文件，消息渲染在 dingtalk_card.go。

// 处理定时任务列表
func (h *DingTalkHandler) handleListScheduledTasks(data *chatbot.BotCallbackDataModel, scope map[string]bool, restricted bool) {
	tasks, err := h.taskRepo.List("pending", "")
	if err != nil {
		log.Printf("获取定时任务失败: %v", err)
		h.sendSimpleCardMessage(data.SessionWebhook, "❌ 获取失败", "- **对象**: 定时任务列表\n- **处理**: 请稍后重试")
		return
	}

	// 与 Web 端一致：只看自己创建的，同时叠加任务范围约束。
	own := make([]model.ScheduledTask, 0, len(tasks))
	for _, t := range tasks {
		if t.CreatedBy != data.SenderStaffId {
			continue
		}
		if restricted && !scope[t.TaskID] {
			continue
		}
		own = append(own, t)
	}
	tasks = own

	if len(tasks) == 0 {
		h.sendSimpleCardMessage(data.SessionWebhook, "定时任务", "当前没有待执行的定时任务。")
		return
	}

	// 按时间排序，分页逻辑交给统一的列表渲染。
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ScheduleTime.Before(tasks[j].ScheduleTime) })

	card := buildScheduleListCard(tasks, h.envNameMap(), h.jobCodeMap())
	h.sendCardMessage(data.SessionWebhook, actionCard(card.Title, card.Text, "", ""))
}

// 处理定时命令
func (h *DingTalkHandler) handleScheduledCommand(data *chatbot.BotCallbackDataModel, command string, scope map[string]bool, restricted bool) {
	re := regexp.MustCompile(`定时\((\d{4})(\d{2})(\d{2})(\d{2})?(\d{2})?\)\s+([^\s]+)\s*(.*)?`)
	matches := re.FindStringSubmatch(command)

	if len(matches) < 7 {
		h.sendInvalidScheduleFormatMessage(data.SessionWebhook)
		return
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])
	hour, minute := 0, 0
	if matches[4] != "" {
		hour, _ = strconv.Atoi(matches[4])
	}
	if matches[5] != "" {
		minute, _ = strconv.Atoi(matches[5])
	}

	scheduleTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	taskID := h.resolveJobID(matches[6])
	refValue := ""
	if len(matches) > 7 {
		refValue = matches[7]
	}

	if scheduleTime.Before(time.Now()) {
		h.sendPastScheduleTimeMessage(data.SessionWebhook, scheduleTime)
		return
	}

	// 查找任务
	job, err := h.jobRepo.Get(taskID)
	if err != nil {
		h.sendUnknownTaskCardMessage(data.SessionWebhook, taskID)
		return
	}

	// 任务级授权：受限用户只能为授权范围内的任务创建定时计划。
	if restricted && !scope[taskID] {
		h.sendNoTaskPermissionCardMessage(data.SessionWebhook, taskID)
		return
	}

	// 检查是否已存在相同任务ID的待执行定时任务
	tasks, _ := h.taskRepo.List("pending", "")
	for _, t := range tasks {
		if t.TaskID == taskID {
			code := jobCode(job)
			var b strings.Builder
			fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(job.ID))
			b.WriteString("- **状态**: 已存在待执行的定时计划\n")
			fmt.Fprintf(&b, "- **取消**: @机器人 取消定时 %s\n", mdCell(code))
			h.sendSimpleCardMessage(data.SessionWebhook, "⏰ 定时任务已存在", b.String())
			return
		}
	}

	// 环境名实时取自环境配置，任务表不再保存环境名副本。
	envName := ""
	if env, err := h.envRepo.Get(job.EnvID); err == nil {
		envName = env.Name
	}
	if envName == "" {
		envName = "未配置环境"
	}

	task := model.ScheduledTask{
		ID:           uuid.New().String(),
		TaskID:       taskID,
		RefValue:     refValue,
		ScheduleTime: scheduleTime,
		CreatedBy:    data.SenderStaffId,
		CreatedName:  data.SenderNick,
		Status:       "pending",
		WebhookURL:   data.SessionWebhook,
	}

	// 保存到数据库
	if err := h.taskRepo.Create(task); err != nil {
		log.Printf("保存定时任务失败: %v", err)
		h.sendSimpleCardMessage(data.SessionWebhook, "❌ 保存失败", "- **状态**: 保存定时任务失败\n- **处理**: 请稍后重试")
		return
	}

	// 发送确认消息
	h.sendScheduleConfirmationMessage(data.SessionWebhook, job.ID, envName, refValue, scheduleTime)
}

// 处理取消定时任务
func (h *DingTalkHandler) handleCancelScheduledTask(data *chatbot.BotCallbackDataModel, parts []string) {
	if len(parts) < 2 {
		var b strings.Builder
		b.WriteString("- **问题**: 缺少要取消的任务名称\n")
		b.WriteString("- **正确格式**: @机器人 取消定时 线上-8080\n")
		h.sendSimpleCardMessage(data.SessionWebhook, "⚠️ 取消定时格式错误", b.String())
		return
	}
	taskToCancelID := h.resolveJobID(parts[1])

	// 查找该任务ID对应的待执行定时任务
	tasks, _ := h.taskRepo.List("pending", "")
	var foundTask *model.ScheduledTask

	for _, task := range tasks {
		if task.TaskID == taskToCancelID {
			foundTask = &task
			break
		}
	}

	if foundTask == nil {
		var b strings.Builder
		fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(taskToCancelID))
		b.WriteString("- **状态**: 没有待执行的定时计划\n")
		b.WriteString("- **处理**: 发送 @机器人 定时列表 查看\n")
		h.sendSimpleCardMessage(data.SessionWebhook, "❌ 未找到任务", b.String())
		return
	}

	if foundTask.CreatedBy != data.SenderStaffId {
		var b strings.Builder
		b.WriteString("- **原因**: 只能取消自己创建的定时任务\n")
		b.WriteString("- **处理**: 联系创建人或管理员\n")
		h.sendSimpleCardMessage(data.SessionWebhook, "🚫 权限不足", b.String())
		return
	}

	// 取消任务
	affected, err := h.taskRepo.Cancel(foundTask.ID)
	if err != nil {
		log.Printf("取消任务失败: %v", err)
		h.sendSimpleCardMessage(data.SessionWebhook, "❌ 取消失败", "- **状态**: 取消操作未成功\n- **处理**: 请稍后重试")
		return
	}
	if affected == 0 {
		h.sendSimpleCardMessage(data.SessionWebhook, "⚠️ 取消未生效", "- **状态**: 该定时任务已不在待执行状态\n- **处理**: 发送 @机器人 定时列表 查看")
		return
	}

	// 获取任务信息：环境名实时取自环境配置
	envName := h.envNameMap()[foundTask.TaskID]
	if envName == "" {
		envName = "未配置环境"
	}

	ref := foundTask.RefValue
	if ref == "" {
		ref = "默认分支"
	}
	name := foundTask.TaskID
	if name == "" {
		name = parts[1]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "- **项目环境**: <font color='%s'>%s</font>\n", colorByName(envName), mdCell(envName))
	fmt.Fprintf(&b, "- **任务名称**: %s\n", mdCell(name))
	fmt.Fprintf(&b, "- **项目分支**: %s\n", mdCell(ref))
	fmt.Fprintf(&b, "- **计划时间**: %s\n", foundTask.ScheduleTime.Format("2006-01-02 15:04"))
	b.WriteString("- **状态**: 已取消\n")
	h.sendSimpleCardMessage(data.SessionWebhook, "🟢 定时任务已取消", b.String())
}
