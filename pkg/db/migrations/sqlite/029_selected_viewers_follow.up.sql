-- A "selected followers" post is only for people who still follow the author.
-- When someone unfollows (or gets removed as a follower) they lose access.
DELETE FROM post_viewers
WHERE NOT EXISTS (
    SELECT 1
    FROM posts
    JOIN user_followers ON user_followers.target_id = posts.user_id
    WHERE posts.id = post_viewers.post_id
      AND user_followers.follower_id = post_viewers.viewer_id
      AND user_followers.status = 1
);

CREATE TRIGGER IF NOT EXISTS remove_selected_viewer_on_unfollow
AFTER DELETE ON user_followers
FOR EACH ROW
BEGIN
    DELETE FROM post_viewers
    WHERE viewer_id = OLD.follower_id
      AND post_id IN (SELECT id FROM posts WHERE user_id = OLD.target_id);
END;
