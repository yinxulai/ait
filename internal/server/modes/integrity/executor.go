package integrity

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/modes/integrity/assertion"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/task"
	"github.com/yinxulai/ait/internal/server/types"
)

type RunnerFactory func(types.Input, types.IntegrityCase) (CaseRunner, error)

type CaseRunner interface {
	RunWithCallback(standard.RequestDoneCallback) (*types.ReportData, error)
	Stop()
}

type Executor struct {
	TaskID        string
	Input         types.Input
	Suite         types.IntegritySuite
	RunnerFactory RunnerFactory
	OnCaseStarted func(types.IntegrityCase)
	OnCaseDone    func(types.IntegrityCaseResult)
	OnRequestDone func(types.IntegrityCase, *client.ResponseMetrics, int, error, []types.AssertionResult)

	mu            sync.Mutex
	currentRunner CaseRunner
	stopped       bool
}

func NewExecutor(taskID string, input types.Input, suite types.IntegritySuite) *Executor {
	return &Executor{
		TaskID: taskID,
		Input:  input,
		Suite:  suite,
		RunnerFactory: func(caseInput types.Input, _ types.IntegrityCase) (CaseRunner, error) {
			return standard.NewRunner(taskID, caseInput)
		},
	}
}

func (e *Executor) Run() (*types.IntegrityResult, error) {
	started := time.Now()
	result := &types.IntegrityResult{
		SuiteID:    e.Suite.ID,
		Status:     "running",
		StartedAt:  started,
		Cases:      []types.IntegrityCaseResult{},
		Assertions: []types.AssertionResult{},
	}

	for _, c := range e.Suite.Cases {
		caseResult, err := e.runCase(c)
		result.Cases = append(result.Cases, caseResult)
		result.Assertions = append(result.Assertions, caseResult.Assertions...)
		if err != nil && caseResult.Status == "" {
			caseResult.Status = "failed"
		}
		if e.Input.Integrity.FailFast && caseResult.Required && caseResult.Status == "failed" {
			break
		}
	}

	finished := time.Now()
	result.FinishedAt = &finished
	result.Duration = finished.Sub(started)
	result.TotalCases = len(result.Cases)
	for _, c := range result.Cases {
		switch c.Status {
		case "passed":
			result.PassedCases++
		case "warned":
			result.WarnedCases++
		case "skipped":
			result.SkippedCases++
		default:
			result.FailedCases++
			if c.Required {
				result.RequiredFailedCases++
			}
		}
	}
	if result.FailedCases > 0 || result.RequiredFailedCases > 0 {
		result.Status = "failed"
	} else {
		result.Status = "completed"
	}
	return result, nil
}

func (e *Executor) Stop() {
	e.mu.Lock()
	e.stopped = true
	r := e.currentRunner
	e.mu.Unlock()
	if r != nil {
		r.Stop()
	}
}

func (e *Executor) runCase(c types.IntegrityCase) (types.IntegrityCaseResult, error) {
	started := time.Now()
	if e.OnCaseStarted != nil {
		e.OnCaseStarted(c)
	}

	caseInput := e.Input
	caseInput.Turbo = false
	caseInput.Concurrency = 1
	caseInput.Count = 1
	caseInput.PromptMode = "raw"
	if c.TimeoutMS > 0 {
		caseInput.Timeout = time.Duration(c.TimeoutMS) * time.Millisecond
	} else if caseInput.Integrity.CaseTimeoutMS > 0 {
		caseInput.Timeout = time.Duration(caseInput.Integrity.CaseTimeoutMS) * time.Millisecond
	}

	caseInput, err := task.HydrateInput(caseInput)
	if err != nil {
		return failedCase(c, started, err.Error()), err
	}

	extracted := map[string]any{}
	var allRequestObs []map[string]any

	for i, req := range c.Requests {
		body := interpolateRequest(string(req.Body), caseInput.Model, extracted)
		reqInput := caseInput
		reqInput.PromptText = body
		reqInput.PromptSource = nil
		reqInput, err = task.HydrateInput(reqInput)
		if err != nil {
			return failedCase(c, started, err.Error()), err
		}

		r, err := e.RunnerFactory(reqInput, c)
		if err != nil {
			return failedCase(c, started, err.Error()), err
		}

		e.mu.Lock()
		if e.stopped {
			e.mu.Unlock()
			r.Stop()
			return failedCase(c, started, "integrity run stopped"), nil
		}
		e.currentRunner = r
		e.mu.Unlock()

		_, runErr := r.RunWithCallback(func(metrics *client.ResponseMetrics, idx int, cbErr error) {
			obs := BuildObservation(reqInput, c, metrics, idx, cbErr)
			allRequestObs = append(allRequestObs, obs)
			if e.OnRequestDone != nil {
				e.OnRequestDone(c, metrics, idx, cbErr, nil)
			}
		})

		e.mu.Lock()
		if e.currentRunner == r {
			e.currentRunner = nil
		}
		e.mu.Unlock()

		if runErr != nil {
			return failedCase(c, started, runErr.Error()), runErr
		}

		// 提取变量供后续请求使用
		if len(allRequestObs) > 0 {
			lastObs := allRequestObs[len(allRequestObs)-1]
			for varName, jsonPath := range req.Extract {
				val, found, _ := assertion.ResolvePath(lastObs, jsonPath)
				if found {
					extracted[varName] = val
				}
			}
		}

		_ = i // request index (used in observation)
	}

	// 构建合并 observation 并评估所有断言
	mergedObs := buildMergedObservation(caseInput, c, allRequestObs)
	caseAssertions, evalErr := assertion.EvaluateAll(mergedObs, c.Assertions)
	if evalErr != nil {
		caseAssertions = append(caseAssertions, types.AssertionResult{
			AssertionID: "assertion.evaluate",
			Level:       "error",
			Passed:      false,
			Message:     evalErr.Error(),
		})
	}

	finished := time.Now()
	caseResult := types.IntegrityCaseResult{
		CaseID:     c.ID,
		Name:       c.Name,
		Capability: c.Capability,
		Required:   c.Required,
		Status:     "passed",
		StartedAt:  started,
	}
	caseResult.FinishedAt = &finished
	caseResult.Duration = finished.Sub(started)
	caseResult.Assertions = caseAssertions
	caseResult.TotalAssertions = len(caseAssertions)
	for _, a := range caseAssertions {
		if a.Passed {
			caseResult.PassedAssertions++
			continue
		}
		if a.Level == "warn" {
			caseResult.WarnedAssertions++
		} else {
			caseResult.FailedAssertions++
		}
	}
	if caseResult.FailedAssertions > 0 {
		caseResult.Status = "failed"
	} else if caseResult.WarnedAssertions > 0 {
		caseResult.Status = "warned"
	}
	if e.OnCaseDone != nil {
		e.OnCaseDone(caseResult)
	}
	return caseResult, nil
}

var templateRe = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func interpolateRequest(body, model string, vars map[string]any) string {
	return templateRe.ReplaceAllStringFunc(body, func(match string) string {
		key := match[2 : len(match)-2] // strip {{ and }}
		if key == "model" {
			return model
		}
		if val, ok := vars[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match // unknown template, keep as-is
	})
}

func failedCase(c types.IntegrityCase, started time.Time, message string) types.IntegrityCaseResult {
	finished := time.Now()
	return types.IntegrityCaseResult{
		CaseID:       c.ID,
		Name:         c.Name,
		Capability:   c.Capability,
		Required:     c.Required,
		Status:       "failed",
		StartedAt:    started,
		FinishedAt:   &finished,
		Duration:     finished.Sub(started),
		ErrorMessage: message,
	}
}
