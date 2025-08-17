package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ProxyConfig struct {
	Port                   string
	MonolithURL           string
	MoviesServiceURL      string
	EventsServiceURL      string
	GradualMigration      bool
	MoviesMigrationPercent int
}

func main() {
	config := loadConfig()
	
	log.Printf("Starting Proxy Service on port %s", config.Port)
	log.Printf("Monolith URL: %s", config.MonolithURL)
	log.Printf("Movies Service URL: %s", config.MoviesServiceURL)
	log.Printf("Events Service URL: %s", config.EventsServiceURL)
	log.Printf("Gradual Migration: %t", config.GradualMigration)
	log.Printf("Movies Migration Percent: %d%%", config.MoviesMigrationPercent)

	router := mux.NewRouter()
	
	// Health check endpoint
	router.HandleFunc("/health", healthHandler).Methods("GET")
	
	// Movies API - applies Strangler Fig pattern
	router.PathPrefix("/api/movies").HandlerFunc(moviesProxyHandler(config))
	
	// Events API - direct proxy to events service
	router.PathPrefix("/api/events").HandlerFunc(eventsProxyHandler(config))
	
	// All other API calls go to monolith
	router.PathPrefix("/api/").HandlerFunc(monolithProxyHandler(config))
	
	// Default handler for non-API routes
	router.PathPrefix("/").HandlerFunc(monolithProxyHandler(config))

	log.Printf("Proxy service listening on :%s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, router))
}

func loadConfig() *ProxyConfig {
	config := &ProxyConfig{
		Port:                   getEnv("PORT", "8000"),
		MonolithURL:           getEnv("MONOLITH_URL", "http://localhost:8080"),
		MoviesServiceURL:      getEnv("MOVIES_SERVICE_URL", "http://localhost:8081"),
		EventsServiceURL:      getEnv("EVENTS_SERVICE_URL", "http://localhost:8082"),
		GradualMigration:      getEnv("GRADUAL_MIGRATION", "true") == "true",
		MoviesMigrationPercent: getEnvInt("MOVIES_MIGRATION_PERCENT", 50),
	}
	
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "proxy"}`))
}

// Strangler Fig implementation for movies API
func moviesProxyHandler(config *ProxyConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var targetURL string
		
		if config.GradualMigration {
			// Generate random number 1-100
			randomPercent := rand.Intn(100) + 1
			
			if randomPercent <= config.MoviesMigrationPercent {
				// Route to new Movies Service
				targetURL = config.MoviesServiceURL
				log.Printf("Routing to Movies Service: %s %s (random: %d, threshold: %d)", 
					r.Method, r.URL.Path, randomPercent, config.MoviesMigrationPercent)
			} else {
				// Route to Monolith
				targetURL = config.MonolithURL
				log.Printf("Routing to Monolith: %s %s (random: %d, threshold: %d)", 
					r.Method, r.URL.Path, randomPercent, config.MoviesMigrationPercent)
			}
		} else {
			// If gradual migration is disabled, route everything to Movies Service
			targetURL = config.MoviesServiceURL
			log.Printf("Routing to Movies Service (migration disabled): %s %s", r.Method, r.URL.Path)
		}
		
		proxyRequest(w, r, targetURL)
	}
}

func eventsProxyHandler(config *ProxyConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Routing to Events Service: %s %s", r.Method, r.URL.Path)
		proxyRequest(w, r, config.EventsServiceURL)
	}
}

func monolithProxyHandler(config *ProxyConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Routing to Monolith: %s %s", r.Method, r.URL.Path)
		proxyRequest(w, r, config.MonolithURL)
	}
}

func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string) {
	// Parse target URL
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Error parsing target URL %s: %v", targetURL, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
		
		// Add proxy headers
		req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Proxy-Service", "cinemaabyss-proxy")
	}
	
	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for %s %s to %s: %v", r.Method, r.URL.Path, targetURL, err)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
	
	// Modify response
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Add custom headers to indicate which service handled the request
		if strings.Contains(targetURL, "8081") {
			resp.Header.Set("X-Served-By", "movies-service")
		} else if strings.Contains(targetURL, "8082") {
			resp.Header.Set("X-Served-By", "events-service")
		} else {
			resp.Header.Set("X-Served-By", "monolith")
		}
		resp.Header.Set("X-Proxy", "cinemaabyss-proxy")
		return nil
	}

	// Execute proxy request
	proxy.ServeHTTP(w, r)
}

func init() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
}
