package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	clients     map[string]map[*websocket.Conn]bool // quizID -> clients
	mutex       sync.Mutex
}

func NewServer(quizService services.QuizServiceInterface) *Server {
	s := &Server{
		Router:      mux.NewRouter(),
		quizService: quizService,
		clients:     make(map[string]map[*websocket.Conn]bool),
	}
	s.Router.HandleFunc("/ws", s.handleWebSocket)
	s.Router.HandleFunc("/leaderboard", s.handleGetLeaderboard).Methods("GET")
	return s
}

func (s *Server) handleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	quizID := r.URL.Query().Get("quiz_id")
	if quizID == "" {
		http.Error(w, "Missing quiz_id", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Printf("Invalid page number: %v", err)
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	if page < 1 {
		http.Error(w, "Page should be greater than 0", http.StatusBadRequest)
		return
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		log.Printf("Invalid page size number: %v", err)
		http.Error(w, "Invalid page size", http.StatusBadRequest)
		return
	}
	if pageSize < 1 || pageSize > 100 {
		http.Error(w, "Page size should be between 1 and 100", http.StatusBadRequest)
		return
	}

	leaderboard, err := s.quizService.GetLeaderboard(quizID, page, pageSize)
	if err != nil {
		log.Printf("Error fetching leaderboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(leaderboard); err != nil {
		log.Printf("Error encoding leaderboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
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
	leaderboard, err := s.quizService.GetLeaderboard(quizID, 1, 1000) // Large page size to get all
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

		updatedLeaderboard, err := s.quizService.GetLeaderboard(quizID, 1, 1000) // Large page size
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
