CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    age INTEGER,
    gender VARCHAR(50),
    about_me TEXT,
    games TEXT[] DEFAULT '{}'
);