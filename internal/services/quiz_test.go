package services

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"realtime_leaderboard/internal/database"
	"realtime_leaderboard/internal/models"
	"testing"
)

func TestProcessAnswer_CorrectAnswer(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	redisClient, redisMock := redismock.NewClientMock()
	s := NewQuizService(&database.DB{db}, redisClient)

	// Mock GetQuestion
	rows := sqlmock.NewRows([]string{"id", "quiz_id", "question_text", "options", "correct_answer"}).
		AddRow("q1", "quiz1", "What cleans best?", "{Water,Soap}", "Soap")
	mock.ExpectQuery("SELECT id, quiz_id, question_text, options, correct_answer FROM questions WHERE id = \\$1").
		WithArgs("q1").
		WillReturnRows(rows)

	// Mock UpdateUserScore
	mock.ExpectExec("INSERT INTO user_scores").
		WithArgs("quiz1", "user1", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock GetLeaderboard
	leaderboardRows := sqlmock.NewRows([]string{"id", "username", "score"}).
		AddRow("user1", "Alice", 1)
	mock.ExpectQuery("SELECT u.id, u.username, us.score FROM user_scores us").
		WithArgs("quiz1").
		WillReturnRows(leaderboardRows)

	// Mock Redis Set
	leaderboard := []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}}
	jsonData, _ := json.Marshal(leaderboard)
	redisMock.ExpectSet("quiz:quiz1:leaderboard", jsonData, 0).SetVal("OK")

	err = s.ProcessAnswer("quiz1", "user1", "q1", "Soap")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestGetLeaderboard_FromRedis(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	redisClient, redisMock := redismock.NewClientMock()
	s := NewQuizService(&database.DB{db}, redisClient)
	context.Background()

	leaderboard := []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}}
	jsonData, _ := json.Marshal(leaderboard)
	redisMock.ExpectGet("quiz:quiz1:leaderboard").SetVal(string(jsonData))

	result, err := s.GetLeaderboard("quiz1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "user1", result[0].UserID)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestGetLeaderboard_FromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	redisClient, redisMock := redismock.NewClientMock()
	s := NewQuizService(&database.DB{db}, redisClient)
	context.Background()

	redisMock.ExpectGet("quiz:quiz1:leaderboard").RedisNil()

	rows := sqlmock.NewRows([]string{"id", "username", "score"}).
		AddRow("user1", "Alice", 1)
	mock.ExpectQuery("SELECT u.id, u.username, us.score FROM user_scores us").
		WithArgs("quiz1").
		WillReturnRows(rows)

	leaderboard := []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}}
	jsonData, _ := json.Marshal(leaderboard)
	redisMock.ExpectSet("quiz:quiz1:leaderboard", jsonData, 0).SetVal("OK")

	result, err := s.GetLeaderboard("quiz1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "user1", result[0].UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
