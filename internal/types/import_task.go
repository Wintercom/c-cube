package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImportTaskStatus string

const (
	ImportTaskStatusPending    ImportTaskStatus = "pending"
	ImportTaskStatusProcessing ImportTaskStatus = "processing"
	ImportTaskStatusCompleted  ImportTaskStatus = "completed"
	ImportTaskStatusFailed     ImportTaskStatus = "failed"
	ImportTaskStatusCancelled  ImportTaskStatus = "cancelled"
)

type ImportTask struct {
	ID              string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint             `json:"tenant_id" gorm:"index"`
	KnowledgeBaseID string           `json:"knowledge_base_id" gorm:"type:varchar(36);index"`
	BaseURL         string           `json:"base_url"`
	Status          ImportTaskStatus `json:"status" gorm:"type:varchar(20);index"`
	TotalURLs       int              `json:"total_urls"`
	ProcessedURLs   int              `json:"processed_urls"`
	SuccessCount    int              `json:"success_count"`
	FailedCount     int              `json:"failed_count"`
	DuplicateCount  int              `json:"duplicate_count"`
	CurrentURL      string           `json:"current_url"`
	ErrorMessage    string           `json:"error_message"`
	Config          JSON             `json:"config" gorm:"type:json"`
	Results         JSON             `json:"results" gorm:"type:json"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at"`
}

func (t *ImportTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

func (t *ImportTask) TableName() string {
	return "import_tasks"
}

type ImportTaskConfig struct {
	MaxPages         int   `json:"max_pages"`
	EnableMultimodel *bool `json:"enable_multimodel"`
}

type ImportTaskResult struct {
	URL         string `json:"url"`
	Status      string `json:"status"`
	KnowledgeID string `json:"knowledge_id,omitempty"`
	Error       string `json:"error,omitempty"`
}
