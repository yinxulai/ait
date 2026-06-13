package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	aitserver "github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/types"
)

type apiHandler struct {
	svc aitserver.Server
}

type errorResponse struct {
	Error string `json:"error"`
}

type taskConfigRequest struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type proxyRequest struct {
	ProxyURL string `json:"proxy_url"`
}

type pathParts []string

func newAPIHandler(svc aitserver.Server) http.Handler {
	return &apiHandler{svc: svc}
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := splitAPIPath(r.URL.Path)
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "api endpoint not found")
		return
	}

	switch parts[0] {
	case "tasks":
		h.handleTasks(w, r, parts)
	case "runs":
		h.handleRuns(w, r, parts)
	case "config":
		h.handleConfig(w, r, parts)
	case "meta":
		h.handleMeta(w, r, parts)
	case "integrity":
		h.handleIntegrity(w, r, parts)
	default:
		writeError(w, http.StatusNotFound, "api endpoint not found")
	}
}

func (h *apiHandler) handleTasks(w http.ResponseWriter, r *http.Request, parts pathParts) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			tasks, err := h.svc.ListTasks()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"tasks": taskOverviewDTOs(tasks)})
		case http.MethodPost:
			cfg, err := decodeTaskConfig(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			created, err := h.svc.CreateTask(cfg)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, taskDTO(created))
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "validate" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		cfg, err := decodeTaskConfig(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		validated, err := h.svc.ValidateTaskConfig(cfg)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"valid": true, "task": taskConfigDTO(validated)})
		return
	}

	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "task endpoint not found")
		return
	}
	taskID := parts[1]

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			task, err := h.svc.GetTask(taskID)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, taskDTO(task))
		case http.MethodPut:
			cfg, err := decodeTaskConfig(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			updated, err := h.svc.UpdateTask(taskID, cfg)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, taskDTO(updated))
		case http.MethodDelete:
			if err := h.svc.DeleteTask(taskID); err != nil {
				writeServiceError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
		}
		return
	}

	if len(parts) == 3 && parts[2] == "duplicate" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		created, err := h.svc.DuplicateTask(taskID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, taskDTO(created))
		return
	}

	if len(parts) == 3 && parts[2] == "runs" {
		switch r.Method {
		case http.MethodGet:
			limit := parseLimit(r, 0)
			runs, err := h.svc.ListTaskRunHistory(taskID, limit)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"runs": runSummaryDTOs(runs)})
		case http.MethodPost:
			runID, err := h.svc.StartRun(taskID)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]any{"run_id": string(runID)})
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	writeError(w, http.StatusNotFound, "task endpoint not found")
}

func (h *apiHandler) handleRuns(w http.ResponseWriter, r *http.Request, parts pathParts) {
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "run endpoint not found")
		return
	}
	runID := aitserver.RunID(parts[1])

	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		state, ok := h.svc.GetRunState(runID)
		if !ok {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, runStateDTO(state, nil))
		return
	}

	switch parts[2] {
	case "stop":
		if len(parts) != 3 || r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		if err := h.svc.StopRun(runID); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
	case "events":
		if len(parts) != 3 || r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.streamRunEvents(w, r, runID)
	case "requests":
		h.handleRunRequests(w, r, runID, parts)
	case "report":
		if len(parts) != 3 || r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.handleRunReport(w, r, runID)
	default:
		writeError(w, http.StatusNotFound, "run endpoint not found")
	}
}

func (h *apiHandler) handleRunRequests(w http.ResponseWriter, r *http.Request, runID aitserver.RunID, parts pathParts) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	state, ok := h.svc.GetRunState(runID)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	requests := requestDTOs(state.Requests)
	if len(parts) == 3 {
		offset := parseIntQuery(r, "offset", 0)
		limit := parseIntQuery(r, "limit", len(requests))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if status != "" {
			filtered := requests[:0]
			for _, req := range requests {
				if req["status"] == status {
					filtered = append(filtered, req)
				}
			}
			requests = filtered
		}
		requests = paginateRequests(requests, offset, limit)
		writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
		return
	}

	if len(parts) == 4 {
		index, err := strconv.Atoi(parts[3])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid request index")
			return
		}
		for _, req := range requests {
			if req["index"] == index {
				writeJSON(w, http.StatusOK, req)
				return
			}
		}
		writeError(w, http.StatusNotFound, "request not found")
		return
	}

	writeError(w, http.StatusNotFound, "request endpoint not found")
}

