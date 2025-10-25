-- Create import_tasks table for batch documentation import tasks
CREATE TABLE IF NOT EXISTS import_tasks (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    base_url TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_urls INTEGER NOT NULL DEFAULT 0,
    processed_urls INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    duplicate_count INTEGER NOT NULL DEFAULT 0,
    current_url TEXT,
    error_message TEXT,
    config JSON NOT NULL,
    results JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    INDEX idx_import_tasks_tenant_id (tenant_id),
    INDEX idx_import_tasks_knowledge_base_id (knowledge_base_id),
    INDEX idx_import_tasks_status (status),
    INDEX idx_import_tasks_created_at (created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Batch documentation import tasks tracking table';
