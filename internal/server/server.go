package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"realtime_leaderboard/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	Router      *mux.Router
	quizService services.QuizServiceInterface
	clients     map[string]map[*websocket.Conn]bool
	mutex       sync.Mutex
}

func NewServer(quizService services.QuizServiceInterface) *Server {
	s := &Server{
		Router:      mux.NewRouter(),
		quizService: quizService,
		clients:     make(map[string]map[*websocket.Conn]bool),
	}
	s.Router.HandleFunc("/ws", s.handleWebSocket)
	return s
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	quizID := r.URL.Query().Get("quiz_id")
	userID := r.URL.Query().Get("user_id")
	if quizID == "" || userID == "" {
		http.Error(w, "Missing quiz_id or user_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	s.mutex.Lock()
	if s.clients[quizID] == nil {
		s.clients[quizID] = make(map[*websocket.Conn]bool)
	}
	s.clients[quizID][conn] = true
	s.mutex.Unlock()

	// Send initial leaderboard
	leaderboard, err := s.quizService.GetLeaderboard(quizID)
	if err != nil {
		log.Println(err)
		return
	}
	if err := conn.WriteJSON(leaderboard); err != nil {
		log.Println(err)
		return
	}

	for {
		var msg struct {
			QuestionID string `json:"question_id"`
			Answer     string `json:"answer"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			s.mutex.Lock()
			delete(s.clients[quizID], conn)
			s.mutex.Unlock()
			return
		}

		if err := s.quizService.ProcessAnswer(quizID, userID, msg.QuestionID, msg.Answer); err != nil {
			log.Println(err)
			continue
		}

		updatedLeaderboard, err := s.quizService.GetLeaderboard(quizID)
		if err != nil {
			log.Println(err)
			continue
		}

		s.mutex.Lock()
		for client := range s.clients[quizID] {
			if err := client.WriteJSON(updatedLeaderboard); err != nil {
				log.Println(err)
				delete(s.clients[quizID], client)
				client.Close()
			}
		}
		s.mutex.Unlock()
	}
}
