// Package pages 包含 TUI 各页面的渲染与按键处理逻辑。
// 本包只依赖 server 和 types 包，不 import 父包 tui，避免循环依赖。
// 页面状态结构体、渲染函数、按键处理函数均定义于此。
package pages

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// NavTarget 导航目标枚举。
type NavTarget int

const (
	NavNone       NavTarget = iota
	NavTaskList             // 返回任务列表
	NavTaskDetail           // 进入任务详情（需 TaskID）
	NavWizard               // 打开向导（EditTask == nil 为新建）
	NavDashboard            // 进入仪表盘（需 RunID + TaskID）
	NavTurboDash            // 进入 Turbo 仪表盘（需 RunID + TaskID）
	NavRunDetail            // 从历史记录进入某次运行的仪表盘（需 RunID）
	NavReqDetail            // 进入请求详情（需 ReqIndex）
	NavProxy                // 进入代理配置页
	NavHelp                 // 打开帮助页
	NavQuit                 // 退出程序
)

// NavAction 页面处理函数的导航意图，由 root Model 统一处理。
type NavAction struct {
	To       NavTarget
	TaskID   string
	RunID    server.RunID
	ReqIndex int
	EditTask *types.TaskDefinition // 向导编辑模式时非空；nil 表示新建
	Summary  *types.TaskRunSummary // NavRunDetail 时，磁盘文件缺失的回退数据
}

// Client 定义 pages 包对外依赖的操作集合。
// tui.Client 实现此接口（Go duck typing）。
type Client interface {
	// 任务 CRUD
	LoadTasksCmd() tea.Cmd
	CreateTaskCmd(cfg server.TaskConfig, autoStart bool) tea.Cmd
	UpdateTaskCmd(id string, cfg server.TaskConfig) tea.Cmd
	DeleteTaskCmd(id string) tea.Cmd
	CopyTaskCmd(id string) tea.Cmd

	// 运行管理
	StartRunCmd(taskID string) tea.Cmd
	StopRunCmd(runID server.RunID) tea.Cmd

	// 历史 & 报告
	LoadHistoryCmd(taskID string, limit int) tea.Cmd
	GetRunStateCmd(runID server.RunID) tea.Cmd
	GetRunStateForHistoryCmd(runID server.RunID, summary *types.TaskRunSummary) tea.Cmd
	GenerateReportCmd(runID server.RunID, format server.ReportFormat) tea.Cmd

	// 全局配置
	SaveProxyConfigCmd(proxyURL string) tea.Cmd
	LoadProxyConfigCmd() tea.Cmd
}
