DROP TABLE IF EXISTS post_viewers;
DROP INDEX IF EXISTS idx_posts_privacy;
ALTER TABLE posts DROP COLUMN privacy;
