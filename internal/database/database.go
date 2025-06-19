package database

import (
	"context"
	"database/sql"
	"realtime_leaderboard/internal/models"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(connStr string) (*DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) GetQuestion(ctx context.Context, questionID string) (*models.Question, error) {
	q := &models.Question{}
	err := db.QueryRowContext(ctx, "SELECT id, quiz_id, question_text, options, correct_answer FROM questions WHERE id = $1", questionID).
		Scan(&q.ID, &q.QuizID, &q.QuestionText, &q.Options, &q.CorrectAnswer)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (db *DB) UpdateUserScore(ctx context.Context, quizID, userID string, increment int) error {
	_, err := db.ExecContext(ctx, `
        INSERT INTO user_scores (quiz_id, user_id, score)
        VALUES ($1, $2, $3)
        ON CONFLICT (quiz_id, user_id)
        DO UPDATE SET score = user_scores.score + $3
    `, quizID, userID, increment)
	return err
}

func (db *DB) GetLeaderboard(ctx context.Context, quizID string, page, pageSize int) ([]models.LeaderboardEntry, int, error) {
	offset := (page - 1) * pageSize

	var totalCount int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_scores WHERE quiz_id = $1", quizID).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT u.id, u.username, us.score
		FROM user_scores us
		JOIN users u ON us.user_id = u.id
		WHERE us.quiz_id = $1
		ORDER BY us.score DESC
		LIMIT $2 OFFSET $3
	`, quizID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var leaderboard []models.LeaderboardEntry
	for rows.Next() {
		var e models.LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.Score); err != nil {
			return nil, 0, err
		}
		leaderboard = append(leaderboard, e)
	}

	return leaderboard, totalCount, nil
}
