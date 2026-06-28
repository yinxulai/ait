package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server/types"
)

// TaskDetailPage 任务详情页，展示任务配置摘要。
type TaskDetailPage struct {
	task *types.TaskOverview
	root *tview.Flex
	tv   *tview.TextView
}

// NewTaskDetailPage 创建任务详情页。
func NewTaskDetailPage(task *types.TaskOverview) *TaskDetailPage {
	p := &TaskDetailPage{task: task}

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	header.SetText(fmt.Sprintf("◆ Task Detail — %s", task.Name))

	p.tv = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	p.tv.SetBorder(true).SetTitle(" Configuration ")

	var content string
	content += fmt.Sprintf("ID:       %s\n", task.ID)
	content += fmt.Sprintf("Name:     %s\n", task.Name)
	content += fmt.Sprintf("Mode:     %s\n", task.Input.RunMode())
	content += fmt.Sprintf("Protocol: %s\n", task.Input.NormalizedProtocol())
	content += fmt.Sprintf("Model:    %s\n", task.Input.Model)
	content += fmt.Sprintf("Endpoint: %s\n", task.Input.ResolvedEndpointURL())
	content += fmt.Sprintf("Created:  %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("Updated:  %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))

	if task.Input.Concurrency > 0 {
		content += fmt.Sprintf("Concurrency: %d\n", task.Input.Concurrency)
	}
	if task.Input.Count > 0 {
		content += fmt.Sprintf("Count:       %d\n", task.Input.Count)
	}
	if task.Input.Stream {
		content += "Stream:      true\n"
	}

	if task.LatestRun != nil {
		content += "\n── Latest Run ──\n"
		content += fmt.Sprintf("Run ID:     %s\n", task.LatestRun.RunID)
		content += fmt.Sprintf("Status:     %s\n", task.LatestRun.Status)
		content += fmt.Sprintf("Success:    %.1f%%\n", task.LatestRun.SuccessRate*100)
		content += fmt.Sprintf("Avg TPS:    %.1f\n", task.LatestRun.AvgTPS)
		content += fmt.Sprintf("Avg TTFT:   %s\n", task.LatestRun.AvgTTFT)
		content += fmt.Sprintf("Cache Hit:  %.1f%%\n", task.LatestRun.CacheHitRate*100)
		dur := task.LatestRun.FinishedAt.Sub(task.LatestRun.StartedAt)
		content += fmt.Sprintf("Duration:   %s\n", dur.Truncate(1e6))
	}

	p.tv.SetText(content)

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	footer.SetText("[::b]Esc=返回  Enter=运行[::-]")

	p.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(p.tv, 0, 1, true).
		AddItem(footer, 1, 0, false)

	return p
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (p *TaskDetailPage) Name() string                   { return "task_detail" }
func (p *TaskDetailPage) Primitive() tview.Primitive      { return p.root }
func (p *TaskDetailPage) FocusTarget() tview.Primitive    { return p.tv }
func (p *TaskDetailPage) OnActivate()                     {}
func (p *TaskDetailPage) OnDeactivate()                   {}

func (p *TaskDetailPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		router.GoBack()
		return nil
	}
	return event
}
