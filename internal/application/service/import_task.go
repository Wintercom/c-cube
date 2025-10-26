package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gocolly/colly/v2"
)

type importTaskService struct {
	repo           interfaces.ImportTaskRepository
	kgService      interfaces.KnowledgeService
	crawlerService interfaces.CrawlerService
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

// shouldSkipURL checks if a URL should be skipped based on file extensions and patterns
func shouldSkipURL(urlStr string) bool {
	skipPatterns := []string{
		".pdf", ".zip", ".tar", ".gz", ".jpg", ".jpeg", ".png", ".gif",
		".mp4", ".mp3", ".avi", ".mov", ".css", ".js", ".woff", ".ttf",
		"/api/", "/download/", "/file/", "/asset/", "/static/",
	}

	lowerURL := strings.ToLower(urlStr)
	for _, pattern := range skipPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}

// ProcessTask implements serial crawling and importing
// Each URL is crawled and immediately imported, single failures don't block subsequent tasks
func (s *importTaskService) ProcessTask(ctx context.Context, task *types.ImportTask) {
	logger.Infof(ctx, "Starting to process import task (serial mode): %s", task.ID)

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

	logger.Infof(ctx, "Starting serial crawl and import for: %s (maxPages: %d)", task.BaseURL, maxPages)

	parsedBase, err := url.Parse(task.BaseURL)
	if err != nil {
		logger.Errorf(ctx, "Invalid base URL: %v", err)
		s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusFailed, fmt.Sprintf("Invalid base URL: %v", err))
		return
	}

	// Track visited URLs to avoid duplicates
	visited := &sync.Map{}
	visitedCount := 0
	successCount := 0
	failedCount := 0
	duplicateCount := 0
	totalURLs := 0

	// Mutex for counters
	counterMutex := &sync.Mutex{}

	// Create collector for serial processing
	c := colly.NewCollector(
		colly.AllowedDomains(parsedBase.Host),
		colly.MaxDepth(5),
		colly.Async(false), // Synchronous mode for serial processing
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1, // Serial processing: one at a time
		Delay:       300 * time.Millisecond,
	})

	// OnHTML: Find links and queue them
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Check if task was cancelled
		currentTask, err := s.GetTaskByID(ctx, task.ID)
		if err == nil && currentTask.Status == types.ImportTaskStatusCancelled {
			logger.Infof(ctx, "Task %s was cancelled, stopping crawl", task.ID)
			return
		}

		// Check if we've reached the max pages limit
		counterMutex.Lock()
		currentVisited := visitedCount
		counterMutex.Unlock()

		if currentVisited >= maxPages {
			return
		}

		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		if absoluteURL == "" {
			return
		}

		parsedURL, err := url.Parse(absoluteURL)
		if err != nil {
			return
		}

		// Only process URLs from the same domain
		if parsedURL.Host != parsedBase.Host {
			return
		}

		// Remove fragment
		parsedURL.Fragment = ""
		cleanURL := parsedURL.String()

		// Skip non-HTML resources
		if shouldSkipURL(cleanURL) {
			return
		}

		// Check if URL was already visited (atomic operation)
		_, alreadyVisited := visited.LoadOrStore(cleanURL, true)
		if alreadyVisited {
			return
		}

		// Visit the URL (will trigger OnResponse)
		e.Request.Visit(cleanURL)
	})

	// OnResponse: Import each successful page immediately
	c.OnResponse(func(r *colly.Response) {
		// Check if task was cancelled
		currentTask, err := s.GetTaskByID(ctx, task.ID)
		if err == nil && currentTask.Status == types.ImportTaskStatusCancelled {
			logger.Infof(ctx, "Task %s was cancelled, stopping import", task.ID)
			return
		}

		// Check if we've reached the max pages limit
		counterMutex.Lock()
		if visitedCount >= maxPages {
			counterMutex.Unlock()
			return
		}
		counterMutex.Unlock()

		// Only process HTML pages
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			contentType := r.Headers.Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				return
			}

			urlStr := r.Request.URL.String()

			counterMutex.Lock()
			visitedCount++
			currentVisited := visitedCount
			totalURLs = visitedCount
			counterMutex.Unlock()

			logger.Infof(ctx, "[%d/%d] Processing URL: %s", currentVisited, maxPages, urlStr)

			// Update progress
			s.UpdateTaskProgress(ctx, task.ID, currentVisited, successCount, failedCount, duplicateCount, urlStr)

			// Immediately import this URL
			knowledge, err := s.kgService.CreateKnowledgeFromURL(ctx, task.KnowledgeBaseID, urlStr, config.EnableMultimodel)

			result := &types.ImportTaskResult{
				URL: urlStr,
			}

			if err != nil {
				if _, ok := err.(*types.DuplicateKnowledgeError); ok {
					counterMutex.Lock()
					duplicateCount++
					counterMutex.Unlock()
					result.Status = "duplicate"
					logger.Infof(ctx, "Duplicate URL skipped: %s", urlStr)
				} else {
					counterMutex.Lock()
					failedCount++
					counterMutex.Unlock()
					result.Status = "failed"
					result.Error = err.Error()
					logger.Warnf(ctx, "Failed to import URL %s: %v", urlStr, err)
				}
			} else {
				counterMutex.Lock()
				successCount++
				counterMutex.Unlock()
				result.Status = "success"
				result.KnowledgeID = knowledge.ID
				logger.Infof(ctx, "Successfully imported URL: %s", urlStr)
			}

			// Record result
			s.AddTaskResult(ctx, task.ID, result)

			// Small delay to avoid overwhelming the system
			time.Sleep(100 * time.Millisecond)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		logger.Debugf(ctx, "Visiting: %s", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		if strings.Contains(err.Error(), "already visited") {
			return
		}
		logger.Warnf(ctx, "Failed to crawl %s: %v", r.Request.URL.String(), err)
	})

	// Update total URLs to unknown initially
	s.repo.Update(ctx, &types.ImportTask{
		ID:        task.ID,
		TotalURLs: maxPages,
	})

	// Start crawling from base URL
	if err := c.Visit(task.BaseURL); err != nil {
		logger.Errorf(ctx, "Failed to start crawling: %v", err)
		s.UpdateTaskStatus(ctx, task.ID, types.ImportTaskStatusFailed, fmt.Sprintf("Crawl failed: %v", err))
		return
	}

	// Wait for all requests to complete
	c.Wait()

	// Final progress update
	s.UpdateTaskProgress(ctx, task.ID, totalURLs, successCount, failedCount, duplicateCount, "")

	// Update total URLs to actual count
	s.repo.Update(ctx, &types.ImportTask{
		ID:        task.ID,
		TotalURLs: totalURLs,
	})

	// Complete the task
	s.CompleteTask(ctx, task.ID)

	logger.Infof(ctx, "Task %s completed: total=%d, success=%d, duplicate=%d, failed=%d",
		task.ID, totalURLs, successCount, duplicateCount, failedCount)
}
