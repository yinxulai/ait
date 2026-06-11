package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// SlogLogger 是基于 slog 的结构化日志记录器。
// 支持多种输出目标：终端、文件、或两者兼有。
type SlogLogger struct {
	logger *slog.Logger
	mu     sync.Mutex
}

// 全局日志记录器实例
var (
	defaultLogger     *SlogLogger
	defaultLoggerInit sync.Once
)

// NewSlog 创建新的 slog 日志记录器。
// output: "terminal" 仅终端, "file" 仅文件, "both" 两者
// level: slog 级别 (debug/info/warn/error)
func NewSlog(output string, level slog.Level) *SlogLogger {
	l := &SlogLogger{}

	var handler slog.Handler
	switch output {
	case "terminal":
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	case "file":
		file, err := os.OpenFile("ait.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		} else {
			handler = slog.NewTextHandler(file, &slog.HandlerOptions{Level: level})
		}
	case "both":
		// 同时输出到终端和文件
		file, err := os.OpenFile("ait.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		} else {
			writer := io.MultiWriter(os.Stderr, file)
			handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
		}
	default:
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	l.logger = slog.New(handler)
	return l
}

// Default 返回默认的全局日志记录器。
// 线程安全，只初始化一次。
func Default() *SlogLogger {
	defaultLoggerInit.Do(func() {
		defaultLogger = NewSlog("terminal", slog.LevelInfo)
	})
	return defaultLogger
}

// Debug 记录调试级别日志
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Debug(msg, args...)
}

// DebugCtx 带 Context 的调试日志
func (l *SlogLogger) DebugCtx(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.DebugContext(ctx, msg, args...)
}

// Info 记录信息级别日志
func (l *SlogLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Info(msg, args...)
}

// InfoCtx 带 Context 的信息日志
func (l *SlogLogger) InfoCtx(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn 记录警告级别日志
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Warn(msg, args...)
}

// WarnCtx 带 Context 的警告日志
func (l *SlogLogger) WarnCtx(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.WarnContext(ctx, msg, args...)
}

// Error 记录错误级别日志
func (l *SlogLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Error(msg, args...)
}

// ErrorCtx 带 Context 的错误日志
func (l *SlogLogger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.ErrorContext(ctx, msg, args...)
}

// With 返回带有额外属性的日志记录器
func (l *SlogLogger) With(args ...any) *SlogLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	return &SlogLogger{logger: l.logger.With(args...)}
}

// ─── 全局便捷方法 ───────────────────────────────────────────────────────

// Debug 是 Default().Debug 的便捷方法
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info 是 Default().Info 的便捷方法
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn 是 Default().Warn 的便捷方法
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error 是 Default().Error 的便捷方法
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}
