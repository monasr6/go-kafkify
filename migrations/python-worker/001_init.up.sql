-- Create processed_events table for Python worker
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS processed_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),
    task_id VARCHAR(100),
    action VARCHAR(50),
    payload JSONB NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_processed_events_event_type ON processed_events(event_type);
CREATE INDEX idx_processed_events_resource_id ON processed_events(resource_id);
CREATE INDEX idx_processed_events_task_id ON processed_events(task_id);
CREATE INDEX idx_processed_events_processed_at ON processed_events(processed_at);
