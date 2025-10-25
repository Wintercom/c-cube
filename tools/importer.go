package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type PassageRequest struct {
	Passages    []string               `json:"passages"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type QABatchImporter struct {
	apiURL          string
	token           string
	knowledgeBaseID string
	batchSize       int
	stats           ImportStats
	failedRecords   []FailedRecord
}

type ImportStats struct {
	Total   int
	Success int
	Failed  int
}

type FailedRecord struct {
	Index int    `json:"index"`
	QAID  string `json:"qa_id"`
	Title string `json:"title"`
	Error string `json:"error,omitempty"`
}

func NewQABatchImporter(apiURL, token, kbID string, batchSize int) *QABatchImporter {
	return &QABatchImporter{
		apiURL:          strings.TrimRight(apiURL, "/"),
		token:           token,
		knowledgeBaseID: kbID,
		batchSize:       batchSize,
		stats:           ImportStats{},
		failedRecords:   []FailedRecord{},
	}
}

func (imp *QABatchImporter) ImportSinglePassage(data TransformedQA) error {
	url := fmt.Sprintf("%s/api/v1/knowledge-bases/%s/knowledge/passage",
		imp.apiURL, imp.knowledgeBaseID)

	payload := PassageRequest{
		Passages:    []string{data.Passage},
		Title:       data.Title,
		Description: data.Description,
		Metadata:    data.Metadata,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", imp.token))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(body))
}

func (imp *QABatchImporter) ImportBatch(qaList []TransformedQA, startIndex int) {
	imp.stats.Total = len(qaList)

	fmt.Printf("\n开始批量导入 (从第 %d 条开始)...\n", startIndex+1)
	fmt.Printf("批次大小: %d\n", imp.batchSize)
	fmt.Printf("总数: %d 条\n\n", len(qaList))

	for i := startIndex; i < len(qaList); i++ {
		qaData := qaList[i]
		qaID := getQAID(qaData.Metadata)
		title := truncateString(qaData.Title, 50)

		fmt.Printf("[%d/%d] 导入 QA ID: %s - %s...\n", i+1, len(qaList), qaID, title)

		err := imp.ImportSinglePassage(qaData)
		if err != nil {
			imp.stats.Failed++
			fmt.Printf("  ❌ %v\n", err)
			imp.failedRecords = append(imp.failedRecords, FailedRecord{
				Index: i,
				QAID:  qaID,
				Title: title,
				Error: err.Error(),
			})
		} else {
			imp.stats.Success++
			fmt.Println("  ✅ 成功")
		}

		if (i+1)%imp.batchSize == 0 {
			fmt.Printf("\n--- 已完成 %d/%d 条，暂停 0.5 秒 ---\n\n", i+1, len(qaList))
			time.Sleep(500 * time.Millisecond)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (imp *QABatchImporter) PrintStats() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("导入统计:")
	fmt.Printf("  总计: %d 条\n", imp.stats.Total)
	fmt.Printf("  成功: %d 条\n", imp.stats.Success)
	fmt.Printf("  失败: %d 条\n", imp.stats.Failed)
	if imp.stats.Total > 0 {
		fmt.Printf("  成功率: %.2f%%\n", float64(imp.stats.Success)/float64(imp.stats.Total)*100)
	}
	fmt.Println(strings.Repeat("=", 60))
}

func (imp *QABatchImporter) SaveFailedRecords(outputFile string) {
	if len(imp.failedRecords) == 0 {
		fmt.Println("\n✅ 所有记录导入成功！")
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
		fmt.Printf("  - [索引 %d] QA ID: %s - %s\n", record.Index, record.QAID, record.Title)
	}
}

func getQAID(metadata map[string]interface{}) string {
	if qaID, ok := metadata["qa_id"].(string); ok {
		return qaID
	}
	return "unknown"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func main() {
	var (
		apiURL      string
		token       string
		kbID        string
		batchSize   int
		startIndex  int
		failedLog   string
		showHelp    bool
	)

	flag.StringVar(&apiURL, "api-url", "", "API 基础 URL (必填)")
	flag.StringVar(&token, "token", "", "认证 token (必填)")
	flag.StringVar(&kbID, "kb-id", "", "知识库 ID (必填)")
	flag.IntVar(&batchSize, "batch-size", 10, "每批次导入数量")
	flag.IntVar(&startIndex, "start-index", 0, "起始索引，用于断点续传")
	flag.StringVar(&failedLog, "failed-log", "failed_imports.json", "失败记录保存文件")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")

	flag.Parse()

	if showHelp || flag.NArg() < 1 {
		fmt.Println("C-Cube 知识库批量导入工具 (Go 版本)")
		fmt.Println("\n用法:")
		fmt.Println("  importer [选项] <转换后的JSON文件>")
		fmt.Println("\n必需参数:")
		fmt.Println("  --api-url     API 基础 URL (例如: http://localhost:8080)")
		fmt.Println("  --token       认证 token")
		fmt.Println("  --kb-id       知识库 ID")
		fmt.Println("\n可选参数:")
		fmt.Println("  --batch-size  每批次导入数量 (默认: 10)")
		fmt.Println("  --start-index 起始索引，用于断点续传 (默认: 0)")
		fmt.Println("  --failed-log  失败记录保存文件 (默认: failed_imports.json)")
		fmt.Println("\n示例:")
		fmt.Println("  importer --api-url http://localhost:8080 \\")
		fmt.Println("           --token YOUR_TOKEN \\")
		fmt.Println("           --kb-id kb-123456 \\")
		fmt.Println("           --batch-size 10 \\")
		fmt.Println("           transformed_qa_data.json")
		os.Exit(0)
	}

	if apiURL == "" || token == "" || kbID == "" {
		fmt.Println("❌ 错误: 必须提供 --api-url, --token 和 --kb-id 参数")
		fmt.Println("使用 --help 查看帮助信息")
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("C-Cube 知识库批量导入工具 (Go 版本)")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("输入文件: %s\n", inputFile)
	fmt.Printf("API URL: %s\n", apiURL)
	fmt.Printf("知识库 ID: %s\n", kbID)
	fmt.Printf("批次大小: %d\n", batchSize)
	fmt.Println(strings.Repeat("=", 60))

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("❌ 错误: 文件 '%s' 不存在或无法读取\n", inputFile)
		os.Exit(1)
	}

	var qaList []TransformedQA
	if err := json.Unmarshal(data, &qaList); err != nil {
		fmt.Printf("❌ 错误: JSON 格式无效 - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ 成功读取 %d 条记录\n", len(qaList))

	if startIndex > 0 {
		fmt.Printf("⚠️  从第 %d 条开始导入（断点续传）\n", startIndex+1)
	}

	importer := NewQABatchImporter(apiURL, token, kbID, batchSize)

	startTime := time.Now()

	importer.ImportBatch(qaList, startIndex)

	elapsedTime := time.Since(startTime)

	importer.PrintStats()
	importer.SaveFailedRecords(failedLog)

	fmt.Printf("\n总耗时: %.2f 秒\n", elapsedTime.Seconds())
	if len(qaList) > 0 {
		fmt.Printf("平均速度: %.2f 条/秒\n", float64(len(qaList))/elapsedTime.Seconds())
	}

	if importer.stats.Failed > 0 {
		fmt.Printf("\n⚠️  部分记录导入失败，请检查 %s\n", failedLog)
		fmt.Println("可使用 --start-index 参数重试失败的记录")
		os.Exit(1)
	} else {
		fmt.Println("\n✅ 全部导入成功！")
		os.Exit(0)
	}
}
