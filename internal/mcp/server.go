package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

type Server struct {
	svc server.Server
}

func New(svc server.Server) *Server {
	return &Server{svc: svc}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *respError  `json:"error,omitempty"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolsCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) Run(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		payload, err := readFrame(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var req request
		if err := json.Unmarshal(payload, &req); err != nil {
			if err := writeResp(out, response{JSONRPC: "2.0", Error: &respError{Code: -32700, Message: "invalid json"}}); err != nil {
				return err
			}
			continue
		}

		// Notification: no id means no response is required.
		if len(req.ID) == 0 {
			continue
		}

		id := decodeID(req.ID)
		resp := s.handleRequest(id, req)
		if err := writeResp(out, resp); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(id any, req request) response {
	switch req.Method {
	case "initialize":
		return response{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]any{
					"name":    "ait",
					"version": "0.1.0",
				},
			},
		}
	case "tools/list":
		return response{JSONRPC: "2.0", ID: id, Result: toolsListResult{Tools: toolDefs()}}
	case "tools/call":
		var p toolsCallParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return errResp(id, -32602, "invalid tools/call params")
		}
		res, err := s.callTool(p.Name, p.Arguments)
		if err != nil {
			return response{JSONRPC: "2.0", ID: id, Result: toolsCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true}}
		}
		return response{JSONRPC: "2.0", ID: id, Result: res}
	default:
		return errResp(id, -32601, "method not found")
	}
}

func (s *Server) callTool(name string, args map[string]any) (toolsCallResult, error) {
	switch name {
	case "ait.list_tasks":
		tasks, err := s.svc.ListTasks()
		if err != nil {
			return toolsCallResult{}, fmt.Errorf("list tasks failed: %w", err)
		}
		return textResult(tasks)
	case "ait.create_task":
		cfg, err := buildTaskConfig(args)
		if err != nil {
			return toolsCallResult{}, err
		}
		task, err := s.svc.CreateTask(cfg)
		if err != nil {
			return toolsCallResult{}, fmt.Errorf("create task failed: %w", err)
		}
		return textResult(task)
	case "ait.start_run":
		taskID := strArg(args, "task_id", "")
		if strings.TrimSpace(taskID) == "" {
			return toolsCallResult{}, fmt.Errorf("task_id is required")
		}
		runID, err := s.svc.StartRun(taskID)
		if err != nil {
			return toolsCallResult{}, fmt.Errorf("start run failed: %w", err)
		}
		return textResult(map[string]any{"run_id": runID})
	case "ait.get_run_state":
		runID := server.RunID(strArg(args, "run_id", ""))
		if strings.TrimSpace(string(runID)) == "" {
			return toolsCallResult{}, fmt.Errorf("run_id is required")
		}
		state, ok := s.svc.GetRunState(runID)
		if !ok {
			return toolsCallResult{}, fmt.Errorf("run not found: %s", runID)
		}
		return textResult(state)
	default:
		return toolsCallResult{}, fmt.Errorf("unknown tool: %s", name)
	}
}

func toolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "ait.list_tasks",
			Description: "List all AIT tasks with latest run info",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "ait.create_task",
			Description: "Create an AIT task (inherits internal/server task creation capability)",
			InputSchema: map[string]any{
				"type": "object",
				"required": []string{"name", "protocol", "model", "api_key"},
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"protocol": map[string]any{"type": "string", "enum": []string{"openai-completions", "openai-responses", "anthropic-messages"}},
					"endpoint_url": map[string]any{"type": "string"},
					"base_url": map[string]any{"type": "string"},
					"api_key": map[string]any{"type": "string"},
					"model": map[string]any{"type": "string"},
					"stream": map[string]any{"type": "boolean"},
					"concurrency": map[string]any{"type": "integer", "minimum": 1},
					"count": map[string]any{"type": "integer", "minimum": 1},
					"timeout_sec": map[string]any{"type": "integer", "minimum": 1},
					"prompt_mode": map[string]any{"type": "string", "enum": []string{"text", "file", "generated", "raw"}},
					"prompt_text": map[string]any{"type": "string"},
					"prompt_file": map[string]any{"type": "string"},
					"prompt_length": map[string]any{"type": "integer", "minimum": 1},
				},
			},
		},
		{
			Name:        "ait.start_run",
			Description: "Start run for a task",
			InputSchema: map[string]any{
				"type": "object",
				"required": []string{"task_id"},
				"properties": map[string]any{"task_id": map[string]any{"type": "string"}},
			},
		},
		{
			Name:        "ait.get_run_state",
			Description: "Get current run state snapshot",
			InputSchema: map[string]any{
				"type": "object",
				"required": []string{"run_id"},
				"properties": map[string]any{"run_id": map[string]any{"type": "string"}},
			},
		},
	}
}

func buildTaskConfig(args map[string]any) (server.TaskConfig, error) {
	name := strArg(args, "name", "")
	protocol := strArg(args, "protocol", "")
	model := strArg(args, "model", "")
	apiKey := strArg(args, "api_key", "")
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

	in := types.Input{
		Protocol:     protocol,
		EndpointURL:  strArg(args, "endpoint_url", ""),
		BaseUrl:      strArg(args, "base_url", ""),
		ApiKey:       apiKey,
		Model:        model,
		Stream:       boolArg(args, "stream", true),
		Concurrency:  intArg(args, "concurrency", 10),
		Count:        intArg(args, "count", 100),
		PromptMode:   strArg(args, "prompt_mode", "generated"),
		PromptText:   strArg(args, "prompt_text", ""),
		PromptFile:   strArg(args, "prompt_file", ""),
		PromptLength: intArg(args, "prompt_length", 4096),
		Timeout:      time.Duration(intArg(args, "timeout_sec", 30)) * time.Second,
	}
	if in.PromptMode == "" {
		in.PromptMode = "generated"
	}
	if in.PromptMode == "text" && strings.TrimSpace(in.PromptText) == "" {
		in.PromptText = "你好，介绍一下你自己。"
	}
	if in.PromptMode == "generated" && in.PromptLength <= 0 {
		in.PromptLength = 4096
	}

	return server.TaskConfig{Name: name, Input: in}, nil
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "content-length:") {
			v := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid content-length: %w", err)
			}
			length = n
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeResp(out io.Writer, resp response) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	_, err = out.Write(payload)
	return err
}

func decodeID(raw json.RawMessage) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func errResp(id any, code int, msg string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &respError{Code: code, Message: msg}}
}

func textResult(v any) (toolsCallResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolsCallResult{}, err
	}
	return toolsCallResult{Content: []toolContent{{Type: "text", Text: string(b)}}}, nil
}

func strArg(args map[string]any, key, def string) string {
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func intArg(args map[string]any, key string, def int) int {
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return def
	}
}

func boolArg(args map[string]any, key string, def bool) bool {
	if args == nil {
		return def
	}
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}
