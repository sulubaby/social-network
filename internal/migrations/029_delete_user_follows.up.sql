-- user_followers has no foreign key, so a deleted account left its follows
-- behind and the other people kept wrong follower / following numbers.
CREATE TRIGGER IF NOT EXISTS trg_delete_user_follows
AFTER DELETE ON user
FOR EACH ROW
BEGIN
    DELETE FROM user_followers
    WHERE follower_id = OLD.id OR target_id = OLD.id;
END;

-- the +1 on accept only counts a real change from request (0) to follower (1),
-- before it counted again on every update of an accepted row
DROP TRIGGER IF EXISTS trg_increase_followers_update;
DROP TRIGGER IF EXISTS trg_increase_following_update;

CREATE TRIGGER trg_increase_followers_update
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status <> 1 AND NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_followers = num_of_followers + 1
    WHERE user_id = NEW.target_id;
END;

CREATE TRIGGER trg_increase_following_update
AFTER UPDATE OF status ON user_followers
FOR EACH ROW
WHEN OLD.status <> 1 AND NEW.status = 1
BEGIN
    UPDATE profile
    SET num_of_following = num_of_following + 1
    WHERE user_id = NEW.follower_id;
END;

-- clean up what already got left behind and count everything again
DELETE FROM user_followers
WHERE follower_id NOT IN (SELECT id FROM user)
   OR target_id NOT IN (SELECT id FROM user);

UPDATE profile
SET num_of_followers = (
        SELECT COUNT(*) FROM user_followers
        WHERE target_id = profile.user_id AND status = 1
    ),
    num_of_following = (
        SELECT COUNT(*) FROM user_followers
        WHERE follower_id = profile.user_id AND status = 1
    );
