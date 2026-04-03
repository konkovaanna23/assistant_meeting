CREATE SCHEMA IF NOT EXISTS recognition AUTHORIZATION current_user;
   

GRANT USAGE, CREATE ON SCHEMA recognition  TO current_user;

CREATE TABLE IF NOT EXISTS recognition.users (
    id BIGINT PRIMARY KEY,
    chat_id BIGINT,
    username TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recognition.users_audio (
    id uuid PRIMARY KEY,
    user_id BIGINT,
    path varchar,
    is_voice bool,
    file_id uuid,
    task_id uuid,
    status varchar,
    result json,
    result_text text,
    result_short text,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS recognition.users_audio_words (
    id uuid,
    word varchar,
    created_at TIMESTAMP DEFAULT NOW()
);