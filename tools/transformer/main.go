package main

import (
	"encoding/json"
	"fmt"
	stdhtml "html"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Wintercom/c-cube/tools/common"
	"golang.org/x/net/html"
)

type HistoricalQA struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Replies     []Reply `json:"replies"`
}

type Reply struct {
	Content string `json:"content"`
	Owner   string `json:"owner"`
}

type QADataTransformer struct {
	stats Stats
}

type Stats struct {
	Total      int
	Success    int
	Failed     int
	Skipped    int
	LowQuality int
}

type QualityScore struct {
	Score   int
	Reasons []string
}

func NewQADataTransformer() *QADataTransformer {
	return &QADataTransformer{
		stats: Stats{},
	}
}

func (t *QADataTransformer) CleanHTMLContent(htmlText string) string {
	if htmlText == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return t.stripHTMLSimple(htmlText)
	}

	var text strings.Builder
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}
	extractText(doc)

	result := stdhtml.UnescapeString(text.String())
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

func (t *QADataTransformer) stripHTMLSimple(htmlText string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(htmlText, " ")
	text = stdhtml.UnescapeString(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (t *QADataTransformer) BuildConversationalPassage(qa HistoricalQA) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("问题标题: %s\n", t.CleanHTMLContent(qa.Title)))
	sb.WriteString("\n")

	if qa.Description != "" {
		sb.WriteString(fmt.Sprintf("问题描述: %s\n", t.CleanHTMLContent(qa.Description)))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("分类: %s\n", qa.Category))
	sb.WriteString("\n")

	sb.WriteString("对话记录:\n")
	sb.WriteString("\n")

	conversationNum := 0
	for _, reply := range qa.Replies {
		ownerLabel := "客户"
		if reply.Owner == "agent" {
			ownerLabel = "客服"
		}

		content := t.CleanHTMLContent(reply.Content)
		conversationNum++

		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", conversationNum, ownerLabel, content))
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("=", 60))

	return sb.String()
}

func (t *QADataTransformer) ExtractMetadata(qa HistoricalQA) map[string]interface{} {
	return map[string]interface{}{
		"qa_id":       fmt.Sprintf("%d", qa.ID),
		"category":    qa.Category,
		"source":      "historical_qa",
		"import_date": time.Now().Format("2006-01-02"),
		"reply_count": len(qa.Replies),
	}
}

func (t *QADataTransformer) CalculateQualityScore(qa HistoricalQA) QualityScore {
	score := 50
	var reasons []string

	agentReplies := []string{}
	for _, reply := range qa.Replies {
		if reply.Owner == "agent" {
			agentReplies = append(agentReplies, t.CleanHTMLContent(reply.Content))
		}
	}

	if len(agentReplies) == 0 {
		score -= 40
		reasons = append(reasons, "缺少客服回复")
		return QualityScore{Score: score, Reasons: reasons}
	}

	replyCount := len(qa.Replies)
	if replyCount < 2 {
		score -= 30
		reasons = append(reasons, "对话轮次过少")
	} else if replyCount >= 3 {
		score += 10
	}

	avgLength := 0
	for _, content := range agentReplies {
		avgLength += len([]rune(content))
	}
	avgLength /= len(agentReplies)

	if avgLength < 10 {
		score -= 25
		reasons = append(reasons, "客服回复过于简短")
	} else if avgLength > 50 {
		score += 15
	}

	techKeywords := []string{
		"API", "SDK", "token", "配置", "参数", "代码",
		"文档", "接口", "错误", "报错", "日志", "http",
		"bucket", "空间", "域名", "证书", "转码",
	}
	hasTechContent := false
	for _, reply := range agentReplies {
		for _, keyword := range techKeywords {
			if strings.Contains(reply, keyword) {
				hasTechContent = true
				break
			}
		}
		if hasTechContent {
			break
		}
	}
	if hasTechContent {
		score += 20
	} else {
		score -= 10
		reasons = append(reasons, "缺少技术内容")
	}

	lowValuePatterns := []string{
		"您再看下", "已处理", "手动介入", "已经帮您",
		"稍等", "正在处理", "麻烦您提供", "联系客服",
	}
	hasLowValueReply := false
	for _, reply := range qa.Replies {
		content := t.CleanHTMLContent(reply.Content)
		for _, pattern := range lowValuePatterns {
			if strings.Contains(content, pattern) && len([]rune(content)) < 20 {
				hasLowValueReply = true
				break
			}
		}
		if hasLowValueReply {
			break
		}
	}
	if hasLowValueReply && !hasTechContent {
		score -= 20
		reasons = append(reasons, "包含低价值模板回复")
	}

	title := t.CleanHTMLContent(qa.Title)
	if len([]rune(title)) < 5 {
		score -= 10
		reasons = append(reasons, "标题过短")
	}

	return QualityScore{Score: score, Reasons: reasons}
}

