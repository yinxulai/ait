// Package i18n provides simple Chinese/English UI string translations.
// Default language is ZH (Chinese). Call SetLang(EN) to switch to English.
//
// Alignment problem: TUI label groups (e.g. metrics panel) pad labels to equal
// display width. Because CJK chars are 2 display columns wide, use
// DisplayWidth() to measure and maxLabelWidth() in helpers.go to auto-compute.
package i18n

import (
	"sync/atomic"

	"github.com/mattn/go-runewidth"
)

// Lang represents a supported language.
type Lang int32

const (
	ZH Lang = iota // Chinese (default)
	EN             // English
)

var active atomic.Int32

// SetLang switches the active UI language.
func SetLang(l Lang) { active.Store(int32(l)) }

// Active returns the currently active language.
func Active() Lang { return Lang(active.Load()) }

// Key is a typed string resource key.
type Key int

const (
	// ─── Hotkeys ─────────────────────────────────────────────────────────────
	KHelp Key = iota
	KViewDetails
	KRun
	KNewTask
	KEdit
	KDelete
	KCopy
	KDuplicateTask
	KProxyConfig
	KStop
	KSelectRecord
	KViewRunDetails
	KRunAgain
	KExportJSONReport
	KExportHistoryJSON
	KGoToLiveDash
	KSwitchField
	KSwitchProtocol
	KNextStep
	KBackToList
	KSwitchOption
	KGoBack
	KScroll
	KPageTurn
	KSave
	KSaveAndRun
	KBackToEdit
	KGenerateReport
	KViewRequest
	KSelectRequest
	KViewLevelReqs
	KSelectItem
	KBackToDash
	KPrevNextReq
	KSwitchType
	KClear
	KTopBottom
	KConfirmDelete
	KCancel

	// ─── Hint texts (used in NewPageHotkeys hints) ────────────────────────────
	KHintQuit      // [q] 退出
	KHintCtrlCQuit // [Ctrl+C] 退出
	KHintBack      // [b/Esc] 返回
	KHintEscBack   // [Esc] 返回
	KHintGoBack    // [b/Esc] 返回上一页
	KHintSelect    // [↑↓] 选择
	KHintNew       // [a] 新建

	// ─── Metric labels ───────────────────────────────────────────────────────
	KSuccessRate
	KAvgTPS
	KAvgTTFT
	KCacheHit
	KRPM
	KTPM
	KStatus
	KTotalTime
	KTTFT
	KOutputTPS
	KToken
	KCache
	KError
	KDNS
	KTCPConnect
	KTLSHandshake
	KTargetIP

	// ─── Status values ───────────────────────────────────────────────────────
	KRunning
	KCompleted
	KRunFailed
	KStopped
	KWaitingStatus

	// ─── Dashboard / progress ─────────────────────────────────────────────────
	KProgress
	KSuccessCount
	KFailureCount
	KWaitingDots
	KRamp
	KPerLevel
	KStopCondLabel
	KTurboMode
	KStandardMode
	KTurboMonitor
	KTurboModeMeta
	KSuccessRateFmt   // "成功率 %.1f%%"
	KTurboCurLevelFmt // "当前级别实时指标 [并发 = %d]"
	KTurboDashSuffix  // "  %d/%d  并发 %d  进度 %s 级"

	// ─── Layout ──────────────────────────────────────────────────────────────
	KWindowTooSmall
	KWaitingData
	KNotRecorded
	KScrollMore
	KTerminalLabel
	KInProgress
	KNoHotkeys // "当前页暂无快捷操作"

	// ─── Page subtitles ───────────────────────────────────────────────────────
	KStdMonitorTitle    // "标准运行监控"
	KStdMonitorSubtitle // "实时查看运行进度、吞吐和单请求明细"
	KTurboSubtitle      // "观察并发爬坡过程、级别指标与稳定区间"
	KTaskListSubtitle   // "创建任务、运行压测、查看执行记录与导出报告"
	KTaskDetailSubtitle // "查看任务配置、当前运行状态与历史记录"
	KReqDetailSubtitle  // "查看单次请求的耗时、网络阶段和完整报文"
	KConcFmt            // "并发%d"

	// ─── TaskDetail / ReqDetail fields ───────────────────────────────────────
	KProtocol
	KEndpoint
	KProxy
	KModel
	KMode
	KConcurrency
	KStepLabel
	KRequests
	KTimeout
	KStream
	KPromptLabel
	KNoRunRecords
	KRecordDetails
	KStart
	KEnd
	KElapsed
	KErrorSummary
	KRequestBody
	KResponseBody

	// ─── Table column headers ─────────────────────────────────────────────────
	KColTime
	KTime // alias for table display
	KColMode
	KColProtocol
	KColSuccessRate
	KColCacheHit
	KColAvgTTFT
	KColAvgTPS
	KColLevel
	KColInput
	KColOutput

	// ─── TaskList ────────────────────────────────────────────────────────────
	KTaskName
	KTaskID
	KLastRun
	KIrreversible
	KNoTasks
	KRunHistory
	KTaskCenter
	KNoRunHistory
	KConfirmDeletePrompt

	// ─── Proxy ───────────────────────────────────────────────────────────────
	KExSOCKS5
	KExSSH
	KExHTTP
	KProxySubtitle
	KProxySaveHint
	KGlobalConfig
	KProxyType
	KProxyURL

	// ─── Help page ───────────────────────────────────────────────────────────
	KHelpTitle
	KHelpSubtitle
	KHelpMeta
	KHelpSecConcepts
	KHelpSecMetrics
	KHelpSecProtocols
	KHelpSecGlobal
	KHelpSecTaskList
	KHelpSecTaskDetail
	KHelpSecDashboard
	KHelpSecExport
	KHelpTermTask
	KHelpDescTask
	KHelpTermRun
	KHelpDescRun
	KHelpTermStandard
	KHelpDescStandard
	KHelpTermTurboMode
	KHelpDescTurboMode
	KHelpTermTPS
	KHelpDescTPS
	KHelpTermAvgTPS
	KHelpDescAvgTPS
	KHelpTermTTFT
	KHelpDescTTFT
	KHelpTermAvgTTFT
	KHelpDescAvgTTFT
	KHelpTermSuccessRate
	KHelpDescSuccessRate
	KHelpTermCacheHit
	KHelpDescCacheHit
	KHelpTermConcurrencyTurbo
	KHelpDescConcurrencyTurbo
	KHelpTermOpenAI
	KHelpDescOpenAI
	KHelpTermAnthropic
	KHelpDescAnthropic
	KHelpTermQuit
	KHelpDescQuit
	KHelpTermQuestionMark
	KHelpDescQuestionMark
	KHelpTermBack
	KHelpDescBack
	KHelpTermLangToggle
	KHelpDescLangToggle
	KHelpTermSelectTask
	KHelpDescSelectTask
	KHelpTermEnterDetail
	KHelpDescEnterDetail
	KHelpTermRunTask
	KHelpDescRunTask
	KHelpTermStopTask
	KHelpDescStopTask
	KHelpTermNewTask
	KHelpDescNewTask
	KHelpTermEditTask
	KHelpDescEditTask
	KHelpTermDeleteTask
	KHelpDescDeleteTask
	KHelpTermDuplicateTask
	KHelpDescDuplicateTask
	KHelpTermProxy
	KHelpDescProxy
	KHelpTermSelectHistory
	KHelpDescSelectHistory
	KHelpTermEnterDash
	KHelpDescEnterDash
	KHelpTermRunAgain
	KHelpDescRunAgain
	KHelpTermExport
	KHelpDescExport
	KHelpTermEditConfig
	KHelpDescEditConfig
	KHelpTermDuplicateTask2
	KHelpDescDuplicateTask2
	KHelpTermDeleteTask2
	KHelpDescDeleteTask2
	KHelpTermSelectReq
	KHelpDescSelectReq
	KHelpTermViewReq
	KHelpDescViewReq
	KHelpTermStopDash
	KHelpDescStopDash
	KHelpTermGenerateReport
	KHelpDescGenerateReport
	KHelpTermBackDash
	KHelpDescBackDash
	KHelpTermJSONReport
	KHelpDescJSONReport
	KHelpTermCSVReport
	KHelpDescCSVReport

	// ─── Wizard ──────────────────────────────────────────────────────────────
	KWzTaskName
	KWzProtocol
	KWzEndpoint
	KWzAPIKey
	KWzTestModel
	KWzTestMode
	KWzTurboMode
	KWzStandardMode
	KWzConcurrency
	KWzTotalRequests
	KWzTimeoutSecs
	KWzInitConc
	KWzMaxConc
	KWzStepSize
	KWzLevelReqs
	KWzMinSuccessRate
	KWzStreamMode
	KWzInputMode
	KWzInputDirect
	KWzInputFile
	KWzInputGenerated
	KWzInputRaw
	KWzPromptConfig
	KWzSelectModeHint
	KWzTurboModeLabel
	KWzStepFmt // "步骤 %d/3"
	KWzStep1Label
	KWzStep2Label
	KWzStep3Label
	KWzStep1Desc
	KWzStep2Desc
	KWzStep3Desc
	KWzUntitled
	KWzNotFilled
	KWzExecParams
	KWzConcurrencyRamp
	KWzStopCondition
	KWzTimeoutLabel
	KWzContentSummary
	KWzBodyBytes
	KWzSaveLocation
	KWzPromptSection
	KWzHintDirect
	KWzHintFile
	KWzHintRaw
	KWzHintCacheToken
	KWzHintRawBody
	KWzJSONBody
	KWzPromptLabelShort
	KWzRAWBody
	KWzFileSummary
	KWzGeneratedFmt   // "生成 %d 字符"
	KWzPromptContent  // field label for prompt content input
	KWzNoConfirmItems // "暂无确认项"
	KWzConfirmRange   // "确认项 %d-%d/%d"
	KWzConfirmTotal   // "共 %d 项待确认"
	KWzNoFields       // "暂无配置项"
	KWzFieldProgress  // "当前字段 %d/%d"

	// ─── Misc ────────────────────────────────────────────────────────────────
	KEnabled
	KDisabled
	KFileSummaryPfx // "文件: "
	KNotSet         // "(未设置)"
	KJustNow        // "刚刚"
	KMinutesAgoFmt  // "%d 分钟前"
	KHoursAgoFmt    // "%d 小时前"
	KDaysAgoFmt     // "%d 天前"
	KToggleLang     // "切换语言" / "Toggle Lang"
)

