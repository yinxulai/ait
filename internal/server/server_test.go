package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/store"
	"github.com/yinxulai/ait/internal/types"
)

// ── test helpers ──────────────────────────────────────────────────────────────

func newTestServer(t *testing.T) *serverImpl {
	t.Helper()
	dir := t.TempDir()
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		t.Fatalf("mkdir history: %v", err)
	}
	ts := store.NewTaskStore(filepath.Join(dir, "tasks.json"))
	if err := ts.Load(); err != nil {
		t.Fatalf("load task store: %v", err)
	}
	return &serverImpl{
		taskStore:  ts,
		bus:        newEventBus(),
		activeRuns: make(map[RunID]*activeRun),
		historyDir: historyDir,
	}
}

func makeTaskConfig(name string) TaskConfig {
	return TaskConfig{
		Name: name,
		Input: types.Input{
			Protocol:    types.ProtocolOpenAICompletions,
			EndpointURL: "http://localhost:19999",
			Model:       "test-model",
			Concurrency: 1,
			Count:       1,
			PromptMode:  "text",
			PromptText:  "hello",
		},
	}
}

// ── eventBus ──────────────────────────────────────────────────────────────────

func TestEventBus_PublishDelivered(t *testing.T) {
	bus := newEventBus()
	rid := RunID("run_1")
	ch, cancel := bus.Subscribe(rid)
	defer cancel()

	want := Event{RunID: rid, Kind: EventRequestDone}
	bus.Publish(want)

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := newEventBus()
	rid := RunID("run_multi")
	const n = 3
	chs := make([]<-chan Event, n)
	for i := range chs {
		ch, cancel := bus.Subscribe(rid)
		chs[i] = ch
		defer cancel()
	}

	ev := Event{RunID: rid, Kind: EventRunComplete}
	bus.Publish(ev)

	for i, ch := range chs {
		select {
		case got := <-ch:
			if got != ev {
				t.Errorf("subscriber %d: got %v, want %v", i, got, ev)
			}
		case <-time.After(time.Second):
			t.Errorf("subscriber %d: timeout", i)
		}
	}
}

func TestEventBus_CancelClosesChannel(t *testing.T) {
	bus := newEventBus()
	rid := RunID("run_cancel")
	ch, cancel := bus.Subscribe(rid)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: channel not closed after cancel")
	}
}

func TestEventBus_CloseRunClosesAllChannels(t *testing.T) {
	bus := newEventBus()
	rid := RunID("run_close")
	ch1, _ := bus.Subscribe(rid)
	ch2, _ := bus.Subscribe(rid)

	bus.CloseRun(rid)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("ch%d should be closed after CloseRun", i+1)
			}
		case <-time.After(time.Second):
			t.Errorf("ch%d: timeout waiting for close", i+1)
		}
	}
}