func (h *apiHandler) handleRunReport(w http.ResponseWriter, r *http.Request, runID aitserver.RunID) {
	format := aitserver.ReportFormat(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = aitserver.ReportFormatJSON
	}
	if format != aitserver.ReportFormatJSON && format != aitserver.ReportFormatCSV {
		writeError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}

	path, err := h.svc.GenerateRunReport(runID, format)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if r.URL.Query().Get("download") == "1" {
		http.ServeFile(w, r, path)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "format": string(format)})
}

func (h *apiHandler) streamRunEvents(w http.ResponseWriter, r *http.Request, runID aitserver.RunID) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if state, ok := h.svc.GetRunState(runID); ok {
		writeSSE(w, "snapshot", runStateDTO(state, nil))
		flusher.Flush()
	}

	events, cancel := h.svc.SubscribeRunEvents(runID)
	defer cancel()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				writeSSE(w, "close", map[string]any{"run_id": string(runID)})
				flusher.Flush()
				return
			}
			writeSSE(w, string(event.Kind), eventDTO(event))
			flusher.Flush()
		}
	}
}

func (h *apiHandler) handleConfig(w http.ResponseWriter, r *http.Request, parts pathParts) {
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		cfg, err := h.svc.GetAppConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				cfg = &config.Config{}
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, cfg)
		return
	}

	if len(parts) == 2 && parts[1] == "proxy" {
		if r.Method != http.MethodPut {
			writeMethodNotAllowed(w, http.MethodPut)
			return
		}
		var req proxyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		if err := h.svc.UpdateProxyURL(req.ProxyURL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"proxy_url": req.ProxyURL})
		return
	}

	writeError(w, http.StatusNotFound, "config endpoint not found")
}

func (h *apiHandler) handleMeta(w http.ResponseWriter, r *http.Request, parts pathParts) {
	if len(parts) == 2 && parts[1] == "protocols" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"protocols": h.svc.ListProtocols()})
		return
	}
	writeError(w, http.StatusNotFound, "meta endpoint not found")
}

func (h *apiHandler) handleIntegrity(w http.ResponseWriter, r *http.Request, parts pathParts) {
	if len(parts) < 2 || parts[1] != "suites" {
		writeError(w, http.StatusNotFound, "integrity endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	protocol := strings.TrimSpace(r.URL.Query().Get("protocol"))
	if protocol == "" {
		protocol = types.ProtocolOpenAICompletions
	}

	if len(parts) == 2 {
		suites, err := h.svc.ListIntegritySuites(protocol)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"suites": suites})
		return
	}

	if len(parts) == 3 {
		suite, err := h.svc.GetIntegritySuite(protocol, parts[2])
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, suite)
		return
	}

	writeError(w, http.StatusNotFound, "integrity endpoint not found")
}

func decodeTaskConfig(body io.Reader) (aitserver.TaskConfig, error) {
	var req taskConfigRequest
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return aitserver.TaskConfig{}, fmt.Errorf("invalid json body: %w", err)
	}
	if len(req.Input) == 0 {
		return aitserver.TaskConfig{}, errors.New("input is required")
	}

	input, err := decodeInput(req.Input)
	if err != nil {
		return aitserver.TaskConfig{}, err
	}
	return aitserver.TaskConfig{Name: strings.TrimSpace(req.Name), Input: input}, nil
}

func decodeInput(raw json.RawMessage) (types.Input, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return types.Input{}, fmt.Errorf("invalid input: %w", err)
	}
	if err := normalizeDurationField(obj, "timeout"); err != nil {
		return types.Input{}, err
	}
	if turbo, ok := obj["turbo_config"].(map[string]any); ok {
		if err := normalizeDurationField(turbo, "max_latency"); err != nil {
			return types.Input{}, err
		}
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return types.Input{}, err
	}
	var input types.Input
	if err := json.Unmarshal(normalized, &input); err != nil {
		return types.Input{}, fmt.Errorf("invalid input: %w", err)
	}
	return input, nil
}

