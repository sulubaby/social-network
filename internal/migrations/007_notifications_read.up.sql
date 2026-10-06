ALTER TABLE notifications ADD COLUMN is_read INTEGER NOT NULL DEFAULT 0;
ALTER TABLE notifications_types ADD COLUMN comment_id_tag INTEGER REFERENCES comments(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_notifications_user_read
ON notifications(user_id, is_read);
