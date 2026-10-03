-- Pending follow requests (status = 0) are not relationships yet.
-- Rebuild the counters first so older databases are corrected as well.
UPDATE profile
SET
    num_of_followers = (
        SELECT COUNT(*)
        FROM user_followers
        WHERE target_id = profile.user_id AND status = 1
    ),
    num_of_following = (
        SELECT COUNT(*)
        FROM user_followers
        WHERE follower_id = profile.user_id AND status = 1
    );

DROP TRIGGER IF EXISTS increase_following_count;
DROP TRIGGER IF EXISTS increase_follower_count;
DROP TRIGGER IF EXISTS decrease_following_count;
DROP TRIGGER IF EXISTS decrease_follower_count;

CREATE TRIGGER increase_following_count
AFTER INSERT ON user_followers
FOR EACH ROW
WHEN NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;

CREATE TRIGGER increase_follower_count
AFTER INSERT ON user_followers
FOR EACH ROW
WHEN NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;

CREATE TRIGGER decrease_following_count
AFTER DELETE ON user_followers
FOR EACH ROW
WHEN OLD.status = 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following - 1
    WHERE user_id = OLD.follower_id;
END;

CREATE TRIGGER decrease_follower_count
AFTER DELETE ON user_followers
FOR EACH ROW
WHEN OLD.status = 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers - 1
    WHERE user_id = OLD.target_id;
END;

CREATE TRIGGER increase_following_count_on_accept
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status <> 1 AND NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;

CREATE TRIGGER increase_follower_count_on_accept
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status <> 1 AND NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;

CREATE TRIGGER decrease_following_count_on_revoke
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status = 1 AND NEW.status <> 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following - 1
    WHERE user_id = NEW.follower_id;
END;

CREATE TRIGGER decrease_follower_count_on_revoke
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status = 1 AND NEW.status <> 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers - 1
    WHERE user_id = NEW.target_id;
END;
