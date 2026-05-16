package tui

import "github.com/yinxulai/ait/internal/types"

type progressMsg struct {
	stats types.StatsData
}

type runCompleteMsg struct {
	taskID      string
	result      *types.ReportData
	reportPaths []string
}

type turboCompleteMsg struct {
	taskID string
	result *types.TurboResult
}

type asyncErrorMsg struct {
	err error
}

type requestLogMsg struct {
	entry string
}
