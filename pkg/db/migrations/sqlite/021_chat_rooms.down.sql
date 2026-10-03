DROP INDEX IF EXISTS messages_chat_created;
DROP INDEX IF EXISTS chats_one_private_room_per_pair;
DROP INDEX IF EXISTS chats_one_room_per_group;

ALTER TABLE chats DROP COLUMN private_user_high_id;
ALTER TABLE chats DROP COLUMN private_user_low_id;
