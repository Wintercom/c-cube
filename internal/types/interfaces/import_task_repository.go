package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ImportTaskRepository interface {
	Create(ctx context.Context, task *types.ImportTask) error
	GetByID(ctx context.Context, tenantID uint, taskID string) (*types.ImportTask, error)
	List(ctx context.Context, tenantID uint, knowledgeBaseID string, pagination *types.Pagination) ([]*types.ImportTask, int64, error)
	Update(ctx context.Context, task *types.ImportTask) error
	UpdateStatus(ctx context.Context, taskID string, status types.ImportTaskStatus, errorMsg string) error
	UpdateProgress(ctx context.Context, taskID string, processedURLs, successCount, failedCount, duplicateCount int, currentURL string) error
	AddResult(ctx context.Context, taskID string, result *types.ImportTaskResult) error
}
