package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStore 是泛型 JSON 文件持久化基类。
// 内置进程级互斥锁，防止同一进程内并发读写；文件操作通过原子写入保证安全。
type JSONStore[T any] struct {
	path     string
	mu       sync.Mutex
	debounce time.Duration
}

type debouncedPathState struct {
	path         string
	mu           sync.Mutex
	cond         *sync.Cond
	timer        *time.Timer
	generation   uint64
	flushRunning bool
	pending      []byte
	lastErr      error
}

var debouncedPathStates = struct {
	mu     sync.Mutex
	states map[string]*debouncedPathState
}{states: make(map[string]*debouncedPathState)}

// NewJSONStore 创建指向指定路径的 JSONStore。
func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

// NewDebouncedJSONStore 创建带写入防抖的 JSONStore。
// 同一路径的多个实例会共享同一个防抖器，用于合并高频 Save 调用。
func NewDebouncedJSONStore[T any](path string, debounce time.Duration) *JSONStore[T] {
	return &JSONStore[T]{path: path, debounce: debounce}
}

func getDebouncedPathState(path string) *debouncedPathState {
	debouncedPathStates.mu.Lock()
	defer debouncedPathStates.mu.Unlock()

	if state, ok := debouncedPathStates.states[path]; ok {
		return state
	}

	state := &debouncedPathState{path: path}
	state.cond = sync.NewCond(&state.mu)
	debouncedPathStates.states[path] = state
	return state
}

func (s *debouncedPathState) schedule(data []byte, delay time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending = append([]byte(nil), data...)
	s.lastErr = nil
	s.generation++
	gen := s.generation

	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(delay, func() {
		s.flushGeneration(gen)
	})
	return nil
}

func (s *debouncedPathState) saveNow(data []byte) error {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.pending = append([]byte(nil), data...)
	s.generation++
	s.lastErr = nil
	s.mu.Unlock()
	return s.flush()
}

func (s *debouncedPathState) flushGeneration(gen uint64) {
	s.mu.Lock()
	if gen != s.generation {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	_ = s.flush()
}

func (s *debouncedPathState) flush() error {
	s.mu.Lock()
	for s.flushRunning {
		s.cond.Wait()
	}
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.pending == nil {
		err := s.lastErr
		s.mu.Unlock()
		return err
	}
	data := append([]byte(nil), s.pending...)
	s.pending = nil
	s.flushRunning = true
	s.mu.Unlock()

	err := writeJSONFileAtomic(s.path, data)

	s.mu.Lock()
	s.flushRunning = false
	s.lastErr = err
	hasPending := s.pending != nil
	s.cond.Broadcast()
	s.mu.Unlock()

	if err != nil {
		return err
	}
	if hasPending {
		return s.flush()
	}
	return nil
}

func writeJSONFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load 从文件读取并反序列化为 T。文件不存在时返回零值（无错误）。
func (s *JSONStore[T]) Load() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var zero T
	if err := getDebouncedPathState(s.path).flush(); err != nil {
		return zero, err
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, nil
	}
	if err != nil {
		return zero, err
	}

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Save 将 v 序列化为 JSON 并写入文件，写入前自动创建父目录。
// 使用"写临时文件后重命名"的原子写入方式，避免文件写到一半导致损坏。
func (s *JSONStore[T]) Save(v T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	state := getDebouncedPathState(s.path)
	if s.debounce > 0 {
		return state.schedule(data, s.debounce)
	}
	return state.saveNow(data)
}

// Flush 立即落盘当前路径上的 pending 数据。
func (s *JSONStore[T]) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return getDebouncedPathState(s.path).flush()
}
