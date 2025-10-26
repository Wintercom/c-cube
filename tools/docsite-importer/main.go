package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

type URLRequest struct {
	URL              string `json:"url"`
	EnableMultimodel *bool  `json:"enable_multimodel,omitempty"`
}

type ImportStats struct {
	Total     int
	Success   int
	Failed    int
	Duplicate int
	Skipped   int
}

type FailedRecord struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

type DocsiteImporter struct {
	apiURL            string
	token             string
	knowledgeBaseID   string
	concurrent        int
	enableMultimodel  *bool
	stats             ImportStats
	failedRecords     []FailedRecord
	progressFile      string
	importedURLs      map[string]bool
	importedURLsMutex sync.Mutex
}

func NewDocsiteImporter(apiURL, token, kbID string, concurrent int, enableMultimodel *bool, progressFile string) *DocsiteImporter {
	imp := &DocsiteImporter{
		apiURL:           strings.TrimRight(apiURL, "/"),
		token:            token,
		knowledgeBaseID:  kbID,
		concurrent:       concurrent,
		enableMultimodel: enableMultimodel,
		stats:            ImportStats{},
		failedRecords:    []FailedRecord{},
		progressFile:     progressFile,
		importedURLs:     make(map[string]bool),
	}

	imp.loadProgress()
	return imp
}

func (imp *DocsiteImporter) loadProgress() {
	if imp.progressFile == "" {
		return
	}

	data, err := os.ReadFile(imp.progressFile)
	if err != nil {
		return
	}

	var urls []string
	if err := json.Unmarshal(data, &urls); err != nil {
		return
	}

	for _, u := range urls {
		imp.importedURLs[u] = true
	}

	fmt.Printf("✅ 从进度文件加载了 %d 个已导入的 URL\n", len(imp.importedURLs))
}

func (imp *DocsiteImporter) saveProgress() {
	if imp.progressFile == "" {
		return
	}

	imp.importedURLsMutex.Lock()
	urls := make([]string, 0, len(imp.importedURLs))
	for u := range imp.importedURLs {
		urls = append(urls, u)
	}
	imp.importedURLsMutex.Unlock()

	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		fmt.Printf("⚠️  保存进度文件失败: %v\n", err)
		return
	}

	if err := os.WriteFile(imp.progressFile, data, 0644); err != nil {
		fmt.Printf("⚠️  保存进度文件失败: %v\n", err)
	}
}

func (imp *DocsiteImporter) isImported(url string) bool {
	imp.importedURLsMutex.Lock()
	defer imp.importedURLsMutex.Unlock()
	return imp.importedURLs[url]
}

func (imp *DocsiteImporter) markImported(url string) {
	imp.importedURLsMutex.Lock()
	imp.importedURLs[url] = true
	imp.importedURLsMutex.Unlock()
}

func (imp *DocsiteImporter) ImportSingleURL(urlStr string) error {
	apiURL := fmt.Sprintf("%s/api/v1/knowledge-bases/%s/knowledge/url",
		imp.apiURL, imp.knowledgeBaseID)

	payload := URLRequest{
		URL:              urlStr,
		EnableMultimodel: imp.enableMultimodel,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-API-Key", imp.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	if resp.StatusCode == http.StatusConflict {
		return &DuplicateError{Message: string(body)}
	}

	return fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(body))
}

type DuplicateError struct {
	Message string
}

func (e *DuplicateError) Error() string {
	return e.Message
}

