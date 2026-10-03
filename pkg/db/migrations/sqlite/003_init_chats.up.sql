CREATE TABLE IF NOT EXISTS groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(45) NOT NULL,
    description VARCHAR(500) NOT NULL
);

CREATE TABLE IF NOT EXISTS chats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(45) NOT NULL,
    group_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    num_of_members INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (group_id)
        REFERENCES groups(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    sender_id INTEGER NOT NULL,
    chat_id INTEGER NOT NULL,

    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (sender_id)
        REFERENCES user(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (chat_id)
        REFERENCES chats(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);


CREATE TABLE IF NOT EXISTS chat_users (
    user_id INTEGER NOT NULL,
    chat_id INTEGER NOT NULL,
    is_owner INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (user_id, chat_id),

    FOREIGN KEY (user_id)
        REFERENCES user(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (chat_id)
        REFERENCES chats(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    title VARCHAR(50) NOT NULL,
    content VARCHAR(500) NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    group_id INTEGER NOT NULL,
    creator_id INTEGER NOT NULL,

    FOREIGN KEY (group_id)
        REFERENCES groups(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (creator_id)
        REFERENCES user(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS votes_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    option VARCHAR(45) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS event_votes (
    event_id INTEGER NOT NULL,
    vote_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,

    PRIMARY KEY (event_id, user_id),

    FOREIGN KEY (event_id)
        REFERENCES events(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (vote_id)
        REFERENCES votes_options(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,

    FOREIGN KEY (user_id)
        REFERENCES user(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);


