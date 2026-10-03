DROP TRIGGER IF EXISTS delete_group_notifications;

CREATE TRIGGER delete_group_notifications
AFTER DELETE ON groups
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'groups'
      AND related_id = OLD.id;
END;
