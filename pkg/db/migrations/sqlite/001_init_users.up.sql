CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(75) NOT NULL UNIQUE,
    first_name VARCHAR(15) NOT NULL,
    last_name VARCHAR(15) NOT NULL,
    password TEXT NOT NULL,
    username VARCHAR(12) UNIQUE,
    dob DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS profile (
    user_id INTEGER PRIMARY KEY,
    num_of_followers INTEGER NOT NULL DEFAULT 0,
    num_of_following INTEGER NOT NULL DEFAULT 0,
    num_of_posts INTEGER NOT NULL DEFAULT 0,
    avatar_path TEXT,
    about VARCHAR(1000),
    is_private INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_about (
    user_id INTEGER PRIMARY KEY,
    work VARCHAR(200),
    hobbies VARCHAR(200),
    education VARCHAR(200),
    intrests VARCHAR(200),
    travel VARCHAR(200),
    website VARCHAR(200),
    linkedin VARCHAR(200),
    instgram VARCHAR(200),
    twitter VARCHAR(200), 
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_followers (
    follower_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    status INTEGER NOT NULL,
    PRIMARY KEY (follower_id, target_id)
);

CREATE TRIGGER IF NOT EXISTS trg_create_about
AFTER INSERT ON user
FOR EACH ROW
BEGIN
    INSERT INTO user_about (user_id,work,hobbies,education,intrests,travel, website, linkedin, instgram, twitter)
    VALUES (NEW.id, '','','','','','','','','');
END;

CREATE TRIGGER IF NOT EXISTS trg_create_profile 
AFTER INSERT ON user 
FOR EACH ROW 
BEGIN
    INSERT INTO profile (user_id)
    VALUES (NEW.id);
END;

CREATE TRIGGER IF NOT EXISTS trg_avatar_default
AFTER INSERT ON profile
FOR EACH ROW
WHEN NEW.avatar_path IS NULL OR NEW.avatar_path = ''
BEGIN
    UPDATE profile
    SET avatar_path = 'avatars/default.png'
    WHERE user_id = NEW.user_id;
END;
