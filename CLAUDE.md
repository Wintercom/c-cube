# WeKnora - 项目分析与开发指南

> 本文档由 Claude Code 自动生成，基于对 WeKnora 项目的全面分析

##  ️ 项目架构概览

WeKnora 是一个基于大语言模型的文档理解与语义检索框架，采用模块化架构构建完整的 RAG 流水线。

### 核心技术栈

- **后端**: Go 1.24 + Gin 框架 + Uber Dig 依赖注入
- **前端**: Vue 3 + TypeScript + TDesign UI + Vite
- **文档解析**: Python gRPC 微服务
- **基础设施**: Docker + Docker Compose
- **数据库**: PostgreSQL (pgvector) + Redis + Neo4j
- **存储**: MinIO (对象存储)
- **监控**: Jaeger (分布式追踪)

##  📁 项目目录结构

```
WeKnora/
├── cmd/server/                 # Go 应用入口
│    └── main.go                 # 主程序入口，依赖注入容器初始化
├── config/                     # 配置文件
│   └── config.yaml             # 服务配置（对话配置、提示词模板等）
├── docker/                     # Docker 构建文件
│   ├── Dockerfile.app          # 后端应用镜像
│    └── Dockerfile.docreader    # 文档解析服务镜像
├── frontend/                   # 前端应用
│   ├── src/                    # 源码目录
│   │   ├── views/              # 页面组件
│   │   │   ├── auth/           # 认证页面
│   │   │   ├── chat/           # 聊天界面
│   │   │   ├── knowledge/      # 知识库管理
│   │   │    └── settings/       # 系统设置
│   │   ├── components/         # 通用组件
│   │   ├── stores/             # Pinia 状态管理
│   │   ├── api/                # API 接口定义
│   │    └── router/             # 路由配置
│   ├── package.json            # 前端依赖管理
│    └── vite.config.ts          # Vite 构建配置
├── internal/                   # Go 内部包（核心业务逻辑）
│   ├── application/            # 应用层
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务服务层
│   │    └── chat_pipline/       # 聊天流水线插件系统
│   ├── config/                 # 配置管理
│   ├── handler/                # HTTP 请求处理器
│   ├── middleware/             # Gin 中间件
│   ├── models/                 # 模型定义（LLM、Embedding 等）
│   ├── router/                 # 路由定义
│   ├── types/                  # 类型定义和接口
│   └── utils/                  # 工具函数
├── migrations/                 # 数据库迁移脚本
├── scripts/                    # 构建和部署脚本
├── services/                   # 微服务实现
│    └── docreader/              # Python 文档解析服务
└── docs/                       # 项目文档
```

## 🔧 核心模块分析

### 后端架构 (Go)

#### 1. 依赖注入系统 (`internal/container/`)
- 使用 Uber Dig 实现依赖注入
- 统一管理所有服务的生命周期
- 支持接口隔离和模块化测试

#### 2. 聊天流水线系统 (`internal/application/service/chat_pipline/`)
- 插件化架构，支持事件驱动处理
- 主要事件类型：预处理、检索、重排、生成等
- 支持熔断机制和错误处理

#### 3. 检索引擎 (`internal/application/service/retriever/`)
- 混合检索策略：向量检索 + 关键词检索
- 支持多种向量数据库后端
- 基于接口的插件化设计

#### 4. API 路由 (`internal/router/`)
- RESTful API 设计
- 认证中间件和权限控制
- 一致的错误处理机制

### 前端架构 (Vue 3)

#### 1. 路由配置 (`frontend/src/router/index.ts`)
- 基于文件的路由配置
- 路由守卫实现认证检查
- 支持知识库级别的路由嵌套

#### 2. 状态管理 (`frontend/src/stores/`)
- 使用 Pinia 进行状态管理
- 模块化状态设计：认证、知识库、菜单等
- 类型安全的 TypeScript 支持

#### 3. 组件设计
- 基于 TDesign Vue Next 组件库
- 模块化组件设计，职责单一
- 支持响应式布局和主题定制

### 文档解析微服务 (Python)

#### 1. gRPC 服务设计 (`services/docreader/src/server/server.py`)
- 基于 gRPC 的高性能文档解析
- 支持多格式文档：PDF、Word、图片、网页等
- 多模态处理支持 OCR 和图像字幕

#### 2. 解析器架构 (`services/docreader/src/parser/`)
- 插件化文档解析器
- 支持分块配置和自定义分隔符
- 集成视觉语言模型 (VLM) 处理图像

##  🚀 开发环境配置

### 1. 环境要求
```bash
# 必备工具
Docker & Docker Compose
Git
Go 1.24+ (后端开发)
Node.js 18+ (前端开发)
Python 3.8+ (文档解析服务开发)
```

