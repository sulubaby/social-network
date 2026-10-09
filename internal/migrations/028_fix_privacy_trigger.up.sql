-- The old trigger ran on every profile update (bio, avatar...), so a private
-- account accepted all its pending follow requests just by editing the bio.
-- Now it only runs when the profile really changes from private to public,
-- and the request notifications that can no longer be answered are removed.
DROP TRIGGER IF EXISTS trg_change_privay;

CREATE TRIGGER IF NOT EXISTS trg_change_privay
AFTER UPDATE OF is_private ON profile
FOR EACH ROW
WHEN OLD.is_private = 1 AND NEW.is_private = 0
BEGIN
    DELETE FROM notifications
    WHERE user_id = NEW.user_id
      AND id IN (
          SELECT nt.notifications_id
          FROM notifications_types nt
          JOIN user_followers uf
            ON uf.follower_id = nt.follow_request_user_id
           AND uf.target_id = NEW.user_id
           AND uf.status = 0
      );

    UPDATE user_followers
    SET status = 1
    WHERE status = 0 AND target_id = NEW.user_id;
END;
