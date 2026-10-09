DROP TRIGGER IF EXISTS trg_change_privay;

CREATE TRIGGER IF NOT EXISTS trg_change_privay
AFTER UPDATE ON profile
FOR EACH ROW
BEGIN
    UPDATE user_followers
    SET status = 1
    WHERE status = 0 AND target_id = NEW.user_id;
END;
