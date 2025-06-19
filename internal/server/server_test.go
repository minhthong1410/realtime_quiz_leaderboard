package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"realtime_leaderboard/internal/models"
	"realtime_leaderboard/internal/services"
)

type mockQuizService struct {
	leaderboard []models.LeaderboardEntry
}

func (m *mockQuizService) ProcessAnswer(quizID, userID, questionID, answer string) error {
	if answer == "Soap" && len(m.leaderboard) > 0 {
		m.leaderboard[0].Score++
	}
	return nil
}

func (m *mockQuizService) GetLeaderboard(quizID string, page, pageSize int) (services.PaginatedLeaderboard, error) {
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > len(m.leaderboard) {
		end = len(m.leaderboard)
	}
	paged := m.leaderboard
	if start < len(m.leaderboard) {
		paged = m.leaderboard[start:end]
	} else {
		paged = []models.LeaderboardEntry{}
	}
	return services.PaginatedLeaderboard{
		Leaderboard: paged,
		TotalCount:  len(m.leaderboard),
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func TestHandleWebSocket(t *testing.T) {
	quizService := &mockQuizService{
		leaderboard: []models.LeaderboardEntry{{UserID: "user1", Username: "Alice", Score: 1}},
	}
	server := NewServer(quizService)

	s := httptest.NewServer(server.Router)
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?quiz_id=quiz1&user_id=user1"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer ws.Close()

	_, msg, err := ws.ReadMessage()
	assert.NoError(t, err)
	var received services.PaginatedLeaderboard
	assert.NoError(t, json.Unmarshal(msg, &received))
	assert.Len(t, received.Leaderboard, 1)
	assert.Equal(t, "user1", received.Leaderboard[0].UserID)

	answer := struct {
		QuestionID string `json:"question_id"`
		Answer     string `json:"answer"`
	}{QuestionID: "q1", Answer: "Soap"}
	ws.WriteJSON(answer)

	_, msg, err = ws.ReadMessage()
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(msg, &received))
	assert.Len(t, received.Leaderboard, 1)
	assert.Equal(t, 2, received.Leaderboard[0].Score)
}

func TestHandleGetLeaderboard(t *testing.T) {
	quizService := &mockQuizService{
		leaderboard: []models.LeaderboardEntry{
			{UserID: "user1", Username: "Alice", Score: 10},
			{UserID: "user2", Username: "Bob", Score: 5},
			{UserID: "user3", Username: "Charlie", Score: 3},
		},
	}
	server := NewServer(quizService)

	s := httptest.NewServer(server.Router)
	defer s.Close()

	// Test page 1, page_size 2
	req, err := http.NewRequest("GET", s.URL+"/leaderboard?quiz_id=quiz1&page=1&page_size=2", nil)
	assert.NoError(t, err)
	resp := httptest.NewRecorder()
	server.Router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var result services.PaginatedLeaderboard
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Leaderboard, 2)
	assert.Equal(t, "user1", result.Leaderboard[0].UserID)
	assert.Equal(t, 10, result.Leaderboard[0].Score)
	assert.Equal(t, "user2", result.Leaderboard[1].UserID)
	assert.Equal(t, 5, result.Leaderboard[1].Score)
	assert.Equal(t, 3, result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PageSize)

	// Test page 2, page_size 2
	req, err = http.NewRequest("GET", s.URL+"/leaderboard?quiz_id=quiz1&page=2&page_size=2", nil)
	assert.NoError(t, err)
	resp = httptest.NewRecorder()
	server.Router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Len(t, result.Leaderboard, 1)
	assert.Equal(t, "user3", result.Leaderboard[0].UserID)
	assert.Equal(t, 3, result.Leaderboard[0].Score)
	assert.Equal(t, 3, result.TotalCount)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 2, result.PageSize)

	// Test missing quiz_id
	req, err = http.NewRequest("GET", s.URL+"/leaderboard", nil)
	assert.NoError(t, err)
	resp = httptest.NewRecorder()
	server.Router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Missing quiz_id")

	// Test invalid page
	req, err = http.NewRequest("GET", s.URL+"/leaderboard?quiz_id=quiz1&page=-1&page_size=2", nil)
	assert.NoError(t, err)
	resp = httptest.NewRecorder()
	server.Router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 1, result.Page)

	// Test invalid page_size
	req, err = http.NewRequest("GET", s.URL+"/leaderboard?quiz_id=quiz1&page=1&page_size=200", nil)
	assert.NoError(t, err)
	resp = httptest.NewRecorder()
	server.Router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, 10, result.PageSize)
}
