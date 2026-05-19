package server

import (
	"sync"

	"github.com/yinxulai/ait/internal/config"
	"github.com/yinxulai/ait/internal/store"
	"github.com/yinxulai/ait/internal/types"
)

// Server 是业务逻辑层的统一入口，TUI 层通过此接口与业务交互。
// 所有方法均为线程安全。
type Server interface {
	// --- 任务 CRUD ---
	ListTasks() ([]types.TaskOverview, error)
	GetTask(id string) (types.TaskDefinition, error)
	CreateTask(cfg TaskConfig) (types.TaskDefinition, error)
	UpdateTask(id string, cfg TaskConfig) (types.TaskDefinition, error)
	DeleteTask(id string) error
	CopyTask(id string) (types.TaskDefinition, error)

	// --- 运行管理 ---

	// StartRun 根据任务配置启动一次运行，立即返回 RunID。
	// 运行在后台 goroutine 中执行，进度通过 Subscribe 获取。
	StartRun(taskID string) (RunID, error)

	// StopRun 请求停止指定运行（软停止，等待当前批次完成）。
	StopRun(runID RunID) error

	// GetRunState 返回指定运行的当前状态快照（线程安全的深度拷贝）。
	GetRunState(runID RunID) (*RunState, bool)

	// Subscribe 订阅指定运行的事件流。返回只读通道和取消函数。
	// 通道在运行结束后自动关闭，调用方可 range 消费。
	Subscribe(runID RunID) (<-chan Event, CancelFunc)

	// GetHistory 返回任务的运行历史，最新在前。limit<=0 表示不限条数。
	GetHistory(taskID string, limit int) ([]types.TaskRunSummary, error)

	// GenerateReport 为已完成的运行生成报告文件，返回文件路径。
	GenerateReport(runID RunID, format ReportFormat) (string, error)
}

// serverImpl 是 Server 的具体实现。
type serverImpl struct {
	mu         sync.RWMutex
	taskStore  *store.TaskStore
	taskViews  *store.TaskViewStore
	runStore   *store.RunStore
	bus        *eventBus
	activeRuns map[RunID]*activeRun
}

// New 创建并初始化 Server 实例。
// 会自动加载 ~/.ait/tasks/ 与 ~/.ait/runs/ 下的业务数据。
func New() (Server, error) {
	if _, err := config.EnsureAppDir(); err != nil {
		return nil, err
	}

	tasksDir, err := config.TasksDir()
	if err != nil {
		return nil, err
	}

	runsDir, err := config.RunsDir()
	if err != nil {
		return nil, err
	}

	ts := store.NewTaskStore(tasksDir)
	rs := store.NewRunStore(runsDir)

	return &serverImpl{
		taskStore:  ts,
		taskViews:  store.NewTaskViewStore(ts, rs),
		runStore:   rs,
		bus:        newEventBus(),
		activeRuns: make(map[RunID]*activeRun),
	}, nil
}
