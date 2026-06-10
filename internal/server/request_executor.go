package server

import (
	"context"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/types"
)

// RequestJob 描述一次可由统一请求队列执行的模型请求。
type RequestJob struct {
	RunID  RunID
	Index  int
	Input  types.Input
	Level  int
	CaseID string
}

// RequestResult 是 RequestJob 的执行结果。
type RequestResult struct {
	Job     RequestJob
	Metrics *client.ResponseMetrics
	Err     error
}

// RequestExecutor 执行单个 RequestJob。
type RequestExecutor struct {
	client client.ModelClient
}

func NewRequestExecutor(c client.ModelClient) *RequestExecutor {
	return &RequestExecutor{client: c}
}

func (e *RequestExecutor) Execute(ctx context.Context, job RequestJob) RequestResult {
	result := RequestResult{Job: job}
	if e.client == nil {
		result.Err = context.Canceled
		return result
	}
	if job.Input.PromptMode == "raw" {
		rawBody := job.Input.PromptSource.GetContentByIndex(job.Index)
		result.Metrics, result.Err = e.client.RawRequest(ctx, rawBody)
		return result
	}
	systemPrompt := job.Input.PromptSource.GetSystemContent()
	userPrompt := job.Input.PromptSource.GetContentByIndex(job.Index)
	result.Metrics, result.Err = e.client.Request(ctx, systemPrompt, userPrompt, job.Input.Stream)
	return result
}
