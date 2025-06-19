package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-redis/redis/v8"
	"realtime_leaderboard/internal/database"
	"realtime_leaderboard/internal/models"
)

//go:generate mockgen -destination=quiz_mock.go -package=services . QuizServiceInterface

type QuizServiceInterface interface {
	ProcessAnswer(quizID, userID, questionID, answer string) error
	GetLeaderboard(quizID string) ([]models.LeaderboardEntry, error)
}

type QuizService struct {
	db    *database.DB
	redis *redis.Client
}

func NewQuizService(db *database.DB, redis *redis.Client) QuizServiceInterface {
	return &QuizService{db, redis}
}

func (s *QuizService) ProcessAnswer(quizID, userID, questionID, answer string) error {
	ctx := context.Background()
	question, err := s.db.GetQuestion(ctx, questionID)
	if err != nil {
		return err
	}
	if question.CorrectAnswer == answer {
		if err := s.db.UpdateUserScore(ctx, quizID, userID, 1); err != nil {
			return err
		}
		leaderboard, err := s.db.GetLeaderboard(ctx, quizID)
		if err != nil {
			return err
		}
		jsonData, _ := json.Marshal(leaderboard)
		if err := s.redis.Set(ctx, "quiz:"+quizID+":leaderboard", jsonData, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (s *QuizService) GetLeaderboard(quizID string) ([]models.LeaderboardEntry, error) {
	ctx := context.Background()
	val, err := s.redis.Get(ctx, "quiz:"+quizID+":leaderboard").Result()
	if errors.Is(err, redis.Nil) {
		leaderboard, err := s.db.GetLeaderboard(ctx, quizID)
		if err != nil {
			return nil, err
		}
		jsonData, _ := json.Marshal(leaderboard)
		s.redis.Set(ctx, "quiz:"+quizID+":leaderboard", jsonData, 0)
		return leaderboard, nil
	} else if err != nil {
		return nil, err
	}
	var leaderboard []models.LeaderboardEntry
	if err := json.Unmarshal([]byte(val), &leaderboard); err != nil {
		return nil, err
	}
	return leaderboard, nil
}
