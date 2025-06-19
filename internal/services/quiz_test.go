package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"realtime_leaderboard/internal/database"
	"realtime_leaderboard/internal/models"
)

func TestProcessAnswer_CorrectAnswer(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	redisClient, redisMock := redismock.NewClientMock()
	s := NewQuizService(&database.DB{db}, redisClient)
	context.Background()

	// Mock GetQuestion
	rows := sqlmock.NewRows([]string{"id", "quiz_id", "question_text", "options", "correct_answer"}).
		AddRow("q1", "quiz1", "What cleans best?", "{Water,Soap}", "Soap")
	mock.ExpectQuery(`SELECT id, quiz_id, question_text, options, correct_answer FROM questions WHERE id = \$1`).
		WithArgs("q1").
		WillReturnRows(rows)

	// Mock UpdateUserScore
	mock.ExpectExec(`INSERT INTO user_scores`).
		WithArgs("quiz1", "user1", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock COUNT(*) query for GetLeaderboard
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_scores WHERE quiz_id = \$1`).
		WithArgs("quiz1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Mock paginated leaderboard query
	leaderboardRows := sqlmock.NewRows([]string{"id", "username", "score"}).
		AddRow("user1", "Alice", 1)
	mock.ExpectQuery(`SELECT u\.id, u\.username, us\.score FROM user_scores us`).
		WithArgs("quiz1", 10, 0). // Default page=1, page_size=10
		WillReturnRows(leaderboardRows)

	// Mock Redis cache invalidation
	redisMock.ExpectKeys("quiz:quiz1:leaderboard:*").SetVal([]string{"quiz:quiz1:leaderboard:1:10"})
	redisMock.ExpectDel("quiz:quiz1:leaderboard:1:10").SetVal(1)

	// Mock Redis SET for new leaderboard
	result := PaginatedLeaderboard{
		Leaderboard: []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}},
		TotalCount:  1,
		Page:        1,
		PageSize:    10,
	}
	jsonData, _ := json.Marshal(result)
	redisMock.ExpectSet("quiz:quiz1:leaderboard:1:10", jsonData, 0).SetVal("OK")

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
	result := PaginatedLeaderboard{
		Leaderboard: leaderboard,
		TotalCount:  1,
		Page:        1,
		PageSize:    10,
	}
	jsonData, _ := json.Marshal(result)
	redisMock.ExpectGet("quiz:quiz1:leaderboard:1:10").SetVal(string(jsonData))

	got, err := s.GetLeaderboard("quiz1", 1, 10)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, result, *got)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestGetLeaderboard_FromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	redisClient, redisMock := redismock.NewClientMock()
	s := NewQuizService(&database.DB{db}, redisClient)
	context.Background()

	redisMock.ExpectGet("quiz:quiz1:leaderboard:1:2").RedisNil()

	// Mock total count
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_scores WHERE quiz_id = \$1`).
		WithArgs("quiz1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	// Mock paginated leaderboard
	rows := sqlmock.NewRows([]string{"id", "username", "score"}).
		AddRow("user1", "Alice", 1)
	mock.ExpectQuery(`SELECT u\.id, u\.username, us\.score FROM user_scores us`).
		WithArgs("quiz1", 2, 0).
		WillReturnRows(rows)

	result := PaginatedLeaderboard{
		Leaderboard: []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}},
		TotalCount:  4,
		Page:        1,
		PageSize:    2,
	}
	jsonData, _ := json.Marshal(result)
	redisMock.ExpectSet("quiz:quiz1:leaderboard:1:2", jsonData, 0).SetVal("OK")

	got, err := s.GetLeaderboard("quiz1", 1, 2)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, result, *got)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
