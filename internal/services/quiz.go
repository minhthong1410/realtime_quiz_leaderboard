package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"realtime_leaderboard/internal/database"
	"realtime_leaderboard/internal/models"
)

type QuizService struct {
	db    *database.DB
	redis *redis.Client
}

func NewQuizService(db *database.DB, redis *redis.Client) *QuizService {
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
		// Clear all leaderboard caches for this quiz
		if keys, err := s.redis.Keys(ctx, fmt.Sprintf("quiz:%s:leaderboard:*", quizID)).Result(); err == nil {
			for _, key := range keys {
				s.redis.Del(ctx, key)
			}
		}

		// Get and cache the updated leaderboard (first page)
		_, err = s.GetLeaderboard(quizID, 1, 10)
		return err
	}
	return nil
}

type PaginatedLeaderboard struct {
	Leaderboard []models.LeaderboardEntry `json:"leaderboard"`
	TotalCount  int                       `json:"total_count"`
	Page        int                       `json:"page"`
	PageSize    int                       `json:"page_size"`
}

func (s *QuizService) GetLeaderboard(quizID string, page, pageSize int) (*PaginatedLeaderboard, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("quiz:%s:leaderboard:%d:%d", quizID, page, pageSize)

	// Get from Redis
	val, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var result PaginatedLeaderboard
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return &result, nil
		}
	}

	// Fetch from database if not in Redis
	leaderboard, totalCount, err := s.db.GetLeaderboard(ctx, quizID, page, pageSize)
	if err != nil {
		return nil, err
	}

	result := &PaginatedLeaderboard{
		Leaderboard: leaderboard,
		TotalCount:  totalCount,
		Page:        page,
		PageSize:    pageSize,
	}

	// Cache in Redis
	jsonData, _ := json.Marshal(result)
	s.redis.Set(ctx, cacheKey, jsonData, 0)

	return result, nil
}

type QuizServiceInterface interface {
	ProcessAnswer(quizID, userID, questionID, answer string) error
	GetLeaderboard(quizID string, page int, pageSize int) (PaginatedLeaderboard, error)
}
