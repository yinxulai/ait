package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

type Server struct {
	svc server.Server
	sdk *mcpsdk.Server
}

func New(svc server.Server) *Server {
	s := &Server{svc: svc}
	s.sdk = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ait",
		Title:   "AIT MCP Server",
		Version: "0.1.0",
	}, &mcpsdk.ServerOptions{
		Capabilities: &mcpsdk.ServerCapabilities{
			Tools: &mcpsdk.ToolCapabilities{},
		},
	})
	s.registerTools()
	return s
}

func (s *Server) Run(ctx context.Context) error {
	return s.sdk.Run(ctx, &mcpsdk.StdioTransport{})
}

func (s *Server) registerTools() {
	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.list_tasks",
		Description: "List all AIT tasks with latest run info",
	}, s.listTasks)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.create_task",
		Description: "Create an AIT task (inherits internal/server task creation capability)",
	}, s.createTask)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.start_run",
		Description: "Start run for a task",
	}, s.startRun)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.get_run_state",
		Description: "Get current run state snapshot",
	}, s.getRunState)
}

type listTasksArgs struct{}

type createTaskArgs struct {
	Name         string `json:"name" jsonschema:"task name"`
	Protocol     string `json:"protocol" jsonschema:"request protocol: openai-completions, openai-responses, or anthropic-messages"`
	EndpointURL  string `json:"endpoint_url,omitempty" jsonschema:"full endpoint URL"`
	BaseURL      string `json:"base_url,omitempty" jsonschema:"base URL"`
	APIKey       string `json:"api_key" jsonschema:"API key"`
	Model        string `json:"model" jsonschema:"model name"`
	Stream       *bool  `json:"stream,omitempty" jsonschema:"enable streaming"`
	Concurrency  int    `json:"concurrency,omitempty" jsonschema:"request concurrency, minimum 1"`
	Count        int    `json:"count,omitempty" jsonschema:"request count, minimum 1"`
	TimeoutSec   int    `json:"timeout_sec,omitempty" jsonschema:"timeout in seconds, minimum 1"`
	PromptMode   string `json:"prompt_mode,omitempty" jsonschema:"prompt mode: text, file, generated, or raw"`
	PromptText   string `json:"prompt_text,omitempty" jsonschema:"prompt text"`
	PromptFile   string `json:"prompt_file,omitempty" jsonschema:"prompt file path"`
	PromptLength int    `json:"prompt_length,omitempty" jsonschema:"generated prompt length, minimum 1"`
}

type startRunArgs struct {
	TaskID string `json:"task_id" jsonschema:"task ID"`
}

type getRunStateArgs struct {
	RunID string `json:"run_id" jsonschema:"run ID"`
}

func (s *Server) listTasks(_ context.Context, _ *mcpsdk.CallToolRequest, _ listTasksArgs) (*mcpsdk.CallToolResult, any, error) {
	tasks, err := s.svc.ListTasks()
	if err != nil {
		return nil, nil, fmt.Errorf("list tasks failed: %w", err)
	}
	return textResult(tasks)
}

func (s *Server) createTask(_ context.Context, _ *mcpsdk.CallToolRequest, args createTaskArgs) (*mcpsdk.CallToolResult, any, error) {
	cfg, err := buildTaskConfig(args)
	if err != nil {
		return nil, nil, err
	}
	task, err := s.svc.CreateTask(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create task failed: %w", err)
	}
	return textResult(task)
}

func (s *Server) startRun(_ context.Context, _ *mcpsdk.CallToolRequest, args startRunArgs) (*mcpsdk.CallToolResult, any, error) {
	if strings.TrimSpace(args.TaskID) == "" {
		return nil, nil, fmt.Errorf("task_id is required")
	}
	runID, err := s.svc.StartRun(args.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("start run failed: %w", err)
	}
	return textResult(map[string]any{"run_id": runID})
}

func (s *Server) getRunState(_ context.Context, _ *mcpsdk.CallToolRequest, args getRunStateArgs) (*mcpsdk.CallToolResult, any, error) {
	runID := server.RunID(args.RunID)
	if strings.TrimSpace(string(runID)) == "" {
		return nil, nil, fmt.Errorf("run_id is required")
	}
	state, ok := s.svc.GetRunState(runID)
	if !ok {
		return nil, nil, fmt.Errorf("run not found: %s", runID)
	}
	return textResult(state)
}

func buildTaskConfig(args createTaskArgs) (server.TaskConfig, error) {
	name := args.Name
	protocol := args.Protocol
	model := args.Model
	apiKey := args.APIKey
	if strings.TrimSpace(name) == "" {
		return server.TaskConfig{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(protocol) == "" {
		return server.TaskConfig{}, fmt.Errorf("protocol is required")
	}
	if strings.TrimSpace(model) == "" {
		return server.TaskConfig{}, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return server.TaskConfig{}, fmt.Errorf("api_key is required")
	}

	stream := true
	if args.Stream != nil {
		stream = *args.Stream
	}

	in := types.Input{
		Protocol:     protocol,
		EndpointURL:  args.EndpointURL,
		BaseUrl:      args.BaseURL,
		ApiKey:       apiKey,
		Model:        model,
		Stream:       stream,
		Concurrency:  intOrDefault(args.Concurrency, 10),
		Count:        intOrDefault(args.Count, 100),
		PromptMode:   stringOrDefault(args.PromptMode, "generated"),
		PromptText:   args.PromptText,
		PromptFile:   args.PromptFile,
		PromptLength: intOrDefault(args.PromptLength, 4096),
		Timeout:      time.Duration(intOrDefault(args.TimeoutSec, 30)) * time.Second,
	}
	if in.PromptMode == "text" && strings.TrimSpace(in.PromptText) == "" {
		in.PromptText = "你好，介绍一下你自己。"
	}
	if in.PromptMode == "generated" && in.PromptLength <= 0 {
		in.PromptLength = 4096
	}

	return server.TaskConfig{Name: name, Input: in}, nil
}

func textResult(v any) (*mcpsdk.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}}}, nil, nil
}

func intOrDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func stringOrDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
