CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    chat_id BIGINT,
    username TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);