package integrity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RuleIndex 规则索引结构
type RuleIndex struct {
	Version       string                 `json:"version"`
	Repository    string                 `json:"repository"`
	Description   string                 `json:"description"`
	Compatibility VersionRange           `json:"compatibility"`
	Rules         map[string]RuleEntry   `json:"rules"`
	UpdateSources map[string]string      `json:"update_sources"`
	LastUpdated   string                 `json:"last_updated"`
}

// VersionRange 版本范围
type VersionRange struct {
	MinVersion string `json:"min_version"`
	MaxVersion string `json:"max_version"`
}

// RuleEntry 规则条目
type RuleEntry struct {
	Suite       string `json:"suite"`
	Protocol    string `json:"protocol"`
	File        string `json:"file"`
	Description string `json:"description"`
}

// RulesManager 规则管理器（从 AIT 仓库加载）
type RulesManager struct {
	version      string
	cacheDir     string
	index        *RuleIndex
	httpClient   *http.Client
	updateSource string // stable | latest | dev
}

// NewRulesManager 创建规则管理器
func NewRulesManager(version string) (*RulesManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".ait", "rules")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	return &RulesManager{
		version:    version,
		cacheDir:   cacheDir,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		updateSource: determineUpdateSource(version),
	}, nil
}

// determineUpdateSource 根据版本决定更新源
func determineUpdateSource(version string) string {
	if version == "dev" || version == "" {
		return "latest"
	}
	// 检查是否是 release 版本（如 v1.0.0, 1.0.0）
	if strings.HasPrefix(version, "v") || isSemanticVersion(version) {
		return "stable"
	}
	return "latest"
}

// isSemanticVersion 检查是否是语义化版本号
func isSemanticVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// buildSourceURL 构建更新源 URL
func (m *RulesManager) buildSourceURL() string {
	// 如果有已加载的索引，使用其中的更新源
	if m.index != nil && m.index.UpdateSources != nil {
		if sourceTemplate, ok := m.index.UpdateSources[m.updateSource]; ok {
			return strings.ReplaceAll(sourceTemplate, "{version}", m.version)
		}
	}

	// 首次运行或索引不完整，使用默认的 GitHub 仓库
	repoBase := "https://raw.githubusercontent.com/yinxulai/ait"
	switch m.updateSource {
	case "stable":
		return fmt.Sprintf("%s/v%s/data/index.json", repoBase, strings.TrimPrefix(m.version, "v"))
	case "dev":
		return fmt.Sprintf("%s/dev/data/index.json", repoBase)
	default: // latest
		return fmt.Sprintf("%s/main/data/index.json", repoBase)
	}
}

// Initialize 初始化规则（在程序启动时调用）
func (m *RulesManager) Initialize(ctx context.Context) error {
	// 1. 尝试从缓存加载
	if err := m.loadFromCache(); err == nil {
		// 检查版本兼容性
		if m.isCompatible() {
			// 缓存有效，后台检查是否有新版本
			go m.checkAndUpdate(context.Background())
			return nil
		}
	}

	// 2. 缓存不存在、过期或不兼容，尝试从网络更新
	if err := m.updateFromNetwork(ctx); err == nil {
		if m.isCompatible() {
			return nil
		}
		// 网络版本也不兼容
		return fmt.Errorf("downloaded rules version incompatible with current version %s", m.version)
	}

	// 3. 网络更新失败，如果有缓存（即使过期），仍然使用
	if m.index != nil {
		slog.Warn("using expired rules cache due to network failure", "version", m.version)
		return nil
	}

	// 4. 完全没有数据
	return fmt.Errorf("failed to load rules: no cache available and network update failed")
}

// loadFromCache 从缓存加载索引
func (m *RulesManager) loadFromCache() error {
	indexPath := filepath.Join(m.cacheDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read cache index: %w", err)
	}

	var index RuleIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("parse cache index: %w", err)
	}

	m.index = &index
	return nil
}

// hasNewerVersion 检查远程是否有新版本
func (m *RulesManager) hasNewerVersion(ctx context.Context) bool {
	if m.index == nil {
		return true // 本地没有索引，需要下载
	}

	sourceURL := m.buildSourceURL()
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return false
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var remoteIndex RuleIndex
	if err := json.Unmarshal(data, &remoteIndex); err != nil {
		return false
	}

	// 对比版本号
	return remoteIndex.Version != m.index.Version || remoteIndex.LastUpdated != m.index.LastUpdated
}

