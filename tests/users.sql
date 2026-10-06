-- Seeds 30 users and makes all of them follow user_id = 3
-- Run against your SQLite db, e.g.: sqlite3 app.db < seed_followers.sql

BEGIN TRANSACTION;

-- 1. Create 30 users (seeduser1 .. seeduser30)
--    NOTE: 'password' is a placeholder string, NOT a valid bcrypt hash.
--    These users won't be able to log in until you replace it with a real
--    hash from your auth package (e.g. bcrypt.GenerateFromPassword).
WITH RECURSIVE seq(n) AS (
    SELECT 1
    UNION ALL
    SELECT n + 1 FROM seq WHERE n < 30
)
INSERT INTO user (email, first_name, last_name, password, username, dob)
SELECT
    'seeduser' || n || '@example.com',
    'First' || n,
    'Last' || n,
    'CHANGE_ME_NOT_A_REAL_HASH',
    'seeduser' || n,
    date('1995-01-01', '+' || n || ' days')
FROM seq;

-- 2. Give each new user a profile row (no trigger creates this automatically,
--    unlike user_about which has trg_create_about)
INSERT INTO profile (user_id, num_of_followers, num_of_following, num_of_posts)
SELECT id, 0, 0, 0
FROM user
WHERE username LIKE 'seeduser%'
  AND id NOT IN (SELECT user_id FROM profile);

-- Make sure target user (id = 3) has a profile row too
INSERT OR IGNORE INTO profile (user_id)
VALUES (3);

-- 3. Make every seeded user follow user_id = 3
--    status = 1 assumed to mean "accepted/following" - adjust if your
--    app uses a different convention (e.g. 0 = pending, 1 = accepted)
INSERT INTO user_followers (follower_id, target_id, status)
SELECT id, 3, 1
FROM user
WHERE username LIKE 'seeduser%'
  AND id NOT IN (
      SELECT follower_id FROM user_followers WHERE target_id = 3
  );

-- 4. Recompute counters so profile.num_of_followers / num_of_following stay accurate
UPDATE profile
SET num_of_followers = (
    SELECT COUNT(*) FROM user_followers
    WHERE target_id = profile.user_id AND status = 1
)
WHERE user_id = 3;

UPDATE profile
SET num_of_following = (
    SELECT COUNT(*) FROM user_followers
    WHERE follower_id = profile.user_id AND status = 1
)
WHERE user_id IN (SELECT id FROM user WHERE username LIKE 'seeduser%');

COMMIT;