### 2. 快速启动
```bash
# 克隆项目
git clone https://github.com/Tencent/WeKnora.git
cd WeKnora

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件设置必要参数

# 启动完整服务栈
./scripts/start_all.sh

# 或使用 Makefile
make start-all
```

### 3. 开发模式启动
```bash
# 仅启动基础设施服务
docker-compose up postgres redis minio neo4j jaeger

# 后端开发模式 (热重载)
cd cmd/server
go run main.go

# 前端开发模式
cd frontend
npm install
npm run dev

# 文档解析服务开发
cd services/docreader
poetry install
python src/server/server.py
```

## 🔌 关键配置文件

### 1. 后端配置 (`config/config.yaml`)
- 服务器端口和主机配置
- 对话服务参数（重写、重排阈值等）
- 提示词模板配置
- 知识库分块配置

### 2. Docker 编排 (`docker-compose.yml`)
- 微服务容器定义和网络配置
- 环境变量注入和健康检查
- 数据卷持久化配置

### 3. Makefile 构建系统
- 多平台 Docker 镜像构建
- 数据库迁移和清理
- 开发工具链集成

##  📊 数据模型概览

### 核心实体关系
```
Tenant (租户)
    ↓
KnowledgeBase (知识库) → EmbeddingModel (嵌入模型)
    ↓
Knowledge (知识) → Chunk (分块)
    ↓
Session (会话) → Message (消息)
```

### 主要数据库表
- `knowledge_bases`: 知识库元数据
- `knowledge`: 知识文档信息
- `chunks`: 文档分块内容（向量化存储）
- `sessions`: 聊天会话
- `messages`: 对话消息历史

##  🔍 API 接口概览

### 认证相关 (`/api/v1/auth/*`)
- `POST /auth/login` - 用户登录
- `POST /auth/register` - 用户注册
- `GET /auth/me` - 获取当前用户信息

### 知识库管理 (`/api/v1/knowledge-bases/*`)
- `POST /knowledge-bases` - 创建知识库
- `GET /knowledge-bases` - 获取知识库列表
- `GET /knowledge-bases/:id/hybrid-search` - 混合搜索

### 文档管理 (`/api/v1/knowledge/*`)
- `POST /knowledge-bases/:id/knowledge/file` - 文件上传
- `POST /knowledge-bases/:id/knowledge/url` - URL 导入
- `GET /knowledge/:id` - 获取知识详情

### 对话接口 (`/api/v1/knowledge-chat/*`)
- `POST /knowledge-chat/:session_id` - 知识问答
- `POST /knowledge-search` - 知识检索

##  🛠️ 开发扩展指南

### 1. 添加新的检索策略
1. 在 `internal/types/interfaces/retriever.go` 定义接口
2. 在 `internal/application/service/retriever/` 实现检索器
3. 在 `internal/application/service/retriever/registry.go` 注册
4. 配置租户的检索器引擎

### 2. 扩展文档解析格式
1. 在 `services/docreader/src/parser/` 添加新的解析器
2. 实现 `parse_file` 方法
3. 在 `services/docreader/src/parser/parser.py` 注册

### 3. 定制聊天流水线
1. 在 `internal/application/service/chat_pipline/` 添加插件
2. 实现 `Plugin` 接口
3. 注册到事件管理器
4. 配置事件处理链

##  调试和监控

### 1. 日志系统
- 后端使用 structured logging
- 请求 ID 追踪支持
- 日志级别可配置

### 2. 分布式追踪
- 集成 OpenTelemetry
- Jaeger UI: http://localhost:16686
- 完整的请求链路追踪

### 3. 健康检查
- 服务健康检查端点: `/health`
- 数据库连接状态监控
- gRPC 健康检查支持

##  🔒 安全考虑

### 1. 认证授权
- JWT Token 认证
- 租户级别的数据隔离
- API Key 访问控制

### 2. 数据安全
- 敏感信息环境变量配置
- 文件上传安全检查
- SQL 注入防护

### 3. 部署安全
- 内网部署推荐
- 防火墙规则配置
- 定期安全更新

## 📈 性能优化建议

### 1. 数据库优化
- 向量索引优化
- 批量查询优化
- 连接池配置

### 2. 检索优化
- 分块策略调优
- 缓存策略实现
- 异步处理优化

### 3. 前端优化
- 代码分割和懒加载
- 图片优化和压缩
- API 请求合并

---

## 📚 进一步阅读

- [官方文档](./docs/WeKnora.md) - 完整的功能说明
- [API 文档](./docs/API.md) - 详细的接口文档
- [QA 文档](./docs/QA.md) - 常见问题解答

## 🤝 贡献指南

项目采用标准的 Git 工作流：
1. Fork 项目到个人账户
2. 创建功能分支 (`feature/feature-name`)
3. 提交代码变更
4. 创建 Pull Request
5. 代码审查和合并

遵循 Conventional Commits 规范提交消息。