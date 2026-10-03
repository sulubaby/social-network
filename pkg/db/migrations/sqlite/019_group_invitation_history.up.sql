DROP TRIGGER IF EXISTS delete_group_notifications;

CREATE TABLE group_invitations_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    inviter_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES user(id) ON DELETE CASCADE
);

INSERT INTO group_invitations_new (group_id, user_id, inviter_id, status, created_at)
SELECT group_id, user_id, inviter_id, status, created_at
FROM group_invitations;

UPDATE notifications
SET related_id = (
    SELECT gi.id
    FROM group_invitations_new gi
    WHERE gi.group_id = notifications.related_id
      AND gi.user_id = notifications.user_id
)
WHERE category = 'groups'
  AND type = 'invitation'
  AND EXISTS (
      SELECT 1
      FROM group_invitations_new gi
      WHERE gi.group_id = notifications.related_id
        AND gi.user_id = notifications.user_id
  );

DROP TABLE group_invitations;
ALTER TABLE group_invitations_new RENAME TO group_invitations;

CREATE UNIQUE INDEX group_invitations_one_pending
ON group_invitations(group_id, user_id)
WHERE status = 'pending';

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
