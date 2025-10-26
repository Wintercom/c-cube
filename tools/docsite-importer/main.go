package main

import (
	"bytes"
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

	"github.com/PuerkitoBio/goquery"
)

type ImportResult struct {
	URL     string `json:"url"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	KnID    string `json:"knowledge_id,omitempty"`
}

type Config struct {
	APIURL      string
	Token       string
	KBBaseURL  string
	MaxPages    int
	Concurrent  int
	OutputFile  string
	URLFile     string
	ResumeFile  string
}

type Importer struct {
	config       *Config
	client       *http.Client
	visited      map[string]bool
	mu           sync.Mutex
	results      []ImportResult
	successCount int
	failCount    int
	skipCount    int
}

func NewImporter(cfg *Config) *Importer {
	return &Importer{
		config:  cfg,
		client:  &http.Client{Timeout: 60 * time.Second},
		visited: make(map[string]bool),
		results: make([]ImportResult, 0),
	}
}

func (imp *Importer) crawlURLs(baseURL string) ([]string, error) {
	fmt.Printf("🕷️  开始爬取文档站: %s\n", baseURL)
	
	urls := []string{}
	toVisit := []string{baseURL}
	
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	
	for len(toVisit) > 0 && len(urls) < imp.config.MaxPages {
		current := toVisit[0]
		toVisit = toVisit[1:]
		
		imp.mu.Lock()
		if imp.visited[current] {
			imp.mu.Unlock()
			continue
		}
		imp.visited[current] = true
		imp.mu.Unlock()
		
		fmt.Printf("   发现: %s\n", current)
		urls = append(urls, current)
		
		resp, err := imp.client.Get(current)
		if err != nil {
			fmt.Printf("   ⚠️  无法访问 %s: %v\n", current, err)
			continue
		}
		
		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}
		
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			continue
		}
		
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}
			
			linkURL, err := url.Parse(href)
			if err != nil {
				return
			}
			
			absoluteURL := parsedBase.ResolveReference(linkURL)
			
			if absoluteURL.Host != parsedBase.Host {
				return
			}
			
			absoluteURL.Fragment = ""
			urlStr := absoluteURL.String()
			
			imp.mu.Lock()
			if !imp.visited[urlStr] && len(urls)+len(toVisit) < imp.config.MaxPages {
				toVisit = append(toVisit, urlStr)
			}
			imp.mu.Unlock()
		})
		
		time.Sleep(100 * time.Millisecond)
	}
	
	fmt.Printf("✅ 爬取完成，共发现 %d 个页面\n\n", len(urls))
	return urls, nil
}

func (imp *Importer) loadURLsFromFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(data), "\n")
	urls := make([]string, 0, len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}
	
	return urls, nil
}

func (imp *Importer) loadResumeFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var results []ImportResult
	if err := json.Unmarshal(data, &results); err != nil {
		return err
	}
	
	imp.mu.Lock()
	for _, r := range results {
		if r.Success {
			imp.visited[r.URL] = true
			imp.successCount++
		}
	}
	imp.results = results
	imp.mu.Unlock()
	
	fmt.Printf("📁 从断点文件恢复，已导入 %d 个页面\n\n", imp.successCount)
	return nil
}

func (imp *Importer) saveProgress() error {
	data, err := json.MarshalIndent(imp.results, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(imp.config.ResumeFile, data, 0644)
}

func (imp *Importer) importURL(urlStr string) ImportResult {
	imp.mu.Lock()
	if imp.visited[urlStr] {
		imp.skipCount++
		imp.mu.Unlock()
		return ImportResult{URL: urlStr, Success: true, Message: "已导入(跳过)"}
	}
	imp.mu.Unlock()
	
	reqBody := map[string]interface{}{
		"url": urlStr,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return ImportResult{URL: urlStr, Success: false, Message: fmt.Sprintf("JSON编码失败: %v", err)}
	}
	
	apiURL := fmt.Sprintf("%s/api/v1/knowledge-bases/%s/knowledge/url", imp.config.APIURL, imp.config.KBBaseURL)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return ImportResult{URL: urlStr, Success: false, Message: fmt.Sprintf("创建请求失败: %v", err)}
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", imp.config.Token)
	
	resp, err := imp.client.Do(req)
	if err != nil {
		return ImportResult{URL: urlStr, Success: false, Message: fmt.Sprintf("请求失败: %v", err)}
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var apiResp struct {
			Success bool `json:"success"`
			Data    struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Success {
			imp.mu.Lock()
			imp.visited[urlStr] = true
			imp.successCount++
			imp.mu.Unlock()
			
			return ImportResult{URL: urlStr, Success: true, KnID: apiResp.Data.ID}
		}
	}
	
	if resp.StatusCode == http.StatusConflict {
		var apiResp struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Code == "duplicate_url" {
			imp.mu.Lock()
			imp.visited[urlStr] = true
			imp.skipCount++
			imp.mu.Unlock()
			
			return ImportResult{URL: urlStr, Success: true, Message: "URL已存在"}
		}
	}
	
	imp.mu.Lock()
	imp.failCount++
	imp.mu.Unlock()
	
	return ImportResult{URL: urlStr, Success: false, Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))}
}

func (imp *Importer) importURLsConcurrently(urls []string) {
	fmt.Printf("📥 开始导入，共 %d 个页面，并发数: %d\n\n", len(urls), imp.config.Concurrent)
	
	sem := make(chan struct{}, imp.config.Concurrent)
	var wg sync.WaitGroup
	
	for i, urlStr := range urls {
		wg.Add(1)
		sem <- struct{}{}
		
		go func(index int, url string) {
			defer wg.Done()
			defer func() { <-sem }()
			
			result := imp.importURL(url)
			
			imp.mu.Lock()
			imp.results = append(imp.results, result)
			imp.mu.Unlock()
			
			status := "✅"
			if !result.Success {
				status = "❌"
			} else if result.Message != "" {
				status = "⏭️ "
			}
			
			msg := ""
			if result.Message != "" {
				msg = fmt.Sprintf(" - %s", result.Message)
			}
			
			fmt.Printf("[%d/%d] %s %s%s\n", index+1, len(urls), status, url, msg)
			
			if (index+1)%10 == 0 {
				if err := imp.saveProgress(); err != nil {
					fmt.Printf("⚠️  保存进度失败: %v\n", err)
				}
			}
		}(i, urlStr)
	}
	
	wg.Wait()
	
	imp.saveProgress()
}

func (imp *Importer) printSummary() {
	fmt.Printf("\n")
	fmt.Printf("========================================\n")
	fmt.Printf("📊 导入统计\n")
	fmt.Printf("========================================\n")
	fmt.Printf("总计: %d\n", len(imp.results))
	fmt.Printf("✅ 成功: %d\n", imp.successCount)
	fmt.Printf("⏭️  跳过: %d\n", imp.skipCount)
	fmt.Printf("❌ 失败: %d\n", imp.failCount)
	fmt.Printf("========================================\n")
	
	if imp.failCount > 0 {
		fmt.Printf("\n失败的URL:\n")
		for _, r := range imp.results {
			if !r.Success {
				fmt.Printf("  - %s: %s\n", r.URL, r.Message)
			}
		}
	}
	
	if imp.config.OutputFile != "" {
		data, _ := json.MarshalIndent(imp.results, "", "  ")
		os.WriteFile(imp.config.OutputFile, data, 0644)
		fmt.Printf("\n📄 详细结果已保存至: %s\n", imp.config.OutputFile)
	}
}

func main() {
	cfg := &Config{}
	
	flag.StringVar(&cfg.APIURL, "api-url", "http://localhost:8080", "WeKnora API 地址")
	flag.StringVar(&cfg.Token, "token", "", "API Token (x-api-key)")
	flag.StringVar(&cfg.KBBaseURL, "kb-id", "", "知识库 ID")
	flag.StringVar(&cfg.URLFile, "url-file", "", "URL 列表文件路径")
	flag.StringVar(&cfg.ResumeFile, "resume-file", "import_progress.json", "断点续传文件")
	flag.StringVar(&cfg.OutputFile, "output", "import_results.json", "结果输出文件")
	flag.IntVar(&cfg.MaxPages, "max-pages", 200, "最大爬取页面数")
	flag.IntVar(&cfg.Concurrent, "concurrent", 3, "并发导入数")
	
	baseURL := flag.String("base-url", "", "文档站基础 URL (自动爬取)")
	
	flag.Parse()
	
	if cfg.Token == "" {
		fmt.Println("❌ 错误: 必须提供 -token")
		flag.Usage()
		os.Exit(1)
	}
	
	if cfg.KBBaseURL == "" {
		fmt.Println("❌ 错误: 必须提供 -kb-id")
		flag.Usage()
		os.Exit(1)
	}
	
	if *baseURL == "" && cfg.URLFile == "" {
		fmt.Println("❌ 错误: 必须提供 -base-url 或 -url-file")
		flag.Usage()
		os.Exit(1)
	}
	
	imp := NewImporter(cfg)
	
	if cfg.ResumeFile != "" {
		if err := imp.loadResumeFile(cfg.ResumeFile); err != nil {
			fmt.Printf("⚠️  无法加载断点文件: %v\n", err)
		}
	}
	
	var urls []string
	var err error
	
	if cfg.URLFile != "" {
		urls, err = imp.loadURLsFromFile(cfg.URLFile)
		if err != nil {
			fmt.Printf("❌ 读取 URL 文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📁 从文件加载了 %d 个 URL\n\n", len(urls))
	} else {
		urls, err = imp.crawlURLs(*baseURL)
		if err != nil {
			fmt.Printf("❌ 爬取失败: %v\n", err)
			os.Exit(1)
		}
	}
	
	if len(urls) == 0 {
		fmt.Println("⚠️  没有找到需要导入的 URL")
		os.Exit(0)
	}
	
	imp.importURLsConcurrently(urls)
	imp.printSummary()
}
