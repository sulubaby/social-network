CREATE TABLE group_join_requests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',

    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

INSERT INTO group_join_requests_new (group_id, user_id, status)
SELECT group_id, user_id, status
FROM group_join_requests;

UPDATE notifications
SET related_id = (
    SELECT group_join_requests_new.id
    FROM group_join_requests_new
    WHERE group_join_requests_new.group_id = notifications.related_id
      AND group_join_requests_new.user_id = notifications.actor_id
)
WHERE category = 'groups'
  AND type = 'join_request'
  AND EXISTS (
      SELECT 1
      FROM group_join_requests_new
      WHERE group_join_requests_new.group_id = notifications.related_id
        AND group_join_requests_new.user_id = notifications.actor_id
  );

DROP TABLE group_join_requests;

ALTER TABLE group_join_requests_new
RENAME TO group_join_requests;

CREATE UNIQUE INDEX group_join_requests_one_pending
ON group_join_requests (group_id, user_id)
WHERE status = 'pending';

CREATE TRIGGER IF NOT EXISTS delete_group_join_request_notifications
AFTER DELETE ON group_join_requests
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'join_request'
      AND related_id = OLD.id;
END;
