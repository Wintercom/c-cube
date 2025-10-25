-- Create import_tasks table for batch documentation import tasks
CREATE TABLE IF NOT EXISTS import_tasks (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
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
    config JSONB NOT NULL DEFAULT '{}',
    results JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Add indexes for import_tasks
CREATE INDEX IF NOT EXISTS idx_import_tasks_tenant_id ON import_tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_tasks_knowledge_base_id ON import_tasks(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_import_tasks_status ON import_tasks(status);
CREATE INDEX IF NOT EXISTS idx_import_tasks_created_at ON import_tasks(created_at DESC);

-- Add comment
COMMENT ON TABLE import_tasks IS 'Batch documentation import tasks tracking table';
COMMENT ON COLUMN import_tasks.status IS 'Task status: pending, processing, completed, failed, cancelled';
COMMENT ON COLUMN import_tasks.config IS 'Task configuration including max_pages, enable_multimodel';
COMMENT ON COLUMN import_tasks.results IS 'Array of import results for each URL';
