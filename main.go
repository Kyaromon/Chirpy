package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/Kyaromon/Chirpy/handlers"
	"github.com/Kyaromon/Chirpy/internal/database"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable not set")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM environment variable not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	fileserverHits := &atomic.Int32{}

	polkaAPIKey := os.Getenv("POLKA_KEY")
	if polkaAPIKey == "" {
		log.Fatal("POLKA_KEY environment variable not set")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/healthz", handlers.HandlerReadiness)

	userCfg := &handlers.UserConfig{
		DB:        dbQueries,
		JWTSecret: jwtSecret,
	}
	mux.HandleFunc("/api/users", userCfg.HandlerUsers)

	authCfg := &handlers.AuthConfig{
		DB:        dbQueries,
		JWTSecret: jwtSecret,
	}
	mux.HandleFunc("/api/login", authCfg.HandlerLogin)
	mux.HandleFunc("/api/refresh", authCfg.HandlerRefresh)
	mux.HandleFunc("/api/revoke", authCfg.HandlerRevoke)

	chirpCfg := &handlers.ChirpConfig{
		DB:        dbQueries,
		JWTSecret: jwtSecret,
	}
	mux.HandleFunc("/api/chirps", chirpCfg.HandlerChirps)
	mux.HandleFunc("/api/chirps/{chirpID}", chirpCfg.HandlerChirps)

	polkaCfg := &handlers.PolkaConfig{
		DB:     dbQueries,
		APIKey: polkaAPIKey,
	}
	mux.HandleFunc("/api/polka/webhooks", polkaCfg.HandlerPolkaWebhooks)

	adminCfg := &handlers.AdminConfig{
		DB:             dbQueries,
		Platform:       platform,
		FileserverHits: fileserverHits,
	}
	mux.HandleFunc("/admin/metrics", adminCfg.HandlerAdminMetrics)
	mux.HandleFunc("/admin/reset", adminCfg.HandlerAdminReset)

	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", middlewareMetricsInc(fileserverHits, http.StripPrefix("/app/", fileServer)))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}

func middlewareMetricsInc(hits *atomic.Int32, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		next.ServeHTTP(w, r)
	})
}