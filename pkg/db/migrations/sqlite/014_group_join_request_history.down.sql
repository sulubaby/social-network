DROP TRIGGER IF EXISTS delete_group_join_request_notifications;

CREATE TABLE group_join_requests_old (
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',

    PRIMARY KEY (group_id, user_id),

    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

INSERT INTO group_join_requests_old (group_id, user_id, status)
SELECT group_id, user_id, status
FROM group_join_requests
WHERE id IN (
    SELECT MAX(id)
    FROM group_join_requests
    GROUP BY group_id, user_id
);

UPDATE notifications
SET related_id = (
    SELECT group_join_requests.group_id
    FROM group_join_requests
    WHERE group_join_requests.id = notifications.related_id
)
WHERE category = 'groups'
  AND type = 'join_request'
  AND EXISTS (
      SELECT 1
      FROM group_join_requests
      WHERE group_join_requests.id = notifications.related_id
  );

DROP TABLE group_join_requests;

ALTER TABLE group_join_requests_old
RENAME TO group_join_requests;
