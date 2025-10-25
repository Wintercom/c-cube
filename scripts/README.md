# C-Cube 历史问答数据导入工具

本工具用于将历史人工客服问答的 JSON 结构化数据批量导入到 C-Cube 智能客服系统的知识库中。

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

1. **整体对话式数据组织**
   - 将完整的多轮对话作为一个知识条目
   - 保留问题标题、描述、分类和完整对话记录
   - 便于用户理解完整的问题解决过程

2. **智能数据清洗**
   - 自动清理 HTML 标签（`<p>`, `<span>`, `<img>` 等）
   - 移除多余的空白字符和换行
   - HTML 实体解码（如 `&nbsp;` → 空格）

3. **Metadata 元数据管理**
   - 自动提取并保存 `category` 分类信息
   - 记录原始问答 ID、导入日期、回复数量等
   - 便于后续过滤和数据追溯

4. **批量导入优化**
   - 支持分批次导入，避免系统负载过高
   - 断点续传功能，支持从失败位置继续
   - 详细的导入日志和统计信息

---

## 系统要求

- Python 3.7+
- 网络连接（用于调用 C-Cube API）
- C-Cube 系统 API 访问权限

---

## 安装

### 1. 安装依赖

```bash
cd scripts
pip install -r requirements.txt
```

### 2. 验证安装

```bash
python qa_data_transformer.py --help
python qa_batch_importer.py --help
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
    "description": "如题，怎么回事，图片在电脑上访问不到...",
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

使用 `qa_data_transformer.py` 清洗和转换数据：

```bash
python qa_data_transformer.py raw_qa_data.json transformed_qa_data.json
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

使用 `qa_batch_importer.py` 将转换后的数据导入知识库：

```bash
python qa_batch_importer.py transformed_qa_data.json \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --batch-size 10
```

**参数说明：**

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `input_file` | ✅ | 转换后的 JSON 文件 | `transformed_qa_data.json` |
| `--api-url` | ✅ | C-Cube API 地址 | `http://localhost:8080` |
| `--token` | ✅ | API 认证 Token | `your-api-token` |
| `--kb-id` | ✅ | 知识库 ID | `kb-123456` |
| `--batch-size` | ❌ | 每批次导入数量（默认：10） | `20` |
| `--start-index` | ❌ | 起始索引（用于断点续传） | `500` |
| `--failed-log` | ❌ | 失败记录日志文件（默认：`failed_imports.json`） | `errors.json` |

**输出示例：**
```
============================================================
C-Cube 知识库批量导入工具
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

⚠️  部分记录导入失败，请检查 failed_imports.json
可使用 --start-index 参数重试失败的记录
```

### 步骤 4: 处理失败记录（如有）

如果有记录导入失败，查看 `failed_imports.json` 了解详情：

```bash
cat failed_imports.json
```

根据失败原因修复数据后，可以从失败位置继续导入：

```bash
python qa_batch_importer.py transformed_qa_data.json \
  --api-url http://localhost:8080 \
  --token YOUR_API_TOKEN \
  --kb-id YOUR_KNOWLEDGE_BASE_ID \
  --start-index 980
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

### Passage 格式示例

```
============================================================
问题标题: 我的网站访问图片都是图片无法加载？

问题描述: 如题，怎么回事，图片在电脑上访问不到，手机也无法访问图片

分类: 对象存储｜其他类咨询

对话记录:
------------------------------------------------------------
1. [客户] 如题，怎么回事，图片在电脑上访问不到...

2. [客服] 您好，麻烦您提供一个具体的访问异常的资源链接。

============================================================
```

---

## 常见问题

### Q1: 如何获取 API Token？

登录 C-Cube 管理后台，进入「设置」→「API 密钥」，创建或复制现有的 Token。

### Q2: 如何获取知识库 ID？

在知识库管理页面，点击目标知识库，浏览器地址栏中的 ID 即为知识库 ID。

例如：`http://localhost:8080/knowledge-bases/kb-123456` 中的 `kb-123456`。

### Q3: 导入速度慢怎么办？

1. 增加 `--batch-size` 参数（如设置为 20 或 50）
2. 确保网络连接稳定
3. 检查 C-Cube 服务器资源是否充足

### Q4: 部分数据导入失败如何处理？

1. 查看 `failed_imports.json` 文件了解失败原因
2. 检查失败记录的数据完整性
3. 修复数据后使用 `--start-index` 参数重试

### Q5: 可以中断后继续导入吗？

可以！使用 `Ctrl+C` 中断导入后，工具会显示已成功导入的数量。使用 `--start-index` 参数即可从中断位置继续：

```bash
python qa_batch_importer.py transformed_qa_data.json \
  --api-url http://localhost:8080 \
  --token YOUR_TOKEN \
  --kb-id YOUR_KB_ID \
  --start-index 500
```

### Q6: HTML 清洗不彻底怎么办？

工具已内置两种清洗方式：
1. BeautifulSoup（推荐）- 智能解析 HTML
2. 正则表达式（备用）- 简单标签移除

如果仍有问题，可以手动预处理数据，或联系技术支持。

### Q7: 如何按分类过滤导入的知识？

导入后，可以通过 C-Cube 的检索 API 使用 metadata 过滤：

```bash
curl -X POST "http://localhost:8080/api/v1/knowledge/search" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "query": "问题关键词",
    "metadata_filter": {
      "category": "对象存储｜其他类咨询"
    }
  }'
```

### Q8: 是否支持增量导入？

支持！只需将新的问答数据追加到 JSON 文件中，重新执行转换和导入流程即可。系统会自动创建新的知识条目。

---

## 技术支持

如遇到问题，请：

1. 查看本文档的「常见问题」部分
2. 检查日志文件和错误信息
3. 提交 Issue 到 GitHub 仓库
4. 联系技术支持团队

---

## 许可证

Copyright © 2024 C-Cube. All rights reserved.
