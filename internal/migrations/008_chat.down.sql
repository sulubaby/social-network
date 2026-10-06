DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS groups_users;

DROP TRIGGER IF EXISTS trg_groups_users_before;
DROP TRIGGER IF EXISTS trg_user_group_exists;