func normalizeDurationField(obj map[string]any, key string) error {
	value, ok := obj[key]
	if !ok || value == nil || value == "" {
		return nil
	}
	switch v := value.(type) {
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("%s must be a valid duration", key)
		}
		obj[key] = int64(d)
	case float64:
		obj[key] = int64(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return fmt.Errorf("%s must be a valid duration", key)
		}
		obj[key] = n
	default:
		return fmt.Errorf("%s must be a valid duration", key)
	}
	return nil
}

func splitAPIPath(path string) pathParts {
	path = strings.Trim(strings.TrimPrefix(path, "/api"), "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseLimit(r *http.Request, fallback int) int {
	return parseIntQuery(r, "limit", fallback)
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func paginateRequests(requests []map[string]any, offset, limit int) []map[string]any {
	if offset >= len(requests) {
		return []map[string]any{}
	}
	if offset < 0 {
		offset = 0
	}
	end := len(requests)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return requests[offset:end]
}

func taskOverviewDTOs(tasks []types.TaskOverview) []map[string]any {
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		item := taskDTO(task.TaskDefinition)
		if task.LatestRun != nil {
			item["latest_run"] = runSummaryDTO(*task.LatestRun)
		}
		out = append(out, item)
	}
	return out
}

func taskDTO(task types.TaskDefinition) map[string]any {
	return map[string]any{
		"id":         task.ID,
		"name":       task.Name,
		"mode":       task.Input.RunMode(),
		"input":      inputDTO(task.Input),
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
	}
}

func taskConfigDTO(cfg aitserver.TaskConfig) map[string]any {
	return map[string]any{
		"name":  cfg.Name,
		"mode":  cfg.Input.RunMode(),
		"input": inputDTO(cfg.Input),
	}
}

func inputDTO(input types.Input) map[string]any {
	return map[string]any{
		"mode":          input.RunMode(),
		"protocol":      input.NormalizedProtocol(),
		"endpoint_url":  input.ResolvedEndpointURL(),
		"base_url":      input.BaseUrl,
		"proxy_url":     input.ProxyURL,
		"model":         input.Model,
		"concurrency":   input.Concurrency,
		"count":         input.Count,
		"stream":        input.Stream,
		"thinking":      input.Thinking,
		"turbo":         input.Turbo,
		"turbo_config":  turboConfigDTO(input.TurboConfig),
		"integrity":     input.Integrity,
		"prompt_mode":   input.PromptMode,
		"prompt_text":   input.PromptText,
		"prompt_file":   input.PromptFile,
		"prompt_length": input.PromptLength,
		"report":        input.Report,
		"timeout":       durationString(input.Timeout),
		"log":           input.Log,
	}
}

func turboConfigDTO(cfg types.TurboConfig) map[string]any {
	return map[string]any{
		"init_concurrency": cfg.InitConcurrency,
		"max_concurrency":  cfg.MaxConcurrency,
		"step_size":        cfg.StepSize,
		"level_requests":   cfg.LevelRequests,
		"min_success_rate": cfg.MinSuccessRate,
		"max_latency":      durationString(cfg.MaxLatency),
	}
}

func runSummaryDTOs(runs []types.TaskRunSummary) []map[string]any {
	out := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		out = append(out, runSummaryDTO(run))
	}
	return out
}

func runSummaryDTO(run types.TaskRunSummary) map[string]any {
	return map[string]any{
		"run_id":                 run.RunID,
		"task_id":                run.TaskID,
		"mode":                   run.Mode,
		"status":                 run.Status,
		"protocol":               run.Protocol,
		"model":                  run.Model,
		"started_at":             run.StartedAt,
		"finished_at":            run.FinishedAt,
		"success_rate":           run.SuccessRate,
		"avg_ttft":               durationString(run.AvgTTFT),
		"avg_tps":                run.AvgTPS,
		"cache_hit_rate":         run.CacheHitRate,
		"rpm":                    run.RPM,
		"tpm":                    run.TPM,
		"max_stable_concurrency": run.MaxStableConcurrency,
		"error_summary":          run.ErrorSummary,
	}
}

