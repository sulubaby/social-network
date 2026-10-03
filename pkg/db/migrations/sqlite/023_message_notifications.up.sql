-- Notification categories are part of the table check constraint, so SQLite
-- needs a small table rebuild to add messages while preserving existing rows.
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS delete_group_join_request_notifications;
DROP TRIGGER IF EXISTS notify_private_follow_request;
DROP TRIGGER IF EXISTS remove_cancelled_follow_notification;
DROP TRIGGER IF EXISTS delete_group_invitation_notifications;
DROP TRIGGER IF EXISTS delete_group_notifications;
DROP TRIGGER IF EXISTS delete_event_notifications;

ALTER TABLE notifications RENAME TO notifications_old;

CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    actor_id INTEGER,
    category VARCHAR(20) NOT NULL CHECK (category IN ('requests', 'groups', 'events', 'messages')),
    type VARCHAR(50) NOT NULL,
    message VARCHAR(500) NOT NULL,
    related_id INTEGER,
    is_read INTEGER NOT NULL DEFAULT 0 CHECK (is_read IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES user(id) ON DELETE SET NULL
);

INSERT INTO notifications (id, user_id, actor_id, category, type, message, related_id, is_read, created_at)
SELECT id, user_id, actor_id, category, type, message, related_id, is_read, created_at
FROM notifications_old;

DROP TABLE notifications_old;

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC, id DESC);

CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id, is_read);

CREATE TRIGGER delete_group_join_request_notifications
AFTER DELETE ON group_join_requests
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'join_request'
      AND related_id = OLD.id;
END;

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

CREATE TRIGGER delete_group_invitation_notifications
AFTER DELETE ON group_invitations
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'invitation'
      AND related_id = OLD.id;
END;

CREATE TRIGGER delete_group_notifications
BEFORE DELETE ON groups
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'join_request'
      AND related_id IN (
          SELECT id FROM group_join_requests WHERE group_id = OLD.id
      );

    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'invitation'
      AND related_id IN (
          SELECT id FROM group_invitations WHERE group_id = OLD.id
      );

    DELETE FROM notifications
    WHERE category = 'events'
      AND related_id IN (
          SELECT id FROM events WHERE group_id = OLD.id
      );

    DELETE FROM notifications
    WHERE category = 'groups'
      AND type NOT IN ('join_request', 'invitation')
      AND related_id = OLD.id;
END;

CREATE TRIGGER delete_event_notifications
AFTER DELETE ON events
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'events'
      AND related_id = OLD.id;
END;

PRAGMA foreign_keys = ON;
