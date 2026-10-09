-- logging out puts the token here so it can not be used again, even if
-- someone kept a copy of the cookie. rows are cleaned once the token expires
CREATE TABLE IF NOT EXISTS revoked_tokens (
    token_hash TEXT PRIMARY KEY,
    expires_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens (expires_at);
