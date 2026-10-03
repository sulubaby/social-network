DROP TRIGGER IF EXISTS delete_group_notifications;

CREATE TRIGGER delete_group_notifications
BEFORE DELETE ON groups
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND type = 'join_request'
      AND related_id IN (
          SELECT id
          FROM group_join_requests
          WHERE group_id = OLD.id
      );

    DELETE FROM notifications
    WHERE category = 'groups'
      AND type <> 'join_request'
      AND related_id = OLD.id;
END;
