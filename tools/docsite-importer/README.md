# WeKnora 文档站批量导入工具

这是一个用于批量导入文档站内容到 WeKnora 知识库的命令行工具。

## 功能特性

- ✅ **自动爬取**: 自动爬取文档站所有页面链接
- ✅ **文件导入**: 支持从文件读取 URL 列表
- ✅ **并发导入**: 可配置并发数，提高导入效率
- ✅ **断点续传**: 支持中断后继续导入，避免重复
- ✅ **智能去重**: 自动跳过已导入和重复的 URL
- ✅ **进度显示**: 实时显示导入进度和统计信息
- ✅ **错误处理**: 详细记录失败的 URL 和错误信息
- ✅ **多模态支持**: 可选启用多模态内容处理

## 安装

### 编译

```bash
cd tools/docsite-importer
go build -o docsite-importer main.go
```

### 或者使用 go install

```bash
go install github.com/Tencent/WeKnora/tools/docsite-importer@latest
```

## 使用方法

### 基本用法

```bash
docsite-importer \
  --api-url <API地址> \
  --token <认证Token> \
  --kb-id <知识库ID> \
  --base-url <文档站URL>
```

### 参数说明

#### 必需参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--api-url` | WeKnora API 地址 | `http://localhost:8080` |
| `--token` | API 认证 Token (X-API-Key) | `sk-xxxxx` |
| `--kb-id` | 目标知识库 ID | `kb-xxxxx` |

#### 导入方式 (二选一)

| 参数 | 说明 | 示例 |
|------|------|------|
| `--base-url` | 文档站基础 URL (自动爬取) | `https://docs.example.com` |
| `--url-file` | URL 列表文件 (每行一个 URL) | `urls.txt` |

#### 可选参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--max-pages` | 最大爬取页面数 | `200` |
| `--concurrent` | 并发导入数 | `3` |
| `--enable-multimodel` | 启用多模态处理 | `false` |
| `--progress-file` | 断点续传进度文件 | `.docsite-importer-progress.json` |
| `--failed-log` | 失败记录保存文件 | `failed_imports.json` |

## 使用示例

### 示例 1: 自动爬取并导入

爬取文档站所有页面并导入到知识库:

```bash
docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-your-api-key \
  --kb-id kb-123456 \
  --base-url https://docs.example.com \
  --max-pages 200 \
  --concurrent 3
```

### 示例 2: 从文件导入

准备 URL 列表文件 `urls.txt`:

```
https://docs.example.com/guide/intro
https://docs.example.com/guide/getting-started
https://docs.example.com/api/reference
# 这是注释，会被忽略
https://docs.example.com/faq
```

执行导入:

```bash
docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-your-api-key \
  --kb-id kb-123456 \
  --url-file urls.txt \
  --concurrent 5
```

### 示例 3: 启用多模态处理

对于包含图片等多媒体内容的文档:

```bash
docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-your-api-key \
  --kb-id kb-123456 \
  --base-url https://docs.example.com \
  --enable-multimodel
```

### 示例 4: 断点续传

如果导入中途失败，可以直接重新运行相同的命令，工具会自动跳过已导入的 URL:

```bash
docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-your-api-key \
  --kb-id kb-123456 \
  --base-url https://docs.example.com
```

工具会自动加载 `.docsite-importer-progress.json` 中的进度信息。

## 工作流程

1. **URL 发现**
   - 方式 1: 爬取文档站，自动发现所有页面链接
   - 方式 2: 从文件读取预定义的 URL 列表

2. **智能过滤**
   - 自动跳过非文档页面 (图片、CSS、JS 等)
   - 只保留同域名下的链接
   - 去除 URL 锚点和参数

3. **并发导入**
   - 使用可配置的并发数同时导入多个 URL
   - 自动控制请求频率，避免服务器压力

4. **断点续传**
   - 每导入 10 个 URL 自动保存进度
   - 重启后自动跳过已导入的 URL

5. **错误处理**
   - 记录失败的 URL 和错误信息
   - 继续导入其他 URL，不会因单个失败而中断

6. **统计报告**
   - 显示总数、成功、失败、重复、跳过的统计
   - 计算成功率和平均速度

## 输出示例

```
======================================================================
WeKnora 文档站批量导入工具
======================================================================
API URL: http://localhost:8080
知识库 ID: kb-123456
并发数: 3
多模态处理: false
进度文件: .docsite-importer-progress.json
======================================================================

开始爬取文档站: https://docs.example.com

  发现: [1/200] https://docs.example.com/
  发现: [2/200] https://docs.example.com/guide/intro
  发现: [3/200] https://docs.example.com/guide/getting-started
  ...

✅ 爬取完成，共发现 45 个 URL

开始批量导入...
并发数: 3
总 URL 数: 45

[1/45] 导入: https://docs.example.com/...
  ✅ 成功
[2/45] 导入: https://docs.example.com/guide/intro...
  ✅ 成功
[3/45] 导入: https://docs.example.com/guide/getting-started...
  ⚠️  重复 URL
...

======================================================================
导入统计:
  总计: 45 个 URL
  成功: 42 个
  失败: 1 个
  重复: 2 个
  跳过: 0 个 (断点续传)
  成功率: 93.33%

  总耗时: 125.43 秒
  平均速度: 0.36 个/秒
======================================================================

✅ 全部导入成功！
```

## 常见问题

### 1. 如何获取 API Token?

在 WeKnora Web UI 中:
1. 打开开发者工具 (F12)
2. 查看任意 API 请求的 Request Headers
3. 找到 `X-API-Key` 字段，值以 `sk-` 开头

### 2. 如何获取知识库 ID?

在知识库列表页面，URL 中包含知识库 ID:
```
http://localhost/knowledge-bases/kb-xxxxx
                                  ^^^^^^^^
```

### 3. 导入速度太慢怎么办?

可以增加并发数:
```bash
--concurrent 10
```

但注意不要设置过高，以免对服务器造成压力。

### 4. 如何只导入特定的页面?

创建一个 URL 列表文件，只包含需要导入的页面:

```bash
docsite-importer \
  --api-url http://localhost:8080 \
  --token sk-xxxxx \
  --kb-id kb-xxxxx \
  --url-file my-urls.txt
```

### 5. 导入失败了怎么办?

1. 查看 `failed_imports.json` 文件，了解失败原因
2. 直接重新运行相同的命令，工具会自动跳过已成功的 URL
3. 如果某些 URL 持续失败，可以手动在 Web UI 中导入

### 6. 如何清除进度重新开始?

删除进度文件:
```bash
rm .docsite-importer-progress.json
```

然后重新运行导入命令。

## 技术细节

### URL 过滤规则

工具会自动跳过以下类型的 URL:

- **文件资源**: `.jpg`, `.png`, `.pdf`, `.zip` 等
- **前端资源**: `.css`, `.js`, `.woff` 等  
- **多媒体**: `.mp4`, `.mp3` 等
- **特殊路径**: `/api/`, `/assets/`, `/static/` 等
- **特殊协议**: `mailto:`, `tel:`, `javascript:` 等

### 爬虫配置

- **并发数**: 5 个并发爬虫线程
- **请求延迟**: 300ms
- **最大深度**: 5 层
- **域名限制**: 仅爬取与基础 URL 相同域名的页面

### API 调用

工具调用以下 WeKnora API:

```
POST /api/v1/knowledge-bases/{kb_id}/knowledge/url
```

请求体:
```json
{
  "url": "https://docs.example.com/page",
  "enable_multimodel": false
}
```

## 贡献

欢迎提交 Issue 和 Pull Request!

## 许可证

MIT License
