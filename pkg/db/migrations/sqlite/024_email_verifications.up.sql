DROP TABLE IF EXISTS email_verifications;

CREATE TABLE email_verifications (
    email TEXT PRIMARY KEY,
    code TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    verified INTEGER NOT NULL DEFAULT 0,
    token TEXT NOT NULL DEFAULT '',
    sent_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