var translations = [2]map[Key]string{
	ZH: {
		// Hotkeys
		KHelp:              "帮助",
		KViewDetails:       "查看详情",
		KRun:               "运行",
		KNewTask:           "新建任务",
		KEdit:              "编辑",
		KDelete:            "删除",
		KCopy:              "复制",
		KDuplicateTask:     "复制任务",
		KProxyConfig:       "代理配置",
		KStop:              "停止",
		KSelectRecord:      "选择记录",
		KViewRunDetails:    "查看运行详情",
		KRunAgain:          "再次运行",
		KExportJSONReport:  "导出 JSON 报告",
		KExportHistoryJSON: "导出历史 JSON",
		KGoToLiveDash:      "进入运行中仪表盘",
		KSwitchField:       "切换字段",
		KSwitchProtocol:    "切换协议",
		KNextStep:          "下一步",
		KBackToList:        "返回列表",
		KSwitchOption:      "切换选项",
		KGoBack:            "返回上一步",
		KScroll:            "滚动",
		KPageTurn:          "翻页",
		KSave:              "保存",
		KSaveAndRun:        "保存并运行",
		KBackToEdit:        "返回修改",
		KGenerateReport:    "生成报告",
		KViewRequest:       "查看请求详情",
		KSelectRequest:     "选择请求",
		KViewLevelReqs:     "查看该级别请求",
		KSelectItem:        "选择",
		KBackToDash:        "返回仪表盘",
		KPrevNextReq:       "上/下一条请求",
		KSwitchType:        "切换类型",
		KClear:             "清空",
		KTopBottom:         "顶部/底部",
		KConfirmDelete:     "确认删除",
		KCancel:            "取消",

		// Hints
		KHintQuit:      "[q] 退出",
		KHintCtrlCQuit: "[Ctrl+C] 退出",
		KHintBack:      "[b/Esc] 返回",
		KHintEscBack:   "[Esc] 返回",
		KHintGoBack:    "[b/Esc] 返回上一页",
		KHintSelect:    "[↑↓] 上下切换",
		KHintNew:       "[a] 创建任务",

		// Metric labels
		KSuccessRate:  "成功率",
		KAvgTPS:       "TPS均值",
		KAvgTTFT:      "TTFT均值",
		KCacheHit:     "缓存命中",
		KRPM:          "RPM",
		KTPM:          "TPM",
		KStatus:       "状态",
		KTotalTime:    "总耗时",
		KTTFT:         "TTFT",
		KOutputTPS:    "输出TPS",
		KToken:        "Token",
		KCache:        "缓存",
		KError:        "错误",
		KDNS:          "DNS",
		KTCPConnect:   "TCP 连接",
		KTLSHandshake: "TLS 握手",
		KTargetIP:     "目标 IP",

		// Status values
		KRunning:       "运行中",
		KCompleted:     "已完成",
		KRunFailed:     "运行失败",
		KStopped:       "已停止",
		KWaitingStatus: "等待数据",

		// Dashboard / progress
		KProgress:         "进度",
		KSuccessCount:     "成功",
		KFailureCount:     "失败",
		KWaitingDots:      "等待中...",
		KRamp:             "爬坡",
		KPerLevel:         "每级",
		KStopCondLabel:    "停止",
		KTurboMode:        "Turbo 模式",
		KStandardMode:     "标准",
		KTurboMonitor:     "Turbo 探测监控",
		KTurboModeMeta:    "Turbo 模式",
		KSuccessRateFmt:   "成功率 %.1f%%",
		KTurboCurLevelFmt: "当前级别实时指标 [并发 = %d]",
		KTurboDashSuffix:  "  %d/%d  并发 %d  进度 %s 级",

		// Layout
		KWindowTooSmall:     "窗口过小 ↔ 请放大终端",
		KWaitingData:        "等待数据...",
		KNotRecorded:        "(未记录)",
		KScrollMore:         "↑↓ 滚动查看完整内容",
		KTerminalLabel:      "终端",
		KInProgress:         "进行中",
		KNoHotkeys:          "当前页暂无快捷操作",
		KStdMonitorTitle:    "标准运行监控",
		KStdMonitorSubtitle: "实时查看运行进度、吸吐和单请求明细",
		KTurboSubtitle:      "观察并发爬坡过程、级别指标与稳定区间",
		KTaskListSubtitle:   "创建任务、运行压测、查看执行记录与导出报告",
		KTaskDetailSubtitle: "查看任务配置、当前运行状态与历史记录",
		KReqDetailSubtitle:  "查看单次请求的耗时、网络阶段和完整报文",
		KConcFmt:            "并发%d",

		// Fields
		KProtocol:      "协议",
		KEndpoint:      "接口",
		KProxy:         "代理",
		KModel:         "模型",
		KMode:          "模式",
		KConcurrency:   "并发",
		KStepLabel:     "步进",
		KRequests:      "请求",
		KTimeout:       "超时",
		KStream:        "流式",
		KPromptLabel:   "Prompt",
		KNoRunRecords:  "暂无运行记录",
		KRecordDetails: "记录详情",
		KStart:         "开始",
		KEnd:           "结束",
		KElapsed:       "耗时",
		KErrorSummary:  "错误摘要",
		KRequestBody:   "请求体 (Request Body)",
		KResponseBody:  "响应体 (Response Body)",

		// Table column headers
		KColTime:        "时间",
		KTime:           "时间",
		KColMode:        "模式",
		KColProtocol:    "协议",
		KColSuccessRate: "成功率",
		KColCacheHit:    "缓存命中",
		KColAvgTTFT:     "均值TTFT",
		KColAvgTPS:      "均值TPS",
		KColLevel:       "级别",
		KColInput:       "输入",
		KColOutput:      "输出",

		// TaskList
		KTaskName:            "任务名称",
		KTaskID:              "任务 ID",
		KLastRun:             "上次运行",
		KIrreversible:        "此操作不可恢复，任务的历史运行记录将一并删除。",
		KNoTasks:             "暂无任务  按 [a] 新建第一个任务",
		KRunHistory:          "历史运行记录",
		KTaskCenter:          "任务中心",
		KNoRunHistory:        "暂无运行历史",
		KConfirmDeletePrompt: "确认删除任务？",

		// Proxy
		KExSOCKS5:      "示例: socks5://127.0.0.1:1080",
		KExSSH:         "示例: ssh://user@host:22",
		KExHTTP:        "示例: http://127.0.0.1:7890",
		KProxySubtitle: "设置全局 HTTP 代理，适用于所有任务的请求。留空则使用系统环境变量或直连。",
		KProxySaveHint: "配置保存至 ~/.ait/config.json，重启无需重新输入。",
		KGlobalConfig:  "全局配置",
		KProxyType:     "代理类型",
		KProxyURL:      "代理地址",

		// Help page
		KHelpTitle:         "帮助",
		KHelpSubtitle:      "AIT — AI 接口压测工具概念说明与操作指南",
		KHelpMeta:          "帮助",
		KHelpSecConcepts:   "核心概念",
		KHelpSecMetrics:    "性能指标",
		KHelpSecProtocols:  "协议支持",
		KHelpSecGlobal:     "快捷键 — 全局",
		KHelpSecTaskList:   "快捷键 — 任务列表",
		KHelpSecTaskDetail: "快捷键 — 任务详情",
		KHelpSecDashboard:  "快捷键 — 运行仪表盘",
		KHelpSecExport:     "报告导出",

		KHelpTermTask:      "任务 (Task)",
		KHelpDescTask:      "一组压测配置的集合，包含目标接口、模型、并发数、请求数等参数。任务可多次运行，每次运行独立记录结果。",
		KHelpTermRun:       "运行 (Run)",
		KHelpDescRun:       "任务的一次具体执行。每次运行产生独立的指标数据和请求记录，可导出为 JSON/CSV 报告。",
		KHelpTermStandard:  "标准模式",
		KHelpDescStandard:  "以固定并发数执行所有请求，适合衡量稳定负载下的接口性能。",
		KHelpTermTurboMode: "Turbo 模式",
		KHelpDescTurboMode: "自动从低并发逐步爬坡，找出接口在保持成功率要求下能承受的最大稳定并发数。",

		KHelpTermTPS:     "TPS",
		KHelpDescTPS:     "Tokens Per Second，每秒输出 Token 数，衡量模型的文本生成速率。",
		KHelpTermAvgTPS:  "均值TPS",
		KHelpDescAvgTPS:  "本次运行中所有请求的 TPS 均值，反映整体吞吐水平。",
		KHelpTermTTFT:    "TTFT",
		KHelpDescTTFT:    "Time To First Token，从发送请求到收到第一个 Token 的耗时，衡量模型响应延迟。",
		KHelpTermAvgTTFT: "均值TTFT",
		KHelpDescAvgTTFT: "本次运行中所有请求的 TTFT 均值。",

		KHelpTermSuccessRate:      "成功率",
		KHelpDescSuccessRate:      "成功完成的请求数占总请求数的百分比。失败包括超时、HTTP 错误、模型返回错误等。",
		KHelpTermCacheHit:         "缓存命中",
		KHelpDescCacheHit:         "请求中使用了 KV 缓存（Prompt Cache）的比例。命中缓存可显著降低 TTFT 和推理成本。该指标为二值统计：单次请求若有任何 Token 命中缓存则计为命中。",
		KHelpTermConcurrencyTurbo: "并发（Turbo）",
		KHelpDescConcurrencyTurbo: "Turbo 模式下找到的最大稳定并发数，即在满足最低成功率要求的前提下能同时维持的请求数。",

		KHelpTermOpenAI:    "OpenAI",
		KHelpDescOpenAI:    "兼容 OpenAI Chat Completions API（/v1/chat/completions），支持流式和非流式响应。",
		KHelpTermAnthropic: "Anthropic",
		KHelpDescAnthropic: "兼容 Anthropic Messages API（/v1/messages），支持流式和非流式响应。",

		KHelpTermQuit:         "q / Ctrl+C",
		KHelpDescQuit:         "退出程序。",
		KHelpTermQuestionMark: "?",
		KHelpDescQuestionMark: "打开此帮助页。",
		KHelpTermBack:         "b / Esc",
		KHelpDescBack:         "返回上一页。",
		KHelpTermLangToggle:   "F2",
		KHelpDescLangToggle:   "切换界面语言（中文 / 英文）。",

		KHelpTermSelectTask:    "↑↓ / j k",
		KHelpDescSelectTask:    "选择任务。",
		KHelpTermEnterDetail:   "Enter",
		KHelpDescEnterDetail:   "进入任务详情页。",
		KHelpTermRunTask:       "r",
		KHelpDescRunTask:       "立即运行选中任务。",
		KHelpTermStopTask:      "s",
		KHelpDescStopTask:      "停止正在运行的任务（仅任务运行中可用）。",
		KHelpTermNewTask:       "a",
		KHelpDescNewTask:       "新建任务（打开向导）。",
		KHelpTermEditTask:      "e",
		KHelpDescEditTask:      "编辑选中任务配置。",
		KHelpTermDeleteTask:    "d",
		KHelpDescDeleteTask:    "删除选中任务（需确认）。",
		KHelpTermDuplicateTask: "y",
		KHelpDescDuplicateTask: "复制选中任务（生成副本）。",
		KHelpTermProxy:         "p",
		KHelpDescProxy:         "打开代理配置页。",

		KHelpTermSelectHistory:  "↑↓ / j k",
		KHelpDescSelectHistory:  "在历史运行记录中选择条目。",
		KHelpTermEnterDash:      "Enter",
		KHelpDescEnterDash:      "查看选中运行的仪表盘；若任务正在运行，进入实时仪表盘。",
		KHelpTermRunAgain:       "r",
		KHelpDescRunAgain:       "再次运行该任务（无正在运行的实例时可用）。",
		KHelpTermExport:         "g",
		KHelpDescExport:         "将选中的历史运行导出为 JSON 报告。",
		KHelpTermEditConfig:     "e",
		KHelpDescEditConfig:     "编辑任务配置。",
		KHelpTermDuplicateTask2: "y",
		KHelpDescDuplicateTask2: "复制任务。",
		KHelpTermDeleteTask2:    "d",
		KHelpDescDeleteTask2:    "删除任务。",

		KHelpTermSelectReq:      "↑↓ / j k",
		KHelpDescSelectReq:      "选择请求条目。",
		KHelpTermViewReq:        "Enter",
		KHelpDescViewReq:        "查看选中请求的详情（耗时、Token、响应体等）。",
		KHelpTermStopDash:       "s",
		KHelpDescStopDash:       "停止正在运行的任务。",
		KHelpTermGenerateReport: "r",
		KHelpDescGenerateReport: "生成 JSON 报告（运行结束后可用）。",
		KHelpTermBackDash:       "b / Esc",
		KHelpDescBackDash:       "返回任务详情页。",

		KHelpTermJSONReport: "JSON 报告",
		KHelpDescJSONReport: "完整记录每次请求的所有指标、请求/响应体，适合程序化分析。",
		KHelpTermCSVReport:  "CSV 报告",
		KHelpDescCSVReport:  "表格形式的汇总数据，可直接在电子表格中打开。报告默认保存在当前工作目录。",

		// Wizard fields
		KWzTaskName:         "任务名称",
		KWzProtocol:         "协议类型",
		KWzEndpoint:         "接口地址",
		KWzAPIKey:           "API 密钥",
		KWzTestModel:        "测试模型",
		KWzTestMode:         "测试模式",
		KWzTurboMode:        "Turbo 模式",
		KWzStandardMode:     "标准模式",
		KWzConcurrency:      "并发数",
		KWzTotalRequests:    "请求总数",
		KWzTimeoutSecs:      "超时(秒)",
		KWzInitConc:         "初始并发",
		KWzMaxConc:          "最大并发",
		KWzStepSize:         "步进值",
		KWzLevelReqs:        "每级请求数",
		KWzMinSuccessRate:   "最低成功率",
		KWzStreamMode:       "流式模式",
		KWzInputMode:        "输入方式",
		KWzInputDirect:      "直接输入",
		KWzInputFile:        "文件",
		KWzInputGenerated:   "按长度生成",
		KWzInputRaw:         "RAW 请求体",
		KWzPromptConfig:     "Prompt 配置",
		KWzSelectModeHint:   "选择压测模式，并补全并发与 Prompt 参数。",
		KWzTurboModeLabel:   "Turbo 模式",
		KWzStepFmt:          "步骤 %d/3",
		KWzStep1Label:       "1 基本信息",
		KWzStep2Label:       "2 测试参数",
		KWzStep3Label:       "3 确认保存",
		KWzStep1Desc:        "配置任务名称、模型协议和连接信息。",
		KWzStep2Desc:        "选择压测模式，并补全并发与 Prompt 参数。",
		KWzStep3Desc:        "保存前快速检查关键配置。",
		KWzUntitled:         "未命名任务",
		KWzNotFilled:        "未填写",
		KWzExecParams:       "执行参数",
		KWzConcurrencyRamp:  "并发爬坡",
		KWzStopCondition:    "停止条件",
		KWzTimeoutLabel:     "超时",
		KWzContentSummary:   "内容摘要",
		KWzBodyBytes:        "Body 字节数",
		KWzSaveLocation:     "保存位置",
		KWzPromptSection:    "Prompt",
		KWzHintDirect:       "直接粘贴或输入 Prompt 文本，所有请求共享同一段内容",
		KWzHintFile:         "从文件读取 Prompt，支持通配符匹配多个文件（请求按文件轮换）",
		KWzHintRaw:          "粘贴完整的 HTTP 请求 JSON Body，将跳过参数组装直接发送",
		KWzHintCacheToken:   "提示：大多数服务需要 ≥ 1024 tokens 才能命中缓存",
		KWzHintRawBody:      "提示：粘贴 API 请求的完整 JSON Body，将直接作为 HTTP 请求体发送",
		KWzJSONBody:         "JSON Body",
		KWzPromptLabelShort: "Prompt",
		KWzRAWBody:          "RAW 请求体",
		KWzFileSummary:      "文件",
		KWzGeneratedFmt:     "生成 %d 字符",
		KWzPromptContent:    "内容",
		KWzNoConfirmItems:   "暂无确认项",
		KWzConfirmRange:     "确认项 %d-%d/%d",
		KWzConfirmTotal:     "共 %d 项待确认",
		KWzNoFields:         "暂无配置项",
		KWzFieldProgress:    "当前字段 %d/%d",

		// Misc
		KEnabled:        "开启",
		KDisabled:       "关闭",
		KFileSummaryPfx: "文件: ",
		KNotSet:         "(未设置)",
		KJustNow:        "刚刚",
		KMinutesAgoFmt:  "%d 分钟前",
		KHoursAgoFmt:    "%d 小时前",
		KDaysAgoFmt:     "%d 天前",
		KToggleLang:     "切换语言",
	},
	EN: {
		// Hotkeys
		KHelp:              "Help",
		KViewDetails:       "View Details",
		KRun:               "Run",
		KNewTask:           "New Task",
		KEdit:              "Edit",
		KDelete:            "Delete",
		KCopy:              "Copy",
		KDuplicateTask:     "Copy Task",
		KProxyConfig:       "Proxy Config",
		KStop:              "Stop",
		KSelectRecord:      "Select Record",
		KViewRunDetails:    "View Run Details",
		KRunAgain:          "Run Again",
		KExportJSONReport:  "Export JSON Report",
		KExportHistoryJSON: "Export History JSON",
		KGoToLiveDash:      "Live Dashboard",
		KSwitchField:       "Switch Field",
		KSwitchProtocol:    "Switch Protocol",
		KNextStep:          "Next",
		KBackToList:        "Back to List",
		KSwitchOption:      "Switch Option",
		KGoBack:            "Go Back",
		KScroll:            "Scroll",
		KPageTurn:          "Page",
		KSave:              "Save",
		KSaveAndRun:        "Save & Run",
		KBackToEdit:        "Back to Edit",
		KGenerateReport:    "Generate Report",
		KViewRequest:       "View Request",
		KSelectRequest:     "Select Request",
		KViewLevelReqs:     "View Level Requests",
		KSelectItem:        "Select",
		KBackToDash:        "Back to Dashboard",
		KPrevNextReq:       "Prev/Next Request",
		KSwitchType:        "Switch Type",
		KClear:             "Clear",
		KTopBottom:         "Top/Bottom",
		KConfirmDelete:     "Confirm Delete",
		KCancel:            "Cancel",

		// Hints
		KHintQuit:      "[q] Quit",
		KHintCtrlCQuit: "[Ctrl+C] Quit",
		KHintBack:      "[b/Esc] Back",
		KHintEscBack:   "[Esc] Back",
		KHintGoBack:    "[b/Esc] Go Back",
		KHintSelect:    "[↑↓] Navigate",
		KHintNew:       "[a] New Task",

		// Metric labels
		KSuccessRate:  "Success Rate",
		KAvgTPS:       "Avg TPS",
		KAvgTTFT:      "Avg TTFT",
		KCacheHit:     "Cache Hit",
		KRPM:          "RPM",
		KTPM:          "TPM",
		KStatus:       "Status",
		KTotalTime:    "Total Time",
		KTTFT:         "TTFT",
		KOutputTPS:    "Output TPS",
		KToken:        "Token",
		KCache:        "Cache",
		KError:        "Error",
		KDNS:          "DNS",
		KTCPConnect:   "TCP Connect",
		KTLSHandshake: "TLS Handshake",
		KTargetIP:     "Target IP",

		// Status values
		KRunning:       "Running",
		KCompleted:     "Done",
		KRunFailed:     "Failed",
		KStopped:       "Stopped",
		KWaitingStatus: "Waiting",

		// Dashboard / progress
		KProgress:         "Progress",
		KSuccessCount:     "Success",
		KFailureCount:     "Failed",
		KWaitingDots:      "Waiting...",
		KRamp:             "Ramp",
		KPerLevel:         "Per Level",
		KStopCondLabel:    "Stop",
		KTurboMode:        "Turbo Mode",
		KStandardMode:     "Standard",
		KTurboMonitor:     "Turbo Probe Monitor",
		KTurboModeMeta:    "Turbo Mode",
		KSuccessRateFmt:   "Success %.1f%%",
		KTurboCurLevelFmt: "Current Level Metrics [Concurrency = %d]",
		KTurboDashSuffix:  "  %d/%d  Level %d  Progress %s",

		// Layout
		KWindowTooSmall:     "Terminal too small ↔ please resize",
		KWaitingData:        "Waiting for data...",
		KNotRecorded:        "(not recorded)",
		KScrollMore:         "↑↓ scroll to view full content",
		KTerminalLabel:      "Terminal",
		KInProgress:         "In Progress",
		KNoHotkeys:          "No shortcuts on this page",
		KStdMonitorTitle:    "Standard Run Monitor",
		KStdMonitorSubtitle: "Live view of run progress, throughput and per-request details",
		KTurboSubtitle:      "Observe concurrency ramp, level metrics and stable range",
		KTaskListSubtitle:   "Create tasks, run benchmarks, view run history and export reports",
		KTaskDetailSubtitle: "View task configuration, current run state and history",
		KReqDetailSubtitle:  "View latency, network phases and full payload of a single request",
		KConcFmt:            "Conc %d",

		// Fields
		KProtocol:      "Protocol",
		KEndpoint:      "Endpoint",
		KProxy:         "Proxy",
		KModel:         "Model",
		KMode:          "Mode",
		KConcurrency:   "Concurrency",
		KStepLabel:     "Step",
		KRequests:      "Requests",
		KTimeout:       "Timeout",
		KStream:        "Stream",
		KPromptLabel:   "Prompt",
		KNoRunRecords:  "No run records",
		KRecordDetails: "Run Details",
		KStart:         "Start",
		KEnd:           "End",
		KElapsed:       "Elapsed",
		KErrorSummary:  "Error Summary",
		KRequestBody:   "Request Body",
		KResponseBody:  "Response Body",

		// Table column headers
		KColTime:        "Time",
		KTime:           "Time",
		KColMode:        "Mode",
		KColProtocol:    "Protocol",
		KColSuccessRate: "Success%",
		KColCacheHit:    "Cache Hit",
		KColAvgTTFT:     "Avg TTFT",
		KColAvgTPS:      "Avg TPS",
		KColLevel:       "Level",
		KColInput:       "Input",
		KColOutput:      "Output",

		// TaskList
		KTaskName:            "Task Name",
		KTaskID:              "Task ID",
		KLastRun:             "Last Run",
		KIrreversible:        "This action is irreversible. All run history will also be deleted.",
		KNoTasks:             "No tasks · Press [a] to create the first task",
		KRunHistory:          "Run History",
		KTaskCenter:          "Tasks",
		KNoRunHistory:        "No run history",
		KConfirmDeletePrompt: "Delete this task?",

		// Proxy
		KExSOCKS5:      "Example: socks5://127.0.0.1:1080",
		KExSSH:         "Example: ssh://user@host:22",
		KExHTTP:        "Example: http://127.0.0.1:7890",
		KProxySubtitle: "Set a global HTTP proxy for all task requests. Leave blank to use system env or direct connection.",
		KProxySaveHint: "Config saved to ~/.ait/config.json. No need to re-enter after restart.",
		KGlobalConfig:  "Global Config",
		KProxyType:     "Proxy Type",
		KProxyURL:      "Proxy URL",

		// Help page
		KHelpTitle:         "Help",
		KHelpSubtitle:      "AIT — AI Load Testing Tool: Concepts & Usage Guide",
		KHelpMeta:          "Help",
		KHelpSecConcepts:   "Core Concepts",
		KHelpSecMetrics:    "Performance Metrics",
		KHelpSecProtocols:  "Protocol Support",
		KHelpSecGlobal:     "Hotkeys — Global",
		KHelpSecTaskList:   "Hotkeys — Task List",
		KHelpSecTaskDetail: "Hotkeys — Task Detail",
		KHelpSecDashboard:  "Hotkeys — Dashboard",
		KHelpSecExport:     "Report Export",

		KHelpTermTask:      "Task",
		KHelpDescTask:      "A set of load test configurations including target endpoint, model, concurrency, and request count. A task can be run multiple times, each run recorded independently.",
		KHelpTermRun:       "Run",
		KHelpDescRun:       "A single execution of a task. Each run produces independent metric data and request records, exportable as JSON/CSV reports.",
		KHelpTermStandard:  "Standard Mode",
		KHelpDescStandard:  "Executes all requests at a fixed concurrency level, ideal for measuring interface performance under steady load.",
		KHelpTermTurboMode: "Turbo Mode",
		KHelpDescTurboMode: "Automatically ramps up concurrency to find the maximum stable concurrency the interface can sustain while meeting the success rate requirement.",

		KHelpTermTPS:     "TPS",
		KHelpDescTPS:     "Tokens Per Second — output token generation rate of the model.",
		KHelpTermAvgTPS:  "Avg TPS",
		KHelpDescAvgTPS:  "Mean TPS across all requests in this run, reflecting overall throughput.",
		KHelpTermTTFT:    "TTFT",
		KHelpDescTTFT:    "Time To First Token — latency from sending the request to receiving the first token.",
		KHelpTermAvgTTFT: "Avg TTFT",
		KHelpDescAvgTTFT: "Mean TTFT across all requests in this run.",

		KHelpTermSuccessRate:      "Success Rate",
		KHelpDescSuccessRate:      "Percentage of requests that completed successfully. Failures include timeouts, HTTP errors, and model errors.",
		KHelpTermCacheHit:         "Cache Hit",
		KHelpDescCacheHit:         "Ratio of requests that used KV cache (Prompt Cache). Cache hits significantly reduce TTFT and inference cost. Binary metric: a request counts as a hit if any tokens were served from cache.",
		KHelpTermConcurrencyTurbo: "Concurrency (Turbo)",
		KHelpDescConcurrencyTurbo: "Maximum stable concurrency found by Turbo mode — the number of simultaneous requests sustainable while meeting the minimum success rate.",

		KHelpTermOpenAI:    "OpenAI",
		KHelpDescOpenAI:    "Compatible with OpenAI Chat Completions API (/v1/chat/completions), supporting both streaming and non-streaming responses.",
		KHelpTermAnthropic: "Anthropic",
		KHelpDescAnthropic: "Compatible with Anthropic Messages API (/v1/messages), supporting both streaming and non-streaming responses.",

		KHelpTermQuit:         "q / Ctrl+C",
		KHelpDescQuit:         "Quit the program.",
		KHelpTermQuestionMark: "?",
		KHelpDescQuestionMark: "Open this help page.",
		KHelpTermBack:         "b / Esc",
		KHelpDescBack:         "Go back to the previous page.",
		KHelpTermLangToggle:   "F2",
		KHelpDescLangToggle:   "Switch UI language (ZH / EN).",

		KHelpTermSelectTask:    "↑↓ / j k",
		KHelpDescSelectTask:    "Select a task.",
		KHelpTermEnterDetail:   "Enter",
		KHelpDescEnterDetail:   "Enter task detail page.",
		KHelpTermRunTask:       "r",
		KHelpDescRunTask:       "Run the selected task immediately.",
		KHelpTermStopTask:      "s",
		KHelpDescStopTask:      "Stop the running task (only available while running).",
		KHelpTermNewTask:       "a",
		KHelpDescNewTask:       "Create a new task (opens wizard).",
		KHelpTermEditTask:      "e",
		KHelpDescEditTask:      "Edit the selected task configuration.",
		KHelpTermDeleteTask:    "d",
		KHelpDescDeleteTask:    "Delete the selected task (requires confirmation).",
		KHelpTermDuplicateTask: "y",
		KHelpDescDuplicateTask: "Copy the selected task (creates a duplicate).",
		KHelpTermProxy:         "p",
		KHelpDescProxy:         "Open proxy configuration page.",

		KHelpTermSelectHistory:  "↑↓ / j k",
		KHelpDescSelectHistory:  "Select an entry in the run history.",
		KHelpTermEnterDash:      "Enter",
		KHelpDescEnterDash:      "View the dashboard for the selected run; enters live dashboard if the task is running.",
		KHelpTermRunAgain:       "r",
		KHelpDescRunAgain:       "Run the task again (available when no instance is running).",
		KHelpTermExport:         "g",
		KHelpDescExport:         "Export the selected run as a JSON report.",
		KHelpTermEditConfig:     "e",
		KHelpDescEditConfig:     "Edit the task configuration.",
		KHelpTermDuplicateTask2: "y",
		KHelpDescDuplicateTask2: "Copy the task.",
		KHelpTermDeleteTask2:    "d",
		KHelpDescDeleteTask2:    "Delete the task.",

		KHelpTermSelectReq:      "↑↓ / j k",
		KHelpDescSelectReq:      "Select a request entry.",
		KHelpTermViewReq:        "Enter",
		KHelpDescViewReq:        "View request details (latency, tokens, response body, etc.).",
		KHelpTermStopDash:       "s",
		KHelpDescStopDash:       "Stop the running task.",
		KHelpTermGenerateReport: "r",
		KHelpDescGenerateReport: "Generate a JSON report (available after run completes).",
		KHelpTermBackDash:       "b / Esc",
		KHelpDescBackDash:       "Return to task detail page.",

		KHelpTermJSONReport: "JSON Report",
		KHelpDescJSONReport: "Complete record of all metrics, request/response bodies for each request. Suitable for programmatic analysis.",
		KHelpTermCSVReport:  "CSV Report",
		KHelpDescCSVReport:  "Summary data in tabular form, openable directly in spreadsheets. Reports are saved in the current working directory by default.",

		// Wizard fields
		KWzTaskName:         "Task Name",
		KWzProtocol:         "Protocol",
		KWzEndpoint:         "Endpoint URL",
		KWzAPIKey:           "API Key",
		KWzTestModel:        "Model",
		KWzTestMode:         "Test Mode",
		KWzTurboMode:        "Turbo Mode",
		KWzStandardMode:     "Standard Mode",
		KWzConcurrency:      "Concurrency",
		KWzTotalRequests:    "Total Requests",
		KWzTimeoutSecs:      "Timeout (s)",
		KWzInitConc:         "Init Concurrency",
		KWzMaxConc:          "Max Concurrency",
		KWzStepSize:         "Step Size",
		KWzLevelReqs:        "Requests/Level",
		KWzMinSuccessRate:   "Min Success Rate",
		KWzStreamMode:       "Stream Mode",
		KWzInputMode:        "Input Mode",
		KWzInputDirect:      "Direct Input",
		KWzInputFile:        "File",
		KWzInputGenerated:   "Generated",
		KWzInputRaw:         "RAW Body",
		KWzPromptConfig:     "Prompt Config",
		KWzSelectModeHint:   "Select load test mode, then fill in concurrency and Prompt parameters.",
		KWzTurboModeLabel:   "Turbo Mode",
		KWzStepFmt:          "Step %d/3",
		KWzStep1Label:       "1 Basic Info",
		KWzStep2Label:       "2 Parameters",
		KWzStep3Label:       "3 Confirm",
		KWzStep1Desc:        "Configure task name, protocol, and connection info.",
		KWzStep2Desc:        "Choose test mode and fill in concurrency and prompt parameters.",
		KWzStep3Desc:        "Quick review before saving.",
		KWzUntitled:         "Untitled Task",
		KWzNotFilled:        "(empty)",
		KWzExecParams:       "Execution Parameters",
		KWzConcurrencyRamp:  "Concurrency Ramp",
		KWzStopCondition:    "Stop Condition",
		KWzTimeoutLabel:     "Timeout",
		KWzContentSummary:   "Content Summary",
		KWzBodyBytes:        "Body Bytes",
		KWzSaveLocation:     "Save Location",
		KWzPromptSection:    "Prompt",
		KWzHintDirect:       "Paste or type Prompt text directly. All requests share the same content.",
		KWzHintFile:         "Read Prompt from file(s). Supports glob patterns; requests rotate through matching files.",
		KWzHintRaw:          "Paste a complete HTTP request JSON body. Parameter assembly is skipped and the body is sent as-is.",
		KWzHintCacheToken:   "Tip: most services require ≥ 1024 tokens to trigger cache hits.",
		KWzHintRawBody:      "Tip: paste the full JSON body of an API request. It will be sent directly as the HTTP request body.",
		KWzJSONBody:         "JSON Body",
		KWzPromptLabelShort: "Prompt",
		KWzRAWBody:          "RAW Body",
		KWzFileSummary:      "File",
		KWzGeneratedFmt:     "%d chars",
		KWzPromptContent:    "Content",
		KWzNoConfirmItems:   "No confirm items",
		KWzConfirmRange:     "Items %d-%d/%d",
		KWzConfirmTotal:     "%d items to confirm",
		KWzNoFields:         "No fields",
		KWzFieldProgress:    "Field %d/%d",

		// Misc
		KEnabled:        "On",
		KDisabled:       "Off",
		KFileSummaryPfx: "File: ",
		KNotSet:         "(empty)",
		KJustNow:        "just now",
		KMinutesAgoFmt:  "%d min ago",
		KHoursAgoFmt:    "%d hr ago",
		KDaysAgoFmt:     "%d days ago",
		KToggleLang:     "Toggle Lang",
	},
}

// T returns the translation for key k in the active language.
// Falls back to ZH if the key is missing in the active language.
func T(k Key) string {
	l := Active()
	if m := translations[l]; m != nil {
		if s, ok := m[k]; ok {
			return s
		}
	}
	if s, ok := translations[ZH][k]; ok {
		return s
	}
	return ""
}

// DisplayWidth returns the display column width of s,
// counting CJK characters as 2 columns and ASCII as 1.
func DisplayWidth(s string) int {
	return runewidth.StringWidth(s)
}
