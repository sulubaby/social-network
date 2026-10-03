DROP TRIGGER IF EXISTS increase_following_count;
DROP TRIGGER IF EXISTS increase_follower_count;
DROP TRIGGER IF EXISTS decrease_following_count;
DROP TRIGGER IF EXISTS decrease_follower_count;
DROP TRIGGER IF EXISTS increase_following_count_on_accept;
DROP TRIGGER IF EXISTS increase_follower_count_on_accept;
DROP TRIGGER IF EXISTS decrease_following_count_on_revoke;
DROP TRIGGER IF EXISTS decrease_follower_count_on_revoke;

CREATE TRIGGER increase_following_count
AFTER INSERT ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;

CREATE TRIGGER increase_follower_count
AFTER INSERT ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;

CREATE TRIGGER decrease_following_count
AFTER DELETE ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following - 1
    WHERE user_id = OLD.follower_id;
END;

CREATE TRIGGER decrease_follower_count
AFTER DELETE ON user_followers
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers - 1
    WHERE user_id = OLD.target_id;
END;
