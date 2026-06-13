package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	aitserver "github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestSPAHandlerServesIndexAndAssets(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":     {Data: []byte("<html>ait</html>")},
		"assets/app.js":  {Data: []byte("console.log('ait')")},
		"assets/app.css": {Data: []byte("body{}")},
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "root", path: "/", wantStatus: http.StatusOK, wantBody: "ait"},
		{name: "spa route", path: "/tasks/task-1", wantStatus: http.StatusOK, wantBody: "ait"},
		{name: "asset", path: "/assets/app.js", wantStatus: http.StatusOK, wantBody: "console.log"},
		{name: "missing asset", path: "/assets/missing.js", wantStatus: http.StatusNotFound, wantBody: "404"},
	}

	handler := spaHandler(assets)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want to contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestDistFSFindsBuiltDist(t *testing.T) {
	assets, err := distFS()
	if err != nil {
		t.Skipf("web dist is not built: %v", err)
	}
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		t.Fatalf("embedded/dev dist missing index.html: %v", err)
	}
}

func TestAPIHandlerCreatesTaskWithDurationStrings(t *testing.T) {
	svc := newStubServer()
	handler := NewHandler(testAssets(), svc)
	body := []byte(`{
		"name":"web-task",
		"input":{
			"mode":"standard",
			"protocol":"openai-completions",
			"model":"gpt-test",
			"concurrency":2,
			"count":4,
			"prompt_text":"hello",
			"timeout":"30s"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if svc.created.Input.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", svc.created.Input.Timeout)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}
	if got["name"] != "web-task" {
		t.Fatalf("name = %v", got["name"])
	}
}

func TestAPIHandlerListsTasksAndRunRequests(t *testing.T) {
	svc := newStubServer()
	svc.tasks = []types.TaskOverview{{
		TaskDefinition: types.TaskDefinition{ID: "task-1", Name: "Task 1", Input: types.Input{Mode: "standard", Protocol: types.ProtocolOpenAICompletions, Model: "gpt-test"}},
		LatestRun:      &types.TaskRunSummary{RunID: "run-1", TaskID: "task-1", Mode: "standard", Status: "completed", AvgTTFT: 120 * time.Millisecond},
	}}
	svc.runState = &aitserver.RunState{
		RunID:       "run-1",
		TaskID:      "task-1",
		Status:      aitserver.RunStatusCompleted,
		Mode:        "standard",
		TotalReqs:   1,
		DoneReqs:    1,
		SuccessReqs: 1,
		Requests: []*types.RequestMetrics{{
			Index:        0,
			Success:      true,
			TotalTime:    500 * time.Millisecond,
			TTFT:         120 * time.Millisecond,
			TPS:          42,
			RequestBody:  `{}`,
			ResponseBody: `{"ok":true}`,
		}},
	}
	handler := NewHandler(testAssets(), svc)

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "latest_run") {
		t.Fatalf("task list response missing latest_run: %s", listRec.Body.String())
	}

	requestReq := httptest.NewRequest(http.MethodGet, "/api/runs/run-1/requests/0", nil)
	requestRec := httptest.NewRecorder()
	handler.ServeHTTP(requestRec, requestReq)
	if requestRec.Code != http.StatusOK {
		t.Fatalf("request status = %d, body = %s", requestRec.Code, requestRec.Body.String())
	}
	if !strings.Contains(requestRec.Body.String(), `"total_time":"500ms"`) {
		t.Fatalf("request response did not format duration: %s", requestRec.Body.String())
	}
}

func TestAPIHandlerReturnsMetadata(t *testing.T) {
	handler := NewHandler(testAssets(), newStubServer())

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/meta/protocols", nil)
	protocolRec := httptest.NewRecorder()
	handler.ServeHTTP(protocolRec, protocolReq)
	if protocolRec.Code != http.StatusOK {
		t.Fatalf("protocol status = %d", protocolRec.Code)
	}
	if !strings.Contains(protocolRec.Body.String(), types.ProtocolOpenAIResponses) {
		t.Fatalf("protocol response = %s", protocolRec.Body.String())
	}

	suiteReq := httptest.NewRequest(http.MethodGet, "/api/integrity/suites/openai-responses-smoke?protocol=openai-responses", nil)
	suiteRec := httptest.NewRecorder()
	handler.ServeHTTP(suiteRec, suiteReq)
	if suiteRec.Code != http.StatusOK {
		t.Fatalf("suite status = %d", suiteRec.Code)
	}
	if !strings.Contains(suiteRec.Body.String(), "basic-response-shape") {
		t.Fatalf("suite response = %s", suiteRec.Body.String())
	}
}

func testAssets() fs.FS {
	return fstest.MapFS{"index.html": {Data: []byte("<html>ait</html>")}}
}

type stubServer struct {
	tasks    []types.TaskOverview
	created  aitserver.TaskConfig
	runState *aitserver.RunState
	events   chan aitserver.Event
}

func newStubServer() *stubServer {
	return &stubServer{events: make(chan aitserver.Event)}
}

func (s *stubServer) ListTasks() ([]types.TaskOverview, error) { return s.tasks, nil }
func (s *stubServer) GetTask(id string) (types.TaskDefinition, error) {
	for _, task := range s.tasks {
		if task.ID == id {
			return task.TaskDefinition, nil
		}
	}
	return types.TaskDefinition{}, errNotFound("task")
}
func (s *stubServer) ValidateTaskConfig(cfg aitserver.TaskConfig) (aitserver.TaskConfig, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	return cfg, nil
}
func (s *stubServer) CreateTask(cfg aitserver.TaskConfig) (types.TaskDefinition, error) {
	s.created = cfg
	return types.TaskDefinition{ID: "task-created", Name: cfg.Name, Input: cfg.Input}, nil
}
func (s *stubServer) UpdateTask(id string, cfg aitserver.TaskConfig) (types.TaskDefinition, error) {
	return types.TaskDefinition{ID: id, Name: cfg.Name, Input: cfg.Input}, nil
}
func (s *stubServer) DeleteTask(id string) error { return nil }
func (s *stubServer) DuplicateTask(id string) (types.TaskDefinition, error) {
	return types.TaskDefinition{ID: "task-copy", Name: "copy", Input: types.Input{Mode: "standard"}}, nil
}
func (s *stubServer) StartRun(taskID string) (aitserver.RunID, error) { return "run-started", nil }
func (s *stubServer) StopRun(runID aitserver.RunID) error             { return nil }
func (s *stubServer) GetRunState(runID aitserver.RunID) (*aitserver.RunState, bool) {
	if s.runState == nil || s.runState.RunID != runID {
		return nil, false
	}
	return s.runState, true
}
func (s *stubServer) SubscribeRunEvents(runID aitserver.RunID) (<-chan aitserver.Event, aitserver.CancelFunc) {
	return s.events, func() {}
}
func (s *stubServer) ListTaskRunHistory(taskID string, limit int) ([]types.TaskRunSummary, error) {
	return []types.TaskRunSummary{{RunID: "run-1", TaskID: taskID, Mode: "standard", Status: "completed"}}, nil
}
func (s *stubServer) GenerateRunReport(runID aitserver.RunID, format aitserver.ReportFormat) (string, error) {
	return "/tmp/ait-report." + string(format), nil
}
func (s *stubServer) GetAppConfig() (*config.Config, error) { return &config.Config{}, nil }
func (s *stubServer) UpdateProxyURL(proxyURL string) error  { return nil }
func (s *stubServer) ListProtocols() []aitserver.ProtocolMeta {
	return []aitserver.ProtocolMeta{{ID: types.ProtocolOpenAIResponses, Name: "OpenAI Responses", DefaultEndpointURL: types.DefaultEndpointURL(types.ProtocolOpenAIResponses)}}
}
func (s *stubServer) ListIntegritySuites(protocol string) ([]types.IntegritySuite, error) {
	return []types.IntegritySuite{{ID: protocol + "-smoke", Cases: []types.IntegrityCase{{ID: "basic-response-shape"}}}}, nil
}
func (s *stubServer) GetIntegritySuite(protocol, suiteID string) (types.IntegritySuite, error) {
	return types.IntegritySuite{ID: suiteID, Cases: []types.IntegrityCase{{ID: "basic-response-shape"}}}, nil
}
func (s *stubServer) Context() context.Context { return context.Background() }

type errNotFound string

func (e errNotFound) Error() string { return string(e) + " not found" }
