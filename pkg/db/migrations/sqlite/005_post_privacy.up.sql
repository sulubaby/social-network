PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS increase_post_count;
DROP TRIGGER IF EXISTS decrease_post_count;
DROP TRIGGER IF EXISTS increase_comment_count;
DROP TRIGGER IF EXISTS decrease_comment_count;
DROP TRIGGER IF EXISTS increase_reaction_like;
DROP TRIGGER IF EXISTS increase_reaction_dislike;
DROP TRIGGER IF EXISTS decrease_reaction_like;
DROP TRIGGER IF EXISTS decrease_reaction_dislike;

CREATE TABLE posts_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(45) NOT NULL DEFAULT 'post',
    title VARCHAR(50) NOT NULL DEFAULT '',
    content VARCHAR(500) NOT NULL,
    image_path TEXT NOT NULL DEFAULT '',
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    group_id INTEGER,
    privacy VARCHAR(20) NOT NULL DEFAULT 'public'
        CHECK (privacy IN ('public', 'followers', 'selected')),
    like_count INTEGER NOT NULL DEFAULT 0,
    dislike_count INTEGER NOT NULL DEFAULT 0,
    comment_count INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (user_id)
        REFERENCES user(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (group_id)
        REFERENCES groups(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

INSERT INTO posts_new (
    id,
    type,
    title,
    content,
    image_path,
    user_id,
    created_at,
    group_id,
    privacy,
    like_count,
    dislike_count,
    comment_count
)
SELECT
    id,
    type,
    title,
    content,
    image_path,
    user_id,
    created_at,
    group_id,
    'public',
    like_count,
    dislike_count,
    comment_count
FROM posts;

DROP TABLE posts;
ALTER TABLE posts_new RENAME TO posts;

CREATE TABLE IF NOT EXISTS post_viewers (
    post_id INTEGER NOT NULL,
    viewer_id INTEGER NOT NULL,

    PRIMARY KEY (post_id, viewer_id),

    FOREIGN KEY (post_id)
        REFERENCES posts(id)
        ON DELETE CASCADE,

    FOREIGN KEY (viewer_id)
        REFERENCES user(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts (user_id);
CREATE INDEX IF NOT EXISTS idx_posts_group_id ON posts (group_id);
CREATE INDEX IF NOT EXISTS idx_posts_privacy ON posts (privacy);
CREATE INDEX IF NOT EXISTS idx_post_viewers_viewer_id ON post_viewers (viewer_id);

CREATE TRIGGER increase_post_count
AFTER INSERT ON posts
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_posts = num_of_posts + 1
    WHERE user_id = NEW.user_id;
END;

CREATE TRIGGER decrease_post_count
AFTER DELETE ON posts
FOR EACH ROW
BEGIN
    UPDATE profile
    SET num_of_posts = num_of_posts - 1
    WHERE user_id = OLD.user_id;
END;

CREATE TRIGGER increase_comment_count
AFTER INSERT ON comments
FOR EACH ROW
BEGIN
    UPDATE posts
    SET comment_count = comment_count + 1
    WHERE id = NEW.post_id;
END;

CREATE TRIGGER decrease_comment_count
AFTER DELETE ON comments
FOR EACH ROW
BEGIN
    UPDATE posts
    SET comment_count = comment_count - 1
    WHERE id = OLD.post_id;
END;

CREATE TRIGGER increase_reaction_like
AFTER INSERT ON post_reactions
FOR EACH ROW
WHEN NEW.value = 1
BEGIN
    UPDATE posts
    SET like_count = like_count + 1
    WHERE id = NEW.post_id;
END;

CREATE TRIGGER increase_reaction_dislike
AFTER INSERT ON post_reactions
FOR EACH ROW
WHEN NEW.value = -1
BEGIN
    UPDATE posts
    SET dislike_count = dislike_count + 1
    WHERE id = NEW.post_id;
END;

CREATE TRIGGER decrease_reaction_like
AFTER DELETE ON post_reactions
FOR EACH ROW
WHEN OLD.value = 1
BEGIN
    UPDATE posts
    SET like_count = like_count - 1
    WHERE id = OLD.post_id;
END;

CREATE TRIGGER decrease_reaction_dislike
AFTER DELETE ON post_reactions
FOR EACH ROW
WHEN OLD.value = -1
BEGIN
    UPDATE posts
    SET dislike_count = dislike_count - 1
    WHERE id = OLD.post_id;
END;

PRAGMA foreign_keys = ON;
