package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/modes/integrity"
	"github.com/yinxulai/ait/internal/server/store"
	"github.com/yinxulai/ait/internal/server/types"
)

// Server 是业务逻辑层的统一入口，TUI 层通过此接口与业务交互。
// 所有方法均为线程安全。
type Server interface {
	// --- 任务管理 ---
	ListTasks() ([]types.TaskOverview, error)
	GetTask(id string) (types.TaskDefinition, error)
	CreateTask(cfg TaskConfig) (types.TaskDefinition, error)
	UpdateTask(id string, cfg TaskConfig) (types.TaskDefinition, error)
	DeleteTask(id string) error
	DuplicateTask(id string) (types.TaskDefinition, error)

	// --- 运行管理 ---

	// StartRun 根据任务配置启动一次运行，立即返回 RunID。
	// 运行在后台 goroutine 中执行，进度通过 SubscribeRunEvents 获取。
	StartRun(taskID string) (RunID, error)

	// StopRun 请求停止指定运行（软停止，等待当前批次完成）。
	StopRun(runID RunID) error

	// GetRunState 返回指定运行的当前状态快照（线程安全的深度拷贝）。
	GetRunState(runID RunID) (*RunState, bool)

	// SubscribeRunEvents 订阅指定运行的事件流。返回只读通道和取消函数。
	// 通道在运行结束后自动关闭，调用方可 range 消费。
	SubscribeRunEvents(runID RunID) (<-chan Event, CancelFunc)

	// ListTaskRunHistory 返回任务的运行历史，最新在前。limit<=0 表示不限条数。
	ListTaskRunHistory(taskID string, limit int) ([]types.TaskRunSummary, error)

	// GenerateRunReport 为已完成的运行生成报告文件，返回文件路径。
	GenerateRunReport(runID RunID, format ReportFormat) (string, error)

	// --- 全局配置 ---

	// GetAppConfig 返回当前全局配置。
	GetAppConfig() (*config.Config, error)

	// UpdateProxyURL 更新并持久化全局代理 URL。
	UpdateProxyURL(proxyURL string) error

	// Context 返回 Server 的生命周期 Context，用于子操作。
	// 当 Server 关闭时，此 Context 会被取消。
	Context() context.Context
}

// serverImpl 是 Server 的具体实现。
type serverImpl struct {
	mu           sync.RWMutex
	taskStore    *store.TaskStore
	taskViews    *store.TaskViewStore
	runStore     *store.RunStore
	bus          *eventBus
	activeRuns   map[RunID]*activeRun
	scheduler    *RunScheduler
	rulesManager *integrity.RulesManager

	// 生命周期 Context，用于优雅关闭
	ctx    context.Context
	cancel context.CancelFunc
}

// New 创建并初始化 Server 实例。
// 会自动加载 ~/.ait/tasks/ 与 ~/.ait/runs/ 下的业务数据。
func New() (Server, error) {
	return NewWithVersion("dev")
}

// NewWithVersion 创建并初始化 Server 实例，指定版本号。
func NewWithVersion(version string) (Server, error) {
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

	// 初始化规则管理器
	rulesManager, err := integrity.NewRulesManager(version)
	if err != nil {
		// 规则管理器初始化失败不应阻止 Server 创建
		rulesManager = nil
	}

	// 创建生命周期 Context，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())

	srv := &serverImpl{
		taskStore:    ts,
		taskViews:    store.NewTaskViewStore(ts, rs),
		runStore:     rs,
		bus:          newEventBus(),
		activeRuns:   make(map[RunID]*activeRun),
		rulesManager: rulesManager,
		ctx:         ctx,
		cancel:      cancel,
	}
	srv.scheduler = newRunScheduler(1, srv.dispatchQueuedRun)

	// 后台初始化规则（使用生命周期 Context）
	if rulesManager != nil {
		go func() {
			initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
			defer initCancel()
			if err := rulesManager.Initialize(initCtx); err != nil {
				// 初始化失败只记录，不影响程序运行
			}
		}()
	}

	return srv, nil
}

// Shutdown 优雅关闭 Server，释放所有资源。
// timeout 时间内等待运行完成，超过后强制取消。
func (s *serverImpl) Shutdown(timeout time.Duration) error {
	// 1. 发送取消信号，停止接受新请求
	s.cancel()

	// 2. 等待调度器完成（如果支持）
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 3. 等待所有运行完成
	if s.scheduler != nil {
		if err := s.scheduler.Shutdown(ctx); err != nil {
			return fmt.Errorf("scheduler shutdown: %w", err)
		}
	}

	return nil
}
