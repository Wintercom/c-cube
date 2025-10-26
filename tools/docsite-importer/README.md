# WeKnora 文档站批量导入工具

一个用于批量导入文档站内容到 WeKnora 知识库的命令行工具。该工具通过调用单个 URL 导入接口实现批量导入，支持自动爬取和断点续传。

## 功能特性

- ✅ **自动爬取**: 从文档站首页自动发现并爬取所有页面链接
- ✅ **文件导入**: 从预先准备的 URL 列表文件批量导入
- ✅ **并发控制**: 可配置并发数，避免对服务器造成过大压力
- ✅ **断点续传**: 支持中断后从上次位置继续导入
- ✅ **错误处理**: 完善的错误处理和失败重试机制
- ✅ **进度显示**: 实时显示导入进度和统计信息
- ✅ **结果导出**: 导出详细的导入结果到 JSON 文件

## 安装

### 方式1: 直接编译

```bash
cd tools/docsite-importer
go mod download
go build -o docsite-importer main.go
```

### 方式2: 使用 go install

```bash
go install github.com/Tencent/WeKnora/tools/docsite-importer@latest
```

## 使用方法

### 基本用法

#### 方式1: 自动爬取文档站

从文档站首页开始自动爬取所有页面并导入:

```bash
./docsite-importer \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --base-url https://docs.example.com \
  --max-pages 200 \
  --concurrent 3
```

#### 方式2: 从文件导入

从预先准备的 URL 列表文件导入:

```bash
./docsite-importer \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --url-file urls.txt \
  --concurrent 3
```

### 参数说明

| 参数 | 说明 | 必填 | 默认值 |
|------|------|------|--------|
| `--api-url` | WeKnora API 地址 | 否 | `http://localhost:8080` |
| `--token` | API 认证 Token (x-api-key) | **是** | - |
| `--kb-id` | 目标知识库 ID | **是** | - |
| `--base-url` | 文档站基础 URL (自动爬取模式) | 否* | - |
| `--url-file` | URL 列表文件路径 (文件导入模式) | 否* | - |
| `--max-pages` | 最大爬取页面数 | 否 | `200` |
| `--concurrent` | 并发导入数量 | 否 | `3` |
| `--resume-file` | 断点续传文件路径 | 否 | `import_progress.json` |
| `--output` | 结果输出文件路径 | 否 | `import_results.json` |

*注: `--base-url` 和 `--url-file` 二选一必填

### URL 列表文件格式

创建一个文本文件 `urls.txt`，每行一个 URL:

```
https://docs.example.com/guide/introduction
https://docs.example.com/guide/getting-started
https://docs.example.com/api/authentication
# 注释行会被忽略
https://docs.example.com/api/endpoints
```

### 获取 API Token

1. 登录 WeKnora Web 界面
2. 打开浏览器开发者工具 (F12)
3. 查看网络请求的 `x-api-key` 请求头
4. 复制该值作为 `--token` 参数

### 获取知识库 ID

1. 在 WeKnora Web 界面打开目标知识库
2. 从 URL 中获取知识库 ID，例如:
   ```
   http://localhost/platform/knowledge-bases/kb-123456/knowledge
                                              ^^^^^^^^^ 这就是 kb-id
   ```

## 使用示例

### 示例1: 导入 Vue.js 官方文档

```bash
./docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-abc123def456 \
  --kb-id kb-20250101-001 \
  --base-url https://vuejs.org/guide/ \
  --max-pages 150 \
  --concurrent 5
```

### 示例2: 从文件导入特定页面

1. 创建 `urls.txt`:
```
https://docs.python.org/3/tutorial/introduction.html
https://docs.python.org/3/tutorial/controlflow.html
https://docs.python.org/3/tutorial/datastructures.html
```

2. 运行导入:
```bash
./docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-abc123def456 \
  --kb-id kb-20250101-002 \
  --url-file urls.txt
```

### 示例3: 断点续传

如果导入过程中中断，再次运行相同命令即可从断点继续:

```bash
# 第一次运行 (假设中断了)
./docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-abc123def456 \
  --kb-id kb-20250101-001 \
  --base-url https://docs.example.com

# 第二次运行 (从断点继续)
./docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-abc123def456 \
  --kb-id kb-20250101-001 \
  --base-url https://docs.example.com
```

工具会自动读取 `import_progress.json` 文件，跳过已成功导入的 URL。

## 输出说明

### 控制台输出

```
🕷️  开始爬取文档站: https://docs.example.com
   发现: https://docs.example.com/guide/intro
   发现: https://docs.example.com/guide/installation
✅ 爬取完成，共发现 50 个页面

📥 开始导入，共 50 个页面，并发数: 3

[1/50] ✅ https://docs.example.com/guide/intro
[2/50] ✅ https://docs.example.com/guide/installation
[3/50] ⏭️  https://docs.example.com/guide/basics - URL已存在
[4/50] ❌ https://docs.example.com/guide/advanced - HTTP 500: ...

========================================
📊 导入统计
========================================
总计: 50
✅ 成功: 45
⏭️  跳过: 3
❌ 失败: 2
========================================
```

### 输出文件

#### `import_progress.json` (断点文件)

记录每个 URL 的导入状态，用于断点续传:

```json
[
  {
    "url": "https://docs.example.com/guide/intro",
    "success": true,
    "knowledge_id": "kn-20250101-001"
  },
  {
    "url": "https://docs.example.com/guide/advanced",
    "success": false,
    "message": "HTTP 500: Internal Server Error"
  }
]
```

#### `import_results.json` (结果文件)

完整的导入结果记录，包含所有详细信息。

## 常见问题

### Q: 如何提高导入速度？

A: 可以适当增加 `--concurrent` 参数值，但需注意:
- 不要设置过大，避免对服务器造成过大压力
- 建议值: 3-10，根据服务器性能调整
- 过大的并发可能导致请求失败率上升

### Q: 导入失败怎么办？

A: 工具会自动记录失败的 URL 和原因:
1. 查看控制台输出中的错误信息
2. 检查 `import_results.json` 文件获取详细错误
3. 修复问题后，重新运行命令（会自动跳过已成功的）

### Q: 如何只导入失败的 URL？

A: 可以从 `import_results.json` 中提取失败的 URL，创建新的 URL 列表文件，然后使用 `--url-file` 参数重新导入。

### Q: 可以中途停止并继续吗？

A: 可以。工具每导入 10 个 URL 就会保存一次进度。重新运行相同命令即可继续。

### Q: 如何清空之前的导入记录重新开始？

A: 删除 `import_progress.json` 文件即可:
```bash
rm import_progress.json
./docsite-importer ...
```

## 技术实现

- 使用 `goquery` 进行 HTML 解析和链接提取
- 支持并发导入，使用 semaphore 控制并发数
- 使用互斥锁保证并发安全
- 定期保存进度，支持断点续传
- 完善的错误处理和重试机制

## 注意事项

1. **尊重 robots.txt**: 爬取时请遵守网站的 robots.txt 规则
2. **合理设置并发**: 不要设置过大的并发数，避免对目标网站造成负担
3. **API 限流**: 如果遇到 429 错误，请降低并发数或增加请求间隔
4. **网络稳定**: 建议在网络稳定的环境下运行，避免频繁失败
5. **数据备份**: 重要的导入任务建议先在测试环境验证

## 许可证

MIT License
