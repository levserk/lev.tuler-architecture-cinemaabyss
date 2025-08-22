package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
)

type EventsService struct {
	kafkaBrokers string
	writers      map[string]*kafka.Writer
	readers      map[string]*kafka.Reader
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
		writers:      make(map[string]*kafka.Writer),
		readers:      make(map[string]*kafka.Reader),
	}

	service.initKafka()
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

	if err := s.publishEvent("movie-events", event); err != nil {
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

	if err := s.publishEvent("user-events", event); err != nil {
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

	if err := s.publishEvent("payment-events", event); err != nil {
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

func (s *EventsService) initKafka() {
	topics := []string{"movie-events", "user-events", "payment-events"}
	
	for _, topic := range topics {
		writer := &kafka.Writer{
			Addr:     kafka.TCP(s.kafkaBrokers),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
		s.writers[topic] = writer
		
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{s.kafkaBrokers},
			Topic:   topic,
			GroupID: "events-service-group",
		})
		s.readers[topic] = reader
		
		log.Printf("✅ Kafka writer and reader initialized for topic: %s", topic)
	}
}

func (s *EventsService) publishEvent(topic string, event interface{}) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	writer, exists := s.writers[topic]
	if !exists {
		log.Printf("❌ Writer not found for topic: %s", topic)
		return fmt.Errorf("writer not found for topic: %s", topic)
	}

	message := kafka.Message{
		Key:   []byte(topic),
		Value: eventData,
		Time:  time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, message)
	if err != nil {
		log.Printf("❌ Failed to write message to topic %s: %v", topic, err)
		return err
	}

	log.Printf("📢 Event published to topic '%s': %s", topic, string(eventData))
	return nil
}

func (s *EventsService) startConsumers() {
	topics := []string{"movie-events", "user-events", "payment-events"}
	
	log.Printf("🎧 Starting consumers for topics: %v", topics)
	log.Printf("🔗 Kafka brokers: %s", s.kafkaBrokers)
	
	for _, topic := range topics {
		go s.consumeTopic(topic)
	}
}

func (s *EventsService) consumeTopic(topic string) {
	reader, exists := s.readers[topic]
	if !exists {
		log.Printf("❌ Reader not found for topic: %s", topic)
		return
	}
	
	log.Printf("🎧 Consumer started for topic: %s", topic)
	
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		message, err := reader.ReadMessage(ctx)
		cancel()
		
		if err != nil {
			if err == context.DeadlineExceeded {
				log.Printf("🔄 Consumer for topic '%s' is running (waiting for messages...)", topic)
				continue
			}
			log.Printf("❌ Error reading message from topic %s: %v", topic, err)
			time.Sleep(5 * time.Second)
			continue
		}
		
		log.Printf("📨 Message consumed from topic '%s': %s", topic, string(message.Value))
		
		switch topic {
		case "movie-events":
			s.processMovieEvent(message.Value)
		case "user-events":
			s.processUserEvent(message.Value)
		case "payment-events":
			s.processPaymentEvent(message.Value)
		}
	}
}

func (s *EventsService) processMovieEvent(data []byte) {
	var event MovieEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("❌ Error unmarshaling movie event: %v", err)
		return
	}
	log.Printf("🎬 Processing movie event: %+v", event)
}

func (s *EventsService) processUserEvent(data []byte) {
	var event UserEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("❌ Error unmarshaling user event: %v", err)
		return
	}
	log.Printf("👤 Processing user event: %+v", event)
}

func (s *EventsService) processPaymentEvent(data []byte) {
	var event PaymentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("❌ Error unmarshaling payment event: %v", err)
		return
	}
	log.Printf("💳 Processing payment event: %+v", event)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
