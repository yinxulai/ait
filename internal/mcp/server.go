package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	fmt.Fprintf(os.Stderr, "AIT MCP server starting on stdio; tools=%s\n", strings.Join(toolNames(), ", "))
	err := s.sdk.Run(ctx, &mcpsdk.StdioTransport{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "AIT MCP server stopped with error: %v\n", err)
		return err
	}
	fmt.Fprintln(os.Stderr, "AIT MCP server stopped")
	return nil
}

func toolNames() []string {
	return []string{"ait.list_tasks", "ait.create_task", "ait.run_task", "ait.get_task_state"}
}

func (s *Server) registerTools() {
	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.list_tasks",
		Description: "List all AIT tasks with latest run info",
	}, s.listTasks)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.create_task",
		Description: "Create an AIT task",
	}, s.createTask)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.run_task",
		Description: "Run an AIT task",
	}, s.runTask)

	mcpsdk.AddTool(s.sdk, &mcpsdk.Tool{
		Name:        "ait.get_task_state",
		Description: "Get the current state snapshot for a running or completed task",
	}, s.getTaskState)
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

type runTaskArgs struct {
	TaskID string `json:"task_id" jsonschema:"task ID"`
}

type getTaskStateArgs struct {
	TaskID string `json:"task_id,omitempty" jsonschema:"task ID; when set, returns the task's current or latest state"`
	RunID  string `json:"run_id,omitempty" jsonschema:"optional run ID returned by ait.run_task for exact lookup"`
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

func (s *Server) runTask(_ context.Context, _ *mcpsdk.CallToolRequest, args runTaskArgs) (*mcpsdk.CallToolResult, any, error) {
	if strings.TrimSpace(args.TaskID) == "" {
		return nil, nil, fmt.Errorf("task_id is required")
	}
	runID, err := s.svc.StartRun(args.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("run task failed: %w", err)
	}
	return textResult(map[string]any{"run_id": runID})
}

func (s *Server) getTaskState(_ context.Context, _ *mcpsdk.CallToolRequest, args getTaskStateArgs) (*mcpsdk.CallToolResult, any, error) {
	runID := server.RunID(strings.TrimSpace(args.RunID))
	if runID == "" {
		var err error
		runID, err = s.resolveTaskRunID(args.TaskID)
		if err != nil {
			return nil, nil, err
		}
	}
	state, ok := s.svc.GetRunState(runID)
	if !ok {
		return nil, nil, fmt.Errorf("task state not found for run_id %s", runID)
	}
	return textResult(state)
}

func (s *Server) resolveTaskRunID(taskID string) (server.RunID, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", fmt.Errorf("task_id or run_id is required")
	}

	tasks, err := s.svc.ListTasks()
	if err != nil {
		return "", fmt.Errorf("list tasks failed: %w", err)
	}
	for _, task := range tasks {
		if task.ID != taskID {
			continue
		}
		if task.LatestRun == nil || strings.TrimSpace(task.LatestRun.RunID) == "" {
			return "", fmt.Errorf("task has no run state: %s", taskID)
		}
		return server.RunID(task.LatestRun.RunID), nil
	}
	return "", fmt.Errorf("task not found: %s", taskID)
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
