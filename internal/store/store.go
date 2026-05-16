package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// JSONStore 是泛型 JSON 文件持久化基类。
// 内置进程级互斥锁，防止同一进程内并发读写；文件操作通过原子写入保证安全。
type JSONStore[T any] struct {
	path string
	mu   sync.Mutex
}

// NewJSONStore 创建指向指定路径的 JSONStore。
func NewJSONStore[T any](path string) *JSONStore[T] {
	return &JSONStore[T]{path: path}
}

// Load 从文件读取并反序列化为 T。文件不存在时返回零值（无错误）。
func (s *JSONStore[T]) Load() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var zero T
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

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	// 原子写入：先写临时文件，再重命名
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
