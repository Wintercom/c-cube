package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrImportTaskNotFound = errors.New("import task not found")

type importTaskRepository struct {
	db *gorm.DB
}

func NewImportTaskRepository(db *gorm.DB) interfaces.ImportTaskRepository {
	return &importTaskRepository{db: db}
}

func (r *importTaskRepository) Create(ctx context.Context, task *types.ImportTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *importTaskRepository) GetByID(ctx context.Context, tenantID uint, taskID string) (*types.ImportTask, error) {
	var task types.ImportTask
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, taskID).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImportTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *importTaskRepository) List(ctx context.Context, tenantID uint, knowledgeBaseID string, pagination *types.Pagination) ([]*types.ImportTask, int64, error) {
	var tasks []*types.ImportTask
	var total int64

	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if knowledgeBaseID != "" {
		query = query.Where("knowledge_base_id = ?", knowledgeBaseID)
	}

	if err := query.Model(&types.ImportTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

func (r *importTaskRepository) Update(ctx context.Context, task *types.ImportTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *importTaskRepository) UpdateStatus(ctx context.Context, taskID string, status types.ImportTaskStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status":      status,
		"updated_at":  time.Now(),
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	if status == types.ImportTaskStatusCompleted || status == types.ImportTaskStatusFailed || status == types.ImportTaskStatusCancelled {
		now := time.Now()
		updates["completed_at"] = &now
	}
	
	return r.db.WithContext(ctx).
		Model(&types.ImportTask{}).
		Where("id = ?", taskID).
		Updates(updates).Error
}

func (r *importTaskRepository) UpdateProgress(ctx context.Context, taskID string, processedURLs, successCount, failedCount, duplicateCount int, currentURL string) error {
	return r.db.WithContext(ctx).
		Model(&types.ImportTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"processed_urls":  processedURLs,
			"success_count":   successCount,
			"failed_count":    failedCount,
			"duplicate_count": duplicateCount,
			"current_url":     currentURL,
			"updated_at":      time.Now(),
		}).Error
}

func (r *importTaskRepository) AddResult(ctx context.Context, taskID string, result *types.ImportTaskResult) error {
	var task types.ImportTask
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return err
	}

	var results []types.ImportTaskResult
	if len(task.Results) > 0 {
		if err := json.Unmarshal(task.Results, &results); err != nil {
			return err
		}
	}

	results = append(results, *result)
	
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&types.ImportTask{}).
		Where("id = ?", taskID).
		Update("results", resultsJSON).Error
}