func (t *QADataTransformer) ValidateQA(qa HistoricalQA) bool {
	if qa.Title == "" && qa.Description == "" {
		return false
	}

	if len(qa.Replies) == 0 {
		return false
	}

	hasValidContent := false
	for _, reply := range qa.Replies {
		content := t.CleanHTMLContent(reply.Content)
		if content != "" {
			hasValidContent = true
			break
		}
	}

	if !hasValidContent {
		return false
	}

	qualityScore := t.CalculateQualityScore(qa)
	threshold := 40

	if qualityScore.Score < threshold {
		t.stats.LowQuality++
		fmt.Printf("  QA ID %d 质量分数 %d 低于阈值 %d, 原因: %v\n",
			qa.ID, qualityScore.Score, threshold, qualityScore.Reasons)
		return false
	}

	return true
}

func (t *QADataTransformer) TransformSingleQA(qa HistoricalQA) (*common.TransformedQA, error) {
	if !t.ValidateQA(qa) {
		return nil, fmt.Errorf("问答数据无效或内容为空")
	}

	passage := t.BuildConversationalPassage(qa)
	metadata := t.ExtractMetadata(qa)
	title := t.CleanHTMLContent(qa.Title)
	description := t.CleanHTMLContent(qa.Description)

	return &common.TransformedQA{
		Title:       title,
		Description: description,
		Passage:     passage,
		Metadata:    metadata,
	}, nil
}

func (t *QADataTransformer) TransformBatch(qaList []HistoricalQA) ([]common.TransformedQA, error) {
	t.stats.Total = len(qaList)
	var transformedList []common.TransformedQA

	for i, qa := range qaList {
		transformed, err := t.TransformSingleQA(qa)
		if err != nil {
			t.stats.Skipped++
			fmt.Printf("跳过第 %d 条记录 (ID: %d): %v\n", i+1, qa.ID, err)
			continue
		}

		transformedList = append(transformedList, *transformed)
		t.stats.Success++

		if (i+1)%100 == 0 {
			fmt.Printf("已处理 %d/%d 条记录\n", i+1, t.stats.Total)
		}
	}

	return transformedList, nil
}

func (t *QADataTransformer) PrintStats() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("数据转换统计:")
	fmt.Printf("  总计: %d 条\n", t.stats.Total)
	fmt.Printf("  成功: %d 条\n", t.stats.Success)
	fmt.Printf("  跳过: %d 条\n", t.stats.Skipped)
	fmt.Printf("  低质量过滤: %d 条\n", t.stats.LowQuality)
	fmt.Printf("  失败: %d 条\n", t.stats.Failed)
	fmt.Println(strings.Repeat("=", 60))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("用法: transformer <输入JSON文件> <输出JSON文件>")
		fmt.Println("\n示例:")
		fmt.Println("  transformer raw_qa_data.json transformed_qa_data.json")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	fmt.Printf("正在读取文件: %s\n", inputFile)

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("❌ 错误: 文件 '%s' 不存在或无法读取\n", inputFile)
		os.Exit(1)
	}

	var qaList []HistoricalQA
	if err := json.Unmarshal(data, &qaList); err != nil {
		fmt.Printf("❌ 错误: JSON 格式无效 - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("成功读取 %d 条问答记录\n\n", len(qaList))

	transformer := NewQADataTransformer()
	fmt.Println("开始转换数据...")

	transformedList, err := transformer.TransformBatch(qaList)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
		os.Exit(1)
	}

	transformer.PrintStats()

	fmt.Printf("\n正在保存到文件: %s\n", outputFile)
	outputData, err := json.MarshalIndent(transformedList, "", "  ")
	if err != nil {
		fmt.Printf("❌ 错误: 无法序列化 JSON - %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		fmt.Printf("❌ 错误: 无法写入文件 - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 转换完成！已保存 %d 条记录\n", len(transformedList))
}
