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
	maxDepth     int
	maxPages     int
	visitedMutex sync.RWMutex
	visited      map[string]bool
}

func NewCrawlerService() (interfaces.CrawlerService, error) {
	return &crawlerService{
		maxDepth: 5,
		maxPages: 500,
		visited:  make(map[string]bool),
	}, nil
}

func (s *crawlerService) CrawlWebsite(ctx context.Context, baseURL string, maxPages int) (*interfaces.CrawlResult, error) {
	logger.Infof(ctx, "Starting website crawl: %s, maxPages: %d", baseURL, maxPages)

	if maxPages <= 0 {
		maxPages = s.maxPages
	}

	s.visitedMutex.Lock()
	s.visited = make(map[string]bool)
	s.visitedMutex.Unlock()

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

		s.visitedMutex.Lock()
		alreadyVisited := s.visited[cleanURL]
		urlCount := len(result.URLs)
		if !alreadyVisited && urlCount < maxPages {
			s.visited[cleanURL] = true
		}
		s.visitedMutex.Unlock()

		if alreadyVisited || urlCount >= maxPages {
			return
		}

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
