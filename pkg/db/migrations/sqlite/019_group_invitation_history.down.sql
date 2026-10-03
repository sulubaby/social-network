DROP TRIGGER IF EXISTS delete_group_notifications;
DROP TRIGGER IF EXISTS delete_group_invitation_notifications;
DROP INDEX IF EXISTS group_invitations_one_pending;

UPDATE notifications
SET related_id = (
    SELECT group_id
    FROM group_invitations
    WHERE group_invitations.id = notifications.related_id
)
WHERE category = 'groups'
  AND type = 'invitation'
  AND EXISTS (
      SELECT 1
      FROM group_invitations
      WHERE group_invitations.id = notifications.related_id
  );

CREATE TABLE group_invitations_old (
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    inviter_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES user(id) ON DELETE CASCADE
);

INSERT INTO group_invitations_old (group_id, user_id, inviter_id, status, created_at)
SELECT gi.group_id, gi.user_id, gi.inviter_id, gi.status, gi.created_at
FROM group_invitations gi
WHERE gi.id = (
    SELECT MAX(gi2.id)
    FROM group_invitations gi2
    WHERE gi2.group_id = gi.group_id
      AND gi2.user_id = gi.user_id
);

DROP TABLE group_invitations;
ALTER TABLE group_invitations_old RENAME TO group_invitations;

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
      AND type <> 'join_request'
      AND related_id = OLD.id;
END;
