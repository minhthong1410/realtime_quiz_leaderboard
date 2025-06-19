package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Quiz struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type Question struct {
	ID            string         `json:"id"`
	QuizID        string         `json:"quiz_id"`
	QuestionText  string         `json:"question_text"`
	Options       pq.StringArray `json:"options" db:"options"`
	CorrectAnswer string         `json:"correct_answer"`
}

// Custom marshal/unmarshal for Options to appear as []string in JSON
func (q *Question) MarshalJSON() ([]byte, error) {
	type Alias Question
	return json.Marshal(&struct {
		Options []string `json:"options"`
		*Alias
	}{
		Options: []string(q.Options),
		Alias:   (*Alias)(q),
	})
}

func (q *Question) UnmarshalJSON(data []byte) error {
	type Alias Question
	aux := &struct {
		Options []string `json:"options"`
		*Alias
	}{
		Alias: (*Alias)(q),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	q.Options = pq.StringArray(aux.Options)
	return nil
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type UserScore struct {
	QuizID string `json:"quiz_id"`
	UserID string `json:"user_id"`
	Score  int    `json:"score"`
}

type LeaderboardEntry struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Score    int    `json:"score"`
}
