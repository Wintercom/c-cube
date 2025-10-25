package interfaces

import (
	"context"
	"time"
)

type CrawlResult struct {
	URLs      []string
	Visited   int
	Failed    []string
	StartedAt time.Time
}

type CrawlerService interface {
	CrawlWebsite(ctx context.Context, baseURL string, maxPages int) (*CrawlResult, error)
}
