CREATE TABLE IF NOT EXISTS quizzes (
                         id VARCHAR(50) PRIMARY KEY,
                         title TEXT,
                         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS questions (
                           id VARCHAR(50) PRIMARY KEY,
                           quiz_id VARCHAR(50) REFERENCES quizzes(id),
                           question_text TEXT,
                           options TEXT[],
                           correct_answer TEXT
);

CREATE TABLE IF NOT EXISTS users (
                       id VARCHAR(50) PRIMARY KEY,
                       username VARCHAR(100) UNIQUE
);

CREATE TABLE IF NOT EXISTS user_scores (
                             quiz_id VARCHAR(50),
                             user_id VARCHAR(50),
                             score INTEGER DEFAULT 0,
                             PRIMARY KEY (quiz_id, user_id),
                             FOREIGN KEY (quiz_id) REFERENCES quizzes(id),
                             FOREIGN KEY (user_id) REFERENCES users(id)
);