func TestEventBus_FullChannelDoesNotBlock(t *testing.T) {
	bus := newEventBus()
	rid := RunID("run_full")
	// Subscribe but never drain the channel.
	_, cancel := bus.Subscribe(rid)
	defer cancel()

	done := make(chan struct{})
	go func() {
		// Publish more events than the channel capacity (64) to verify non-blocking.
		for i := 0; i < 100; i++ {
			bus.Publish(Event{RunID: rid, Kind: EventRequestDone})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on full subscriber channel")
	}
}

func TestEventBus_EventsOnlyDeliveredToMatchingRunID(t *testing.T) {
	bus := newEventBus()
	ch1, cancel1 := bus.Subscribe(RunID("run_a"))
	ch2, cancel2 := bus.Subscribe(RunID("run_b"))
	defer cancel1()
	defer cancel2()

	bus.Publish(Event{RunID: "run_a", Kind: EventRequestDone})

	select {
	case <-ch1:
		// expected
	case <-time.After(time.Second):
		t.Fatal("run_a subscriber should have received event")
	}

	// run_b should NOT receive the event.
	select {
	case <-ch2:
		t.Fatal("run_b subscriber should not receive event for run_a")
	default:
	}
}

// ── mapRequestMetrics ─────────────────────────────────────────────────────────

func TestMapRequestMetrics_NilMetricsNoError(t *testing.T) {
	rm := mapRequestMetrics(nil, 3, nil)
	if rm.Index != 3 {
		t.Errorf("Index: got %d, want 3", rm.Index)
	}
	if rm.Success {
		t.Error("expected Success=false")
	}
	if rm.ErrorMessage != "" {
		t.Errorf("expected empty ErrorMessage, got %q", rm.ErrorMessage)
	}
}

func TestMapRequestMetrics_NilMetricsWithError(t *testing.T) {
	err := errors.New("connection refused")
	rm := mapRequestMetrics(nil, 0, err)
	if rm.Success {
		t.Error("expected Success=false")
	}
	if rm.ErrorMessage != err.Error() {
		t.Errorf("ErrorMessage: got %q, want %q", rm.ErrorMessage, err.Error())
	}
}

func TestMapRequestMetrics_SuccessFields(t *testing.T) {
	m := &client.ResponseMetrics{
		TotalTime:         2 * time.Second,
		TimeToFirstToken:  100 * time.Millisecond,
		CompletionTokens:  100,
		PromptTokens:      200,
		CachedInputTokens: 50,
		TargetIP:          "1.2.3.4",
		DNSTime:           5 * time.Millisecond,
		ConnectTime:       10 * time.Millisecond,
		TLSHandshakeTime:  15 * time.Millisecond,
	}
	rm := mapRequestMetrics(m, 5, nil)

	if !rm.Success {
		t.Error("expected Success=true")
	}
	if rm.Index != 5 {
		t.Errorf("Index: got %d, want 5", rm.Index)
	}
	// TPS = CompletionTokens / TotalTime.Seconds() = 100 / 2 = 50
	if rm.TPS != 50.0 {
		t.Errorf("TPS: got %v, want 50", rm.TPS)
	}
	// CacheHitRate = CachedInputTokens / PromptTokens = 50 / 200 = 0.25
	if rm.CacheHitRate != 0.25 {
		t.Errorf("CacheHitRate: got %v, want 0.25", rm.CacheHitRate)
	}
	if rm.TargetIP != "1.2.3.4" {
		t.Errorf("TargetIP: got %q, want %q", rm.TargetIP, "1.2.3.4")
	}
	if rm.TTFT != 100*time.Millisecond {
		t.Errorf("TTFT: got %v, want 100ms", rm.TTFT)
	}
	if rm.CompletionTokens != 100 {
		t.Errorf("CompletionTokens: got %d, want 100", rm.CompletionTokens)
	}
	if rm.PromptTokens != 200 {
		t.Errorf("PromptTokens: got %d, want 200", rm.PromptTokens)
	}
	if rm.CachedTokens != 50 {
		t.Errorf("CachedTokens: got %d, want 50", rm.CachedTokens)
	}
}

func TestMapRequestMetrics_FailureFromErrorMessage(t *testing.T) {
	m := &client.ResponseMetrics{ErrorMessage: "rate limit exceeded"}
	rm := mapRequestMetrics(m, 0, nil)
	if rm.Success {
		t.Error("expected Success=false when ErrorMessage is set")
	}
	if rm.ErrorMessage != "rate limit exceeded" {
		t.Errorf("ErrorMessage: got %q", rm.ErrorMessage)
	}
}

func TestMapRequestMetrics_ErrorOverridesEmptyMessage(t *testing.T) {
	m := &client.ResponseMetrics{}
	err := errors.New("transport error")
	rm := mapRequestMetrics(m, 0, err)
	if rm.Success {
		t.Error("expected Success=false")
	}
	if rm.ErrorMessage != err.Error() {
		t.Errorf("ErrorMessage: got %q, want %q", rm.ErrorMessage, err.Error())
	}
}

func TestMapRequestMetrics_ZeroTotalTimeSkipsTPS(t *testing.T) {
	m := &client.ResponseMetrics{CompletionTokens: 100} // TotalTime == 0
	rm := mapRequestMetrics(m, 0, nil)
	if rm.TPS != 0 {
		t.Errorf("expected TPS=0 when TotalTime=0, got %v", rm.TPS)
	}
}

func TestMapRequestMetrics_ZeroPromptTokensSkipsCacheHitRate(t *testing.T) {
	m := &client.ResponseMetrics{CachedInputTokens: 10} // PromptTokens == 0
	rm := mapRequestMetrics(m, 0, nil)
	if rm.CacheHitRate != 0 {
		t.Errorf("expected CacheHitRate=0 when PromptTokens=0, got %v", rm.CacheHitRate)
	}
}

// ── snapshotState ─────────────────────────────────────────────────────────────

func TestSnapshotState_DeepCopiesRequests(t *testing.T) {
	original := &RequestMetrics{Index: 0, Success: true}
	ar := &activeRun{
		state: &RunState{
			Requests: []*RequestMetrics{original},
		},
	}
	snap := ar.snapshotState()

	// Mutate original slice — snapshot must remain unchanged.
	ar.state.Requests[0] = &RequestMetrics{Index: 99}
	if snap.Requests[0].Index != 0 {
		t.Error("Requests slice was not deep-copied: snapshot reflects mutation of original")
	}
}

func TestSnapshotState_DeepCopiesLevels(t *testing.T) {
	ar := &activeRun{
		state: &RunState{
			Levels: []types.TurboLevelResult{{Concurrency: 5}},
		},
	}
	snap := ar.snapshotState()

	ar.state.Levels[0] = types.TurboLevelResult{Concurrency: 99}
	if snap.Levels[0].Concurrency != 5 {
		t.Error("Levels slice was not deep-copied: snapshot reflects mutation of original")
	}
}

func TestSnapshotState_EmptySlicesNotCopied(t *testing.T) {
	ar := &activeRun{
		state: &RunState{
			RunID:  "run_snap",
			Status: RunStatusRunning,
		},
	}
	snap := ar.snapshotState()
	if snap.RunID != "run_snap" {
		t.Errorf("RunID: got %q, want %q", snap.RunID, "run_snap")
	}
	if snap.Requests != nil {
		t.Error("expected nil Requests for empty state")
	}
}

// ── historyPath ───────────────────────────────────────────────────────────────

func TestHistoryPath(t *testing.T) {
	got := historyPath("/data/history", "task-abc")
	want := filepath.Join("/data/history", "task-abc.json")
	if got != want {
		t.Errorf("historyPath: got %q, want %q", got, want)
	}
}

// ── task CRUD ─────────────────────────────────────────────────────────────────

func TestListTasks_Empty(t *testing.T) {
	s := newTestServer(t)
	tasks := s.ListTasks()
	if len(tasks) != 0 {
		t.Errorf("expected empty list, got %d tasks", len(tasks))
	}
}

func TestCreateTask_ReturnsTaskWithID(t *testing.T) {
	s := newTestServer(t)
	got, err := s.CreateTask(makeTaskConfig("my-task"))
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if got.Name != "my-task" {
		t.Errorf("Name: got %q, want %q", got.Name, "my-task")
	}
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateTask_AppearsInList(t *testing.T) {
	s := newTestServer(t)
	s.CreateTask(makeTaskConfig("task-a"))
	all := s.ListTasks()
	if len(all) != 1 {
		t.Errorf("expected 1 task, got %d", len(all))
	}
	if all[0].Name != "task-a" {
		t.Errorf("Name: got %q, want task-a", all[0].Name)
	}
}

func TestCreateTask_MultipleTasksAllListed(t *testing.T) {
	s := newTestServer(t)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if _, err := s.CreateTask(makeTaskConfig(name)); err != nil {
			t.Fatalf("CreateTask %q: %v", name, err)
		}
	}
	if len(s.ListTasks()) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(s.ListTasks()))
	}
}

