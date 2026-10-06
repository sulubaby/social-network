DROP INDEX IF EXISTS idx_post_reactions_user_id;

DROP INDEX IF EXISTS idx_comments_post_id;
DROP INDEX IF EXISTS idx_comments_user_id;
DROP INDEX IF EXISTS idx_comments_reply_to;

DROP INDEX IF EXISTS idx_comment_votes_user_id;

DROP TABLE IF EXISTS comment_votes;

DROP TABLE IF EXISTS comments;

DROP TABLE IF EXISTS post_reactions;