func (imp *DocsiteImporter) ImportURLs(urls []string) {
	imp.stats.Total = len(urls)

	sem := make(chan struct{}, imp.concurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Printf("\n开始批量导入...\n")
	fmt.Printf("并发数: %d\n", imp.concurrent)
	fmt.Printf("总 URL 数: %d\n\n", len(urls))

	startTime := time.Now()

	for i, urlStr := range urls {
		if imp.isImported(urlStr) {
			mu.Lock()
			imp.stats.Skipped++
			mu.Unlock()
			fmt.Printf("[%d/%d] 跳过 (已导入): %s\n", i+1, len(urls), urlStr)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, url string) {
			defer wg.Done()
			defer func() { <-sem }()

			shortURL := truncateString(url, 80)
			fmt.Printf("[%d/%d] 导入: %s...\n", idx+1, len(urls), shortURL)

			err := imp.ImportSingleURL(url)

			mu.Lock()
			if err != nil {
				if _, ok := err.(*DuplicateError); ok {
					imp.stats.Duplicate++
					fmt.Printf("  ⚠️  重复 URL\n")
				} else {
					imp.stats.Failed++
					fmt.Printf("  ❌ %v\n", err)
					imp.failedRecords = append(imp.failedRecords, FailedRecord{
						URL:   url,
						Error: err.Error(),
					})
				}
			} else {
				imp.stats.Success++
				fmt.Printf("  ✅ 成功\n")
				imp.markImported(url)
			}
			mu.Unlock()

			if (idx+1)%10 == 0 {
				imp.saveProgress()
			}
		}(i, urlStr)

		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()
	imp.saveProgress()

	elapsed := time.Since(startTime)
	imp.PrintStats(elapsed)
}

func (imp *DocsiteImporter) PrintStats(elapsed time.Duration) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("导入统计:")
	fmt.Printf("  总计: %d 个 URL\n", imp.stats.Total)
	fmt.Printf("  成功: %d 个\n", imp.stats.Success)
	fmt.Printf("  失败: %d 个\n", imp.stats.Failed)
	fmt.Printf("  重复: %d 个\n", imp.stats.Duplicate)
	fmt.Printf("  跳过: %d 个 (断点续传)\n", imp.stats.Skipped)
	if imp.stats.Total > 0 {
		successRate := float64(imp.stats.Success) / float64(imp.stats.Total-imp.stats.Skipped) * 100
		fmt.Printf("  成功率: %.2f%%\n", successRate)
	}
	fmt.Printf("\n  总耗时: %.2f 秒\n", elapsed.Seconds())
	if imp.stats.Total > 0 {
		fmt.Printf("  平均速度: %.2f 个/秒\n", float64(imp.stats.Total-imp.stats.Skipped)/elapsed.Seconds())
	}
	fmt.Println(strings.Repeat("=", 70))
}

func (imp *DocsiteImporter) SaveFailedRecords(outputFile string) {
	if len(imp.failedRecords) == 0 {
		return
	}

	fmt.Printf("\n保存失败记录到: %s\n", outputFile)

	data, err := json.MarshalIndent(imp.failedRecords, "", "  ")
	if err != nil {
		fmt.Printf("❌ 序列化失败记录失败: %v\n", err)
		return
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		fmt.Printf("❌ 保存失败记录失败: %v\n", err)
		return
	}

	fmt.Printf("已保存 %d 条失败记录\n", len(imp.failedRecords))
	fmt.Println("\n失败记录详情:")
	for i, record := range imp.failedRecords {
		if i >= 10 {
			fmt.Printf("  ... 还有 %d 条\n", len(imp.failedRecords)-10)
			break
		}
		fmt.Printf("  - %s\n", record.URL)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func CrawlDocsite(ctx context.Context, baseURL string, maxPages int) ([]string, error) {
	fmt.Printf("\n开始爬取文档站: %s\n", baseURL)
	fmt.Printf("最大页面数: %d\n\n", maxPages)

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("无效的 URL: %w", err)
	}

	c := colly.NewCollector(
		colly.AllowedDomains(parsedBase.Host),
		colly.MaxDepth(5),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       300 * time.Millisecond,
	})

	urls := make([]string, 0)
	visited := make(map[string]bool)
	var mu sync.Mutex

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

		mu.Lock()
		alreadyVisited := visited[cleanURL]
		urlCount := len(urls)
		if !alreadyVisited && urlCount < maxPages {
			visited[cleanURL] = true
			urls = append(urls, cleanURL)
			fmt.Printf("  发现: [%d/%d] %s\n", len(urls), maxPages, cleanURL)
		}
		mu.Unlock()

		if alreadyVisited || urlCount >= maxPages {
			return
		}

		if err := e.Request.Visit(cleanURL); err != nil {
			if err.Error() != "URL already visited" {
			}
		}
	})

	c.OnRequest(func(r *colly.Request) {
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("  ⚠️  爬取失败: %s - %v\n", r.Request.URL, err)
	})

	if err := c.Visit(baseURL); err != nil {
		return nil, fmt.Errorf("无法访问起始 URL: %w", err)
	}

	c.Wait()

	fmt.Printf("\n✅ 爬取完成，共发现 %d 个 URL\n", len(urls))
	return urls, nil
}

func shouldSkipURL(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)

	skipExtensions := []string{
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico",
		".pdf", ".zip", ".tar", ".gz", ".rar",
		".mp4", ".avi", ".mov", ".mp3", ".wav",
		".css", ".js", ".woff", ".woff2", ".ttf", ".eot",
	}

	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}

	skipPatterns := []string{
		"/api/", "/assets/", "/static/",
		"mailto:", "tel:", "javascript:",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}

func ReadURLsFromFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("无法读取文件: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	urls := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	return urls, nil
}

