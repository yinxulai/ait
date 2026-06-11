package integrity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRulesManager(t *testing.T) {
	m, err := NewRulesManager("v1.0.0")
	if err != nil {
		t.Fatalf("NewRulesManager failed: %v", err)
	}

	if m.version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", m.version)
	}

	if m.updateSource != "stable" {
		t.Errorf("expected update source stable, got %s", m.updateSource)
	}
}

func TestDetermineUpdateSource(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"dev", "latest"},
		{"", "latest"},
		{"v1.0.0", "stable"},
		{"1.0.0", "stable"},
		{"v1.2.3", "stable"},
		{"abc123", "latest"},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			result := determineUpdateSource(tc.version)
			if result != tc.expected {
				t.Errorf("version %q: expected %q, got %q", tc.version, tc.expected, result)
			}
		})
	}
}

func TestIsSemanticVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"1.0.0", true},
		{"0.1.0", true},
		{"10.20.30", true},
		{"v1.0.0", false},
		{"1.0", false},
		{"1.0.0.0", false},
		{"a.b.c", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			result := isSemanticVersion(tc.version)
			if result != tc.expected {
				t.Errorf("version %q: expected %v, got %v", tc.version, tc.expected, result)
			}
		})
	}
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int // -1: v1 < v2, 0: v1 == v2, 1: v1 > v2
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.0", 1},
		{"2.0.0", "1.9.9", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"dev", "1.0.0", 1},
		{"1.0.0", "dev", -1},
	}

	for _, tc := range tests {
		t.Run(tc.v1+"_vs_"+tc.v2, func(t *testing.T) {
			result := compareVersion(tc.v1, tc.v2)
			var resultSign int
			if result < 0 {
				resultSign = -1
			} else if result > 0 {
				resultSign = 1
			}
			if resultSign != tc.expected {
				t.Errorf("compareVersion(%q, %q) = %d (sign %d), want sign %d",
					tc.v1, tc.v2, result, resultSign, tc.expected)
			}
		})
	}
}

func TestInitialize(t *testing.T) {
	// 创建测试 HTTP 服务器
	indexJSON := `{
		"version": "1.0.0",
		"repository": "https://github.com/test/repo",
		"description": "Test rules",
		"compatibility": {
			"min_version": "0.1.0",
			"max_version": "99.99.99"
		},
		"rules": {
			"test-rule": {
				"suite": "test-suite",
				"protocol": "test-protocol",
				"file": "integrity/test.json",
				"description": "Test rule"
			}
		},
		"update_sources": {
			"stable": "http://test.example.com/data/index.json",
			"latest": "http://test.example.com/data/index.json"
		},
		"last_updated": "2026-06-11T00:00:00Z"
	}`

	ruleJSON := `{
		"version": "ait.integrity.rules/v1",
		"suite": "test-suite",
		"assertions": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/index.json") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(indexJSON))
		} else if strings.HasSuffix(r.URL.Path, "/integrity/test.json") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(ruleJSON))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 清理测试缓存
	tempDir := t.TempDir()
	m, err := NewRulesManager("v1.0.0")
	if err != nil {
		t.Fatalf("NewRulesManager failed: %v", err)
	}
	m.cacheDir = tempDir

	// 模拟首次初始化，需要提供默认的更新源
	// 修改 index 以使用测试服务器
	m.index = &RuleIndex{
		UpdateSources: map[string]string{
			"stable": server.URL + "/data/index.json",
			"latest": server.URL + "/data/index.json",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.updateFromNetwork(ctx)
	if err != nil {
		t.Fatalf("updateFromNetwork failed: %v", err)
	}

	if m.index == nil {
		t.Fatal("index should be initialized")
	}

	if m.index.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.index.Version)
	}
}

func TestGetRuleFiles(t *testing.T) {
	m, err := NewRulesManager("v1.0.0")
	if err != nil {
		t.Fatalf("NewRulesManager failed: %v", err)
	}

	// 手动设置索引和创建规则文件
	m.index = &RuleIndex{
		Version: "1.0.0",
		Rules: map[string]RuleEntry{
			"test-rule": {
				Suite:    "test-suite",
				Protocol: "test-protocol",
				File:     "integrity/test.json",
			},
		},
		Compatibility: VersionRange{
			MinVersion: "0.1.0",
			MaxVersion: "99.99.99",
		},
	}

	// 创建规则文件
	ruleContent := `{"version":"ait.integrity.rules/v1","suite":"test-suite","assertions":[]}`
	rulePath := filepath.Join(m.cacheDir, "integrity", "test.json")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o755); err != nil {
		t.Fatalf("Failed to create rule directory: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte(ruleContent), 0o600); err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}

	// 测试获取规则文件
	files, err := m.GetRuleFiles("test-protocol", "test-suite")
	if err != nil {
		t.Fatalf("GetRuleFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Error("should have at least one rule file")
	}

	// 验证文件存在
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("rule file does not exist: %s", file)
		}
	}
}

func TestUpdateFromNetwork(t *testing.T) {
	// 创建测试 HTTP 服务器
	indexJSON := `{
		"version": "1.0.1",
		"repository": "https://github.com/test/repo",
		"description": "Test rules",
		"compatibility": {
			"min_version": "0.1.0",
			"max_version": "99.99.99"
		},
		"rules": {
			"test-rule": {
				"suite": "test-suite",
				"protocol": "test-protocol",
				"file": "integrity/test.json",
				"description": "Test rule"
			}
		},
		"update_sources": {
			"stable": "http://test.example.com/data/index.json"
		},
		"last_updated": "2026-06-11T00:00:00Z"
	}`

	ruleJSON := `{
		"version": "ait.integrity.rules/v1",
		"suite": "test-suite",
		"assertions": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data/index.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(indexJSON))
		} else if r.URL.Path == "/data/integrity/test.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(ruleJSON))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	m, err := NewRulesManager("v1.0.0")
	if err != nil {
		t.Fatalf("NewRulesManager failed: %v", err)
	}
	m.cacheDir = tempDir

	// 设置更新源为测试服务器
	m.index = &RuleIndex{
		UpdateSources: map[string]string{
			"stable": server.URL + "/data/index.json",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.updateFromNetwork(ctx)
	if err != nil {
		t.Fatalf("updateFromNetwork failed: %v", err)
	}

	if m.index.Version != "1.0.1" {
		t.Errorf("expected version 1.0.1, got %s", m.index.Version)
	}

	// 验证缓存文件存在
	indexPath := filepath.Join(m.cacheDir, "index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("cache index file should exist")
	}
}

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		version      string
		minVersion   string
		maxVersion   string
		shouldMatch  bool
	}{
		{"1.0.0", "0.1.0", "2.0.0", true},
		{"0.0.1", "0.1.0", "2.0.0", false},
		{"3.0.0", "0.1.0", "2.0.0", false},
		{"1.5.0", "1.0.0", "2.0.0", true},
		{"dev", "0.1.0", "99.99.99", true},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			m, err := NewRulesManager(tc.version)
			if err != nil {
				t.Fatalf("NewRulesManager failed: %v", err)
			}

			m.index = &RuleIndex{
				Compatibility: VersionRange{
					MinVersion: tc.minVersion,
					MaxVersion: tc.maxVersion,
				},
			}

			result := m.isCompatible()
			if result != tc.shouldMatch {
				t.Errorf("version %q with range [%s, %s]: expected %v, got %v",
					tc.version, tc.minVersion, tc.maxVersion, tc.shouldMatch, result)
			}
		})
	}
}
