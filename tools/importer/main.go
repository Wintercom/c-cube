package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Wintercom/c-cube/tools/common"
	_ "github.com/lib/pq"
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
	db              *sql.DB
	skipExisting    bool
	maxImportNum    int
}

type ImportStats struct {
	Total    int
	Success  int
	Failed   int
	Skipped  int
}

type FailedRecord struct {
	Index int    `json:"index"`
	QAID  string `json:"qa_id"`
	Title string `json:"title"`
	Error string `json:"error,omitempty"`
}

func NewQABatchImporter(apiURL, token, kbID string, batchSize int, db *sql.DB, skipExisting bool, maxImportNum int) *QABatchImporter {
	return &QABatchImporter{
		apiURL:          strings.TrimRight(apiURL, "/"),
		token:           token,
		knowledgeBaseID: kbID,
		batchSize:       batchSize,
		stats:           ImportStats{},
		failedRecords:   []FailedRecord{},
		db:              db,
		skipExisting:    skipExisting,
		maxImportNum:    maxImportNum,
	}
}

func (imp *QABatchImporter) ImportSinglePassage(data common.TransformedQA) error {
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

	req.Header.Set("X-API-Key", imp.token)
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

func (imp *QABatchImporter) KnowledgeExists(qaID string) (bool, error) {
	if imp.db == nil {
		return false, nil
	}

	var count int
	query := `
		SELECT COUNT(*) 
		FROM knowledges 
		WHERE knowledge_base_id = $1 
		  AND metadata->>'qa_id' = $2 
		  AND deleted_at IS NULL
	`
	err := imp.db.QueryRow(query, imp.knowledgeBaseID, qaID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询数据库失败: %w", err)
	}

	return count > 0, nil
}

func (imp *QABatchImporter) ImportBatch(qaList []common.TransformedQA, startIndex int) {
	imp.stats.Total = len(qaList)

	fmt.Printf("\n开始批量导入 (从第 %d 条开始)...\n", startIndex+1)
	fmt.Printf("批次大小: %d\n", imp.batchSize)
	fmt.Printf("总数: %d 条\n", len(qaList))
	if imp.skipExisting {
		fmt.Println("跳过已存在: 是")
	} else {
		fmt.Println("跳过已存在: 否")
	}
	if imp.maxImportNum > 0 {
		fmt.Printf("导入数量限制: %d 条\n", imp.maxImportNum)
	} else {
		fmt.Println("导入数量限制: 无限制")
	}
	fmt.Println()

	importedCount := 0
	for i := startIndex; i < len(qaList); i++ {
		qaData := qaList[i]
		qaID := getQAID(qaData.Metadata)
		title := truncateString(qaData.Title, 50)

		fmt.Printf("[%d/%d] 导入 QA ID: %s - %s...\n", i+1, len(qaList), qaID, title)

		if imp.skipExisting {
			exists, err := imp.KnowledgeExists(qaID)
			if err != nil {
				fmt.Printf("  ⚠️  检查失败: %v，继续导入\n", err)
			} else if exists {
				imp.stats.Skipped++
				fmt.Println("  ⏭️  已存在，跳过")
				continue
			}
		}

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
			importedCount++

			if imp.maxImportNum > 0 && importedCount >= imp.maxImportNum {
				fmt.Printf("\n✅ 已达到导入数量限制 (%d 条)，停止导入\n", imp.maxImportNum)
				return
			}
		}

		if (i+1)%imp.batchSize == 0 {
			fmt.Printf("\n--- 已完成 %d/%d 条，暂停 0.5 秒 ---\n\n", i+1, len(qaList))
			time.Sleep(1000 * time.Millisecond)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (imp *QABatchImporter) PrintStats() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("导入统计:")
	fmt.Printf("  总计: %d 条\n", imp.stats.Total)
	fmt.Printf("  成功: %d 条\n", imp.stats.Success)
	fmt.Printf("  跳过: %d 条\n", imp.stats.Skipped)
	fmt.Printf("  失败: %d 条\n", imp.stats.Failed)
	processed := imp.stats.Success + imp.stats.Skipped
	if imp.stats.Total > 0 {
		fmt.Printf("  成功率: %.2f%%\n", float64(imp.stats.Success)/float64(imp.stats.Total)*100)
		fmt.Printf("  处理率: %.2f%%\n", float64(processed)/float64(imp.stats.Total)*100)
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func connectDatabase(dbHost, dbPort, dbUser, dbPassword, dbName string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	return db, nil
}

func main() {
	var (
		apiURL      string
		token       string
		kbID        string
		batchSize   int
		startIndex  int
		failedLog   string
		skipExist   bool
		dbHost      string
		dbPort      string
		dbUser      string
		dbPassword  string
		dbName      string
		showHelp    bool
		importNum   int
	)

	flag.StringVar(&apiURL, "api-url", "", "API 基础 URL (必填)")
	flag.StringVar(&token, "token", "", "认证 token (必填)")
	flag.StringVar(&kbID, "kb-id", "", "知识库 ID (必填)")
	flag.IntVar(&batchSize, "batch-size", 10, "每批次导入数量")
	flag.IntVar(&startIndex, "start-index", 0, "起始索引，用于断点续传")
	flag.StringVar(&failedLog, "failed-log", "failed_imports.json", "失败记录保存文件")
	flag.BoolVar(&skipExist, "skip-existing", false, "跳过已存在的知识（需要提供数据库配置）")
	flag.StringVar(&dbHost, "db-host", os.Getenv("DB_HOST"), "数据库主机 (默认从 DB_HOST 环境变量读取)")
	flag.StringVar(&dbPort, "db-port", getEnvOrDefault("DB_PORT", "5432"), "数据库端口 (默认从 DB_PORT 环境变量读取)")
	flag.StringVar(&dbUser, "db-user", getEnvOrDefault("DB_USER", "postgres"), "数据库用户 (默认从 DB_USER 环境变量读取)")
	flag.StringVar(&dbPassword, "db-password", os.Getenv("DB_PASSWORD"), "数据库密码 (默认从 DB_PASSWORD 环境变量读取)")
	flag.StringVar(&dbName, "db-name", getEnvOrDefault("DB_NAME", "WeKnora"), "数据库名称 (默认从 DB_NAME 环境变量读取)")
	flag.IntVar(&importNum, "num", 0, "导入数量限制 (0 表示无限制，>0 表示导入指定条数后退出)")
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
		fmt.Println("  --batch-size      每批次导入数量 (默认: 10)")
		fmt.Println("  --start-index     起始索引，用于断点续传 (默认: 0)")
		fmt.Println("  --failed-log      失败记录保存文件 (默认: failed_imports.json)")
		fmt.Println("  --skip-existing   跳过已存在的知识（基于 metadata.qa_id）")
		fmt.Println("  --num             导入数量限制 (0=无限制, >0=导入指定条数后退出, 默认: 0)")
		fmt.Println("\n数据库配置 (用于 --skip-existing):")
		fmt.Println("  --db-host         数据库主机 (默认从 DB_HOST 环境变量读取)")
		fmt.Println("  --db-port         数据库端口 (默认从 DB_PORT 环境变量读取，默认: 5432)")
		fmt.Println("  --db-user         数据库用户 (默认从 DB_USER 环境变量读取，默认: postgres)")
		fmt.Println("  --db-password     数据库密码 (默认从 DB_PASSWORD 环境变量读取)")
		fmt.Println("  --db-name         数据库名称 (默认从 DB_NAME 环境变量读取，默认: WeKnora)")
		fmt.Println("\n示例:")
		fmt.Println("  # 基本导入")
		fmt.Println("  importer --api-url http://localhost:8080 \\")
		fmt.Println("           --token YOUR_TOKEN \\")
		fmt.Println("           --kb-id kb-123456 \\")
		fmt.Println("           transformed_qa_data.json")
		fmt.Println("")
		fmt.Println("  # 跳过已存在的记录")
		fmt.Println("  importer --api-url http://localhost:8080 \\")
		fmt.Println("           --token YOUR_TOKEN \\")
		fmt.Println("           --kb-id kb-123456 \\")
		fmt.Println("           --skip-existing \\")
		fmt.Println("           --db-host localhost \\")
		fmt.Println("           --db-password yourpass \\")
		fmt.Println("           transformed_qa_data.json")
		fmt.Println("")
		fmt.Println("  # 限制导入数量")
		fmt.Println("  importer --api-url http://localhost:8080 \\")
		fmt.Println("           --token YOUR_TOKEN \\")
		fmt.Println("           --kb-id kb-123456 \\")
		fmt.Println("           --num 100 \\")
		fmt.Println("           transformed_qa_data.json")
		os.Exit(0)
	}

	if apiURL == "" || token == "" || kbID == "" {
		fmt.Println("❌ 错误: 必须提供 --api-url, --token 和 --kb-id 参数")
		fmt.Println("使用 --help 查看帮助信息")
		os.Exit(1)
	}

	var db *sql.DB
	if skipExist {
		if dbHost == "" {
			fmt.Println("❌ 错误: 启用 --skip-existing 时必须提供数据库主机 (--db-host 或 DB_HOST 环境变量)")
			os.Exit(1)
		}

		fmt.Println("\n连接数据库...")
		fmt.Printf("  主机: %s\n", dbHost)
		fmt.Printf("  端口: %s\n", dbPort)
		fmt.Printf("  用户: %s\n", dbUser)
		fmt.Printf("  数据库: %s\n", dbName)

		var err error
		db, err = connectDatabase(dbHost, dbPort, dbUser, dbPassword, dbName)
		if err != nil {
			fmt.Printf("❌ 数据库连接失败: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		fmt.Println("✅ 数据库连接成功")
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

	var qaList []common.TransformedQA
	if err := json.Unmarshal(data, &qaList); err != nil {
		fmt.Printf("❌ 错误: JSON 格式无效 - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ 成功读取 %d 条记录\n", len(qaList))

	if startIndex > 0 {
		fmt.Printf("⚠️  从第 %d 条开始导入（断点续传）\n", startIndex+1)
	}

	importer := NewQABatchImporter(apiURL, token, kbID, batchSize, db, skipExist, importNum)

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