func TestGetTask_Found(t *testing.T) {
	s := newTestServer(t)
	created, _ := s.CreateTask(makeTaskConfig("task-get"))
	got, ok := s.GetTask(created.ID)
	if !ok {
		t.Fatal("GetTask returned not found")
	}
	if got.ID != created.ID {
		t.Errorf("ID mismatch: %q vs %q", got.ID, created.ID)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	s := newTestServer(t)
	_, ok := s.GetTask("nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent ID")
	}
}

func TestUpdateTask_Success(t *testing.T) {
	s := newTestServer(t)
	created, _ := s.CreateTask(makeTaskConfig("original"))
	updated, err := s.UpdateTask(created.ID, makeTaskConfig("renamed"))
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("Name: got %q, want renamed", updated.Name)
	}
	// Verify persistence via GetTask.
	fetched, ok := s.GetTask(created.ID)
	if !ok || fetched.Name != "renamed" {
		t.Errorf("GetTask after update: ok=%v name=%q", ok, fetched.Name)
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	s := newTestServer(t)
	_, err := s.UpdateTask("missing-id", makeTaskConfig("x"))
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestDeleteTask_Success(t *testing.T) {
	s := newTestServer(t)
	created, _ := s.CreateTask(makeTaskConfig("to-delete"))
	if err := s.DeleteTask(created.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, ok := s.GetTask(created.ID); ok {
		t.Error("task still accessible after delete")
	}
	if len(s.ListTasks()) != 0 {
		t.Error("expected empty list after delete")
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	s := newTestServer(t)
	if err := s.DeleteTask("missing-id"); err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestCopyTask_CreatesNewTask(t *testing.T) {
	s := newTestServer(t)
	original, _ := s.CreateTask(makeTaskConfig("original"))
	copied, err := s.CopyTask(original.ID)
	if err != nil {
		t.Fatalf("CopyTask: %v", err)
	}
	if copied.ID == original.ID {
		t.Error("copy should have a new ID")
	}
	if copied.Name != "original (copy)" {
		t.Errorf("Name: got %q, want %q", copied.Name, "original (copy)")
	}
	if len(s.ListTasks()) != 2 {
		t.Errorf("expected 2 tasks after copy, got %d", len(s.ListTasks()))
	}
}

func TestCopyTask_NotFound(t *testing.T) {
	s := newTestServer(t)
	_, err := s.CopyTask("missing-id")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

// ── run management ────────────────────────────────────────────────────────────

func TestStartRun_TaskNotFound(t *testing.T) {
	s := newTestServer(t)
	_, err := s.StartRun("no-such-task")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
}

func TestStartRun_ReturnsRunIDAndRegistersActiveRun(t *testing.T) {
	s := newTestServer(t)
	task, _ := s.CreateTask(makeTaskConfig("run-task"))
	runID, err := s.StartRun(task.ID)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if runID == "" {
		t.Fatal("expected non-empty RunID")
	}
	state, ok := s.GetRunState(runID)
	if !ok {
		t.Fatal("GetRunState: run not found immediately after StartRun")
	}
	if state.TaskID != task.ID {
		t.Errorf("TaskID: got %q, want %q", state.TaskID, task.ID)
	}
	// Initial status should be running (goroutine may not have progressed yet).
	if state.Status != RunStatusRunning {
		t.Errorf("Status: got %q, want %q", state.Status, RunStatusRunning)
	}
}

func TestGetRunState_NotFound(t *testing.T) {
	s := newTestServer(t)
	_, ok := s.GetRunState("run_nonexistent")
	if ok {
		t.Fatal("expected not found for unknown RunID")
	}
}

func TestStopRun_NotFound(t *testing.T) {
	s := newTestServer(t)
	err := s.StopRun("run_nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown RunID")
	}
}

func TestStopRun_ActiveRunNoRunner(t *testing.T) {
	s := newTestServer(t)
	// Inject an activeRun with no runner/engine (neither rnr nor turboEngine).
	runID := RunID("run_no_engine")
	s.mu.Lock()
	s.activeRuns[runID] = &activeRun{
		state: &RunState{RunID: runID, Status: RunStatusRunning},
	}
	s.mu.Unlock()

	// Should not panic; both rnr and engine are nil — stop is a no-op.
	if err := s.StopRun(runID); err != nil {
		t.Fatalf("StopRun: unexpected error: %v", err)
	}
}

func TestGetHistory_EmptyForNewTask(t *testing.T) {
	s := newTestServer(t)
	task, _ := s.CreateTask(makeTaskConfig("hist-task"))
	history, err := s.GetHistory(task.ID, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d entries", len(history))
	}
}

func TestGetHistory_PersistsAfterRun(t *testing.T) {
	s := newTestServer(t)
	task, _ := s.CreateTask(makeTaskConfig("persist-task"))

	summary := types.TaskRunSummary{
		RunID:   "run_test",
		TaskID:  task.ID,
		Mode:    "standard",
		Status:  string(RunStatusCompleted),
		StartedAt: time.Now().Add(-time.Second),
		FinishedAt: time.Now(),
	}
	s.persistRunResult(task.ID, s.historyDir, summary)

	history, err := s.GetHistory(task.ID, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if history[0].RunID != "run_test" {
		t.Errorf("RunID: got %q, want run_test", history[0].RunID)
	}
}

func TestGetHistory_LimitRespected(t *testing.T) {
	s := newTestServer(t)
	task, _ := s.CreateTask(makeTaskConfig("limit-task"))

	for i := 0; i < 5; i++ {
		s.persistRunResult(task.ID, s.historyDir, types.TaskRunSummary{
			RunID:      "run_" + string(rune('0'+i)),
			TaskID:     task.ID,
			StartedAt:  time.Now(),
			FinishedAt: time.Now(),
		})
	}

	history, err := s.GetHistory(task.ID, 3)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 entries with limit=3, got %d", len(history))
	}
}

// ── GenerateReport ────────────────────────────────────────────────────────────

func TestGenerateReport_RunNotFound(t *testing.T) {
	s := newTestServer(t)
	_, err := s.GenerateReport("run_missing", ReportFormatJSON)
	if err == nil {
		t.Fatal("expected error for missing run")
	}
}

func TestGenerateReport_StillRunning(t *testing.T) {
	s := newTestServer(t)
	runID := RunID("run_in_progress")
	s.mu.Lock()
	s.activeRuns[runID] = &activeRun{
		state: &RunState{RunID: runID, Status: RunStatusRunning, Mode: "standard"},
	}
	s.mu.Unlock()

	_, err := s.GenerateReport(runID, ReportFormatJSON)
	if err == nil {
		t.Fatal("expected error for in-progress run")
	}
	if !strings.Contains(err.Error(), "in progress") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateReport_TurboNotSupported(t *testing.T) {
	s := newTestServer(t)
	runID := RunID("run_turbo")
	s.mu.Lock()
	s.activeRuns[runID] = &activeRun{
		state: &RunState{RunID: runID, Status: RunStatusCompleted, Mode: "turbo"},
	}
	s.mu.Unlock()

	_, err := s.GenerateReport(runID, ReportFormatJSON)
	if err == nil {
		t.Fatal("expected error for turbo run")
	}
	if !strings.Contains(err.Error(), "turbo") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateReport_NoResultData(t *testing.T) {
	s := newTestServer(t)
	runID := RunID("run_no_result")
	s.mu.Lock()
	s.activeRuns[runID] = &activeRun{
		state: &RunState{RunID: runID, Status: RunStatusFailed, Mode: "standard"},
	}
	s.mu.Unlock()

	_, err := s.GenerateReport(runID, ReportFormatJSON)
	if err == nil {
		t.Fatal("expected error for run with no result data")
	}
}

// ── Subscribe ─────────────────────────────────────────────────────────────────

func TestSubscribe_DelegatesEventBus(t *testing.T) {
	s := newTestServer(t)
	runID := RunID("run_sub")
	ch, cancel := s.Subscribe(runID)
	defer cancel()

	ev := Event{RunID: runID, Kind: EventRunComplete}
	s.bus.Publish(ev)

	select {
	case got := <-ch:
		if got != ev {
			t.Fatalf("got %v, want %v", got, ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event via Subscribe")
	}
}

func TestSubscribe_ChannelClosedAfterCloseRun(t *testing.T) {
	s := newTestServer(t)
	runID := RunID("run_lifecycle")
	ch, _ := s.Subscribe(runID)

	s.bus.CloseRun(runID)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after CloseRun")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: channel not closed after CloseRun")
	}
}
