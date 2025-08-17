package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type EventsService struct {
	kafkaBrokers string
}

type MovieEvent struct {
	MovieID int     `json:"movie_id"`
	Title   string  `json:"title"`
	Action  string  `json:"action"`
	UserID  *int    `json:"user_id,omitempty"`
	Rating  *float64 `json:"rating,omitempty"`
}

type UserEvent struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Action   string `json:"action"`
}

type PaymentEvent struct {
	PaymentID int     `json:"payment_id"`
	UserID    int     `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
}

type EventResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	
	service := &EventsService{
		kafkaBrokers: brokers,
	}

	go service.startConsumers()

	router := mux.NewRouter()

	router.HandleFunc("/api/events/health", service.healthCheck).Methods("GET")
	router.HandleFunc("/api/events/movie", service.createMovieEvent).Methods("POST")
	router.HandleFunc("/api/events/user", service.createUserEvent).Methods("POST")
	router.HandleFunc("/api/events/payment", service.createPaymentEvent).Methods("POST")

	port := getEnv("PORT", "8082")
	log.Printf("Events Service starting on port %s", port)
	log.Printf("Kafka brokers: %v", brokers)
	
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func (s *EventsService) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func (s *EventsService) createMovieEvent(w http.ResponseWriter, r *http.Request) {
	var event MovieEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.publishEvent("movies", event); err != nil {
		log.Printf("Failed to publish movie event: %v", err)
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	log.Printf("Movie event published: %+v", event)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(EventResponse{
		Status:  "success",
		Message: "Movie event created successfully",
	})
}

func (s *EventsService) createUserEvent(w http.ResponseWriter, r *http.Request) {
	var event UserEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.publishEvent("users", event); err != nil {
		log.Printf("Failed to publish user event: %v", err)
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	log.Printf("User event published: %+v", event)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(EventResponse{
		Status:  "success",
		Message: "User event created successfully",
	})
}

func (s *EventsService) createPaymentEvent(w http.ResponseWriter, r *http.Request) {
	var event PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.publishEvent("payments", event); err != nil {
		log.Printf("Failed to publish payment event: %v", err)
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment event published: %+v", event)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(EventResponse{
		Status:  "success",
		Message: "Payment event created successfully",
	})
}

func (s *EventsService) publishEvent(topic string, event interface{}) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	log.Printf("📢 Event published to topic '%s' (simulated): %s", topic, string(eventData))
	log.Printf("🎯 Kafka brokers: %s", s.kafkaBrokers)
	
	return nil
}

func (s *EventsService) startConsumers() {
	topics := []string{"movies", "users", "payments"}
	
	log.Printf("🎧 Starting consumers for topics: %v", topics)
	log.Printf("🔗 Kafka brokers: %s", s.kafkaBrokers)
	
	for _, topic := range topics {
		go s.consumeTopic(topic)
	}
}

func (s *EventsService) consumeTopic(topic string) {
	log.Printf("🎧 Consumer started for topic: %s (simulated)", topic)
	
	for {
		select {
		case <-time.After(5 * time.Second):
			log.Printf("🔄 Consumer for topic '%s' is running (waiting for messages...)", topic)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
