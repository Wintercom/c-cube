package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ImportTaskService interface {
	CreateTask(ctx context.Context, task *types.ImportTask) error
	GetTaskByID(ctx context.Context, taskID string) (*types.ImportTask, error)
	ListTasks(ctx context.Context, tenantID uint, knowledgeBaseID string, pagination *types.Pagination) (*types.PageResult, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status types.ImportTaskStatus, errorMsg string) error
	UpdateTaskProgress(ctx context.Context, taskID string, processedURLs, successCount, failedCount, duplicateCount int, currentURL string) error
	CancelTask(ctx context.Context, taskID string) error
	CompleteTask(ctx context.Context, taskID string) error
	StartProcessing(ctx context.Context, taskID string) error
	AddTaskResult(ctx context.Context, taskID string, result *types.ImportTaskResult) error
	ProcessTask(ctx context.Context, task *types.ImportTask)
}
