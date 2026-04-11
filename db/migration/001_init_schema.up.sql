CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT        NOT NULL UNIQUE,
    full_name  TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT        NOT NULL,
    user_agent    TEXT        NOT NULL DEFAULT '',
    client_ip     TEXT        NOT NULL DEFAULT '',
    is_blocked    BOOLEAN     NOT NULL DEFAULT false,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions(user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS tags (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    color      TEXT        NOT NULL DEFAULT '#6366f1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS tags_user_id_idx ON tags(user_id);

CREATE TYPE todo_status AS ENUM ('pending', 'in_progress', 'done');
CREATE TYPE todo_priority AS ENUM ('low', 'medium', 'high');

CREATE TABLE IF NOT EXISTS todos (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT          NOT NULL,
    description TEXT          NOT NULL DEFAULT '',
    status      todo_status   NOT NULL DEFAULT 'pending',
    priority    todo_priority NOT NULL DEFAULT 'medium',
    due_date    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS todos_user_id_idx     ON todos(user_id);
CREATE INDEX IF NOT EXISTS todos_status_idx      ON todos(status);
CREATE INDEX IF NOT EXISTS todos_user_status_idx ON todos(user_id, status);

CREATE TABLE IF NOT EXISTS todo_tags (
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (todo_id, tag_id)
);

CREATE INDEX IF NOT EXISTS todo_tags_tag_id_idx ON todo_tags(tag_id);
