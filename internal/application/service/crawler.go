package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gocolly/colly/v2"
)

type crawlerService struct {
	maxDepth int
	maxPages int
	visited  sync.Map // 使用 sync.Map 替代 map+RWMutex，避免并发竞态
}

func NewCrawlerService() (interfaces.CrawlerService, error) {
	return &crawlerService{
		maxDepth: 5,
		maxPages: 500,
	}, nil
}

func (s *crawlerService) CrawlWebsite(ctx context.Context, baseURL string, maxPages int) (*interfaces.CrawlResult, error) {
	logger.Infof(ctx, "Starting website crawl: %s, maxPages: %d", baseURL, maxPages)

	if maxPages <= 0 {
		maxPages = s.maxPages
	}

	// 重置 visited map（删除所有旧条目）
	s.visited.Range(func(key, value interface{}) bool {
		s.visited.Delete(key)
		return true
	})

	result := &interfaces.CrawlResult{
		URLs:      make([]string, 0),
		Failed:    make([]string, 0),
		StartedAt: time.Now(),
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	c := colly.NewCollector(
		colly.AllowedDomains(parsedBase.Host),
		colly.MaxDepth(s.maxDepth),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       300 * time.Millisecond,
	})

	urlsMutex := sync.Mutex{}

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(link)

		if absoluteURL == "" {
			return
		}

		parsedURL, err := url.Parse(absoluteURL)
		if err != nil {
			return
		}

		if parsedURL.Host != parsedBase.Host {
			return
		}

		parsedURL.Fragment = ""
		cleanURL := parsedURL.String()

		if shouldSkipURL(cleanURL) {
			return
		}

		// 使用 sync.Map 的 LoadOrStore 实现原子操作
		// 只有第一个成功存储的 goroutine 会返回 false，其他都返回 true
		_, alreadyVisited := s.visited.LoadOrStore(cleanURL, true)
		if alreadyVisited {
			// URL 已被其他 goroutine 访问，直接返回
			return
		}

		// 检查是否已达到最大页面数
		urlsMutex.Lock()
		urlCount := len(result.URLs)
		urlsMutex.Unlock()
		
		if urlCount >= maxPages {
			return
		}

		// 原子操作确保只有一个 goroutine 会访问此 URL
		e.Request.Visit(cleanURL)
	})

	c.OnRequest(func(r *colly.Request) {
		logger.Debugf(ctx, "Visiting: %s", r.URL.String())
	})

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			contentType := r.Headers.Get("Content-Type")
			if strings.Contains(contentType, "text/html") {
				urlsMutex.Lock()
				if len(result.URLs) < maxPages {
					result.URLs = append(result.URLs, r.Request.URL.String())
					logger.Infof(ctx, "Added URL [%d/%d]: %s", len(result.URLs), maxPages, r.Request.URL.String())
				}
				urlsMutex.Unlock()
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		if strings.Contains(err.Error(), "already visited") {
			return
		}
		logger.Warnf(ctx, "Failed to crawl %s: %v", r.Request.URL.String(), err)
		urlsMutex.Lock()
		result.Failed = append(result.Failed, r.Request.URL.String())
		urlsMutex.Unlock()
	})

	if err := c.Visit(baseURL); err != nil {
		return nil, fmt.Errorf("failed to start crawling: %w", err)
	}

	c.Wait()

	result.Visited = len(result.URLs)
	logger.Infof(ctx, "Crawl completed: %d URLs found, %d failed", result.Visited, len(result.Failed))

	return result, nil
}

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