// updateFromNetwork 从网络更新规则
func (m *RulesManager) updateFromNetwork(ctx context.Context) error {
	sourceURL := m.buildSourceURL()

	// 下载索引
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var newIndex RuleIndex
	if err := json.Unmarshal(data, &newIndex); err != nil {
		return fmt.Errorf("parse index: %w", err)
	}

	// 保存到缓存
	indexPath := filepath.Join(m.cacheDir, "index.json")
	if err := os.WriteFile(indexPath, data, 0o600); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	// 下载规则文件
	if err := m.downloadRuleFiles(ctx, &newIndex); err != nil {
		// 下载规则文件失败不是致命错误，继续使用索引
		slog.Warn("failed to download rule files, using cached version", "error", err)
	}

	m.index = &newIndex
	return nil
}

// downloadRuleFiles 下载所有规则文件
func (m *RulesManager) downloadRuleFiles(ctx context.Context, index *RuleIndex) error {
	baseURL := strings.TrimSuffix(m.index.UpdateSources[m.updateSource], "/index.json")
	baseURL = strings.ReplaceAll(baseURL, "{version}", m.version)

	for _, rule := range index.Rules {
		ruleURL := baseURL + "/" + rule.File
		localPath := filepath.Join(m.cacheDir, rule.File)

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			return fmt.Errorf("create rule dir: %w", err)
		}

		// 下载规则文件
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ruleURL, nil)
		if err != nil {
			return fmt.Errorf("create request for %s: %w", rule.File, err)
		}

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("download %s: %w", rule.File, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download %s: http %d", rule.File, resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read %s: %w", rule.File, err)
		}

		if err := os.WriteFile(localPath, data, 0o600); err != nil {
			return fmt.Errorf("save %s: %w", rule.File, err)
		}
	}

	return nil
}

// checkAndUpdate 检查并更新规则（后台运行）
func (m *RulesManager) checkAndUpdate(ctx context.Context) {
	// 检查是否有新版本
	if !m.hasNewerVersion(ctx) {
		return // 没有新版本
	}

	// 有新版本，尝试更新
	if err := m.updateFromNetwork(ctx); err != nil {
		// 更新失败，静默处理（继续使用本地缓存）
		slog.Info("failed to update rules in background, using cached", "error", err)
	}
}

// isCompatible 检查当前版本是否兼容
func (m *RulesManager) isCompatible() bool {
	if m.index == nil {
		return false
	}

	// dev 版本总是兼容
	if m.version == "dev" || m.version == "" {
		return true
	}

	// 简单版本比较（可以使用更复杂的语义化版本比较）
	return compareVersion(m.version, m.index.Compatibility.MinVersion) >= 0 &&
		compareVersion(m.version, m.index.Compatibility.MaxVersion) <= 0
}

// compareVersion 简单版本比较
func compareVersion(v1, v2 string) int {
	// 移除 'v' 前缀
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// "dev" 版本视为最新
	if v1 == "dev" {
		return 1
	}
	if v2 == "dev" {
		return -1
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < 3; i++ {
		var n1, n2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &n2)
		}
		if n1 != n2 {
			return n1 - n2
		}
	}
	return 0
}

// GetRuleFiles 获取指定协议的规则文件路径
func (m *RulesManager) GetRuleFiles(protocol, suite string) ([]string, error) {
	if m.index == nil {
		return nil, fmt.Errorf("rules not initialized")
	}

	var files []string

	for _, rule := range m.index.Rules {
		// 匹配协议和套件
		if (rule.Protocol == protocol || rule.Protocol == "*") &&
			(rule.Suite == suite || rule.Suite == "*") {
			
			// 从缓存读取文件
			cachedPath := filepath.Join(m.cacheDir, rule.File)
			if _, err := os.Stat(cachedPath); err == nil {
				files = append(files, cachedPath)
			} else {
				// 文件不存在，返回错误
				return nil, fmt.Errorf("rule file not found in cache: %s (try reinitializing)", rule.File)
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no matching rules found for protocol=%s, suite=%s", protocol, suite)
	}

	return files, nil
}

// GetIndex 获取当前索引
func (m *RulesManager) GetIndex() *RuleIndex {
	return m.index
}

// ClearCache 清除缓存
func (m *RulesManager) ClearCache() error {
	return os.RemoveAll(m.cacheDir)
}
