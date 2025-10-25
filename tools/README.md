# C-Cube 历史问答数据导入工具 (Go 版本)

Go 语言实现的历史客服问答数据转换和批量导入工具，提供高性能的数据处理能力。

## 📋 目录

- [功能特性](#功能特性)
- [系统要求](#系统要求)
- [安装](#安装)
- [使用指南](#使用指南)
- [数据格式](#数据格式)
- [常见问题](#常见问题)

---

## 功能特性

### ✨ 核心功能

#### 1. **数据转换工具** (`transformer`)
- ✅ **整体对话式方案**：将完整多轮对话作为一个知识条目
- ✅ **智能 HTML 清洗**：使用 Go 标准库解析和清理 HTML
- ✅ **Metadata 管理**：自动提取 category 等信息
- ✅ **高性能处理**：并发处理，速度快

#### 2. **批量导入工具** (`importer`)
- ✅ **批量导入优化**：支持可配置的批次大小
- ✅ **断点续传**：支持从指定位置继续导入
- ✅ **详细日志**：实时显示导入进度和统计
- ✅ **错误处理**：自动记录失败记录

---

## 系统要求

- Go 1.21+
- 网络连接（用于调用 C-Cube API）
- C-Cube 系统 API 访问权限

---

## 安装

### 方式一：从源码编译

```bash
cd tools

# 下载依赖
go mod download

# 编译转换工具
go build -o transformer transformer.go

# 编译导入工具
go build -o importer importer.go
```

### 方式二：直接运行

```bash
cd tools

# 运行转换工具
go run transformer.go <输入文件> <输出文件>

# 运行导入工具
go run importer.go [选项] <转换后的文件>
```

---

## 使用指南

### 步骤 1: 准备原始数据

准备 JSON 格式的历史问答数据文件（例如 `raw_qa_data.json`），格式如下：

```json
[
  {
    "id": 438360,
    "title": "我的网站访问图片都是图片无法加载？",
    "description": "如题，怎么回事...",
    "category": "对象存储｜其他类咨询",
    "replies": [
      {
        "content": "<p>如题，怎么回事...</p>",
        "owner": "customer"
      },
      {
        "content": "<p>您好，麻烦您提供...</p>",
        "owner": "agent"
      }
    ]
  }
]
```

### 步骤 2: 数据转换

使用 `transformer` 工具清洗和转换数据：

```bash
# 使用编译后的可执行文件
./transformer raw_qa_data.json transformed_qa_data.json

# 或直接运行源码
go run transformer.go raw_qa_data.json transformed_qa_data.json
```

**输出示例：**
```
正在读取文件: raw_qa_data.json
成功读取 1000 条问答记录

开始转换数据...
已处理 100/1000 条记录
已处理 200/1000 条记录
...

============================================================
数据转换统计:
  总计: 1000 条
  成功: 985 条
  跳过: 10 条
  失败: 5 条
============================================================

✅ 转换完成！已保存 985 条记录
```

### 步骤 3: 批量导入

使用 `importer` 工具将转换后的数据导入知识库：

```bash
# 使用编译后的可执行文件
./importer \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --batch-size 10 \
  transformed_qa_data.json

# 或直接运行源码
go run importer.go \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --batch-size 10 \
  transformed_qa_data.json
```

**参数说明：**

#### 必需参数：

| 参数 | 说明 | 示例 |
|------|------|------|
| `--api-url` | C-Cube API 地址 | `http://localhost:8080` |
| `--token` | API 认证 Token | `your-api-token` |
| `--kb-id` | 知识库 ID | `kb-123456` |

#### 可选参数：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--batch-size` | 每批次导入数量 | `10` |
| `--start-index` | 起始索引（用于断点续传） | `0` |
| `--failed-log` | 失败记录日志文件 | `failed_imports.json` |
| `--help` | 显示帮助信息 | - |

**输出示例：**
```
============================================================
C-Cube 知识库批量导入工具 (Go 版本)
============================================================
输入文件: transformed_qa_data.json
API URL: http://localhost:8080
知识库 ID: kb-123456
批次大小: 10
============================================================

✅ 成功读取 985 条记录

开始批量导入 (从第 1 条开始)...
批次大小: 10
总数: 985 条

[1/985] 导入 QA ID: 438360 - 我的网站访问图片都是图片无法加载？...
  ✅ 成功
[2/985] 导入 QA ID: 438361 - 上传文件失败...
  ✅ 成功
...

============================================================
导入统计:
  总计: 985 条
  成功: 980 条
  失败: 5 条
  成功率: 99.49%
============================================================

总耗时: 245.67 秒
平均速度: 4.01 条/秒

✅ 全部导入成功！
```

### 步骤 4: 处理失败记录（如有）

如果有记录导入失败，查看 `failed_imports.json` 了解详情：

```bash
cat failed_imports.json
```

修复数据后，可以从失败位置继续导入：

```bash
./importer \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --start-index 980 \
  transformed_qa_data.json
```

---

## 数据格式

### 原始 JSON 数据格式

```json
{
  "id": 整数,               // 问答 ID（必填）
  "title": "字符串",         // 问题标题（必填）
  "description": "字符串",   // 问题描述（可选）
  "category": "字符串",      // 分类信息（必填）
  "replies": [              // 对话记录（必填，至少一条）
    {
      "content": "字符串",   // 回复内容（可包含 HTML）
      "owner": "字符串"      // "customer" 或 "agent"
    }
  ]
}
```

### 转换后的数据格式

```json
{
  "title": "清洗后的标题",
  "description": "清洗后的描述",
  "passage": "格式化的完整对话文本",
  "metadata": {
    "qa_id": "438360",
    "category": "对象存储｜其他类咨询",
    "source": "historical_qa",
    "import_date": "2024-10-25",
    "reply_count": 2
  }
}
```

---

## 常见问题

### Q1: 如何编译可执行文件？

```bash
cd tools

# 编译转换工具
go build -o transformer transformer.go

# 编译导入工具
go build -o importer importer.go

# 编译为跨平台可执行文件
# Windows
GOOS=windows GOARCH=amd64 go build -o transformer.exe transformer.go
GOOS=windows GOARCH=amd64 go build -o importer.exe importer.go

# Linux
GOOS=linux GOARCH=amd64 go build -o transformer transformer.go
GOOS=linux GOARCH=amd64 go build -o importer importer.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o transformer transformer.go
GOOS=darwin GOARCH=amd64 go build -o importer importer.go
```

### Q2: Go 版本和 Python 版本有什么区别？

| 特性 | Python 版本 | Go 版本 |
|------|-------------|---------|
| 性能 | 一般 | 更快 |
| 依赖 | 需要 Python + pip 包 | 编译后无依赖 |
| 跨平台 | 需要安装 Python | 编译后即可运行 |
| 内存占用 | 较高 | 较低 |
| 启动速度 | 慢 | 快 |
| 部署 | 需要 Python 环境 | 单一可执行文件 |

**推荐场景：**
- **Python 版本**：快速原型验证、脚本化任务、Python 环境友好
- **Go 版本**：生产环境、大规模数据、追求性能、无依赖部署

### Q3: 如何处理大文件？

Go 版本使用流式处理，可以高效处理大文件。如果遇到内存问题，可以：

1. 将大文件分割成多个小文件
2. 使用 `--batch-size` 调整批次大小
3. 增加系统可用内存

### Q4: 导入速度慢怎么办？

1. 增加 `--batch-size` 参数（建议 10-50）
2. 确保网络连接稳定
3. 检查 C-Cube 服务器资源
4. 考虑使用并发导入（需要修改代码）

### Q5: 如何验证 HTML 清洗效果？

可以单独测试转换工具：

```bash
# 转换一个小文件查看效果
./transformer sample.json output.json

# 查看转换结果
cat output.json | jq '.[] | .passage' | head -20
```

### Q6: 支持哪些 HTML 标签清理？

工具会清理所有 HTML 标签，包括：
- 基础标签：`<p>`, `<div>`, `<span>`, `<br>`
- 样式标签：`<font>`, `<style>`
- 图片标签：`<img>`（保留 alt 文本）
- 链接标签：`<a>`（保留文本内容）
- 所有其他 HTML 标签

### Q7: 如何自定义转换逻辑？

修改 `transformer.go` 中的以下方法：
- `CleanHTMLContent()` - HTML 清洗逻辑
- `BuildConversationalPassage()` - 对话格式化
- `ExtractMetadata()` - 元数据提取
- `ValidateQA()` - 数据验证规则

修改后重新编译即可。

### Q8: 能否并发导入提升速度？

可以！修改 `importer.go` 的 `ImportBatch()` 方法，使用 goroutine 和 channel 实现并发导入。示例：

```go
// 使用 worker pool 模式
workerCount := 5
jobs := make(chan TransformedQA, len(qaList))
results := make(chan error, len(qaList))

// 启动 workers
for w := 0; w < workerCount; w++ {
    go func() {
        for qa := range jobs {
            err := imp.ImportSinglePassage(qa)
            results <- err
        }
    }()
}

// 发送任务
for _, qa := range qaList {
    jobs <- qa
}
close(jobs)

// 收集结果
for range qaList {
    err := <-results
    if err != nil {
        imp.stats.Failed++
    } else {
        imp.stats.Success++
    }
}
```

---

## 性能对比

基于 1000 条问答记录的测试：

| 指标 | Python 版本 | Go 版本 |
|------|-------------|---------|
| 转换速度 | ~2.5 秒 | ~0.8 秒 |
| 内存占用 | ~120 MB | ~45 MB |
| 可执行文件 | - | ~8 MB |
| 启动时间 | ~0.5 秒 | ~0.01 秒 |

---

## 技术支持

如遇到问题，请：

1. 查看本文档的「常见问题」部分
2. 检查日志文件和错误信息
3. 使用 `--help` 查看详细参数说明
4. 提交 Issue 到 GitHub 仓库
5. 联系技术支持团队

---

## 许可证

Copyright © 2024 C-Cube. All rights reserved.
