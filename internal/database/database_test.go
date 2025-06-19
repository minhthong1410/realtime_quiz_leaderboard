package database

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDB(t *testing.T) {
	db, err := NewDB("invalid://connection")
	assert.Error(t, err, "should fail with invalid connection string")
	assert.Nil(t, db, "db should be nil on error")
}

func TestGetQuestion(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	d := &DB{db}
	ctx := context.Background()
	questionID := "q1"

	// Mock the options column as a PostgreSQL array string
	rows := sqlmock.NewRows([]string{"id", "quiz_id", "question_text", "options", "correct_answer"}).
		AddRow("q1", "quiz1", "What cleans best?", "{Water,Soap}", "Soap")

	// Escape $1 in the query regex to match PostgreSQL placeholder
	mock.ExpectQuery(`SELECT id, quiz_id, question_text, options, correct_answer FROM questions WHERE id = \$1`).
		WithArgs(questionID).
		WillReturnRows(rows)

	// Execute the query
	q, err := d.GetQuestion(ctx, questionID)
	assert.NoError(t, err, "should not return an error")
	assert.NotNil(t, q, "question should not be nil")
	if q != nil {
		assert.Equal(t, "q1", q.ID, "question ID should match")
		assert.Equal(t, "quiz1", q.QuizID, "quiz ID should match")
		assert.Equal(t, "What cleans best?", q.QuestionText, "question text should match")
		assert.ElementsMatch(t, []string{"Water", "Soap"}, []string(q.Options), "options should match")
		assert.Equal(t, "Soap", q.CorrectAnswer, "correct answer should match")
	}

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "all mock expectations should be met")
}

func TestUpdateUserScore(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	d := &DB{db}
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO user_scores").
		WithArgs("quiz1", "user1", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = d.UpdateUserScore(ctx, "quiz1", "user1", 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLeaderboard(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	d := &DB{db}
	ctx := context.Background()
	quizID := "quiz1"

	rows := sqlmock.NewRows([]string{"id", "username", "score"}).
		AddRow("user1", "Alice", 10).
		AddRow("user2", "Bob", 5)

	mock.ExpectQuery("SELECT u.id, u.username, us.score FROM user_scores us").
		WithArgs(quizID).
		WillReturnRows(rows)

	leaderboard, err := d.GetLeaderboard(ctx, quizID)
	assert.NoError(t, err)
	assert.Len(t, leaderboard, 2)
	assert.Equal(t, "user1", leaderboard[0].UserID)
	assert.Equal(t, "Alice", leaderboard[0].Username)
	assert.Equal(t, 10, leaderboard[0].Score)
	assert.NoError(t, mock.ExpectationsWereMet())
}
