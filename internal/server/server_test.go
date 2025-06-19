package server

import (
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"realtime_leaderboard/internal/models"
	"strings"
	"testing"
)

type mockQuizService struct {
	leaderboard []models.LeaderboardEntry
}

func (m *mockQuizService) ProcessAnswer(quizID, userID, questionID, answer string) error {
	return nil
}

func (m *mockQuizService) GetLeaderboard(quizID string) ([]models.LeaderboardEntry, error) {
	return m.leaderboard, nil
}

func TestHandleWebSocket(t *testing.T) {
	// Mock quiz service
	quizService := &mockQuizService{
		leaderboard: []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}},
	}
	server := NewServer(quizService)
	// Create test server
	s := httptest.NewServer(server.Router)
	defer s.Close()

	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?quiz_id=quiz1&user_id=user1"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer c.Close()

	// Read initial leaderboard
	var leaderboard []models.LeaderboardEntry
	err = c.ReadJSON(&leaderboard)
	assert.NoError(t, err)
	assert.Len(t, leaderboard, 1)
	assert.Equal(t, "user1", leaderboard[0].UserID)
}
