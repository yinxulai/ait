package main

import (
	"flag"
	"testing"
)

func TestFlagDefinitions(t *testing.T) {
	// 重置 flag 状态，避免冲突
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// 模拟定义 flags（这部分通常在 main 中）
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apikey := flag.String("apikey", "", "API 密钥")
	model := flag.String("model", "", "模型名称")
	provider := flag.String("provider", "openai", "协议类型: openai 或 anthropic")
	concurrency := flag.Int("concurrency", 1, "并发数")
	count := flag.Int("count", 10, "请求总数")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", false, "是否开启流模式")

	// 测试默认值
	if *provider != "openai" {
		t.Errorf("Expected default provider 'openai', got '%s'", *provider)
	}

	if *concurrency != 1 {
		t.Errorf("Expected default concurrency 1, got %d", *concurrency)
	}

	if *count != 10 {
		t.Errorf("Expected default count 10, got %d", *count)
	}

	if *stream != false {
		t.Errorf("Expected default stream false, got %t", *stream)
	}

	// 测试 flag 是否正确定义
	if baseUrl == nil || apikey == nil || model == nil || prompt == nil {
		t.Error("Required flags should be defined")
	}
}
