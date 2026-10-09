DROP TRIGGER IF EXISTS trg_delete_user_follows;

DROP TRIGGER IF EXISTS trg_increase_followers_update;
DROP TRIGGER IF EXISTS trg_increase_following_update;

CREATE TRIGGER trg_increase_followers_update
AFTER UPDATE ON user_followers
FOR EACH ROW
WHEN NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;

CREATE TRIGGER trg_increase_following_update
AFTER UPDATE ON user_followers
FOR EACH ROW
WHEN NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;
