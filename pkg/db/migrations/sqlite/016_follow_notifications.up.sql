-- Keep notifications in the same transaction as the request that creates them.
CREATE TRIGGER notify_private_follow_request
AFTER INSERT ON user_followers
WHEN NEW.status = 0
BEGIN
    INSERT INTO notifications (user_id, actor_id, category, type, message)
    VALUES (NEW.target_id, NEW.follower_id, 'requests', 'follow_request',
        (SELECT first_name || ' ' || last_name || ' wants to follow you' FROM user WHERE id = NEW.follower_id));
END;

CREATE TRIGGER remove_cancelled_follow_notification
AFTER DELETE ON user_followers
BEGIN
    DELETE FROM notifications WHERE category = 'requests' AND type = 'follow_request'
        AND user_id = OLD.target_id AND actor_id = OLD.follower_id;
END;

-- Existing pending requests also need a visible action after upgrading.
INSERT INTO notifications (user_id, actor_id, category, type, message)
SELECT f.target_id, f.follower_id, 'requests', 'follow_request', u.first_name || ' ' || u.last_name || ' wants to follow you'
FROM user_followers f JOIN user u ON u.id = f.follower_id
WHERE f.status = 0 AND NOT EXISTS (
    SELECT 1 FROM notifications n WHERE n.user_id = f.target_id AND n.actor_id = f.follower_id
    AND n.type = 'follow_request' AND n.category = 'requests'
);
