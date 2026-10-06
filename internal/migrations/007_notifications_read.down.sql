DROP INDEX IF EXISTS idx_notifications_user_read;
ALTER TABLE notifications_types DROP COLUMN comment_id_tag;
ALTER TABLE notifications DROP COLUMN is_read;
