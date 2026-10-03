ALTER TABLE chats
ADD COLUMN private_user_low_id INTEGER REFERENCES user(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE chats
ADD COLUMN private_user_high_id INTEGER REFERENCES user(id) ON DELETE CASCADE ON UPDATE CASCADE;

UPDATE chats
SET private_user_low_id = (
        SELECT MIN(cu.user_id)
        FROM chat_users cu
        WHERE cu.chat_id = chats.id
    ),
    private_user_high_id = (
        SELECT MAX(cu.user_id)
        FROM chat_users cu
        WHERE cu.chat_id = chats.id
    )
WHERE type = 'private'
  AND group_id IS NULL
  AND (SELECT COUNT(*) FROM chat_users cu WHERE cu.chat_id = chats.id) = 2;

CREATE TEMP TABLE chat_room_merge (
    old_id INTEGER PRIMARY KEY,
    keep_id INTEGER NOT NULL
);

INSERT INTO chat_room_merge (old_id, keep_id)
SELECT c.id, (
    SELECT MIN(other.id)
    FROM chats other
    WHERE other.type = 'group'
      AND other.group_id = c.group_id
)
FROM chats c
WHERE c.type = 'group'
  AND c.group_id IS NOT NULL
  AND c.id <> (
      SELECT MIN(other.id)
      FROM chats other
      WHERE other.type = 'group'
        AND other.group_id = c.group_id
  );

INSERT OR IGNORE INTO chat_users (user_id, chat_id, is_owner)
SELECT cu.user_id, merge.keep_id, cu.is_owner
FROM chat_users cu
JOIN chat_room_merge merge ON merge.old_id = cu.chat_id;

UPDATE chat_users
SET is_owner = 1
WHERE EXISTS (
    SELECT 1
    FROM chat_room_merge merge
    JOIN chat_users old_member
      ON old_member.chat_id = merge.old_id
     AND old_member.user_id = chat_users.user_id
     AND old_member.is_owner = 1
    WHERE merge.keep_id = chat_users.chat_id
);

UPDATE messages
SET chat_id = (
    SELECT merge.keep_id
    FROM chat_room_merge merge
    WHERE merge.old_id = messages.chat_id
)
WHERE EXISTS (
    SELECT 1 FROM chat_room_merge merge WHERE merge.old_id = messages.chat_id
);

DELETE FROM chats
WHERE id IN (SELECT old_id FROM chat_room_merge);

DELETE FROM chat_room_merge;

INSERT INTO chat_room_merge (old_id, keep_id)
SELECT c.id, (
    SELECT MIN(other.id)
    FROM chats other
    WHERE other.type = 'private'
      AND other.private_user_low_id = c.private_user_low_id
      AND other.private_user_high_id = c.private_user_high_id
)
FROM chats c
WHERE c.type = 'private'
  AND c.private_user_low_id IS NOT NULL
  AND c.private_user_high_id IS NOT NULL
  AND c.id <> (
      SELECT MIN(other.id)
      FROM chats other
      WHERE other.type = 'private'
        AND other.private_user_low_id = c.private_user_low_id
        AND other.private_user_high_id = c.private_user_high_id
  );

INSERT OR IGNORE INTO chat_users (user_id, chat_id, is_owner)
SELECT cu.user_id, merge.keep_id, cu.is_owner
FROM chat_users cu
JOIN chat_room_merge merge ON merge.old_id = cu.chat_id;

UPDATE messages
SET chat_id = (
    SELECT merge.keep_id
    FROM chat_room_merge merge
    WHERE merge.old_id = messages.chat_id
)
WHERE EXISTS (
    SELECT 1 FROM chat_room_merge merge WHERE merge.old_id = messages.chat_id
);

DELETE FROM chats
WHERE id IN (SELECT old_id FROM chat_room_merge);

DROP TABLE chat_room_merge;

CREATE UNIQUE INDEX chats_one_room_per_group
ON chats(group_id)
WHERE type = 'group' AND group_id IS NOT NULL;

CREATE UNIQUE INDEX chats_one_private_room_per_pair
ON chats(private_user_low_id, private_user_high_id)
WHERE type = 'private'
  AND private_user_low_id IS NOT NULL
  AND private_user_high_id IS NOT NULL;

CREATE INDEX messages_chat_created
ON messages(chat_id, created_at, id);
