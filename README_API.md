# C-Cube API Server

## Overview

This is the backend API server for the C-Cube intelligent customer service system.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile commands)

### Installation

```bash
# Download dependencies
go mod download

# Build the server
make build
# or
go build -o bin/server cmd/server/main.go
```

### Running the Server

```bash
# Using Make
make run

# Or directly with Go
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "time": "2024-01-01T00:00:00Z"
}
```

### Create Knowledge from Passage

Create knowledge entries from text passages.

```http
POST /api/v1/knowledge-bases/:id/knowledge/passage
```

**Headers:**
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "passages": ["知识内容"],
  "title": "标题",
  "description": "描述",
  "metadata": {
    "category": "FAQ",
    "qa_id": "001"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-01T00:00:00Z",
  "message": "Knowledge passage created successfully"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/knowledge-bases/kb-123/knowledge/passage \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "passages": ["知识内容"],
    "title": "标题",
    "description": "描述",
    "metadata": {
      "category": "FAQ",
      "qa_id": "001"
    }
  }'
```

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── handler/
│   │   └── knowledge.go      # HTTP handlers
│   ├── middleware/
│   │   ├── cors.go          # CORS middleware
│   │   └── logger.go        # Logging middleware
│   └── model/
│       └── knowledge.go      # Data models
├── go.mod                    # Go module definition
├── Makefile                  # Build automation
└── README_API.md            # This file
```

## Development

### Build Commands

```bash
# Build the binary
make build

# Run the server
make run

# Run tests
make test

# Clean build artifacts
make clean

# Download and tidy dependencies
make deps
```

## Integration with Import Tools

This API is designed to work with the import tools located in the `tools/` directory:

- `tools/importer` - Batch import historical QA data
- `tools/transformer` - Transform JSON QA data to the required format

See the [tools README](tools/README.md) for more information.

## License

Copyright © 2024 C-Cube. All rights reserved.
