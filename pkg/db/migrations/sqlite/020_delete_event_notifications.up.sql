CREATE TRIGGER delete_event_notifications
AFTER DELETE ON events
FOR EACH ROW
BEGIN
    DELETE FROM notifications
    WHERE category = 'events'
      AND related_id = OLD.id;
END;
