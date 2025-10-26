package main

import (
	"encoding/json"
	"fmt"
	stdhtml "html"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

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

type KeywordInfo struct {
	Keyword   string `json:"keyword"`
	Frequency int    `json:"frequency"`
	Category  string `json:"category"`
}

type KeywordExtractor struct {
	keywordFreq map[string]int
	stopWords   map[string]bool
}

func NewKeywordExtractor() *KeywordExtractor {
	stopWords := map[string]bool{
		"您好": true, "您": true, "我": true, "的": true, "了": true,
		"是": true, "在": true, "有": true, "和": true, "就": true,
		"不": true, "人": true, "都": true, "一": true, "个": true,
		"上": true, "也": true, "很": true, "到": true, "说": true,
		"要": true, "去": true, "你": true, "会": true, "着": true,
		"没有": true, "什么": true, "这个": true, "那个": true, "这样": true,
		"怎么": true, "为什么": true, "可以": true, "已经": true, "还是": true,
		"稍等": true, "麻烦": true, "好的": true, "谢谢": true, "您再看下": true,
		"已处理": true, "手动介入": true, "已经帮您": true, "正在处理": true,
		"麻烦您提供": true, "联系客服": true, "这边": true, "帮您": true,
		"看下": true, "提供": true, "一下": true, "这里": true, "那边": true,
	}

	return &KeywordExtractor{
		keywordFreq: make(map[string]int),
		stopWords:   stopWords,
	}
}

func (e *KeywordExtractor) CleanHTMLContent(htmlText string) string {
	if htmlText == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return e.stripHTMLSimple(htmlText)
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

	return strings.TrimSpace(result)
}

func (e *KeywordExtractor) stripHTMLSimple(htmlText string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(htmlText, " ")
	text = stdhtml.UnescapeString(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func (e *KeywordExtractor) IsTechnicalKeyword(word string) bool {
	word = strings.TrimSpace(word)
	if len(word) < 2 {
		return false
	}

	if e.stopWords[word] {
		return false
	}

	hasEnglish := false
	hasDigit := false
	for _, r := range word {
		if unicode.IsLetter(r) && r < 256 {
			hasEnglish = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if hasEnglish {
		return true
	}

	technicalPatterns := []string{
		"错误", "报错", "异常", "失败", "问题",
		"配置", "设置", "参数", "选项", "功能",
		"文件", "上传", "下载", "存储", "空间",
		"域名", "证书", "解析", "绑定", "备案",
		"接口", "调用", "请求", "响应", "返回",
		"代码", "脚本", "命令", "语句", "方法",
		"数据", "字段", "记录", "内容", "信息",
		"格式", "类型", "版本", "编码", "解码",
		"权限", "认证", "授权", "密钥", "签名",
		"转码", "处理", "分析", "检测", "识别",
		"流量", "带宽", "速度", "延迟", "超时",
		"日志", "监控", "统计", "报表", "数据",
		"回调", "通知", "推送", "订阅", "发布",
		"查询", "搜索", "过滤", "排序", "分页",
		"缓存", "队列", "任务", "进程", "线程",
		"模板", "样式", "主题", "布局", "组件",
	}

	for _, pattern := range technicalPatterns {
		if strings.Contains(word, pattern) {
			return true
		}
	}

	if hasDigit && len(word) >= 3 {
		return true
	}

	return false
}

func (e *KeywordExtractor) ExtractKeywords(text string) []string {
	var keywords []string

	englishWordRegex := regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_\-\.]*`)
	englishWords := englishWordRegex.FindAllString(text, -1)
	for _, word := range englishWords {
		word = strings.ToLower(word)
		if len(word) >= 2 && e.IsTechnicalKeyword(word) {
			keywords = append(keywords, word)
		}
	}

	chineseText := englishWordRegex.ReplaceAllString(text, " ")

	chineseWords := e.extractChineseWords(chineseText)
	for _, word := range chineseWords {
		if e.IsTechnicalKeyword(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func (e *KeywordExtractor) extractChineseWords(text string) []string {
	var words []string

	for length := 15; length >= 2; length-- {
		runes := []rune(text)
		for i := 0; i <= len(runes)-length; i++ {
			word := string(runes[i : i+length])
			word = strings.TrimSpace(word)

			if len(word) == 0 {
				continue
			}

			hasOnlyChinese := true
			for _, r := range word {
				if !unicode.Is(unicode.Han, r) && !unicode.IsDigit(r) {
					hasOnlyChinese = false
					break
				}
			}

			if hasOnlyChinese && e.IsTechnicalKeyword(word) {
				words = append(words, word)
			}
		}
	}

	return words
}

func (e *KeywordExtractor) ProcessQAList(qaList []HistoricalQA) {
	fmt.Printf("开始提取关键词，共 %d 条问答记录\n\n", len(qaList))

	for i, qa := range qaList {
		for _, reply := range qa.Replies {
			if reply.Owner == "agent" {
				content := e.CleanHTMLContent(reply.Content)
				keywords := e.ExtractKeywords(content)

				for _, keyword := range keywords {
					e.keywordFreq[keyword]++
				}
			}
		}

		if (i+1)%10 == 0 {
			fmt.Printf("已处理 %d/%d 条记录\n", i+1, len(qaList))
		}
	}

	fmt.Printf("\n提取完成！共发现 %d 个不重复的关键词\n", len(e.keywordFreq))
}

func (e *KeywordExtractor) hasSignificantOverlap(word1, word2 string) bool {
	runes1 := []rune(word1)
	runes2 := []rune(word2)
	
	len1 := len(runes1)
	len2 := len(runes2)
	minLen := len1
	if len2 < minLen {
		minLen = len2
	}
	
	if minLen < 2 {
		return false
	}
	
	overlapThreshold := minLen - 1
	if overlapThreshold < 2 {
		overlapThreshold = 2
	}
	
	for i := overlapThreshold; i <= len1; i++ {
		suffix := string(runes1[len1-i:])
		if strings.HasPrefix(word2, suffix) && len([]rune(suffix)) >= overlapThreshold {
			return true
		}
	}
	
	for i := overlapThreshold; i <= len2; i++ {
		suffix := string(runes2[len2-i:])
		if strings.HasPrefix(word1, suffix) && len([]rune(suffix)) >= overlapThreshold {
			return true
		}
	}
	
	matchCount := 0
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}
	for i := 0; i < minLen; i++ {
		if i < len1 && i < len2 && runes1[i] == runes2[i] {
			matchCount++
		}
	}
	
	if len1 == len2 && matchCount >= minLen*7/10 {
		return true
	}
	
	return false
}

func (e *KeywordExtractor) removeSubstringKeywords(keywords []KeywordInfo) []KeywordInfo {
	sort.Slice(keywords, func(i, j int) bool {
		lenI := len([]rune(keywords[i].Keyword))
		lenJ := len([]rune(keywords[j].Keyword))
		if lenI != lenJ {
			return lenI > lenJ
		}
		if keywords[i].Frequency != keywords[j].Frequency {
			return keywords[i].Frequency > keywords[j].Frequency
		}
		return keywords[i].Keyword < keywords[j].Keyword
	})

	allWords := make(map[string]KeywordInfo)
	for _, kw := range keywords {
		allWords[kw.Keyword] = kw
	}

	var result []KeywordInfo
	removed := make(map[string]bool)

	for _, kw := range keywords {
		if removed[kw.Keyword] {
			continue
		}

		shouldRemove := false
		for otherWord := range allWords {
			if otherWord == kw.Keyword || removed[otherWord] {
				continue
			}

			otherLen := len([]rune(otherWord))
			kwLen := len([]rune(kw.Keyword))

			if strings.Contains(otherWord, kw.Keyword) {
				if otherLen > kwLen {
					shouldRemove = true
					break
				}
			}

			if e.hasSignificantOverlap(kw.Keyword, otherWord) {
				if otherLen > kwLen {
					shouldRemove = true
					break
				} else if otherLen == kwLen {
					if otherWord < kw.Keyword {
						shouldRemove = true
						break
					}
				}
			}
		}

		if !shouldRemove {
			result = append(result, kw)
		} else {
			removed[kw.Keyword] = true
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Frequency != result[j].Frequency {
			return result[i].Frequency > result[j].Frequency
		}
		return result[i].Keyword < result[j].Keyword
	})

	return result
}

func (e *KeywordExtractor) GetTopKeywords(minFreq int) []KeywordInfo {
	var keywords []KeywordInfo

	for keyword, freq := range e.keywordFreq {
		if freq >= minFreq {
			category := e.categorizeKeyword(keyword)
			keywords = append(keywords, KeywordInfo{
				Keyword:   keyword,
				Frequency: freq,
				Category:  category,
			})
		}
	}

	keywords = e.removeSubstringKeywords(keywords)

	return keywords
}

func (e *KeywordExtractor) categorizeKeyword(keyword string) string {
	keyword = strings.ToLower(keyword)

	apiKeywords := []string{"api", "sdk", "token", "接口", "调用", "请求"}
	for _, k := range apiKeywords {
		if strings.Contains(keyword, k) {
			return "API相关"
		}
	}

	storageKeywords := []string{"bucket", "空间", "存储", "上传", "下载", "文件"}
	for _, k := range storageKeywords {
		if strings.Contains(keyword, k) {
			return "存储相关"
		}
	}

	networkKeywords := []string{"域名", "dns", "证书", "ssl", "cdn", "http", "https"}
	for _, k := range networkKeywords {
		if strings.Contains(keyword, k) {
			return "网络相关"
		}
	}

	errorKeywords := []string{"错误", "报错", "异常", "失败", "error"}
	for _, k := range errorKeywords {
		if strings.Contains(keyword, k) {
			return "错误相关"
		}
	}

	configKeywords := []string{"配置", "参数", "设置", "选项"}
	for _, k := range configKeywords {
		if strings.Contains(keyword, k) {
			return "配置相关"
		}
	}

	return "其他技术"
}

func (e *KeywordExtractor) PrintStats(keywords []KeywordInfo) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("关键词提取统计")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("总关键词数量: %d\n", len(e.keywordFreq))
	fmt.Printf("高频关键词数量 (出现次数 ≥ 阈值): %d\n\n", len(keywords))

	categoryCount := make(map[string]int)
	for _, kw := range keywords {
		categoryCount[kw.Category]++
	}

	fmt.Println("分类统计:")
	for category, count := range categoryCount {
		fmt.Printf("  %s: %d 个\n", category, count)
	}

	fmt.Println("\n" + strings.Repeat("-", 70))
	fmt.Println("高频关键词列表 (Top 50):")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("%-30s %-12s %s\n", "关键词", "出现次数", "分类")
	fmt.Println(strings.Repeat("-", 70))

	limit := 50
	if len(keywords) < limit {
		limit = len(keywords)
	}

	for i := 0; i < limit; i++ {
		kw := keywords[i]
		fmt.Printf("%-30s %-12d %s\n", kw.Keyword, kw.Frequency, kw.Category)
	}

	fmt.Println(strings.Repeat("=", 70))
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("用法: keyword_extractor <输入JSON文件> <输出JSON文件> [最小频次]")
		fmt.Println("\n参数说明:")
		fmt.Println("  输入JSON文件: 与 transformer 相同的问答数据文件")
		fmt.Println("  输出JSON文件: 输出的关键词列表文件")
		fmt.Println("  最小频次: 可选，默认为 2，只输出出现次数 >= 该值的关键词")
		fmt.Println("\n示例:")
		fmt.Println("  keyword_extractor example_qa_data.json keywords.json")
		fmt.Println("  keyword_extractor example_qa_data.json keywords.json 3")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]
	minFreq := 2

	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &minFreq)
	}

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

	fmt.Printf("成功读取 %d 条问答记录\n", len(qaList))
	fmt.Printf("最小频次阈值: %d\n\n", minFreq)

	extractor := NewKeywordExtractor()
	extractor.ProcessQAList(qaList)

	keywords := extractor.GetTopKeywords(minFreq)
	extractor.PrintStats(keywords)

	fmt.Printf("\n正在保存到文件: %s\n", outputFile)
	outputData, err := json.MarshalIndent(keywords, "", "  ")
	if err != nil {
		fmt.Printf("❌ 错误: 无法序列化 JSON - %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		fmt.Printf("❌ 错误: 无法写入文件 - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 提取完成！已保存 %d 个关键词\n", len(keywords))
}