func runStateDTO(state *aitserver.RunState, requests []map[string]any) map[string]any {
	if state == nil {
		return nil
	}
	if requests == nil {
		requests = requestDTOs(state.Requests)
	}
	return map[string]any{
		"run_id":         string(state.RunID),
		"task_id":        state.TaskID,
		"status":         string(state.Status),
		"mode":           state.Mode,
		"started_at":     state.StartedAt,
		"finished_at":    state.FinishedAt,
		"total_reqs":     state.TotalReqs,
		"queued_reqs":    state.QueuedReqs,
		"running_reqs":   state.RunningReqs,
		"done_reqs":      state.DoneReqs,
		"success_reqs":   state.SuccessReqs,
		"failed_reqs":    state.FailedReqs,
		"skipped_reqs":   state.SkippedReqs,
		"avg_tps":        state.AvgTPS,
		"avg_ttft":       durationString(state.AvgTTFT),
		"success_rate":   state.SuccessRate,
		"cache_hit_rate": state.CacheHitRate,
		"rpm":            state.RPM,
		"tpm":            state.TPM,
		"requests":       requests,
		"request_states": requestStateDTOs(state.RequestStates),
		"mode_state":     state.ModeState,
		"mode_result":    state.ModeResult,
		"error_msg":      state.ErrorMsg,
	}
}

func requestStateDTOs(states map[int]aitserver.RequestState) []map[string]any {
	out := make([]map[string]any, 0, len(states))
	for _, state := range states {
		out = append(out, map[string]any{
			"index":       state.Index,
			"status":      string(state.Status),
			"level":       state.Level,
			"case_id":     state.CaseID,
			"queued_at":   state.QueuedAt,
			"started_at":  state.StartedAt,
			"finished_at": state.FinishedAt,
			"error_msg":   state.ErrorMsg,
		})
	}
	return out
}

func requestDTOs(requests []*types.RequestMetrics) []map[string]any {
	out := make([]map[string]any, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		out = append(out, requestDTO(*request))
	}
	return out
}

func requestDTO(request types.RequestMetrics) map[string]any {
	status := "ok"
	if !request.Success {
		status = "failed"
	}
	return map[string]any{
		"index":             request.Index,
		"status":            status,
		"success":           request.Success,
		"total_time":        durationString(request.TotalTime),
		"ttft":              durationString(request.TTFT),
		"tps":               request.TPS,
		"prompt_tokens":     request.PromptTokens,
		"completion_tokens": request.CompletionTokens,
		"cached_tokens":     request.CachedTokens,
		"cache_hit_rate":    request.CacheHitRate,
		"dns_time":          durationString(request.DNSTime),
		"connect_time":      durationString(request.ConnectTime),
		"tls_time":          durationString(request.TLSTime),
		"target_ip":         request.TargetIP,
		"error_message":     request.ErrorMessage,
		"request_body":      request.RequestBody,
		"response_body":     request.ResponseBody,
		"level":             request.Level,
	}
}

func eventDTO(event aitserver.Event) map[string]any {
	payload := event.Payload
	if state, ok := payload.(*aitserver.RunState); ok {
		payload = runStateDTO(state, nil)
	}
	return map[string]any{
		"run_id":  string(event.RunID),
		"kind":    string(event.Kind),
		"payload": payload,
	}
}

func durationString(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeServiceError(w http.ResponseWriter, err error) {
	message := err.Error()
	switch {
	case strings.Contains(message, "not found"):
		writeError(w, http.StatusNotFound, message)
	case strings.Contains(message, "currently running") || strings.Contains(message, "still in progress"):
		writeError(w, http.StatusConflict, message)
	default:
		writeError(w, http.StatusBadRequest, message)
	}
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeSSE(w io.Writer, eventName string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		data, _ = json.Marshal(errorResponse{Error: err.Error()})
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", eventName)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}
