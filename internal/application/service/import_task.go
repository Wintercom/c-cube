package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type importTaskService struct {
	repo              interfaces.ImportTaskRepository
	kgService         interfaces.KnowledgeService
	crawlerService    interfaces.CrawlerService
}

func NewImportTaskService(
	repo interfaces.ImportTaskRepository,
	kgService interfaces.KnowledgeService,
	crawlerService interfaces.CrawlerService,
) (interfaces.ImportTaskService, error) {
	return &importTaskService{
		repo:           repo,
		kgService:      kgService,
		crawlerService: crawlerService,
	}, nil
}

func (s *importTaskService) CreateTask(ctx context.Context, task *types.ImportTask) error {
	return s.repo.Create(ctx, task)
}

func (s *importTaskService) GetTaskByID(ctx context.Context, taskID string) (*types.ImportTask, error) {
	tenantID, ok := ctx.Value(types.TenantIDContextKey).(uint)
	if !ok {
		return nil, fmt.Errorf("tenant ID not found in context")
	}
	return s.repo.GetByID(ctx, tenantID, taskID)
}

func (s *importTaskService) ListTasks(ctx context.Context, tenantID uint, knowledgeBaseID string, pagination *types.Pagination) (*types.PageResult, error) {
	tasks, total, err := s.repo.List(ctx, tenantID, knowledgeBaseID, pagination)
	if err != nil {
		return nil, err
	}

	return types.NewPageResult(total, pagination, tasks), nil
}

func (s *importTaskService) UpdateTaskStatus(ctx context.Context, taskID string, status types.ImportTaskStatus, errorMsg string) error {
	return s.repo.UpdateStatus(ctx, taskID, status, errorMsg)
}

func (s *importTaskService) UpdateTaskProgress(ctx context.Context, taskID string, processedURLs, successCount, failedCount, duplicateCount int, currentURL string) error {
	return s.repo.UpdateProgress(ctx, taskID, processedURLs, successCount, failedCount, duplicateCount, currentURL)
}

func (s *importTaskService) CancelTask(ctx context.Context, taskID string) error {
	tenantID, ok := ctx.Value(types.TenantIDContextKey).(uint)
	if !ok {
		return fmt.Errorf("tenant ID not found in context")
	}

	task, err := s.repo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return err
	}

	if task.Status != types.ImportTaskStatusPending && task.Status != types.ImportTaskStatusProcessing {
		return fmt.Errorf("cannot cancel task with status: %s", task.Status)
	}

	return s.repo.UpdateStatus(ctx, taskID, types.ImportTaskStatusCancelled, "Task cancelled by user")
}

func (s *importTaskService) CompleteTask(ctx context.Context, taskID string) error {
	return s.repo.UpdateStatus(ctx, taskID, types.ImportTaskStatusCompleted, "")
}

func (s *importTaskService) StartProcessing(ctx context.Context, taskID string) error {
	return s.repo.UpdateStatus(ctx, taskID, types.ImportTaskStatusProcessing, "")
}

func (s *importTaskService) AddTaskResult(ctx context.Context, taskID string, result *types.ImportTaskResult) error {
	return s.repo.AddResult(ctx, taskID, result)
}

func (s *importTaskService) ProcessTask(ctx context.Context, task *types.ImportTask) {
	logger.Infof(ctx, "Starting to process import task: %s", task.ID)

	if err := s.StartProcessing(ctx, task.ID); err != nil {
		logger.Errorf(ctx, "Failed to update task status to processing: %v", err)
		s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusFailed, err.Error())
		return
	}

	var config types.ImportTaskConfig
	if len(task.Config) > 0 {
		if err := json.Unmarshal(task.Config, &config); err != nil {
			logger.Errorf(ctx, "Failed to parse task config: %v", err)
			s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusFailed, fmt.Sprintf("Invalid config: %v", err))
			return
		}
	}

	maxPages := config.MaxPages
	if maxPages <= 0 {
		maxPages = 100
	}
	if maxPages > 500 {
		maxPages = 500
	}

	crawlResult, err := s.crawlerService.CrawlWebsite(ctx, task.BaseURL, maxPages)
	if err != nil {
		logger.Errorf(ctx, "Failed to crawl website: %v", err)
		s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusFailed, fmt.Sprintf("Crawl failed: %v", err))
		return
	}

	if len(crawlResult.URLs) == 0 {
		logger.Warn(ctx, "No URLs found from crawling")
		s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusCompleted, "")
		return
	}

	totalURLs := len(crawlResult.URLs)
	s.repo.Update(ctx, &types.ImportTask{
		ID:        task.ID,
		TotalURLs: totalURLs,
	})

	logger.Infof(ctx, "Found %d URLs, starting import", totalURLs)

	successCount := 0
	failedCount := 0
	duplicateCount := 0

	for i, urlStr := range crawlResult.URLs {
		currentTask, err := s.GetTaskByID(ctx, task.ID)
		if err != nil {
			logger.Errorf(ctx, "Failed to get task status: %v", err)
			break
		}

		if currentTask.Status == types.ImportTaskStatusCancelled {
			logger.Infof(ctx, "Task %s was cancelled, stopping import", task.ID)
			return
		}

		s.UpdateTaskProgress(ctx, task.ID, i, successCount, failedCount, duplicateCount, urlStr)

		knowledge, err := s.kgService.CreateKnowledgeFromURL(ctx, task.KnowledgeBaseID, urlStr, config.EnableMultimodel)
		
		result := &types.ImportTaskResult{
			URL: urlStr,
		}

		if err != nil {
			if _, ok := err.(*types.DuplicateKnowledgeError); ok {
				duplicateCount++
				result.Status = "duplicate"
				logger.Infof(ctx, "Duplicate URL skipped: %s", urlStr)
			} else {
				failedCount++
				result.Status = "failed"
				result.Error = err.Error()
				logger.Warnf(ctx, "Failed to import URL %s: %v", urlStr, err)
			}
		} else {
			successCount++
			result.Status = "success"
			result.KnowledgeID = knowledge.ID
			logger.Infof(ctx, "Successfully imported URL: %s", urlStr)
		}

		s.AddTaskResult(ctx, task.ID, result)

		time.Sleep(100 * time.Millisecond)
	}

	s.UpdateTaskProgress(ctx, task.ID, totalURLs, successCount, failedCount, duplicateCount, "")
	s.CompleteTask(ctx, task.ID)

	logger.Infof(ctx, "Task %s completed: total=%d, success=%d, duplicate=%d, failed=%d",
		task.ID, totalURLs, successCount, duplicateCount, failedCount)
}
