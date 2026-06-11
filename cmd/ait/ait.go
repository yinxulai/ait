package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/mcp"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/tui"
)

// 版本信息，通过 ldflags 在构建时注入。
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// ── flags ────────────────────────────────────────────────────────────────
	versionFlag := flag.Bool("version", false, "显示版本信息")
	mcpFlag := flag.Bool("mcp", false, "启用 MCP 模式")
	langFlag := flag.String("lang", "", "界面语言：zh 或 en")
	flag.Parse()

	// ── 版本输出 ──────────────────────────────────────────────────────────────
	if *versionFlag {
		fmt.Printf("ait version %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// ── 创建 Server ───────────────────────────────────────────────────────────
	srv, err := server.NewWithVersion(Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 Server 失败: %v\n", err)
		os.Exit(1)
	}

	// ── 初始化界面语言（flag > 配置文件 > 默认 ZH）────────────────────────────
	if *langFlag == "en" {
		i18n.SetLang(i18n.EN)
	} else if *langFlag == "zh" {
		i18n.SetLang(i18n.ZH)
	} else if cfg, err := config.Load(); err == nil && cfg.Lang == "en" {
		i18n.SetLang(i18n.EN)
	}

	if routeByMCPFlag(*mcpFlag) == "mcp" {
		if err := mcp.New(srv).Run(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "MCP 启动失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	tui.SetVersion(Version)
	if err := tui.Run(srv); err != nil {
		fmt.Fprintf(os.Stderr, "TUI 启动失败: %v\n", err)
		os.Exit(1)
	}
}

func routeByMCPFlag(enabled bool) string {
	if enabled {
		return "mcp"
	}
	return "tui"
}
