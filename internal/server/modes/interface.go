package modes

// Runner 统一的模式执行器接口
// 所有运行模式（standard/turbo/integrity）都应实现此接口
type Runner interface {
	// Stop 停止当前运行
	Stop()
}

// StateProvider 可选接口，用于提供模式特定的运行时状态快照
// 实现此接口的 Runner 可以在事件中携带自定义状态
type StateProvider interface {
	Runner
	// GetState 返回当前模式的运行时状态快照（用于事件推送）
	// 返回值会被序列化到 RunState.ModeState
	GetState() any
}

// ResultProvider 可选接口，用于返回模式特定的最终结果
// 实现此接口的 Runner 可以提供结构化的运行结果
type ResultProvider interface {
	Runner
	// GetResult 返回运行结束后的最终结果
	// 返回值会被序列化到 RunState.ModeResult
	GetResult() any
}
