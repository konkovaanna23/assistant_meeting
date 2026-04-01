CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    chat_id BIGINT,
    username TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users_audio (
    id uuid PRIMARY KEY,
    user_id BIGINT,
    path varchar,
    file_id uuid,
    task_id uuid,
    status varchar,
    result json,
    result_text text,
    result_short text,
    created_at TIMESTAMP DEFAULT NOW()
    updated_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS users_audio_words (
    id uuid,
    word varchar,
    created_at TIMESTAMP DEFAULT NOW()
);