func main() {
	var (
		apiURL           string
		token            string
		kbID             string
		baseURL          string
		urlFile          string
		maxPages         int
		concurrent       int
		enableMultimodel bool
		progressFile     string
		failedLog        string
		showHelp         bool
	)

	flag.StringVar(&apiURL, "api-url", "", "API 基础 URL (必填)")
	flag.StringVar(&token, "token", "", "认证 token (必填)")
	flag.StringVar(&kbID, "kb-id", "", "知识库 ID (必填)")
	flag.StringVar(&baseURL, "base-url", "", "文档站基础 URL (与 --url-file 二选一)")
	flag.StringVar(&urlFile, "url-file", "", "URL 列表文件 (与 --base-url 二选一)")
	flag.IntVar(&maxPages, "max-pages", 200, "最大爬取页面数")
	flag.IntVar(&concurrent, "concurrent", 3, "并发导入数")
	flag.BoolVar(&enableMultimodel, "enable-multimodel", false, "启用多模态处理")
	flag.StringVar(&progressFile, "progress-file", ".docsite-importer-progress.json", "断点续传进度文件")
	flag.StringVar(&failedLog, "failed-log", "failed_imports.json", "失败记录保存文件")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")

	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if apiURL == "" || token == "" || kbID == "" {
		fmt.Println("❌ 错误: 必须提供 --api-url, --token 和 --kb-id 参数")
		fmt.Println("使用 --help 查看帮助信息")
		os.Exit(1)
	}

	if baseURL == "" && urlFile == "" {
		fmt.Println("❌ 错误: 必须提供 --base-url 或 --url-file 参数")
		fmt.Println("使用 --help 查看帮助信息")
		os.Exit(1)
	}

	if baseURL != "" && urlFile != "" {
		fmt.Println("❌ 错误: --base-url 和 --url-file 不能同时使用")
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("WeKnora 文档站批量导入工具")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("API URL: %s\n", apiURL)
	fmt.Printf("知识库 ID: %s\n", kbID)
	fmt.Printf("并发数: %d\n", concurrent)
	fmt.Printf("多模态处理: %v\n", enableMultimodel)
	fmt.Printf("进度文件: %s\n", progressFile)
	fmt.Println(strings.Repeat("=", 70))

	var enableMultimodelPtr *bool
	if enableMultimodel {
		enableMultimodelPtr = &enableMultimodel
	}

	importer := NewDocsiteImporter(apiURL, token, kbID, concurrent, enableMultimodelPtr, progressFile)

	var urls []string
	var err error

	if baseURL != "" {
		ctx := context.Background()
		urls, err = CrawlDocsite(ctx, baseURL, maxPages)
		if err != nil {
			fmt.Printf("❌ 爬取失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		urls, err = ReadURLsFromFile(urlFile)
		if err != nil {
			fmt.Printf("❌ 读取文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✅ 从文件读取了 %d 个 URL\n", len(urls))
	}

	if len(urls) == 0 {
		fmt.Println("❌ 没有发现任何 URL")
		os.Exit(1)
	}

	importer.ImportURLs(urls)
	importer.SaveFailedRecords(failedLog)

	if importer.stats.Failed > 0 {
		fmt.Printf("\n⚠️  部分 URL 导入失败，详见: %s\n", failedLog)
		os.Exit(1)
	} else {
		fmt.Println("\n✅ 全部导入成功！")
		os.Exit(0)
	}
}

func printHelp() {
	fmt.Println("WeKnora 文档站批量导入工具")
	fmt.Println("\n用法:")
	fmt.Println("  docsite-importer [选项]")
	fmt.Println("\n必需参数:")
	fmt.Println("  --api-url         API 基础 URL (例如: http://localhost:8080)")
	fmt.Println("  --token           认证 token (X-API-Key)")
	fmt.Println("  --kb-id           知识库 ID")
	fmt.Println("\n导入方式 (二选一):")
	fmt.Println("  --base-url        文档站基础 URL (自动爬取)")
	fmt.Println("  --url-file        URL 列表文件 (每行一个 URL)")
	fmt.Println("\n可选参数:")
	fmt.Println("  --max-pages       最大爬取页面数 (默认: 200)")
	fmt.Println("  --concurrent      并发导入数 (默认: 3)")
	fmt.Println("  --enable-multimodel  启用多模态处理 (默认: false)")
	fmt.Println("  --progress-file   断点续传进度文件 (默认: .docsite-importer-progress.json)")
	fmt.Println("  --failed-log      失败记录保存文件 (默认: failed_imports.json)")
	fmt.Println("\n示例:")
	fmt.Println("\n  1. 自动爬取并导入:")
	fmt.Println("     docsite-importer \\")
	fmt.Println("       --api-url http://localhost:8080 \\")
	fmt.Println("       --token YOUR_TOKEN \\")
	fmt.Println("       --kb-id kb-xxxxx \\")
	fmt.Println("       --base-url https://docs.example.com \\")
	fmt.Println("       --max-pages 200 \\")
	fmt.Println("       --concurrent 3")
	fmt.Println("\n  2. 从文件导入:")
	fmt.Println("     docsite-importer \\")
	fmt.Println("       --api-url http://localhost:8080 \\")
	fmt.Println("       --token YOUR_TOKEN \\")
	fmt.Println("       --kb-id kb-xxxxx \\")
	fmt.Println("       --url-file urls.txt \\")
	fmt.Println("       --concurrent 5")
	fmt.Println("\n  3. 启用多模态处理:")
	fmt.Println("     docsite-importer \\")
	fmt.Println("       --api-url http://localhost:8080 \\")
	fmt.Println("       --token YOUR_TOKEN \\")
	fmt.Println("       --kb-id kb-xxxxx \\")
	fmt.Println("       --base-url https://docs.example.com \\")
	fmt.Println("       --enable-multimodel")
	fmt.Println("\n特性:")
	fmt.Println("  - 自动爬取或从文件读取 URL")
	fmt.Println("  - 并发导入，可配置并发数")
	fmt.Println("  - 断点续传，失败后可继续导入")
	fmt.Println("  - 详细的进度显示和统计")
	fmt.Println("  - 自动去重和跳过已导入")
}
